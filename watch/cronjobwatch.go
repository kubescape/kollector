package watch

import (
	"container/list"
	"runtime/debug"
	"time"

	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"golang.org/x/net/context"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// CronJobWatch watch over services
func (wh *WatchHandler) CronJobWatch(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			logger.L().Ctx(ctx).Error("RECOVER CronJobWatch", helpers.Interface("error", err), helpers.String("stack", string(debug.Stack())))
		}
	}()
	var lastWatchEventCreationTime time.Time
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
	for {
		logger.L().Info("Watching over cronjobs starting")
		cronjobWatcher, err := wh.RestAPIClient.BatchV1().CronJobs("").Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			logger.L().Ctx(ctx).Error("Cannot watch over cronjobs", helpers.Error(err))
			time.Sleep(3 * time.Second)
			continue
		}
		wh.handleCronJobWatch(ctx, cronjobWatcher, newStateChan, &lastWatchEventCreationTime)

		logger.L().Info("Watching over cronjobs ended - since we got timeout")
	}
}

func (wh *WatchHandler) handleCronJobWatch(ctx context.Context, cronjobWatcher watch.Interface, newStateChan <-chan bool, lastWatchEventCreationTime *time.Time) {
	cronjobChan := cronjobWatcher.ResultChan()
	cronJobIDs := make(map[string]int)
	logger.L().Info("Watching over cronjobs started")
	for {
		var event watch.Event
		select {
		case event = <-cronjobChan:
		case <-newStateChan:
			cronjobWatcher.Stop()
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if event.Type == watch.Error {
			logger.L().Ctx(ctx).Error("CronJob watch chan loop", helpers.Interface("error", event.Object))
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
					logger.L().Info("cronjob already exist, will not be reported", helpers.String("name", cronjob.Name))
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
				logger.L().Ctx(ctx).Error("while watching over cronjobs we got an error", helpers.Interface("error", event))
				*lastWatchEventCreationTime = time.Now()
				return
			}
		} else {
			*lastWatchEventCreationTime = time.Now()
			return
		}
	}
}
