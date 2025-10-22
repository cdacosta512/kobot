package checks

// NamespaceResult represents the health summary for a single namespace.
type NamespaceResult struct {
	Name        string
	PodsChecked int
	PodsFailed  int
	FailedPods  []string
}