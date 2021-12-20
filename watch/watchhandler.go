package watch

import (
	"container/list"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/armosec/capacketsgo/k8sinterface"
	"github.com/golang/glog"

	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ResourceMap struct {
	resourceMap map[int]*list.List
	mutex       sync.RWMutex
}

func NewResourceMap() *ResourceMap {
	return &ResourceMap{
		resourceMap: make(map[int]*list.List),
		mutex:       sync.RWMutex{},
	}
}
func (rm *ResourceMap) Init(index int) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	if rm.resourceMap[index] == nil {
		rm.resourceMap[index] = list.New()
	}
}
func (rm *ResourceMap) Remove(index int) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	delete(rm.resourceMap, index)
}
func (rm *ResourceMap) GetIDs() []int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	ids := []int{}
	for i := range rm.resourceMap {
		ids = append(ids, i)
	}
	return ids
}
func (rm *ResourceMap) PushBack(index int, obj interface{}) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	if mapElem := rm.resourceMap[index]; mapElem != nil {
		mapElem.PushBack(obj)
	}
}
func (rm *ResourceMap) Front(index int) *list.Element {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	if mapElem := rm.resourceMap[index]; mapElem != nil {
		return rm.resourceMap[index].Front()
	}
	return nil
}
func (rm *ResourceMap) UpdateFront(index int, obj interface{}) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	if mapElem := rm.resourceMap[index]; mapElem != nil {
		mapElem.Front().Value = obj
		rm.resourceMap[index] = mapElem
	}
}
func (rm *ResourceMap) Len() int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return len(rm.resourceMap)
}

func (rm *ResourceMap) IndexLen(index int) int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	if rm.resourceMap[index] == nil {
		return 0
	}
	return rm.resourceMap[index].Len()
}

// WatchHandler -
type WatchHandler struct {
	extensionsClient apixv1beta1client.ApiextensionsV1beta1Interface
	RestAPIClient    kubernetes.Interface
	K8sApi           *k8sinterface.KubernetesApi
	WebSocketHandle  *WebSocketHandler
	// cluster info
	clusterAPIServerVersion *version.Info
	cloudVendor             string
	// pods list
	pdm map[int]*list.List
	// node list
	ndm map[int]*list.List
	// services list
	sdm map[int]*list.List
	// secrets list
	secretdm *ResourceMap
	// namespaces list
	namespacedm *ResourceMap

	jsonReport             jsonFormat
	informNewDataChannel   chan int
	aggregateFirstDataFlag bool
	// newStateReportChans is calling in a loop whenever new connection to ARMO BE is initialized
	newStateReportChans []chan bool
	IncludeNamespaces   []string
}

// GetAggregateFirstDataFlag return pointer
func (wh *WatchHandler) GetAggregateFirstDataFlag() *bool {
	return &wh.aggregateFirstDataFlag
}

//CreateWatchHandler -
func CreateWatchHandler() *WatchHandler {

	namespacesStr := flag.String("include-namespaces", "", "comma separated namespaces list to watch on. Empty list or omit to watch them all")
	config := parseArgument()
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		log.Print(err.Error())
		return nil
	}
	extensionsClientSet, err := apixv1beta1client.NewForConfig(config)

	if err != nil {
		log.Print(err.Error())
		return nil
	}

	var clusterName string
	if clusterName = os.Getenv("CA_CLUSTER_NAME"); clusterName == "" {
		log.Println("there is no cluster name environment variable, envKey:CA_CLUSTER_NAME")
		//clusterName = "superCluster"
		return nil
	}

	var reportURL string
	if reportURL = os.Getenv("CA_K8S_REPORT_URL"); reportURL == "" {
		log.Println("there is no report url environment variable, envKey:CA_K8S_REPORT_URL")
		//reportURL = "report.eudev2.cyberarmorsoft.com"
		return nil
	}

	var cusGUID string
	if cusGUID = os.Getenv("CA_CUSTOMER_GUID"); cusGUID == "" {
		log.Println("there is no customer guid environment variable, envKey:CA_CUSTOMER_GUID")
		//cusGUID = "1e3a88bf-92ce-44f8-914e-cbe71830d566"
		return nil
	}

	result := WatchHandler{RestAPIClient: clientset,
		WebSocketHandle:  createWebSocketHandler(reportURL, "k8s/cluster-reports", clusterName, cusGUID),
		extensionsClient: extensionsClientSet,
		K8sApi:           k8sinterface.NewKubernetesApi(),
		pdm:              make(map[int]*list.List),
		ndm:              make(map[int]*list.List),
		sdm:              make(map[int]*list.List),
		secretdm:         NewResourceMap(),
		namespacedm:      NewResourceMap(),
		jsonReport: jsonFormat{
			FirstReport: true,
		},
		informNewDataChannel:   make(chan int),
		aggregateFirstDataFlag: true,
		IncludeNamespaces:      strings.Split(*namespacesStr, ","),
	}
	return &result
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

	threFlag := flag.Lookup("stderrthreshold")
	threFlag.DefValue = "WARNING"
	flag.Parse()
	fmt.Printf("Log level: %s, set -stderrthreshold=INFO for detailed logs", threFlag.Value)

	switch *configtype {
	case 0:
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfigpath)
		if err != nil {
			log.Printf("kubeconfig path is %s\n", *kubeconfigpath)
			log.Print(err.Error())
			return nil
		}
	case 1:
		config, err = restclient.InClusterConfig()
		if err != nil {
			log.Print(err.Error())
			return nil
		}
	}

	return config
}

