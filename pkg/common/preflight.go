package common

import (
	"gitlab.com/kobot/kobot/pkg/cluster"
	"gitlab.com/kobot/kobot/pkg/logging"
	"k8s.io/client-go/kubernetes"
)

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