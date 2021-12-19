package watch

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// namespaceWatch watch over namespaces
func (wh *WatchHandler) NamespaceWatch() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("RECOVER NamespaceWatch. error: %v\n %s", err, string(debug.Stack()))
		}
	}()
	newStateChan := make(chan bool)
	resourceMap := make(map[string]string)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
WatchLoop:
	for {
		glog.Infof("Watching over namespaces starting")
		namespacesWatcher, err := wh.RestAPIClient.CoreV1().Namespaces().Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			glog.Errorf("Failed watching over namespaces. %s", err.Error())
			time.Sleep(3 * time.Second)
			continue
		}
		wh.HandleDataMismatch("namespaces", resourceMap)
		namespacesChan := namespacesWatcher.ResultChan()
		glog.Infof("Watching over namespaces started")
	ChanLoop:
		for {
			var event watch.Event
			select {
			case event = <-namespacesChan:
			case <-newStateChan:
				namespacesWatcher.Stop()
				glog.Errorf("namespaces watch - newStateChan signal")
				continue WatchLoop
			}

			if event.Type == watch.Error {
				glog.Errorf("namespaces watch chan loop error: %v", event.Object)
				namespacesWatcher.Stop()
				break ChanLoop
			}
			if err := wh.NamespaceEventHandler(&event, resourceMap); err != nil {
				break ChanLoop
			}
		}
		glog.Infof("Watching over namespaces ended - timeout")
	}
}
func (wh *WatchHandler) NamespaceEventHandler(event *watch.Event, resourceMap map[string]string) error {
	if namespace, ok := event.Object.(*corev1.Namespace); ok {
		namespace.ManagedFields = []metav1.ManagedFieldsEntry{}
		switch event.Type {
		case "ADDED":
			resourceMap[string(namespace.GetUID())] = namespace.GetResourceVersion()
			id := CreateID()
			fmt.Printf("\n------------\n%v\n", wh.namespacedm)
			wh.namespacedm.Init(id)
			fmt.Printf("\n------------\n%v\n", wh.namespacedm)
			wh.namespacedm.PushBack(id, namespace)
			fmt.Printf("\n------------\n%v\n", wh.namespacedm)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(namespace, NAMESPACES, CREATED)
		case "MODIFY":
			resourceMap[string(namespace.GetUID())] = namespace.GetResourceVersion()
			wh.UpdateNamespace(namespace)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(namespace, NAMESPACES, UPDATED)
		case "DELETED":
			delete(resourceMap, string(namespace.GetUID()))
			wh.RemoveNamespace(namespace)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(namespace, NAMESPACES, DELETED)
		case "BOOKMARK": //only the resource version is changed but it's the same object
			return nil
		case "ERROR":
			log.Printf("while watching over namespaces we got an error ")
			return fmt.Errorf("while watching over namespaces we got an error")
		}
	} else {
		log.Printf("Got unexpected namespace from chan: %v", event.Object)
		return fmt.Errorf("got unexpected namespace from chan")
	}
	return nil
}

// UpdateNamespace update websocket when namespace is updated
func (wh *WatchHandler) UpdateNamespace(namespace *corev1.Namespace) {
	for _, id := range wh.namespacedm.GetIDs() {
		front := wh.namespacedm.Front(id)
		if front == nil || front.Value == nil {
			continue
		}
		namespaceData, ok := front.Value.(*corev1.Namespace)
		if !ok {
			continue
		}
		if strings.Compare(namespaceData.Name, namespaceData.Name) != 0 {
			continue
		}

	}
}

// RemoveNamespace update websocket when namespace is removed
func (wh *WatchHandler) RemoveNamespace(namespace *corev1.Namespace) string {
	ids := wh.namespacedm.GetIDs()
	for _, id := range ids {
		front := wh.namespacedm.Front(id)
		for front != nil && front.Value == nil {
			front = front.Next()
		}
		if front == nil {
			continue
		}
		namespaceData, ok := front.Value.(*corev1.Namespace)
		if !ok {
			continue
		}
		log.Printf("Checking namespace %s", namespaceData.Name)

		if strings.Compare(namespaceData.ObjectMeta.Name, namespace.Name) == 0 {
			name := namespaceData.ObjectMeta.Name
			wh.namespacedm.Remove(id)
			log.Printf("namespace %s removed", name)
			return name
		}
	}
	return ""
}
