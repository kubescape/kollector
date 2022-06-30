package watch

import (
	"container/list"
	"runtime/debug"
	"time"

	"github.com/golang/glog"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// CronJobWatch watch over servises
func (wh *WatchHandler) CronJobWatch() {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorf("RECOVER CronJobWatch. error: %v, stack: %s", err, debug.Stack())
		}
	}()
	var lastWatchEventCreationTime time.Time
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
	for {
		glog.Info("Watching over cronjobs starting")
		cronjobWatcher, err := wh.RestAPIClient.BatchV1().CronJobs("").Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			glog.Errorf("Cannot watch over cronjobs. %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		wh.handleCronJobWatch(cronjobWatcher, newStateChan, &lastWatchEventCreationTime)

		glog.Infof("Watching over cronjobs ended - since we got timeout")
	}
}

func (wh *WatchHandler) handleCronJobWatch(cronjobWatcher watch.Interface, newStateChan <-chan bool, lastWatchEventCreationTime *time.Time) {
	cronjobChan := cronjobWatcher.ResultChan()
	cronJobIDs := make(map[string]int)
	glog.Infof("Watching over cronjobs started")
	for {
		var event watch.Event
		select {
		case event = <-cronjobChan:
		case <-newStateChan:
			cronjobWatcher.Stop()
			glog.Errorf("CronJob watch - newStateChan signal")
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if event.Type == watch.Error {
			glog.Errorf("CronJob watch chan loop error: %v", event.Object)
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if cronjob, ok := event.Object.(*batchv1.CronJob); ok {
			if !wh.isNamespaceWatched(cronjob.Namespace) {
				continue
			}
			cronjob.Kind = "CronJob"
			if cronjob.APIVersion == "" {
				cronjob.APIVersion = "batch/v1"
			}
			// handle cases like microservice
			cronjob.ManagedFields = []metav1.ManagedFieldsEntry{}
			switch event.Type {
			case watch.Added:
				if cronjob.CreationTimestamp.Time.Before(*lastWatchEventCreationTime) {
					glog.Infof("cronjob %s already exist, will not be reported", cronjob.Name)
					continue
				}
				id := CreateID()
				od := OwnerDet{
					Name:      cronjob.Name,
					Kind:      cronjob.Kind,
					OwnerData: cronjob,
				}
				wh.pdm[id] = list.New()
				nms := MicroServiceData{Pod: &v1.Pod{Spec: cronjob.Spec.JobTemplate.Spec.Template.Spec, TypeMeta: cronjob.TypeMeta, ObjectMeta: cronjob.ObjectMeta},
					Owner: od, PodSpecId: id}
				// wh.cjm[id].PushBack(nms)
				wh.jsonReport.AddToJsonFormat(nms, MICROSERVICES, CREATED)
				cronJobIDs[string(cronjob.GetUID())] = id
				informNewDataArrive(wh)
			case watch.Modified:
				od := OwnerDet{
					Name:      cronjob.Name,
					Kind:      cronjob.Kind,
					OwnerData: cronjob,
				}
				nms := MicroServiceData{Pod: &v1.Pod{Spec: cronjob.Spec.JobTemplate.Spec.Template.Spec, TypeMeta: cronjob.TypeMeta, ObjectMeta: cronjob.ObjectMeta},
					Owner: od, PodSpecId: cronJobIDs[string(cronjob.GetUID())]}
				wh.jsonReport.AddToJsonFormat(nms, MICROSERVICES, UPDATED)
				informNewDataArrive(wh)
			case watch.Deleted:
				delete(cronJobIDs, string(cronjob.GetUID()))
				od := OwnerDet{
					Name:      cronjob.Name,
					Kind:      cronjob.Kind,
					OwnerData: cronjob,
				}
				nms := MicroServiceData{Pod: &v1.Pod{Spec: cronjob.Spec.JobTemplate.Spec.Template.Spec, TypeMeta: cronjob.TypeMeta, ObjectMeta: cronjob.ObjectMeta},
					Owner: od, PodSpecId: cronJobIDs[string(cronjob.GetUID())]}
				wh.jsonReport.AddToJsonFormat(nms, MICROSERVICES, DELETED)
				informNewDataArrive(wh)
			case watch.Bookmark: //only the resource version is changed but it's the same workload
				continue
			case watch.Error:
				glog.Errorf("while watching over cronjobs we got an error: %v", event)
				*lastWatchEventCreationTime = time.Now()
				return
			}
		} else {
			*lastWatchEventCreationTime = time.Now()
			return
		}
	}
}
