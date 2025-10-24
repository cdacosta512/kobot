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
	// TODO -- add the package breakdowns into a doc so we can clean up this file and remove all these comments.
	"k8s.io/client-go/dynamic" // --- this allows us to create a non-standard kubernetes clientset to work with CRDs like helmreleases
	"k8s.io/apimachinery/pkg/runtime/schema" // --- this allows us to define the GVR for the custom resource we want to work with like the helmrelease resource
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // gives us access to global types and options like GET and List options to get and list resources in the cluster
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured" // allows the dynamic client to read and modify nested fields since that is unknown with CRDs -- not needed when working with standard k8s objects
)


// RunHelmReleaseCheck performs a health check on all helmreleases in one or more namespaces.
// For internal use right now, it will only check the bigbang namespace but we will make it flexible for users who dont need it hardcoded to the bigbang namesapce.
func RunHelmReleaseCheck(dynamicClient dynamic.Interface, namespace string, htmlOutput bool) {

	// ctx := context.Background()
	var namespaces []string

	// Get all namespaces if user didn't pass one
	if namespace != "" {
		namespaces = []string{namespace}
	} else {
		// Dynamic client doesn’t handle namespaces directly, so use corev1 API logic elsewhere
		// For now, we’ll assume helmreleases exist in bigbang or provided NS
		namespaces = []string{"bigbang"}
		logging.Info("Scanning only HelmRelease resources.")
		logging.Warn("No namespace specified — defaulting to bigbang.\n")
	}

	logging.Title("Starting operator-initiated HelmRelease readiness check..\n")
	fmt.Println("")
	time.Sleep(5 * time.Second)

	var totalNamespaces int
	var totalHelmReleases int
	var failedNamespaces int
	failingMap := make(map[string]int)
	// var results []HelmReleaseCheckResult

	helmReleaseGVR := schema.GroupVersionResource{
		Group:    "helm.toolkit.fluxcd.io",
		Version:  "v2",
		Resource: "helmreleases",
	}

	for _, ns := range namespaces {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		totalNamespaces++
		fmt.Printf("Scanning namespace ===>   %s\n", ns)
		fmt.Println("")

		releases, err := dynamicClient.Resource(helmReleaseGVR).Namespace(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				logging.Warn("Timeout listing HelmReleases in %s -- API slow or busy", ns)
			} else {
				logging.Error("Error occurred listing HelmReleases in %s: %v", ns, err)
			}
			color.Red("--- FAIL: %s (error listing HelmReleases)\n", ns)
			failedNamespaces++
			continue
		}

		totalHelmReleases += len(releases.Items)
		var nonReady []string

		for _, hr := range releases.Items {
			name := hr.GetName()
			ready, reason := isHelmReleaseReady(hr)
			if ready {
				color.Green("    PASS: %s", name)
			} else {
				color.Red("    FAIL: %s (Reason: %s)", name, reason)
				nonReady = append(nonReady, fmt.Sprintf("%s (Reason: %s)", name, reason))
			}
			fmt.Println()
		}

		// results = append(results, HelmReleaseCheckResult{
		// 	Name:                ns,
		// 	HelmReleasesChecked: len(releases.Items),
		// 	HelmReleasesFailed:  len(nonReady),
		// 	FailedReleases:      nonReady,
		// })

		if len(nonReady) > 0 {
			color.Red("--- FAIL: %s (%d HelmReleases not Ready)\n", ns, len(nonReady))
			for _, hr := range nonReady {
				fmt.Printf("        HelmRelease %s\n", hr)
			}
			failedNamespaces++
			failingMap[ns] = len(nonReady)
		} else {
			logging.Info("--- COMPLETE: %s (%d HelmReleases Ready)\n", ns, len(releases.Items))
		}
	}

	fmt.Println()
	logging.Title("Kobot HelmRelease Readiness Report\n")
	fmt.Printf("Namespaces checked: %d\n", totalNamespaces)
	fmt.Printf("Total HelmReleases checked: %d\n", totalHelmReleases)
	fmt.Println("")

	if failedNamespaces > 0 {
		logging.Error("%d namespace(s) failed HelmRelease readiness check.\n", failedNamespaces)
		fmt.Println("Failing namespaces:")
		for ns, count := range failingMap {
			fmt.Printf("  - %s (%d HelmReleases not Ready)\n", ns, count)
		}
		fmt.Println()
		logging.Warn("Operators should review the listed HelmReleases to resolve deployment issues or Helm upgrade failures.\n")
	} else {
		logging.Success("%d namespace(s) were scanned and all HelmReleases are Ready.\n", totalNamespaces)
	}

	// if htmlOutput {
	// 	if err := WriteHTMLReportHelm(results, totalHelmReleases, totalNamespaces, failedNamespaces); err != nil {
	// 		logging.Error("Failed to write HTML report: %v", err)
	// 	} else {
	// 		logging.Success("HTML report saved as kobot-helmrelease-report.html\n")
	// 	}
	// }
}

// isHelmReleaseReady checks the HelmRelease .status.conditions for Ready=True.
func isHelmReleaseReady(obj unstructured.Unstructured) (bool, string) {
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if !found || err != nil {
		return false, "no conditions found"
	}

	for _, c := range conditions {
		if cond, ok := c.(map[string]interface{}); ok {
			t, _, _ := unstructured.NestedString(cond, "type")
			s, _, _ := unstructured.NestedString(cond, "status")
			r, _, _ := unstructured.NestedString(cond, "reason")

			if t == "Ready" {
				return s == "True", r
			}
		}
	}
	return false, "Ready condition missing"
}