package checks

import (
	// standard packages
	"context"
	"fmt"
	"time"
	"errors"

	// non-standard or custom packages
	"github.com/fatih/color" // helps with the logging and nice colors
	"gitlab.com/kobot/kobot/pkg/logging" // custom package I made so my logging could look a certain way
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // gives us access to global types and options like GET and List options to get and list resources in the cluster
	"k8s.io/client-go/kubernetes" // allows us to make a clientset to access different resources like corev1, appv1, batchv1 etc.
)

// RunPodCheck performs a health check for pods in one or more namespaces.
// cli usage: kobot check cluster 
// if you dont pass it any namespace (-n, --namespace <namespace>), it will check all namespaces 
func RunPodCheck(clientset *kubernetes.Clientset, namespace string, htmlOutput bool) {

	// set a empty context for the namespace list operation
	ctx := context.Background()

	// holds the names of the namespaces that will be checked
	var namespaces []string

	// checks if the user passed in a namespace(s) and if not fallsback to getting all namespaces from the api
	if namespace != "" {
		namespaces = []string{namespace}
	} else {
		nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		logging.Warn("No namespace specified — checking all namespaces.\n")
		if err != nil {
			logging.Error("Unable to list namespaces: %v", err)
			return
		}
		for _, ns := range nsList.Items {
			namespaces = append(namespaces, ns.Name)
		}
	}

	logging.Title("Starting operator-initiated health check..\n")
	fmt.Println("")
	time.Sleep(5 * time.Second) // grace period for pods still starting

	var totalNamespaces int
	var totalPods int
	var failedNamespaces int
	failingMap := make(map[string]int) // ns -> keeps count of bad pods
	var results []PodCheckResult      // collect data for HTML output

	// looks through all the namespaces and reports if any have non running pods
	for _, ns := range namespaces {

		// context to timeout if the api calls hang (10s)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		totalNamespaces++
		fmt.Printf("Scanning namespace ===>   %s\n", ns)

		pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
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

		results = append(results, PodCheckResult{
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
		logging.Success("%d namespace(s) was scanned and reported healthy. Additional verification may be needed for other resources if kobot didnt scan it.\n", totalNamespaces)
	}

	// checks if the user set --html (true) and builds a html report like robot test
	if htmlOutput {
		if err := WriteHTMLReport(results, totalPods, totalNamespaces, failedNamespaces); err != nil {
			logging.Error("Failed to write HTML report: %v", err)
		} else {
			logging.Success("HTML report saved as kobot-report.html\n")
		}
	}
}
