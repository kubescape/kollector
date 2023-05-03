package watch

import (
	"container/list"
	"runtime/debug"
	"strings"
	"time"

	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"golang.org/x/net/context"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type serviceData struct {
	Service *core.Service `json:",inline"`
}

// ServiceWatch watch over services
func (wh *WatchHandler) ServiceWatch(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			logger.L().Ctx(ctx).Error("RECOVER ServiceWatch", helpers.Interface("error", err), helpers.String("stack", string(debug.Stack())))
		}
	}()
	var lastWatchEventCreationTime time.Time
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
	for {
		logger.L().Info("Watching over services starting")
		serviceWatcher, err := wh.RestAPIClient.CoreV1().Services("").Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			time.Sleep(1 * time.Second)
			lastWatchEventCreationTime = time.Now()
			continue
		}
		wh.handleServiceWatch(serviceWatcher, newStateChan, &lastWatchEventCreationTime)
	}
}
func updateService(service *core.Service, sdm map[int]*list.List) string {
	for _, v := range sdm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(serviceData).Service.ObjectMeta.Name, service.ObjectMeta.Name) == 0 {
			*v.Front().Value.(serviceData).Service = *service
			return v.Front().Value.(serviceData).Service.ObjectMeta.Name
		}
		if strings.Compare(v.Front().Value.(serviceData).Service.ObjectMeta.GenerateName, service.ObjectMeta.Name) == 0 {
			*v.Front().Value.(serviceData).Service = *service
			return v.Front().Value.(serviceData).Service.ObjectMeta.Name
		}
	}
	return ""
}

// RemoveService update websocket when service is removed
func removeService(service *core.Service, sdm map[int]*list.List) string {
	for _, v := range sdm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(serviceData).Service.ObjectMeta.Name, service.ObjectMeta.Name) == 0 {
			name := v.Front().Value.(serviceData).Service.ObjectMeta.Name
			v.Remove(v.Front())
			return name
		}
		if strings.Compare(v.Front().Value.(serviceData).Service.ObjectMeta.GenerateName, service.ObjectMeta.Name) == 0 {
			gName := v.Front().Value.(serviceData).Service.ObjectMeta.Name
			v.Remove(v.Front())
			return gName
		}
	}
	return ""
}

func (wh *WatchHandler) handleServiceWatch(serviceWatcher watch.Interface, newStateChan <-chan bool, lastWatchEventCreationTime *time.Time) {
	serviceChan := serviceWatcher.ResultChan()
	logger.L().Info("Watching over services started")
	for {
		var event watch.Event
		select {
		case event = <-serviceChan:
		case <-newStateChan:
			serviceWatcher.Stop()
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if event.Type == watch.Error {
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if service, ok := event.Object.(*core.Service); ok {
			if !wh.isNamespaceWatched(service.Namespace) {
				continue
			}
			service.ManagedFields = []metav1.ManagedFieldsEntry{}
			switch event.Type {
			case watch.Added:
				if service.CreationTimestamp.Time.Before(*lastWatchEventCreationTime) {
					continue
				}
				id := CreateID()
				if wh.sdm[id] == nil {
					wh.sdm[id] = list.New()
				}
				sd := serviceData{Service: service}
				wh.sdm[id].PushBack(sd)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(service, SERVICES, CREATED)
			case watch.Modified:
				updateService(service, wh.sdm)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(service, SERVICES, UPDATED)
			case watch.Deleted:
				removeService(service, wh.sdm)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(service, SERVICES, DELETED)
			case watch.Bookmark: //only the resource version is changed but it's the same workload
				continue
			case watch.Error:
				*lastWatchEventCreationTime = time.Now()
				return
			}
		} else {
			*lastWatchEventCreationTime = time.Now()
			return
		}
	}
}
