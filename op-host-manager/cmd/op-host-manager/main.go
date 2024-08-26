package main

import (
	"context"
	"flag"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"path/filepath"
	"time"
)

func main() {
	fmt.Println("Starting main")
	// Set up Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig
		home := homedir.HomeDir()
		kubeconfig := flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		flag.Parse()

		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	go updateLabels(clientset)
}

func updateLabels(clientset *kubernetes.Clientset) {
	// Main loop
	i := 0
	for {
		nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Error listing nodes: %v\n", err)
			time.Sleep(5 * time.Minute)
			continue
		}

		for _, node := range nodes.Items {
			// Example: Add a custom label to each node
			newLabels := node.Labels
			newLabels["custom-label"] = fmt.Sprintf("example-value-%d", i)

			node.Labels = newLabels
			_, err := clientset.CoreV1().Nodes().Update(context.TODO(), &node, metav1.UpdateOptions{})
			if err != nil {
				fmt.Printf("Error updating node %s: %v\n", node.Name, err)
			} else {
				fmt.Printf("Updated labels for node %s\n", node.Name)
			}
		}

		// Wait before next iteration
		time.Sleep(15 * time.Minute)
		fmt.Println("Sleeping")
	}
}
