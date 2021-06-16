package watch

import (
	"container/list"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apimachinery/pkg/watch"
)

type NodeData struct {
	// core.NodeSystemInfo
	core.NodeStatus `json:",inline"`
	Name            string `json:"name"`
}

func (updateNode *NodeData) UpdateNodeData(node *core.Node) {
	updateNode.Name = node.ObjectMeta.Name
	updateNode.NodeStatus = node.Status
}

func UpdateNode(node *core.Node, ndm map[int]*list.List) NodeData {

	var nd NodeData
	for _, v := range ndm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(NodeData).Name, node.ObjectMeta.Name) == 0 {
			v.Front().Value.(*NodeData).UpdateNodeData(node)
			log.Printf("node %s updated", v.Front().Value.(NodeData).Name)
			nd = v.Front().Value.(NodeData)
			break
		}
		if strings.Compare(v.Front().Value.(NodeData).Name, node.ObjectMeta.GenerateName) == 0 {
			v.Front().Value.(*NodeData).UpdateNodeData(node)
			log.Printf("node %s updated", v.Front().Value.(NodeData).Name)
			nd = v.Front().Value.(NodeData)
			break
		}
	}
	return nd
}

func RemoveNode(node *core.Node, ndm map[int]*list.List) string {

	var nodeName string
	for _, v := range ndm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(NodeData).Name, node.ObjectMeta.Name) == 0 {
			v.Remove(v.Front())
			log.Printf("node %s updated", v.Front().Value.(NodeData).Name)
			nodeName = v.Front().Value.(NodeData).Name
			break
		}
		if strings.Compare(v.Front().Value.(NodeData).Name, node.ObjectMeta.GenerateName) == 0 {
			v.Remove(v.Front())
			log.Printf("node %s updated", v.Front().Value.(NodeData).Name)
			nodeName = v.Front().Value.(NodeData).Name
			break
		}
	}
	return nodeName
}

func (wh *WatchHandler) CheckInstanceMetadataAPIVendor() string {
	res, _ := getInstanceMetadata()
	return res
}

// // NodeWatch Watching over nodes
// func (wh *WatchHandler) NodeWatch() {
// 	defer func() {
// 		if err := recover(); err != nil {
// 			log.Printf("RECOVER NodeWatch. error: %v", err)
// 		}
// 	}()
// 	newStateChan := make(chan bool)
// 	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
// WatchLoop:
// 	for {
// 		log.Printf("Taking k8s API version")
// 		serverVersion, err := wh.RestAPIClient.Discovery().ServerVersion()
// 		if err != nil {
// 			glog.Errorf("Failed to get API server version, %v", err)
// 			serverVersion = &version.Info{GitVersion: "Unknown"}
// 		} else {
// 			log.Printf("K8s API version :%v", serverVersion)
// 		}
// 		wh.clusterAPIServerVersion = serverVersion
// 		wh.cloudVendor = wh.CheckInstanceMetadataAPIVendor()
// 		if wh.cloudVendor != "" {
// 			wh.clusterAPIServerVersion.GitVersion += ";" + wh.cloudVendor
// 		}
// 		log.Printf("K8s Cloud Vendor : %s", wh.cloudVendor)
// 		log.Printf("Watching over nodes starting")
// 		nodesWatcher, err := wh.RestAPIClient.CoreV1().Nodes().Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
// 		if err != nil {
// 			log.Printf("Cannot wathch over pods. %v", err)
// 			time.Sleep(time.Duration(10) * time.Second)
// 			continue
// 		}
// 		nodesChan := nodesWatcher.ResultChan()
// 		log.Printf("Watching over nodes started")
// 	ChanLoop:
// 		for {
// 			var event watch.Event
// 			select {
// 			case event = <-nodesChan:
// 			case <-newStateChan:
// 				nodesWatcher.Stop()
// 				glog.Errorf("Node watch - newStateChan signal")
// 				continue WatchLoop
// 			}
// 			if event.Type == watch.Error {
// 				glog.Errorf("Node watch chan loop error: %v", event.Object)
// 				nodesWatcher.Stop()
// 				break ChanLoop
// 			}
// 			if node, ok := event.Object.(*core.Node); ok {
// 				node.ManagedFields = []metav1.ManagedFieldsEntry{}
// 				switch event.Type {
// 				case "ADDED":
// 					id := CreateID()
// 					if wh.ndm[id] == nil {
// 						wh.ndm[id] = list.New()
// 					}
// 					nd := &NodeData{Name: node.ObjectMeta.Name,
// 						NodeStatus: node.Status,
// 					}
// 					wh.ndm[id].PushBack(nd)
// 					informNewDataArrive(wh)
// 					wh.jsonReport.AddToJsonFormat(nd, NODE, CREATED)
// 				case "MODIFY":
// 					updateNode := UpdateNode(node, wh.ndm)
// 					informNewDataArrive(wh)
// 					wh.jsonReport.AddToJsonFormat(updateNode, NODE, UPDATED)
// 				case "DELETED":
// 					name := RemoveNode(node, wh.ndm)
// 					informNewDataArrive(wh)
// 					wh.jsonReport.AddToJsonFormat(name, NODE, DELETED)
// 				case "BOOKMARK": //only the resource version is changed but it's the same workload
// 					continue
// 				case "ERROR":
// 					log.Printf("while watching over nodes we got an error")
// 				}
// 			} else {
// 				log.Printf("Got unexpected node from chan: %t, %v", event.Object, event.Object)
// 				break
// 			}
// 		}
// 		log.Printf("Wathching over nodes ended - since we got timeout")
// 	}
// }

