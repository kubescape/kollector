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
	name                    string             `json:"name"`
	machineID               string             `json:"machineID"`
	kernelVersion           string             `json:"kernelVersion"`
	osImage                 string             `json:"osImage"`
	containerRuntimeVersion string             `json:"containerRuntimeVersion"`
	operatingSystem         string             `json:"operatingSystem"`
	architecture            string             `json:"architecture"`
	nodeAddr                []core.NodeAddress `json:"addresses"`
}

func CreateNewNodeData(node *core.Node) NodeData {
	return NodeData{node.ObjectMeta.Name,
		node.Status.NodeInfo.MachineID,
		node.Status.NodeInfo.KernelVersion,
		node.Status.NodeInfo.OSImage,
		node.Status.NodeInfo.ContainerRuntimeVersion,
		node.Status.NodeInfo.OperatingSystem,
		node.Status.NodeInfo.Architecture,
		node.Status.Addresses}
}

func (updateNode *NodeData) UpdateNodeData(node *core.Node) {
	updateNode.name = node.ObjectMeta.Name
	updateNode.machineID = node.Status.NodeInfo.MachineID
	updateNode.kernelVersion = node.Status.NodeInfo.KernelVersion
	updateNode.osImage = node.Status.NodeInfo.OSImage
	updateNode.containerRuntimeVersion = node.Status.NodeInfo.ContainerRuntimeVersion
	updateNode.operatingSystem = node.Status.NodeInfo.OperatingSystem
	updateNode.architecture = node.Status.NodeInfo.Architecture
	updateNode.nodeAddr = node.Status.Addresses
}

func UpdateNode(node *core.Node, ndm map[int]*list.List) NodeData {

	var nd NodeData
	for _, v := range ndm {
		if strings.Compare(v.Front().Value.(NodeData).name, node.ObjectMeta.Name) == 0 {
			v.Front().Value.(*NodeData).UpdateNodeData(node)
			log.Printf("node %s updated", v.Front().Value.(NodeData).name)
			nd = v.Front().Value.(NodeData)
			break
		}
		if strings.Compare(v.Front().Value.(NodeData).name, node.ObjectMeta.GenerateName) == 0 {
			v.Front().Value.(*NodeData).UpdateNodeData(node)
			log.Printf("node %s updated", v.Front().Value.(NodeData).name)
			nd = v.Front().Value.(NodeData)
			break
		}
	}
	return nd
}

func RemoveNode(node *core.Node, ndm map[int]*list.List) string {

	var nodeName string
	for _, v := range ndm {
		if strings.Compare(v.Front().Value.(NodeData).name, node.ObjectMeta.Name) == 0 {
			v.Remove(v.Front())
			log.Printf("node %s updated", v.Front().Value.(NodeData).name)
			nodeName = v.Front().Value.(NodeData).name
			break
		}
		if strings.Compare(v.Front().Value.(NodeData).name, node.ObjectMeta.GenerateName) == 0 {
			v.Remove(v.Front())
			log.Printf("node %s updated", v.Front().Value.(NodeData).name)
			nodeName = v.Front().Value.(NodeData).name
			break
		}
	}
	return nodeName
}

func (wh *WatchHandler) NodeWatch() {
	for {
		log.Printf("Watching over nodes starting")
		podsWatcher, err := wh.RestAPIClient.CoreV1().Nodes().Watch(metav1.ListOptions{Watch: true})
		if err != nil {
			log.Printf("Cannot wathching over pods. %v", err)
			time.Sleep(time.Duration(10) * time.Second)
			continue
		}
		podsChan := podsWatcher.ResultChan()
		for event := range podsChan {
			if node, ok := event.Object.(*core.Node); ok {
				switch event.Type {
				case "ADDED":
					id := CreateID()
					if wh.ndm[id] == nil {
						wh.ndm[id] = list.New()
					}
					nd := CreateNewNodeData(node)
					wh.ndm[id].PushBack(nd)
					wh.jsonReport.AddToJsonFormat(nd, NODE, CREATED)
				case "MODIFY":
					updateNode := UpdateNode(node, wh.ndm)
					wh.jsonReport.AddToJsonFormat(updateNode, NODE, UPDATED)
				case "DELETED":
					name := RemoveNode(node, wh.ndm)
					wh.jsonReport.AddToJsonFormat(name, NODE, DELETED)
				}
			} else {
				log.Printf("Got unexpected pod from chan: %t, %v", event.Object, event.Object)
			}
		}
		log.Printf("Wathching over pods ended")
	}
}
