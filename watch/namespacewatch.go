package watch

import (
	"fmt"
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
			glog.Errorf("RECOVER NamespaceWatch. error: %v\n %s", err, string(debug.Stack()))
		}
	}()
	var last_watch_event_creation_time time.Time
	newStateChan := make(chan bool)
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
			if err := wh.NamespaceEventHandler(&event, last_watch_event_creation_time); err != nil {
				break ChanLoop
			}
		}
		last_watch_event_creation_time = time.Now()
		glog.Infof("Watching over namespaces ended - timeout")
	}
}
func (wh *WatchHandler) NamespaceEventHandler(event *watch.Event, last_watch_event_creation_time time.Time) error {
	if namespace, ok := event.Object.(*corev1.Namespace); ok {
		namespace.ManagedFields = []metav1.ManagedFieldsEntry{}
		switch event.Type {
		case "ADDED":
			if namespace.CreationTimestamp.Time.Before(last_watch_event_creation_time) {
				glog.Infof("namespace %s already exist, will not reported", namespace.ObjectMeta.Name)
				return nil
			}
			id := CreateID()
			wh.namespacedm.Init(id)
			wh.namespacedm.PushBack(id, namespace)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(namespace, NAMESPACES, CREATED)
		case "MODIFY":
			wh.UpdateNamespace(namespace)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(namespace, NAMESPACES, UPDATED)
		case "DELETED":
			wh.RemoveNamespace(namespace)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(namespace, NAMESPACES, DELETED)
		case "BOOKMARK": //only the resource version is changed but it's the same object
			return nil
		case "ERROR":
			glog.Errorf("while watching over namespaces we got an error: %v", event)
			return fmt.Errorf("while watching over namespaces we got an error")
		}
	} else {
		glog.Errorf("Got unexpected namespace from chan: %v", event)
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

		if strings.Compare(namespaceData.ObjectMeta.Name, namespace.Name) == 0 {
			name := namespaceData.ObjectMeta.Name
			wh.namespacedm.Remove(id)
			glog.Infof("namespace %s removed", name)
			return name
		}
	}
	return ""
}
