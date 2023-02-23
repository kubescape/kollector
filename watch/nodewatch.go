package watch

import (
	"container/list"
	"runtime/debug"
	"strings"
	"time"

	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
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

func UpdateNode(node *core.Node, ndm map[int]*list.List) *NodeData {

	var nd *NodeData
	for _, v := range ndm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(*NodeData).Name, node.ObjectMeta.Name) == 0 {
			v.Front().Value.(*NodeData).UpdateNodeData(node)
			logger.L().Debug("node updated", helpers.String("name", v.Front().Value.(*NodeData).Name))
			nd = v.Front().Value.(*NodeData)
			break
		}
		if strings.Compare(v.Front().Value.(*NodeData).Name, node.ObjectMeta.GenerateName) == 0 {
			v.Front().Value.(*NodeData).UpdateNodeData(node)
			logger.L().Debug("node updated", helpers.String("name", v.Front().Value.(*NodeData).Name))
			nd = v.Front().Value.(*NodeData)
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
		if strings.Compare(v.Front().Value.(*NodeData).Name, node.ObjectMeta.Name) == 0 {
			v.Remove(v.Front())
			logger.L().Debug("node removed", helpers.String("name", v.Front().Value.(*NodeData).Name))
			nodeName = v.Front().Value.(*NodeData).Name
			break
		}
		if strings.Compare(v.Front().Value.(*NodeData).Name, node.ObjectMeta.GenerateName) == 0 {
			v.Remove(v.Front())
			logger.L().Debug("node removed", helpers.String("name", v.Front().Value.(*NodeData).Name))
			nodeName = v.Front().Value.(*NodeData).Name
			break
		}
	}
	return nodeName
}

// NodeWatch Watching over nodes
func (wh *WatchHandler) NodeWatch() {
	defer func() {
		if err := recover(); err != nil {
			logger.L().Error("RECOVER NodeWatch", helpers.Interface("error", err), helpers.String("stack", string(debug.Stack())))
		}
	}()
	var lastWatchEventCreationTime time.Time
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
	for {
		wh.clusterAPIServerVersion = wh.getClusterVersion()
		wh.cloudVendor = wh.checkInstanceMetadataAPIVendor()
		if wh.cloudVendor != "" {
			wh.clusterAPIServerVersion.GitVersion += ";" + wh.cloudVendor
		}
		logger.L().Info("K8s Cloud Vendor", helpers.String("cloudVendor", wh.cloudVendor))
		logger.L().Info("Watching over nodes starting")
		nodesWatcher, err := wh.RestAPIClient.CoreV1().Nodes().Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		wh.handleNodeWatch(nodesWatcher, newStateChan, &lastWatchEventCreationTime)

	}
}
func (wh *WatchHandler) handleNodeWatch(nodesWatcher watch.Interface, newStateChan <-chan bool, lastWatchEventCreationTime *time.Time) {
	nodesChan := nodesWatcher.ResultChan()
	for {
		var event watch.Event
		select {
		case event = <-nodesChan:
		case <-newStateChan:
			nodesWatcher.Stop()
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if event.Type == watch.Error {
			nodesWatcher.Stop()
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if node, ok := event.Object.(*core.Node); ok {
			node.ManagedFields = []metav1.ManagedFieldsEntry{}
			switch event.Type {
			case "ADDED":
				if node.CreationTimestamp.Time.Before(*lastWatchEventCreationTime) {
					continue
				}
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
				updateNode := UpdateNode(node, wh.ndm)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(updateNode, NODE, UPDATED)
			case "DELETED":
				name := RemoveNode(node, wh.ndm)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(name, NODE, DELETED)
			case "BOOKMARK": //only the resource version is changed but it's the same workload
				continue
			case "ERROR":
				*lastWatchEventCreationTime = time.Now()
				return
			}
		} else {
			*lastWatchEventCreationTime = time.Now()
			return
		}
	}
}

func (wh *WatchHandler) checkInstanceMetadataAPIVendor() string {
	res, _ := getInstanceMetadata()
	return res
}

func (wh *WatchHandler) getClusterVersion() *version.Info {
	serverVersion, err := wh.RestAPIClient.Discovery().ServerVersion()
	if err != nil {
		serverVersion = &version.Info{GitVersion: "Unknown"}
	}
	logger.L().Info("K8s API version", helpers.Interface("version", serverVersion))
	return serverVersion
}
