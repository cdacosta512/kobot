package checks

import (
	// standard packages
	"context"
	"errors"
	"fmt"
	"time"

	// non-standard or custom packages
	"github.com/fatih/color"             // helps with the logging and nice colors
	"gitlab.com/kobot/kobot/pkg/logging" // custom package I made so my logging could look a certain way
	// TODO -- add the package breakdowns into a doc so we can clean up this file and remove all these comments.
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"       // gives us access to global types and options like GET and List options to get and list resources in the cluster
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured" // allows the dynamic client to read and modify nested fields since that is unknown with CRDs -- not needed when working with standard k8s objects
	"k8s.io/apimachinery/pkg/runtime/schema"            // --- this allows us to define the GVR for the custom resource we want to work with like the helmrelease resource
	"k8s.io/client-go/dynamic"                          // --- this allows us to create a non-standard kubernetes clientset to work with CRDs like helmreleases
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
		// fmt.Println()
		logging.Warn("No namespace specified — defaulting to bigbang.")
	}

	fmt.Println()
	logging.Info("Scanning only HelmRelease resources.")
	logging.Starting("Operator-initiated HelmRelease readiness check")
	time.Sleep(5 * time.Second)

	var totalNamespaces int
	var totalSuspended int
	var totalHelmReleases int
	var failedNamespaces int
	failingMap := make(map[string]int)

	helmReleaseGVR := schema.GroupVersionResource{
		Group:    "helm.toolkit.fluxcd.io",
		Version:  "v2",
		Resource: "helmreleases",
	}

	for _, ns := range namespaces {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		totalNamespaces++
		logging.Info("Scanning namespace: %s", ns)
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
		var suspendedCount int

		if totalHelmReleases == 0 {
			fmt.Printf("No HelmReleases found in %s namespace.\n", ns)
			fmt.Println("")

			elapsed := time.Since(start)
			logging.Info("Attempted scan completed in %s\n",  elapsed.Round(time.Millisecond))
			return
		}

		for _, hr := range releases.Items {
			name := hr.GetName()
			ready, reason := isHelmReleaseReady(hr)
			switch reason {
			case "HelmRelease is suspended":
				color.Yellow("    WARN: %s (Suspended)", name)
				suspendedCount++
				totalSuspended++
			default:
				if ready {
					color.Green("    PASS: %s", name)
				} else {
					color.Red("    FAIL: %s (Reason: %s)", name, reason)
					nonReady = append(nonReady, fmt.Sprintf("%s (Reason: %s)", name, reason))
				}
			}
		}

		if len(nonReady) > 0 {
			fmt.Println("")
			elapsed := time.Since(start)
			logging.Error("Scan completed but with failures in %s", elapsed.Round(time.Millisecond))
			failedNamespaces++
			failingMap[ns] = len(nonReady)
		} else {
			fmt.Println("")
			elapsed := time.Since(start)
			logging.Info("Scan completed in %s", elapsed.Round(time.Millisecond))
		}
	}

	fmt.Println()
	logging.Title("Kobot HelmRelease Readiness Report\n")
	fmt.Printf("Namespaces checked: %d\n", totalNamespaces)
	fmt.Printf("Total HelmReleases checked: %d\n", totalHelmReleases)
	fmt.Printf("Total Suspended HelmReleases: %d\n", totalSuspended)
	fmt.Println("")

	if failedNamespaces > 0 {
		fmt.Println("Failing namespaces:")
		for ns, count := range failingMap {
			fmt.Printf("  - %s (%d HelmRelease(s) not Ready)\n", ns, count)
		}
		fmt.Println()
		logging.Action("Operators should review the listed HelmReleases to resolve deployment issues or Helm upgrade failures.\n")
	} else if totalSuspended > 0{
		logging.Warn("Kobot scanned %d namespace(s) and found %d HelmRelease(s) are Ready but %d HelmRelease(s) were suspended.", totalNamespaces, totalHelmReleases-totalSuspended, totalSuspended)
		logging.Action("Try running 'flux resume hr <helmReleaseName> -n <namespace>' and then re-running the kobot health check.\n")
	} else {
		logging.Success("Kobot scanned %d namespace(s) and found all HelmRelease(s) are Ready.\n", totalNamespaces)
	}
}

// isHelmReleaseReady checks the HelmRelease .status.conditions for Ready=True
// and also ensures the resource is not suspended (.spec.suspend != true).
func isHelmReleaseReady(obj unstructured.Unstructured) (bool, string) {
	// Check if the release is suspended
	suspended, found, err := unstructured.NestedBool(obj.Object, "spec", "suspend")
	if err == nil && found && suspended {
		return false, "HelmRelease is suspended"

	}

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
				if s == "True" {
					return true, r
				}
				return false, r
			}
		}
	}
	return false, "Ready condition missing"
}
