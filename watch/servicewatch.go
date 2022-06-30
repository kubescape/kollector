package watch

import (
	"container/list"
	"runtime/debug"
	"strings"
	"time"

	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type ServiceData struct {
	Service *core.Service `json:",inline"`
}

func UpdateService(service *core.Service, sdm map[int]*list.List) string {
	for _, v := range sdm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(ServiceData).Service.ObjectMeta.Name, service.ObjectMeta.Name) == 0 {
			*v.Front().Value.(ServiceData).Service = *service
			glog.Infof("service %s updated", v.Front().Value.(ServiceData).Service.ObjectMeta.Name)
			return v.Front().Value.(ServiceData).Service.ObjectMeta.Name
		}
		if strings.Compare(v.Front().Value.(ServiceData).Service.ObjectMeta.GenerateName, service.ObjectMeta.Name) == 0 {
			*v.Front().Value.(ServiceData).Service = *service
			glog.Infof("service %s updated", v.Front().Value.(ServiceData).Service.ObjectMeta.Name)
			return v.Front().Value.(ServiceData).Service.ObjectMeta.Name
		}
	}
	return ""
}

// RemoveService update websocket when service is removed
func RemoveService(service *core.Service, sdm map[int]*list.List) string {
	for _, v := range sdm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(ServiceData).Service.ObjectMeta.Name, service.ObjectMeta.Name) == 0 {
			name := v.Front().Value.(ServiceData).Service.ObjectMeta.Name
			v.Remove(v.Front())
			glog.Infof("service %s removed", name)
			return name
		}
		if strings.Compare(v.Front().Value.(ServiceData).Service.ObjectMeta.GenerateName, service.ObjectMeta.Name) == 0 {
			gName := v.Front().Value.(ServiceData).Service.ObjectMeta.Name
			v.Remove(v.Front())
			glog.Infof("service %s removed", gName)
			return gName
		}
	}
	return ""
}

// ServiceWatch watch over servises
func (wh *WatchHandler) ServiceWatch() {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorf("RECOVER ServiceWatch. error: %v, stack: %s", err, debug.Stack())
		}
	}()
	var lastWatchEventCreationTime time.Time
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
	for {
		glog.Info("Watching over services starting")
		serviceWatcher, err := wh.RestAPIClient.CoreV1().Services("").Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			glog.Errorf("Cannot watch over services. %v", err)
			time.Sleep(3 * time.Second)
			lastWatchEventCreationTime = time.Now()
			continue
		}
		wh.handleServiceWatch(serviceWatcher, newStateChan, &lastWatchEventCreationTime)

		glog.Infof("Watching over services ended - since we got timeout")
	}
}

func (wh *WatchHandler) handleServiceWatch(serviceWatcher watch.Interface, newStateChan <-chan bool, lastWatchEventCreationTime *time.Time) {
	serviceChan := serviceWatcher.ResultChan()
	glog.Infof("Watching over services started")
	for {
		var event watch.Event
		select {
		case event = <-serviceChan:
		case <-newStateChan:
			serviceWatcher.Stop()
			glog.Errorf("Service watch - newStateChan signal")
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if event.Type == watch.Error {
			glog.Errorf("Service watch chan loop error: %v", event.Object)
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if service, ok := event.Object.(*core.Service); ok {
			if !wh.isNamespaceWatched(service.Namespace) {
				continue
			}
			service.ManagedFields = []metav1.ManagedFieldsEntry{}
			switch event.Type {
			case "ADDED":
				if service.CreationTimestamp.Time.Before(*lastWatchEventCreationTime) {
					glog.Infof("service %s already exist, will not be reported", service.Name)
					continue
				}
				id := CreateID()
				if wh.sdm[id] == nil {
					wh.sdm[id] = list.New()
				}
				sd := ServiceData{Service: service}
				wh.sdm[id].PushBack(sd)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(service, SERVICES, CREATED)
			case "MODIFY":
				UpdateService(service, wh.sdm)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(service, SERVICES, UPDATED)
			case "DELETED":
				RemoveService(service, wh.sdm)
				informNewDataArrive(wh)
				wh.jsonReport.AddToJsonFormat(service, SERVICES, DELETED)
			case "BOOKMARK": //only the resource version is changed but it's the same workload
				continue
			case "ERROR":
				glog.Errorf("while watching over services we got an error: %v", event)
				*lastWatchEventCreationTime = time.Now()
				return
			}
		} else {
			*lastWatchEventCreationTime = time.Now()
			return
		}
	}
}
