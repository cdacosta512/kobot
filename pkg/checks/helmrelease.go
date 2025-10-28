package checks

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"gitlab.com/kobot/kobot/pkg/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// RunHelmReleaseCheck performs a health check on all HelmReleases
// within a namespace or a default one (bigbang) if none is specified.
func RunHelmReleaseCheck(dynamicClient dynamic.Interface, namespaces []string, htmlOutput bool, fluxGracePeriod int) {
	// If user didn’t specify any namespaces, use "bigbang" by default
	if len(namespaces) == 0 || (len(namespaces) == 1 && namespaces[0] == "") {
		namespaces = []string{"bigbang"}
		fmt.Println()
		logging.Warn("No namespaces specified — defaulting to 'bigbang'.")
	}

	logging.Info("Scanning only HelmRelease resources.")

	if fluxGracePeriod != 5 {
		logging.Info("Flux wait grace period set to %ds.", fluxGracePeriod)
	} else {
		logging.Info("No Flux wait grace period specified — using default of 5 seconds.")
	}

	logging.Starting("Operator-initiated HelmRelease readiness check")
	time.Sleep(2 * time.Second)

	helmReleaseGVR := schema.GroupVersionResource{
		Group:    "helm.toolkit.fluxcd.io",
		Version:  "v2",
		Resource: "helmreleases",
	}

	var results []struct {
		Name   string
		Ready  bool
		Reason string
	}

	var totalHelmReleases, totalSuspended, failed int

	// --- Loop over all provided namespaces
	for _, ns := range namespaces {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		logging.Info("Scanning namespace: %s", ns)

		releases, err := dynamicClient.Resource(helmReleaseGVR).Namespace(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			logging.Error("Unable to list HelmReleases in %s: %v", ns, err)
			continue
		}

		for _, hr := range releases.Items {
			totalHelmReleases++
			name := hr.GetName()

			ready, reason := checkHelmReleaseWithGrace(hr, fluxGracePeriod)
			results = append(results, struct {
				Name   string
				Ready  bool
				Reason string
			}{Name: name, Ready: ready, Reason: reason})

			switch {
			case reason == "HelmRelease is suspended":
				totalSuspended++
			case !ready:
				failed++
			}
		}
	}

	logging.Info("Scan complete. Generating kobot health report.\n")
	time.Sleep(3 * time.Second)

	fmt.Println("=====================================================")
	logging.Title("    Kobot HelmRelease Readiness Report\n")
	fmt.Println("=====================================================\n")

	fmt.Printf("Total HelmReleases checked: %d\n", totalHelmReleases)
	fmt.Printf("Total Suspended HelmReleases: %d\n\n", totalSuspended)

	for _, r := range results {
		switch {
		case r.Reason == "HelmRelease is suspended":
			color.Yellow("WARN: %s (Suspended)", r.Name)
		case r.Ready:
			color.Green("PASS: %s", r.Name)
		default:
			color.Red("FAIL: %s (Reason: %s)", r.Name, r.Reason)
			fmt.Print(logging.Kobot("HelmRelease remained unhealthy after waiting %ds to transition into Ready\n", fluxGracePeriod))
		}
	}

	fmt.Println()
	if failed > 0 {
		logging.Error("Scan completed with %d failing HelmReleases.", failed)
		logging.Action("Operators should review failed HelmReleases and rerun kobot once reconciled.\n")
	} else if totalSuspended > 0 {
		logging.Warn("No failing HelmReleases were detected; however, one or more HelmReleases are currently suspended. An operator should investigate the reason for the suspension.\n")
	} else {
		logging.Success("All HelmReleases are in a Ready state. This reflects only the readiness condition of the HelmReleases themselves and does not confirm that the deployed services or pods are functioning correctly. To perform a full cluster health check, use the 'kobot check cluster' command.\n")
	}
}

// isHelmReleaseReady checks the HelmRelease .status.conditions for Ready=True
// and also ensures the resource is not suspended (.spec.suspend != true).
func isHelmReleaseReady(obj unstructured.Unstructured) (bool, string) {
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
				return s == "True", r
			}
		}
	}
	return false, "Ready condition missing"
}

func checkHelmReleaseWithGrace(obj unstructured.Unstructured, fluxGracePeriod int) (bool, string) {
	ready, reason := isHelmReleaseReady(obj)
	if ready {
		return true, reason
	}

	time.Sleep(time.Duration(fluxGracePeriod) * time.Second)

	ready, reason = isHelmReleaseReady(obj)
	if ready {
		return true, reason
	}

	return false, reason
}