// SetFirstReportFlag set first report flag
func (wh *WatchHandler) SetFirstReportFlag(first bool) {
	if wh.jsonReport.FirstReport == first {
		return
	}
	wh.jsonReport.FirstReport = first
	if first {
		wh.ndm = make(map[int]*list.List)
		wh.pdm = make(map[int]*list.List)
		wh.sdm = make(map[int]*list.List)
		wh.secretdm = NewResourceMap()
		wh.namespacedm = NewResourceMap()
		for chanIdx := range wh.newStateReportChans {
			wh.newStateReportChans[chanIdx] <- true
		}
	}
}

// GetFirstReportFlag get first report flag
func (wh *WatchHandler) GetFirstReportFlag() bool {
	return wh.jsonReport.FirstReport
}

func (wh *WatchHandler) HandleDataMismatch(resource string, resourceMap map[string]string) error {
	if len(resourceMap) == 0 { // ignore if map is empty / nil
		return nil
	}
	mismatch, err := wh.isDataMismatch(resource, resourceMap)
	if err != nil || mismatch {
		glog.Infof("mismatch found in resource: %s, exiting...", resource)
		os.Exit(2)
	}
	return nil
}
func (wh *WatchHandler) isDataMismatch(resource string, resourceMap map[string]string) (bool, error) {
	r, _ := k8sinterface.GetGroupVersionResource(resource)
	workloadList, err := wh.K8sApi.ListWorkloads(&r, "", nil)
	if err != nil {
		return false, err
	}

	if len(workloadList) != len(resourceMap) {
		glog.Infof("found 'resource len' mismatch, resource: '%s', current len: %d, received from server: %d", resource, len(workloadList), len(resourceMap))
		return true, nil
	}
	for i := range workloadList {
		resourceID := GetResourceID(&workloadList[i])
		if r, ok := resourceMap[resourceID]; ok {
			if r != GetResourceVersion(&workloadList[i]) {
				glog.Infof("resource version mismatch, resource: '%s', name: %s, id: %s not found in current resource map", resource, resourceID, GetResourceVersion(&workloadList[i]))
				return true, nil
			}
		} else {
			glog.Infof("resource ID mismatch, resource: '%s', name: '%s', id: %s not found in current resource map", resource, resourceID, GetResourceVersion(&workloadList[i]))
			return true, nil
		}
	}
	return false, nil
}

func GetResourceID(workload *k8sinterface.Workload) string {
	switch workload.GetKind() {
	case "Node":
		return workload.GetName()
	default:
		return workload.GetUID()
	}
}

func GetResourceVersion(workload *k8sinterface.Workload) string {
	switch workload.GetKind() {
	case "Node":
		return workload.GetName()
	default:
		return workload.GetResourceVersion()
	}
}

func (wh *WatchHandler) isNamespaceWatched(namespace string) bool {
	for nsIdx := range wh.IncludeNamespaces {
		if wh.IncludeNamespaces[nsIdx] == "" || wh.IncludeNamespaces[nsIdx] == namespace {
			return true
		}
	}

	glog.Info("Namespace '%s' isn't tracked", namespace)
	return false

}
