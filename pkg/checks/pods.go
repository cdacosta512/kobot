package checks

import (
	// standard packages
	"context"
	"errors"
	"fmt"
	"time"
	"strings"

	// non-standard or custom packages
	"github.com/fatih/color"                      // helps with the logging and nice colors
	"gitlab.com/kobot/kobot/pkg/logging"          // custom package I made so my logging could look a certain way
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // gives us access to global types and options like GET and List options to get and list resources in the cluster
	"k8s.io/client-go/kubernetes"                 // allows us to make a clientset to access different resources like corev1, appv1, batchv1 etc.
)

// RunPodCheck performs a health check for pods in one or more namespaces.
// CLI usage: kobot check cluster
// If no namespace is provided (-n, --namespace), it will check all namespaces.
func RunPodCheck(clientset *kubernetes.Clientset, namespaces []string, htmlOutput bool) {
	ctx := context.Background()

	// visual gap between the commmand in the first log message
	fmt.Println()

	// If no namespace flags were passed, list all namespaces
	if len(namespaces) == 0 || (len(namespaces) == 1 && namespaces[0] == "") {
		logging.Warn("No namespaces specified — checking all namespaces.")

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

	logging.Info("Scanning pod health across %d namespace(s).", len(namespaces))
	logging.Starting("Operator-initiated pod readiness check")
	time.Sleep(2 * time.Second)
	fmt.Println()
	time.Sleep(2 * time.Second) // short grace period for pods that are still starting

	var totalNamespaces, totalPods, failedNamespaces int
	failingMap := make(map[string]int) // ns -> failed pod count
	var results []PodCheckResult       // for HTML output

	// Iterate through all namespaces to check their pod health
	for _, ns := range namespaces {
		nsCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		totalNamespaces++
		logging.Running("Scan job on namespace: %s", ns)

		pods, err := clientset.CoreV1().Pods(ns).List(nsCtx, metav1.ListOptions{})
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				fmt.Printf("   %s %s\n", color.YellowString("WARN:"), fmt.Sprintf("Timeout listing pods in %s — API slow or busy", ns))
			} else {
				fmt.Printf("   %s %s\n", color.RedString("ERROR:"), fmt.Sprintf("Unable to list pods in %s: %v", ns, err))
			}
			failedNamespaces++
			continue
		}

		totalPods += len(pods.Items)
		var nonRunning []string

		for _, pod := range pods.Items {
			if pod.Status.Phase != "Running" && pod.Status.Phase != "Succeeded" {
				nonRunning = append(nonRunning, fmt.Sprintf("%s (%s)", pod.Name, pod.Status.Phase))
			}
		}

		results = append(results, PodCheckResult{
			Name:        ns,
			PodsChecked: len(pods.Items),
			PodsFailed:  len(nonRunning),
			FailedPods:  nonRunning,
		})

		if len(nonRunning) > 0 {
			fmt.Printf("   %s %s (%d pods not running)\n", color.RedString("FAIL:"), ns, len(nonRunning))
			for i, p := range nonRunning {
				prefix := "└──"
				if i < len(nonRunning)-1 {
					prefix = "├──"
				}
				fmt.Printf("        %s %s\n", prefix, color.YellowString(p))
			}
			failedNamespaces++
			failingMap[ns] = len(nonRunning)
		} else {
			fmt.Printf("   %s %s (%d pods running)\n", color.GreenString("PASS:"), ns, len(pods.Items))
		}
	}

	// --- Summary report
	fmt.Println()
	fmt.Println(strings.Repeat("=", 55))
	logging.Title("            Kobot Deep Pod Health Report\n")
	fmt.Println(strings.Repeat("=", 55))
	fmt.Println()

	fmt.Printf("Namespaces checked: %d\n", totalNamespaces)
	fmt.Printf("Total pods checked: %d\n\n", totalPods)

	if failedNamespaces > 0 {
		logging.Error("%d namespace(s) failed pod health check.\n", failedNamespaces)
		fmt.Println("Failing namespaces:")
		for ns, count := range failingMap {
			fmt.Printf("  - %s (%d pods not running)\n", ns, count)
		}
		fmt.Println()
		logging.Warn("Operators should perform a deeper analysis of those namespaces to ensure critical applications are operational.\n")
	} else {
		logging.Success("%d namespace(s) were scanned and reported healthy.\n", totalNamespaces)
	}

	// Generate HTML report if requested
	if htmlOutput {
		if err := WriteHTMLReport(results, totalPods, totalNamespaces, failedNamespaces); err != nil {
			logging.Error("Failed to write HTML report: %v", err)
		} else {
			logging.Success("HTML report saved as kobot-report.html\n")
		}
	}
}
