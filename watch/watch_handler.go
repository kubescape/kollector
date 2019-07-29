package watch

import (
	"container/list"
	"flag"
	"log"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type WatchHandler struct {
	RestAPIClient   kubernetes.Interface
	WebSocketHandle *WebSocketHandler
	pdm             map[int]*list.List
	ndm             map[int]*list.List
	sdm             map[int]*list.List
	jsonReport      jsonFormat
}

func CreateWatchHandler() WatchHandler {

	config := parseArgument()
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}

	result := WatchHandler{RestAPIClient: clientset, WebSocketHandle: CreateWebSocketHandler(), pdm: make(map[int]*list.List), ndm: make(map[int]*list.List), sdm: make(map[int]*list.List), jsonReport: jsonFormat{Nodes: ObjectData{}, Services: ObjectData{}, MicroServices: ObjectData{}, Pods: ObjectData{}}}

	var cn string
	if cn = os.Getenv("CA_CLUSTER_NAME"); cn == "" {
		cn = "123"
	}
	//start websocket
	//result.WebSocketHandle.StartWebSokcetClient("10.42.4.52:7555", "k8s/cluster-reports", cn, "123" /*customer guid*/)
	return result
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
func parseArgument() *restclient.Config {
	var kubeconfigpath *string
	var config *restclient.Config
	var err error

	home := homeDir()
	configtype := flag.Int("configtype", 0, "newForConfig = 0, inClusterConfig = 1")
	if len(os.Args) < 3 && home != "" {
		kubeconfigpath = flag.String("kubeconfigpath", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfigpath = flag.String("kubeconfigpath", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	switch *configtype {
	case 0:
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfigpath)
		if err != nil {
			log.Printf("kubeconfig path is %s\n", *kubeconfigpath)
			panic(err.Error())
		}
	case 1:
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	return config
}
