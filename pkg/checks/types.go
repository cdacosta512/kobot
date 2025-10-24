package checks

// PodCheckResult represents the health results for pods in a single namespace. Supports the summary report at the end of the RunPodCheck function call.
type PodCheckResult struct {
	Name        string
	PodsChecked int
	PodsFailed  int
	FailedPods  []string
}
