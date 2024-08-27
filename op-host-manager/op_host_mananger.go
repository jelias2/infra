package ophostmanager

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type OPHostManager struct {
	clientset kubernetes.Clientset
	pod       string
	host      string
	namespace string
}

func NewOPHostMananger() (*OPHostManager, error) {
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
			return nil, err
		}
	}
	// Get the pod name and namespace from environment variables
	podName := os.Getenv("POD_NAME")
	namespace := os.Getenv("POD_NAMESPACE")
	if podName == "" || namespace == "" {
		return nil, errors.New("POD_NAME or POD_NAMESPACE environment variables are not set. Exiting")
	}

	log.Info("Config aquired. ",
		"Current username", config.Username,
		"pod name", podName,
		"namespace", namespace,
	)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error("Error initalizing clientset",
			"error", err.Error(),
		)
		return nil, err
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Error("Error obtaining pod name",
			"error", err.Error(),
		)
		return nil, err
	}

	// Print the node name
	nodeName := pod.Spec.NodeName
	log.Info("Pod is running on node",
		"nodeName", nodeName,
	)

	return &OPHostManager{
		clientset: *clientset,
		pod:       podName,
		namespace: namespace,
		host:      nodeName,
	}, nil
}

func (o *OPHostManager) updateLabels() {
	// Main loop
	log.Info("Starting to update labels")
	for {
		log.Info("Getting host",
			"host", o.host,
		)
		node, err := o.clientset.CoreV1().Nodes().Get(context.TODO(), o.host, metav1.GetOptions{})
		if err != nil {
			log.Error("Error getting node",
				"node", o.host,
				"error", err.Error(),
			)
			time.Sleep(10 * time.Second)
			continue
		}

		newLabels := node.Labels
		if val, ok := newLabels["custom-label"]; ok {
			log.Info("current node label",
				"node-name", node.Name,
				"label", val,
			)
		}

		randomInt := rand.Intn(101)
		log.Info("updating node",
			"node-name", node.Name,
			"random-int", randomInt,
		)

		newLabels["custom-label"] = fmt.Sprintf("example-value-%d", randomInt)

		node.Labels = newLabels
		_, err = o.clientset.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
		if err != nil {
			log.Error("Error updating node",
				"node", node.Name,
				"error", err,
			)
		}
		IncrementLabelUpdates("bar")
		// Wait before next iteration
		time.Sleep(20 * time.Second)
		log.Info("updated node",
			"node-name", node.Name,
			"random-int", randomInt,
		)
	}
}
