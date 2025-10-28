package checks

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"gitlab.com/kobot/kobot/pkg/logging"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PodFinding represents detailed findings for a single pod.
type PodFinding struct {
	PodName string
	Issues  []string
}

// RunPodDeepCheck performs a deep concurrent inspection of pods and containers.
// It handles API throttling gracefully with exponential backoff and retry logic.
func RunPodDeepCheck(clientset *kubernetes.Clientset, namespaces []string, htmlOutput bool) {
	ctx := context.Background()
	fmt.Println()

	// Discover namespaces if none provided
	if len(namespaces) == 0 || (len(namespaces) == 1 && namespaces[0] == "") {
		logging.Warn("No namespaces specified — checking all namespaces. A deep pod health scan across all namespaces could take slightly longer than a quick pod health scan.")
		logging.Info("Throttling for deep pod health scans are in place by default.")
		nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			logging.Error("Unable to list namespaces: %v", err)
			return
		}
		namespaces = make([]string, len(nsList.Items))
		for i, ns := range nsList.Items {
			namespaces[i] = ns.Name
		}
	}

	logging.Info("Performing deep pod health scan across %d namespace(s).", len(namespaces))
	logging.Starting("Operator-initiated deep pod readiness check")
	fmt.Println()

	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 4) // slightly lower concurrency to reduce throttling

	var totalNamespaces, totalPods, failedNamespaces int
	failingMap := make(map[string]int)
	var results []PodCheckResult

	wg.Add(len(namespaces))
	for _, ns := range namespaces {
		ns := ns
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			nsCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			mu.Lock()
			totalNamespaces++
			mu.Unlock()

			// --- Fetch pods with retry + backoff for throttling
			var pods *v1.PodList
			var err error
			maxRetries := 3

			for attempt := 1; attempt <= maxRetries; attempt++ {
				pods, err = clientset.CoreV1().Pods(ns).List(nsCtx, metav1.ListOptions{})
				if err == nil {
					break
				}

				if apierrors.IsTooManyRequests(err) || strings.Contains(strings.ToLower(err.Error()), "throttl") {
					backoff := time.Duration(rand.Intn(500)+500*attempt) * time.Millisecond
					mu.Lock()
					fmt.Printf("%s Scan job on namespace: %s ... %s (attempt %d/%d, retrying in %s)\n",
						color.YellowString("THROTTLED"),
						ns,
						color.YellowString("client-side API rate limit hit"),
						attempt,
						maxRetries,
						backoff)
					mu.Unlock()
					time.Sleep(backoff)
					continue
				}

				if errors.Is(err, context.DeadlineExceeded) {
					mu.Lock()
					fmt.Printf("%s Scan job on namespace: %s ... %s (attempt %d/%d)\n",
						color.YellowString("TIMEOUT"),
						ns,
						color.YellowString("API slow or busy"),
						attempt,
						maxRetries)
					mu.Unlock()
					time.Sleep(1 * time.Second)
					continue
				}

				// Non-retryable error
				mu.Lock()
				fmt.Printf("%s Scan job on namespace: %s ... %s (%v)\n",
					color.RedString("ERROR"), ns,
					color.RedString("Unable to list pods"), err)
				failedNamespaces++
				mu.Unlock()
				return
			}

			if err != nil {
				mu.Lock()
				fmt.Printf("%s Scan job on namespace: %s ... %s (exceeded retry limit)\n",
					color.RedString("ERROR"), ns,
					color.RedString("Pod listing failed after retries"))
				failedNamespaces++
				mu.Unlock()
				return
			}

			// --- Skip namespaces with no pods
			if len(pods.Items) == 0 {
				mu.Lock()
				fmt.Printf("%s Scan job on namespace: %s ... %s\n",
					color.HiBlackString("RUNNING    "), ns,
					color.HiBlackString("SKIP (no pods found)"))
				mu.Unlock()
				return
			}

			localPods := len(pods.Items)
			var podFindings []PodFinding

			// --- Analyze each pod
			for _, pod := range pods.Items {
				podName := pod.Name
				var issues []string

				// Skip completed pods
				if pod.Status.Phase == v1.PodSucceeded {
					continue
				}

				// Evicted or failed
				if pod.Status.Reason == "Evicted" || pod.Status.Phase == v1.PodFailed {
					issues = append(issues,
						fmt.Sprintf("Pod phase: %s (Reason: %s)", pod.Status.Phase, pod.Status.Reason))
				}

				// Pod conditions
				for _, cond := range pod.Status.Conditions {
					if cond.Type == v1.PodReady && cond.Status != v1.ConditionTrue {
						issues = append(issues, fmt.Sprintf("PodReady=False (%s)", cond.Reason))
					}
					if cond.Type == v1.PodScheduled && cond.Status != v1.ConditionTrue {
						issues = append(issues, fmt.Sprintf("NotScheduled (%s)", cond.Reason))
					}
				}

				// Init containers
				for _, init := range pod.Status.InitContainerStatuses {
					if init.State.Terminated != nil && init.State.Terminated.ExitCode != 0 {
						issues = append(issues,
							fmt.Sprintf("Init container %s failed (exit %d, reason=%s)",
								init.Name,
								init.State.Terminated.ExitCode,
								init.State.Terminated.Reason))
					}
				}

				// Main containers
				for _, c := range pod.Status.ContainerStatuses {
					name := c.Name
					state := c.State

					if state.Waiting != nil {
						reason := state.Waiting.Reason
						if strings.Contains(reason, "BackOff") || strings.Contains(reason, "Err") {
							issues = append(issues,
								fmt.Sprintf("Container %s waiting: %s", name, reason))
						}
					}

					if state.Terminated != nil && state.Terminated.ExitCode != 0 {
						issues = append(issues,
							fmt.Sprintf("Container %s terminated (exit %d, reason=%s)",
								name,
								state.Terminated.ExitCode,
								state.Terminated.Reason))
					}

					if !c.Ready {
						issues = append(issues, fmt.Sprintf("Container %s not ready", name))
					}

					if c.RestartCount > 0 {
						old := c.RestartCount
						time.Sleep(3 * time.Second)
						if c.RestartCount > old {
							issues = append(issues,
								fmt.Sprintf("Container %s restarted recently (%d restarts)", name, c.RestartCount))
						} else {
							issues = append(issues,
								fmt.Sprintf("Container %s has restarted %d time(s)", name, c.RestartCount))
						}
					}
				}

				if len(issues) > 0 {
					podFindings = append(podFindings, PodFinding{
						PodName: podName,
						Issues:  issues,
					})
				}
			}

			// --- Namespace summary
			resultMsg := ""
			if len(podFindings) > 0 {
				resultMsg = color.RedString("FAIL (%d pods unhealthy)", len(podFindings))
			} else {
				resultMsg = color.GreenString("PASS (%d pods healthy)", localPods)
			}

			mu.Lock()
			totalPods += localPods
			if len(podFindings) > 0 {
				failedNamespaces++
				failingMap[ns] = len(podFindings)
			}
			results = append(results, PodCheckResult{
				Name:        ns,
				PodsChecked: localPods,
				PodsFailed:  len(podFindings),
			})

			fmt.Printf("%s Scan job on namespace: %s ... %s\n",
				color.BlueString("RUNNING    "), ns, resultMsg)

			if len(podFindings) > 0 {
				for i, pf := range podFindings {
					prefix := "└──"
					if i < len(podFindings)-1 {
						prefix = "├──"
					}
					fmt.Printf("        %s %s\n", prefix, color.YellowString(pf.PodName))
					for _, issue := range pf.Issues {
						fmt.Printf("             ↳ %s\n", issue)
					}
				}
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// --- Summary
	fmt.Println()
	fmt.Println(strings.Repeat("=", 55))
	logging.Title("            Kobot Deep Pod Health Report\n")
	fmt.Println(strings.Repeat("=", 55))
	fmt.Println()

	fmt.Printf("Namespaces checked: %d\n", totalNamespaces)
	fmt.Printf("Total pods checked: %d\n\n", totalPods)

	if failedNamespaces > 0 {
		logging.Error("%d namespace(s) failed deep pod health check.\n", failedNamespaces)
		for ns, count := range failingMap {
			fmt.Printf("  - %s (%d unhealthy pods)\n", ns, count)
		}
		fmt.Println()
		logging.Warn("Operators should investigate failing containers, restarts, or scheduling issues.\n")
	} else {
		logging.Success("%d namespace(s) were scanned and reported healthy across pod, container, and condition levels.\n", totalNamespaces)
	}

	if htmlOutput {
		if err := WriteHTMLReport(results, totalPods, totalNamespaces, failedNamespaces); err != nil {
			logging.Error("Failed to write HTML report: %v", err)
		} else {
			logging.Success("HTML report saved as kobot-deep-report.html\n")
		}
	}
}
