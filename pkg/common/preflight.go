package common

import (
	"gitlab.com/kobot/kobot/pkg/cluster"
	"gitlab.com/kobot/kobot/pkg/logging"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/dynamic"
)

// returns the clientset
func EnsureClusterConnection() *kubernetes.Clientset {
	clientset, err := cluster.GetClientset()
	if err != nil {
		logging.Error("Failed to connect to cluster. If kubectl can't connect, Kobot can't connect either.")
		return nil
	}
	// commenting this out because we dont want to tell the user that it connected; rather just tell them when it didnt connect right?
	// logging.Success("Connected to cluster successfully") 
	return clientset
}

// returns the dynamicClient
func EnsureDynamicClusterConnection() dynamic.Interface {
	dynamicClient, err := cluster.GetDynamicClientset()
	if err != nil {
		logging.Error("Failed to connect to cluster dynamically. If kubectl can't connect, Kobot can't connect either.")
		return nil
	}

	// intentionally silent on success to keep preflight output clean
	return dynamicClient
}