// NodeWatch Watching over nodes
func (wh *WatchHandler) NodeWatch() {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorf("RECOVER NodeWatch. error: %v, stack: %s", err, debug.Stack())
		}
	}()
	resourceMap := make(map[string]string)
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
	for {
		glog.Infof("Taking k8s API version")
		serverVersion, err := wh.RestAPIClient.Discovery().ServerVersion()
		if err != nil {
			glog.Errorf("Failed to get API server version, %v", err)
			serverVersion = &version.Info{GitVersion: "Unknown"}
		} else {
			glog.Infof("K8s API version :%v", serverVersion)
		}
		wh.clusterAPIServerVersion = serverVersion
		wh.cloudVendor = wh.CheckInstanceMetadataAPIVendor()
		if wh.cloudVendor != "" {
			wh.clusterAPIServerVersion.GitVersion += ";" + wh.cloudVendor
		}
		glog.Infof("K8s Cloud Vendor : %s", wh.cloudVendor)
		glog.Infof("Watching over nodes starting")
		nodesWatcher, err := wh.RestAPIClient.CoreV1().Nodes().Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			glog.Errorf("Cannot watch over nodes. %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		wh.HandleDataMismatch("nodes", resourceMap)
		wh.handleNodeWatch(nodesWatcher, newStateChan, resourceMap)

		glog.Infof("Watching over nodes ended - since we got timeout")
	}
}

func (wh *WatchHandler) handleNodeWatch(nodesWatcher watch.Interface, newStateChan <-chan bool, resourceMap map[string]string) {
	log.Printf("Watching over nodes started")
	nodesChan := nodesWatcher.ResultChan()
	for {
		var event watch.Event
		select {
		case event = <-nodesChan:
		case <-newStateChan:
			nodesWatcher.Stop()
			glog.Errorf("Node watch - newStateChan signal")
			return
		}
		if event.Type == watch.Error {
			glog.Errorf("Node watch chan loop error: %v", event.Object)
			nodesWatcher.Stop()
			return
		}
		if node, ok := event.Object.(*core.Node); ok {
			node.ManagedFields = []metav1.ManagedFieldsEntry{}
			switch event.Type {
			case "ADDED":
				resourceMap[string(node.GetUID())] = node.GetResourceVersion()
				id := CreateID()
				if wh.ndm[id] == nil {
					wh.ndm[id] = list.New()
				}
				nd := &NodeData{Name: node.ObjectMeta.Name,
					NodeStatus: node.Status,
				}
				wh.ndm[id].PushBack(nd)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(nd, NODE, CREATED)
			case "MODIFY":
				resourceMap[string(node.GetUID())] = node.GetResourceVersion()
				updateNode := UpdateNode(node, wh.ndm)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(updateNode, NODE, UPDATED)
			case "DELETED":
				delete(resourceMap, string(node.GetUID()))
				name := RemoveNode(node, wh.ndm)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(name, NODE, DELETED)
			case "BOOKMARK": //only the resource version is changed but it's the same workload
				continue
			case "ERROR":
				log.Printf("while watching over nodes we got an error")
				return
			}
		} else {
			log.Printf("Got unexpected node from chan: %v", event.Object)
			return
		}
	}
}
