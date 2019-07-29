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
	Node *core.Node `json:"data"`
}

func UpdateNode(node *core.Node, ndm map[int]*list.List) string {
	for _, v := range ndm {
		if strings.Compare(v.Front().Value.(NodeData).Node.ObjectMeta.Name, node.ObjectMeta.Name) == 0 {
			*v.Front().Value.(NodeData).Node = *node
			log.Printf("node %s updated", v.Front().Value.(NodeData).Node.ObjectMeta.Name)
			return v.Front().Value.(NodeData).Node.ObjectMeta.Name
		}
		if strings.Compare(v.Front().Value.(NodeData).Node.ObjectMeta.GenerateName, node.ObjectMeta.Name) == 0 {
			*v.Front().Value.(NodeData).Node = *node
			log.Printf("node %s updated", v.Front().Value.(NodeData).Node.ObjectMeta.Name)
			return v.Front().Value.(NodeData).Node.ObjectMeta.Name
		}
	}
	return ""
}

func RemoveNode(node *core.Node, ndm map[int]*list.List) string {
	for _, v := range ndm {
		if strings.Compare(v.Front().Value.(NodeData).Node.ObjectMeta.Name, node.ObjectMeta.Name) == 0 {
			v.Remove(v.Front())
			log.Printf("node %s removed", v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name)
			return v.Front().Value.(NodeData).Node.ObjectMeta.Name
		}
		if strings.Compare(v.Front().Value.(NodeData).Node.ObjectMeta.GenerateName, node.ObjectMeta.Name) == 0 {
			v.Remove(v.Front())
			log.Printf("node %s removed", v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name)
			return v.Front().Value.(NodeData).Node.ObjectMeta.Name
		}
	}
	return ""
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
					nd := NodeData{Node: node}
					wh.ndm[id].PushBack(nd)
					wh.jsonReport.AddToJsonFormat(nd, NODE, CREATED)
				case "MODIFY":
					name := UpdateNode(node, wh.ndm)
					wh.jsonReport.AddToJsonFormat(name, NODE, UPDATED)
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
