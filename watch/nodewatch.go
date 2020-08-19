package watch

import (
	"container/list"
	"log"
	"strings"
	"time"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeData struct {
	Name                    string             `json:"name"`
	MachineID               string             `json:"machineID"`
	KernelVersion           string             `json:"kernelVersion"`
	OsImage                 string             `json:"osImage"`
	ContainerRuntimeVersion string             `json:"containerRuntimeVersion"`
	OperatingSystem         string             `json:"operatingSystem"`
	Architecture            string             `json:"architecture"`
	Addresses               []core.NodeAddress `json:"addresses"`
}

func (updateNode *NodeData) UpdateNodeData(node *core.Node) {
	updateNode.Name = node.ObjectMeta.Name
	updateNode.MachineID = node.Status.NodeInfo.MachineID
	updateNode.KernelVersion = node.Status.NodeInfo.KernelVersion
	updateNode.OsImage = node.Status.NodeInfo.OSImage
	updateNode.ContainerRuntimeVersion = node.Status.NodeInfo.ContainerRuntimeVersion
	updateNode.OperatingSystem = node.Status.NodeInfo.OperatingSystem
	updateNode.Architecture = node.Status.NodeInfo.Architecture
	updateNode.Addresses = node.Status.Addresses
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

// NodeWatch Watching over nodes
func (wh *WatchHandler) NodeWatch() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("RECOVER NodeWatch. error: %v", err)
		}
	}()
	for {
		log.Printf("Watching over nodes starting")
		podsWatcher, err := wh.RestAPIClient.CoreV1().Nodes().Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			log.Printf("Cannot wathch over pods. %v", err)
			time.Sleep(time.Duration(10) * time.Second)
			continue
		}
		podsChan := podsWatcher.ResultChan()
		log.Printf("Watching over nodes started")
		for event := range podsChan {
			if node, ok := event.Object.(*core.Node); ok {
				switch event.Type {
				case "ADDED":
					id := CreateID()
					if wh.ndm[id] == nil {
						wh.ndm[id] = list.New()
					}
					nd := &NodeData{Name: node.ObjectMeta.Name,
						MachineID:               node.Status.NodeInfo.MachineID,
						KernelVersion:           node.Status.NodeInfo.KernelVersion,
						OsImage:                 node.Status.NodeInfo.OSImage,
						ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
						OperatingSystem:         node.Status.NodeInfo.OperatingSystem,
						Architecture:            node.Status.NodeInfo.Architecture,
						Addresses:               node.Status.Addresses}
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
					log.Printf("while watching over nodes we got an error")
				}
			} else {
				log.Printf("Got unexpected pod from chan: %t, %v", event.Object, event.Object)
			}
		}
		log.Printf("Wathching over nodes ended - since we got timeout")
	}
	log.Printf("Wathching over nodes ended")
}
