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
	"math/rand"
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
	// Get the pod name and namespace from environment variables
	podName := os.Getenv("POD_NAME")
	namespace := os.Getenv("POD_NAMESPACE")
	if podName == "" || namespace == "" {
		fmt.Println("POD_NAME or POD_NAMESPACE environment variables are not set")
		return
	}

	log.Info("Config aquired. ",
		"Current username", config.Username,
		"pod name", podName,
		"namespace", namespace,
	)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Crit("Error initalizing clientset",
			"error", err.Error(),
		)
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Crit("Error obtaining pod name",
			"error", err.Error(),
		)
	}

	// Print the node name
	nodeName := pod.Spec.NodeName
	log.Info("Pod is running on node",
		"nodeName", nodeName,
	)
	log.Info("Starting to update labels")
	go updateLabels(clientset, nodeName)

	log.Info("Starting server")
	srv := server.NewServer()
	srv.Start()
}

func updateLabels(clientset *kubernetes.Clientset, nodename string) {
	// Main loop
	log.Info("Beginning update labels")
	for {
		log.Info("Getting node",
			"node", nodename,
		)
		node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodename, metav1.GetOptions{})
		if err != nil {
			log.Error("Error getting node",
				"node", nodename,
				"error", err.Error(),
			)
			time.Sleep(10 * time.Second)
			continue
		}

		randomInt := rand.Intn(101)
		log.Info("updating node",
			"node-name", node.Name,
			"random-int", randomInt,
		)
		// Example: Add a custom label to each node
		newLabels := node.Labels
		if val, ok := newLabels["custom-label"]; ok {
			log.Info("current node label",
				"node-name", node.Name,
				"label", val,
			)
		}
		newLabels["custom-label"] = fmt.Sprintf("example-value-%d", randomInt)

		node.Labels = newLabels
		_, err = clientset.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
		if err != nil {
			log.Error("Error updating node",
				"node", node.Name,
				"error", err,
			)
		}
		// Wait before next iteration
		time.Sleep(20 * time.Second)
		log.Info("updated node",
			"node-name", node.Name,
			"random-int", randomInt,
		)
	}
}

func SetLogLevel(logLevel slog.Leveler) {
	log.SetDefault(log.NewLogger(slog.NewJSONHandler(
		os.Stdout, &slog.HandlerOptions{Level: logLevel})))
}
