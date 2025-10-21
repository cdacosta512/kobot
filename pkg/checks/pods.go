package checks

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/briandowns/spinner"
	"gitlab.com/kobot/kobot/pkg/logging"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RunPodCheck performs a health check for all pods across all namespaces.
func RunPodCheck(clientset *kubernetes.Clientset) {
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Prefix = "\n Starting operator-initiated health check "
	fmt.Print("")
	s.Start()

	ctx := context.Background()
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
	if err != nil {
		logging.Error("Unable to list namespaces: %v", err)
		return
	}

	time.Sleep(5 * time.Second) // grace period for pods still starting
	s.Stop()
	fmt.Println("\n Starting operator-initiated health check")

	var totalNamespaces int
	var totalPods int
	var failedNamespaces int
	failingMap := make(map[string]int) // ns -> count of bad pods

	for _, ns := range namespaces.Items {
		totalNamespaces++
		fmt.Printf("=== RUN   %s\n", ns.Name)

		pods, err := clientset.CoreV1().Pods(ns.Name).List(ctx, v1.ListOptions{})
		if err != nil {
			color.Red("--- FAIL: %s (error listing pods: %v)\n", ns.Name, err)
			failedNamespaces++
			continue
		}

		totalPods += len(pods.Items)
		var nonRunning []string

		for _, pod := range pods.Items {
			if pod.Status.Phase != "Running" {
				nonRunning = append(nonRunning, fmt.Sprintf("%s (%s)", pod.Name, pod.Status.Phase))
			}
		}

		if len(nonRunning) > 0 {
			color.Red("--- FAIL: %s (%d pods not running)\n", ns.Name, len(nonRunning))
			for _, p := range nonRunning {
				fmt.Printf("        Pod %s\n", p)
			}
			failedNamespaces++
			failingMap[ns.Name] = len(nonRunning)
		} else {
			color.Green("--- PASS: %s (%d pods running)\n", ns.Name, len(pods.Items))
		}
	}

	// Summary section
	fmt.Println()
	logging.Title("Kobot Health Check Summary\n")
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
}
