package watch

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// namespaceWatch watch over namespaces
func (wh *WatchHandler) NamespaceWatch(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			logger.L().Ctx(ctx).Error("RECOVER NamespaceWatch", helpers.Interface("error", err), helpers.String("stack", string(debug.Stack())))
		}
	}()
	var lastWatchEventCreationTime time.Time
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
WatchLoop:
	for {
		logger.L().Info("Watching over namespaces starting")
		namespacesWatcher, err := wh.RestAPIClient.CoreV1().Namespaces().Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			logger.L().Ctx(ctx).Error("Failed watching over namespaces", helpers.Error(err))
			time.Sleep(1 * time.Second)
			continue
		}
		namespacesChan := namespacesWatcher.ResultChan()
		logger.L().Info("Watching over namespaces started")
	ChanLoop:
		for {
			var event watch.Event
			select {
			case event = <-namespacesChan:
			case <-newStateChan:
				namespacesWatcher.Stop()
				continue WatchLoop
			}

			if event.Type == watch.Error {
				logger.L().Ctx(ctx).Error("namespaces watch chan loop", helpers.Interface("error", event.Object))
				namespacesWatcher.Stop()
				break ChanLoop
			}
			if err := wh.NamespaceEventHandler(ctx, &event, lastWatchEventCreationTime); err != nil {
				break ChanLoop
			}
		}
		lastWatchEventCreationTime = time.Now()
		logger.L().Debug("Watching over namespaces ended - timeout")
	}
}
func (wh *WatchHandler) NamespaceEventHandler(ctx context.Context, event *watch.Event, lastWatchEventCreationTime time.Time) error {
	if namespace, ok := event.Object.(*corev1.Namespace); ok {
		namespace.ManagedFields = []metav1.ManagedFieldsEntry{}
		switch event.Type {
		case watch.Added:
			if namespace.CreationTimestamp.Time.Before(lastWatchEventCreationTime) {
				logger.L().Debug("namespace already exist, will not be reported", helpers.String("name", namespace.ObjectMeta.Name))
				return nil
			}
			id := CreateID()
			wh.namespacedm.init(id)
			wh.namespacedm.pushBack(id, namespace)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(namespace, NAMESPACES, CREATED)
		case watch.Modified:
			wh.UpdateNamespace(namespace)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(namespace, NAMESPACES, UPDATED)
		case watch.Deleted:
			wh.RemoveNamespace(namespace)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(namespace, NAMESPACES, DELETED)
		case watch.Bookmark: //only the resource version is changed but it's the same object
			return nil
		case watch.Error:
			logger.L().Ctx(ctx).Error("while watching over namespaces", helpers.Interface("error", event.Object))
			return fmt.Errorf("while watching over namespaces we got an error")
		}
	} else {
		return fmt.Errorf("got unexpected namespace from chan")
	}
	return nil
}

// UpdateNamespace update websocket when namespace is updated
func (wh *WatchHandler) UpdateNamespace(namespace *corev1.Namespace) {
	for _, id := range wh.namespacedm.getIDs() {
		front := wh.namespacedm.front(id)
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
	ids := wh.namespacedm.getIDs()
	for _, id := range ids {
		front := wh.namespacedm.front(id)
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
			wh.namespacedm.remove(id)
			return name
		}
	}
	return ""
}
