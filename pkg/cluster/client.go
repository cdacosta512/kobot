package cluster

import (
	"fmt"
	"os" // allows us to get the kubeconfig from the env var
	"path/filepath" // allows us to read the kubeconfig from the os path
	"k8s.io/client-go/tools/clientcmd" // allows to find and connect to the users kubeconfig
	"k8s.io/client-go/kubernetes" // creates the actual clientset to get resources within the cluster
)

func GetClientset() (*kubernetes.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return nil, fmt.Errorf("KUBECONFIG not set and HOME is empty")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	if info, err := os.Stat(kubeconfig); err != nil {
		return nil, fmt.Errorf("error accessing kubeconfig at %s: %w", kubeconfig, err)
	} else if info.IsDir() {
		return nil, fmt.Errorf("expected kubeconfig file but found directory at %s", kubeconfig)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build Kubernetes REST config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return clientset, nil
}
