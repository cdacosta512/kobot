package checks

import (
	"context"
	"fmt"
	"time"
	"errors"

	"github.com/fatih/color"
	"github.com/briandowns/spinner"
	"gitlab.com/kobot/kobot/pkg/logging"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RunPodCheck performs a health check for all pods across all namespaces.
func RunPodCheck(clientset *kubernetes.Clientset, namespace string, htmlOutput bool) {

	// set a empty context for the namespace search
	ctx := context.Background()

	// holds the names of the namespaces that will be checked
	var namespaces []string

	// checks if the user passed in a namespace(s) and if fallsback to getting all namespaces from the api
	if namespace != "" {
		namespaces = []string{namespace}
	} else {
		nsList, err := clientset.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
		logging.Warn("No namespace specified — checking all namespaces.\n")
		if err != nil {
			logging.Error("Unable to list namespaces: %v", err)
			return
		}
		for _, ns := range nsList.Items {
			namespaces = append(namespaces, ns.Name)
		}
	}

	// spinner animation so the user knows that something is happening when the check is starting
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Prefix = "\n Starting operator-initiated health check "
	fmt.Print("")
	s.Start()

	time.Sleep(5 * time.Second) // grace period for pods still starting
	s.Stop()
	fmt.Println("\n Starting operator-initiated health check:\n")

	var totalNamespaces int
	var totalPods int
	var failedNamespaces int
	failingMap := make(map[string]int) // ns -> count of bad pods
	var results []NamespaceResult      // collect data for HTML output

	// looks through all the namespaces and reports if any have non running pods
	for _, ns := range namespaces {

		// context to timeout if the api calls hang
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		totalNamespaces++
		fmt.Printf("=== RUN   %s\n", ns)

		pods, err := clientset.CoreV1().Pods(ns).List(ctx, v1.ListOptions{})
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				logging.Warn("Timeout list pods in %s -- API slow or busy", ns)
			} else {
				logging.Error("Error occurred listing pods in %s: %v, ns, err")
			}

			color.Red("--- FAIL: %s (error listing pods: %v)\n", ns, err)
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

		results = append(results, NamespaceResult{
			Name:        ns,
			PodsChecked: len(pods.Items),
			PodsFailed:  len(nonRunning),
			FailedPods:  nonRunning,
		})

		if len(nonRunning) > 0 {
			color.Red("--- FAIL: %s (%d pods not running)\n", ns, len(nonRunning))
			for _, p := range nonRunning {
				fmt.Printf("        Pod %s\n", p)
			}
			failedNamespaces++
			failingMap[ns] = len(nonRunning)
		} else {
			color.Green("--- PASS: %s (%d pods running)\n", ns, len(pods.Items))
		}
	}

	fmt.Println()
	logging.Title("Kobot Cluster Health Report\n")
	fmt.Printf("Namespaces checked: %d\n", totalNamespaces)
	fmt.Printf("Total pods checked: %d\n", totalPods)
	fmt.Println("")

	if failedNamespaces > 0 {
		logging.Error("%d namespace(s) failed pod health check.\n", failedNamespaces)
		fmt.Println("Failing namespaces:")
		for ns, count := range failingMap {
			fmt.Printf("  - %s (%d pods not running)\n", ns, count)
		}
		fmt.Println()
		logging.Warn("Operators should perform a deeper analysis of those namespaces to ensure critical applications are operational.\n")
	} else {
		logging.Success("All %d namespaces healthy\n", totalNamespaces)
	}

	// html report
	if htmlOutput {
		if err := WriteHTMLReport(results, totalPods, totalNamespaces, failedNamespaces); err != nil {
			logging.Error("Failed to write HTML report: %v", err)
		} else {
			logging.Success("HTML report saved as kobot-report.html\n")
			openReport()
		}
	}
}
