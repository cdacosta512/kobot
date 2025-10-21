package cluster

import (
	"fmt"
	"os" // allows us to get the kubeconfig from the env var
	"path/filepath" // allows us to read the kubeconfig from the os path
	"k8s.io/client-go/tools/clientcmd" // allows to find and connect to the users kubeconfig
	"k8s.io/client-go/kubernetes" // creates the actual clientset to get resources within the cluster
	"log/slog"
)

// get the kubeconfig and return a authenticated clientset (frontdesk to the cluster)
func GetClientset() (*kubernetes.Clientset, error) {
	// look for this env var as most users set this to a different location depending
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		// fallback and try to see if the user has it in the default .kube path
		home := os.Getenv("HOME")
		kubeconfig := filepath.Join(home, ".kube", "config")
		_ = kubeconfig
		slog.Info("Loaded kubeconfig from the default path")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		slog.Error("could not load kubeconfig or in-cluster config: %v", err)
	}
	
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}
	
	return clientset, nil

}

