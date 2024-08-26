package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ethereum-optimism/infra/op-host-manager/pkg/server"
	"github.com/ethereum/go-ethereum/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func main() {
	fmt.Println("Hello world")
	SetLogLevel(slog.LevelInfo)
	log.Info("Starting main")

	// Set up Kubernetes client
	log.Info("Searching for incluster kube config")
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Info("No cluster config found: ", err)
		log.Info("Searching for host kubeconfig")
		// Fallback to kubeconfig
		home := homedir.HomeDir()
		kubeconfig := flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		flag.Parse()

		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	log.Info("Config aquired. ",
		"Current username", config.Username,
	)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Crit("Error initalizing clientset",
			"error", err.Error(),
		)
	}
	log.Info("Starting to update labels")
	go updateLabels(clientset)

	log.Info("Starting server")
	srv := server.NewServer()
	srv.Start()
}

func updateLabels(clientset *kubernetes.Clientset) {
	// Main loop
	i := 0
	log.Info("Beginning update labels")
	for {
		log.Info("Listing nodes")
		nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Error("Error listing nodes: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		for _, node := range nodes.Items {
			log.Info("updating node",
				"node-name", node.Name,
			)
			// Example: Add a custom label to each node
			newLabels := node.Labels
			newLabels["custom-label"] = fmt.Sprintf("example-value-%d", i)

			node.Labels = newLabels
			_, err := clientset.CoreV1().Nodes().Update(context.TODO(), &node, metav1.UpdateOptions{})
			if err != nil {
				log.Error("Error updating node %s: %v\n", node.Name, err)
			} else {
				log.Error("Updated labels for node %s\n", node.Name)
			}
		}
		// Wait before next iteration
		time.Sleep(5 * time.Second)
		log.Info("Sleeping")
	}
}

func SetLogLevel(logLevel slog.Leveler) {
	log.SetDefault(log.NewLogger(slog.NewJSONHandler(
		os.Stdout, &slog.HandlerOptions{Level: logLevel})))
}
