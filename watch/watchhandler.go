package watch

import (
	"container/list"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/kubescape/k8s-interface/k8sinterface"
	"github.com/kubescape/kollector/config"
	"github.com/kubescape/kollector/consts"
	restclient "k8s.io/client-go/rest"

	beClientV1 "github.com/kubescape/backend/pkg/client/v1"
	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
)

type resourceMap struct {
	resourceMap map[int]*list.List
	mutex       sync.RWMutex
}

func newResourceMap() *resourceMap {
	return &resourceMap{
		resourceMap: make(map[int]*list.List),
		mutex:       sync.RWMutex{},
	}
}
func (rm *resourceMap) init(index int) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	if rm.resourceMap[index] == nil {
		rm.resourceMap[index] = list.New()
	}
}
func (rm *resourceMap) remove(index int) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	delete(rm.resourceMap, index)
}
func (rm *resourceMap) getIDs() []int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	ids := []int{}
	for i := range rm.resourceMap {
		ids = append(ids, i)
	}
	return ids
}
func (rm *resourceMap) pushBack(index int, obj interface{}) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	if mapElem := rm.resourceMap[index]; mapElem != nil {
		mapElem.PushBack(obj)
	}
}
func (rm *resourceMap) front(index int) *list.Element {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	if mapElem := rm.resourceMap[index]; mapElem != nil {
		return rm.resourceMap[index].Front()
	}
	return nil
}
func (rm *resourceMap) updateFront(index int, obj interface{}) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	if mapElem := rm.resourceMap[index]; mapElem != nil {
		mapElem.Front().Value = obj
		rm.resourceMap[index] = mapElem
	}
}
func (rm *resourceMap) len() int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return len(rm.resourceMap)
}

func (rm *resourceMap) indexLen(index int) int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	if rm.resourceMap[index] == nil {
		return 0
	}
	return rm.resourceMap[index].Len()
}

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
	// pods list
	cjm map[int]*list.List
	// secrets list
	secretdm *resourceMap
	// namespaces list
	namespacedm *resourceMap

	jsonReport             jsonFormat
	informNewDataChannel   chan int
	aggregateFirstDataFlag bool
	// newStateReportChans is calling in a loop whenever new connection to BE is initialized
	newStateReportChans []chan bool
	includeNamespaces   []string

	config config.IConfig

	notifyUpdates iClusterNotifier // notify other (in-cluster) components about new data
}

func CreateWatchHandler(config config.IConfig) (*WatchHandler, error) {

	componentNamespace := os.Getenv(consts.NamespaceEnvironmentVariable)

	if err := parseArgument(); err != nil {
		return nil, fmt.Errorf("failed to parse args: %s", err.Error())
	}

	// create the clientset
	k8sAPiObj := k8sinterface.NewKubernetesApi()

	restclient.SetDefaultWarningHandler(restclient.NoWarnings{})
	extensionsClientSet, err := apixv1beta1client.NewForConfig(k8sinterface.GetK8sConfig())
	if err != nil {
		return nil, fmt.Errorf("apiV1beta1client.NewForConfig failed: %s", err.Error())
	}

	erURL, err := beClientV1.GetReporterClusterReportsWebsocketUrl(config.EventReceiverWebsocketURL(), config.AccountID(), config.ClusterName())
	if err != nil {
		return nil, fmt.Errorf("failed to set event receiver url: %s", err.Error())
	}

	result := WatchHandler{RestAPIClient: k8sAPiObj.KubernetesClient,
		WebSocketHandle:  createWebSocketHandler(erURL, config.AccessKey()),
		extensionsClient: extensionsClientSet,
		K8sApi:           k8sinterface.NewKubernetesApi(),
		pdm:              make(map[int]*list.List),
		ndm:              make(map[int]*list.List),
		sdm:              make(map[int]*list.List),
		cjm:              make(map[int]*list.List),
		config:           config,
		secretdm:         newResourceMap(),
		namespacedm:      newResourceMap(),
		jsonReport: jsonFormat{
			FirstReport: true,
		},
		informNewDataChannel:   make(chan int),
		aggregateFirstDataFlag: true,
		includeNamespaces:      []string{componentNamespace}, // ignore only the component namespace
		notifyUpdates:          newInClusterNotifier(config),
	}
	return &result, nil
}

func parseArgument() error {

	threFlag := flag.Lookup("stderrthreshold")
	threFlag.DefValue = "WARNING"
	flag.Parse()

	return nil
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
		wh.cjm = make(map[int]*list.List)
		wh.secretdm = newResourceMap()
		wh.namespacedm = newResourceMap()
		for chanIdx := range wh.newStateReportChans {
			wh.newStateReportChans[chanIdx] <- true
		}
	}
}

// getFirstReportFlag get first report flag
func (wh *WatchHandler) getFirstReportFlag() bool {
	return wh.jsonReport.FirstReport
}

func (wh *WatchHandler) isNamespaceWatched(namespace string) bool {
	for nsIdx := range wh.includeNamespaces {
		if wh.includeNamespaces[nsIdx] == "" || wh.includeNamespaces[nsIdx] == namespace {
			return true
		}
	}
	return false
}

// getAggregateFirstDataFlag return pointer
func (wh *WatchHandler) getAggregateFirstDataFlag() *bool {
	return &wh.aggregateFirstDataFlag
}
