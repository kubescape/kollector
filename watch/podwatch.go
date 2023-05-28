package watch

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

type OwnerDet struct {
	Name      string      `json:"name"`
	Kind      string      `json:"kind"`
	OwnerData interface{} `json:"ownerData,omitempty"`
}
type CRDOwnerData struct {
	metav1.TypeMeta
}
type OwnerDetNameAndKindOnly struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type MicroServiceData struct {
	*core.Pod `json:",inline"`
	Owner     OwnerDet `json:"uptreeOwner"`
	PodSpecId int      `json:"podSpecId"`
}

type PodDataForExistMicroService struct {
	PodName           string                  `json:"podName"`
	NodeName          string                  `json:"nodeName"`
	PodIP             string                  `json:"podIP"`
	Namespace         string                  `json:"namespace,omitempty"`
	Owner             OwnerDetNameAndKindOnly `json:"uptreeOwner"`
	PodStatus         string                  `json:"podStatus"`
	CreationTimestamp string                  `json:"startedAt"`
	DeletionTimestamp string                  `json:"terminatedAt,omitempty"`
}

type ScanNewImageData struct {
	Pod        *core.Pod
	Owner      *OwnerDet
	PodsNumber int
}

var (
	collectorCreationTime         time.Time
	scanNotificationCandidateList []*ScanNewImageData
	maxTailLines                  int64 = 50 // Max number of lines to return from the end of the log of a crashed container
)

// PodWatch - an infinite loop which will observe changes in pods and acts accordingly
func (wh *WatchHandler) PodWatch(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			logger.L().Ctx(ctx).Error("RECOVER ListenerAndSender", helpers.Interface("error", err), helpers.String("stack", string(debug.Stack())))
		}
	}()
	var lastWatchEventCreationTime time.Time
	collectorCreationTime = time.Now()
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
	for {
		logger.L().Ctx(ctx).Info("Watching over pods starting")
		podsWatcher, err := wh.RestAPIClient.CoreV1().Pods("").Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		wh.handlePodWatch(ctx, podsWatcher, newStateChan, &lastWatchEventCreationTime)
	}
}
func isPodAlreadyExistInScanCandidateList(ctx context.Context, od *OwnerDet, pod *core.Pod) (bool, int) {
	for i, data := range scanNotificationCandidateList {
		if pod.GetNamespace() == data.Pod.GetNamespace() && data.Owner.Name == od.Name && data.Owner.Kind == od.Kind {
			logger.L().Ctx(ctx).Debug("pod already exist", helpers.String("name", pod.Name))
			return true, i
		}
	}
	return false, -1
}

func addPodScanNotificationCandidateList(ctx context.Context, od *OwnerDet, pod *core.Pod) {
	if exist, index := isPodAlreadyExistInScanCandidateList(ctx, od, pod); !exist {
		logger.L().Debug("pod added to scan list candidate", helpers.String("name", pod.Name))
		nms := &ScanNewImageData{Pod: pod, Owner: od, PodsNumber: 1}
		scanNotificationCandidateList = append(scanNotificationCandidateList, nms)
	} else {
		scanNotificationCandidateList[index].PodsNumber++
	}
}

func removePodScanNotificationCandidateList(od *OwnerDet, pod *core.Pod) {
	for i := range scanNotificationCandidateList {
		data := scanNotificationCandidateList[i]
		if pod.GetNamespace() == data.Pod.GetNamespace() && data.Owner.Name == od.Name && data.Owner.Kind == od.Kind {
			scanNotificationCandidateList[i].PodsNumber--
			if scanNotificationCandidateList[i].PodsNumber == 0 {
				logger.L().Debug("pod removed from scan list candidate", helpers.String("name", pod.Name))
				scanNotificationCandidateList = append(scanNotificationCandidateList[:i], scanNotificationCandidateList[i+1:]...)
				return
			}
		}
	}
}

func isContainersIDSChanged(podWithNewState []core.ContainerStatus, oldPod []core.ContainerStatus) bool {
	if len(podWithNewState) > len(oldPod) {
		return true
	}

	length := len(podWithNewState)
	for i := 0; i < length; i++ {
		if podWithNewState[i].ImageID != oldPod[i].ImageID {
			return true
		}
	}
	return false
}

func isPodIsTheNewOne(pod *core.Pod) bool {
	return pod.CreationTimestamp.Time.Equal(pod.CreationTimestamp.Time) || pod.CreationTimestamp.After(pod.CreationTimestamp.Time)
}

func checkNotificationCandidateList(pod *core.Pod, od *OwnerDet, podStatus string) bool {
	if podStatus != "Running" {
		return false
	}
	for i, data := range scanNotificationCandidateList {
		if pod.GetNamespace() == data.Pod.GetNamespace() && data.Owner.Name == od.Name && data.Owner.Kind == od.Kind {
			if isPodIsTheNewOne(data.Pod) && isContainersIDSChanged(pod.Status.ContainerStatuses, data.Pod.Status.ContainerStatuses) {
				scanNotificationCandidateList[i].Pod = pod
				return true
			} else {
				return false
			}
		}
	}
	return false
}

func (wh *WatchHandler) handlePodWatch(ctx context.Context, podsWatcher watch.Interface, newStateChan <-chan bool, lastWatchEventCreationTime *time.Time) {
	for {
		var event watch.Event
		var chanActive bool
		select {
		case event, chanActive = <-podsWatcher.ResultChan():
			if !chanActive {
				podsWatcher.Stop()
				*lastWatchEventCreationTime = time.Now()
				return
			}
		case <-newStateChan:
			podsWatcher.Stop()
			*lastWatchEventCreationTime = time.Now()
			return
		}
		if event.Type == watch.Error {
			logger.L().Ctx(ctx).Error("Pod watch chan loop", helpers.Interface("error", event.Object))
			podsWatcher.Stop()
			*lastWatchEventCreationTime = time.Now()
			return
		}
		pod, ok := event.Object.(*core.Pod)
		if !ok {
			logger.L().Ctx(ctx).Error("Watch error: cannot convert to core.Pod", helpers.Interface("error", event))
			continue
		}
		if !wh.isNamespaceWatched(pod.Namespace) {
			continue
		}
		pod.ManagedFields = []metav1.ManagedFieldsEntry{}
		podName := pod.ObjectMeta.Name
		if podName == "" {
			podName = pod.ObjectMeta.GenerateName
		}
		podStatus := getPodStatus(pod)
		logger.L().Ctx(ctx).Debug("pod", helpers.String("name", podName), helpers.String("status", podStatus), helpers.String("namespace", pod.Namespace), helpers.String("node", pod.Spec.NodeName))
		od, err := GetAncestorOfPod(ctx, pod, wh)
		if err != nil {
			*lastWatchEventCreationTime = time.Now()
			break
		}
		switch event.Type {
		case watch.Added:
			if pod.CreationTimestamp.Time.Before(*lastWatchEventCreationTime) {
				continue
			}
			first := true
			id, runningPodNum := isPodSpecAlreadyExist(&od, pod.Namespace, wh.pdm)
			if runningPodNum <= 1 {
				// when a new pod microservice (a new pod that is running first in the cluster) is found
				// we want to scan its vulnerabilities so we will use the trigger mechanism to do it
				wh.pdm[id] = list.New()
				nms := MicroServiceData{Pod: pod, Owner: od, PodSpecId: id}
				wh.pdm[id].PushBack(nms)
				if wh.isNamespaceWatched(pod.Namespace) {
					wh.jsonReport.AddToJsonFormat(nms, MICROSERVICES, CREATED)
				}

			} else { // Check if pod is already reported
				if wh.pdm[id].Front() != nil {
					element := wh.pdm[id].Front().Next()
					for element != nil {
						if element.Value.(PodDataForExistMicroService).PodName == podName {
							first = false
							break
						}
						element = element.Next()
					}
				}
			}
			if !first {
				*lastWatchEventCreationTime = time.Now()
				break
			}

			newPod := PodDataForExistMicroService{
				PodName:   podName,
				NodeName:  pod.Spec.NodeName,
				PodIP:     pod.Status.PodIP,
				Namespace: pod.ObjectMeta.Namespace,
				Owner: OwnerDetNameAndKindOnly{
					Name: od.Name,
					Kind: od.Kind,
				},
				PodStatus:         podStatus,
				CreationTimestamp: pod.CreationTimestamp.Time.UTC().Format(time.RFC3339),
			}
			wh.pdm[id].PushBack(newPod)
			if wh.isNamespaceWatched(pod.Namespace) {
				wh.jsonReport.AddToJsonFormat(newPod, PODS, CREATED)
				informNewDataArrive(wh)
			}
			if pod.CreationTimestamp.Time.After(collectorCreationTime) {
				addPodScanNotificationCandidateList(ctx, &od, pod)
			}
		case watch.Modified:
			if checkNotificationCandidateList(pod, &od, podStatus) {
				if err := wh.notifyUpdates.notifyNewMicroServiceCreatedInTheCluster(pod.Namespace, od.Kind, od.Name); err != nil {
					logger.L().Ctx(ctx).Error("failed to notify updates", helpers.Error(err))
				}
			}
			if !wh.isNamespaceWatched(pod.Namespace) {
				continue
			}
			if pod.DeletionTimestamp != nil { // the pod is terminating
				*lastWatchEventCreationTime = time.Now()
				break
			}
			podSpecID, newPodData := wh.updatePod(pod, wh.pdm, podStatus)
			if podSpecID > -2 {
				logger.L().Ctx(ctx).Debug("Pod Modified", helpers.String("name", podName), helpers.String("status", podStatus), helpers.String("namespace", pod.Namespace), helpers.String("node", pod.Spec.NodeName))
				if strings.Contains(strings.ToLower(podStatus), "crashloop") {
					wh.logPodInCrashLoop(ctx, pod)
				}
				wh.jsonReport.AddToJsonFormat(newPodData, PODS, UPDATED)
			}
			if podSpecID > -1 {
				wh.jsonReport.AddToJsonFormat(wh.pdm[podSpecID].Front().Value.(MicroServiceData), MICROSERVICES, UPDATED)
			}
			if podSpecID > -2 {
				informNewDataArrive(wh)
			}
		case watch.Deleted:
			removePodScanNotificationCandidateList(&od, pod)
			if !wh.isNamespaceWatched(pod.Namespace) {
				continue
			}
			wh.DeletePod(ctx, pod, podName)
		case watch.Bookmark:
			logger.L().Ctx(ctx).Debug("Pod Bookmark", helpers.String("name", podName), helpers.String("status", podStatus), helpers.String("namespace", pod.Namespace), helpers.String("node", pod.Spec.NodeName))
		case watch.Error:
			removePodScanNotificationCandidateList(&od, pod)
			logger.L().Ctx(ctx).Debug("Pod Error", helpers.String("name", podName), helpers.String("status", podStatus), helpers.String("namespace", pod.Namespace), helpers.String("node", pod.Spec.NodeName))
			*lastWatchEventCreationTime = time.Now()
			return
		}
	}
}

// logs all container logs of a pod in crash loop. In case the RestartCount of one of the containers is greater than 2, skipping it.
func (wh *WatchHandler) logPodInCrashLoop(ctx context.Context, pod *core.Pod) {
	ctx, span := otel.Tracer("").Start(ctx, "logPodInCrashLoop", trace.WithAttributes(attribute.String("pod", pod.Name)))
	defer span.End()

	containerSlice := make([]core.ContainerStatus, 0, len(pod.Status.ContainerStatuses)+len(pod.Status.InitContainerStatuses))
	containerSlice = append(containerSlice, pod.Status.ContainerStatuses...)
	containerSlice = append(containerSlice, pod.Status.InitContainerStatuses...)

	for contIdx := range containerSlice {
		contName := containerSlice[contIdx].Name

		if containerSlice[contIdx].RestartCount > 2 {
			logger.L().Info(fmt.Sprintf("skipping container %s with RestartCount > 2", contName))
			continue
		}

		// container details for logging
		containerDetails := []helpers.IDetails{
			helpers.String("containerName", contName),
			helpers.Int("restartCount", int(containerSlice[contIdx].RestartCount)),
		}
		if state, err := containerSlice[contIdx].State.Marshal(); err != nil {
			logger.L().Ctx(ctx).Error("failed to marshal container State", helpers.String("containerName", contName), helpers.Error(err))
		} else {
			containerDetails = append(containerDetails, helpers.String("state", string(state)))
		}

		if lastState, err := containerSlice[contIdx].LastTerminationState.Marshal(); err != nil {
			logger.L().Ctx(ctx).Error("failed to marshal container LastTerminationState", helpers.String("containerName", contName), helpers.Error(err))
		} else {
			containerDetails = append(containerDetails, helpers.String("lastState", string(lastState)))
		}

		// previous container logs
		podLogOpts := core.PodLogOptions{Previous: true, Timestamps: true, Container: contName, TailLines: &maxTailLines}
		logsObj := wh.K8sApi.KubernetesClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
		readerObj, err := logsObj.Stream(wh.K8sApi.Context)

		span.AddEvent("getting previous container logs", trace.WithAttributes(attribute.String("containerName", contName)))

		if err != nil {
			logger.L().Ctx(ctx).Error("failed to get previous logs stream of a crashed pod container", helpers.String("containerName", contName), helpers.Error(err))
		} else {
			if logs, err := io.ReadAll(readerObj); err != nil {
				logger.L().Ctx(ctx).Error("failed to read previous logs stream of a crashed pod container", helpers.String("containerName", contName), helpers.Error(err))
			} else {
				logger.L().Ctx(ctx).Error(fmt.Sprintf("previous logs of a crashed pod container:\n %s", string(logs)), containerDetails...)
			}
		}

		// current container logs
		span.AddEvent("getting current container logs", trace.WithAttributes(attribute.String("containerName", contName)))
		podLogOpts.Previous = false
		logsObj = wh.K8sApi.KubernetesClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
		readerObj, err = logsObj.Stream(wh.K8sApi.Context)
		if err != nil {
			logger.L().Ctx(ctx).Error("failed to get logs stream of a crashed pod container", helpers.String("containerName", contName), helpers.Error(err))
		} else {
			if logs, err := io.ReadAll(readerObj); err != nil {
				logger.L().Ctx(ctx).Error("failed to read logs stream of a crashed pod container", helpers.String("containerName", contName), helpers.Error(err))
			} else {
				logger.L().Ctx(ctx).Error(fmt.Sprintf("logs of a crashed pod container:\n %s", string(logs)), containerDetails...)
			}
		}
	}
}

// DeletePod delete a pod
func (wh *WatchHandler) DeletePod(ctx context.Context, pod *core.Pod, podName string) {
	podStatus := "Terminating"
	podSpecID, removeMicroServiceAsWell, owner := wh.RemovePod(pod, wh.pdm)
	if podSpecID == -1 {
		return
	}
	logger.L().Ctx(ctx).Debug("Pod Deleted", helpers.String("name", podName), helpers.String("status", podStatus), helpers.String("namespace", pod.Namespace), helpers.String("node", pod.Spec.NodeName))
	np := PodDataForExistMicroService{PodName: pod.ObjectMeta.Name, NodeName: pod.Spec.NodeName, PodIP: pod.Status.PodIP, Namespace: pod.ObjectMeta.Namespace, Owner: OwnerDetNameAndKindOnly{Name: owner.Name, Kind: owner.Kind}, PodStatus: podStatus, CreationTimestamp: pod.CreationTimestamp.Time.UTC().Format(time.RFC3339)}
	if pod.DeletionTimestamp != nil {
		np.DeletionTimestamp = pod.DeletionTimestamp.Time.UTC().Format(time.RFC3339)
	}
	wh.jsonReport.AddToJsonFormat(np, PODS, DELETED)
	if removeMicroServiceAsWell {
		nms := MicroServiceData{Pod: pod, Owner: owner, PodSpecId: podSpecID}
		wh.jsonReport.AddToJsonFormat(nms, MICROSERVICES, DELETED)
	}
	informNewDataArrive(wh)
}

// IsPodExist check
func IsPodExist(pod *core.Pod, pdm map[int]*list.List) bool {
	for _, v := range pdm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name == pod.ObjectMeta.Name {
			return true
		}
		if v.Front().Value.(MicroServiceData).Pod.ObjectMeta.GenerateName == pod.ObjectMeta.Name {
			return true
		}
		for e := ids.Ids.Front().Next(); e != nil; e = e.Next() {
			if e.Value.(PodDataForExistMicroService).PodName == pod.ObjectMeta.Name {
				return true
			}
		}
	}
	return false
}

func extractPodSpecFromOwner(ownerData interface{}) interface{} {
	if ownerData != nil {
		jsonBytes, err := json.Marshal(ownerData)
		if err != nil {
			return ownerData
		}
		fd := make(map[string]interface{})
		if err := json.Unmarshal(jsonBytes, &fd); err != nil {
			return ownerData
		}
		if fdSpec, ok := fd["spec"]; ok {
			return fdSpec
		}

	}
	return ownerData
}

func isPodSpecAlreadyExist(podOwner *OwnerDet, namespace string, pdm map[int]*list.List) (int, int) {
	newSpec := extractPodSpecFromOwner(podOwner.OwnerData)
	for _, v := range pdm {
		if v == nil || v.Len() <= 1 {
			continue
		}
		p := v.Front().Value.(MicroServiceData)
		existsSpec := extractPodSpecFromOwner(p.Owner.OwnerData)
		// In addition, in case we didn't change the podspec of the OwnerReference of the pod, we cant count on the owner labels changes
		//  but on the labels / volumes of the actual pod we got to identify the changes
		if p.ObjectMeta.Namespace == namespace && reflect.DeepEqual(newSpec, existsSpec) {
			return v.Front().Value.(MicroServiceData).PodSpecId, v.Len()
		}
	}
	return CreateID(), 0
}

// GetOwnerData - get the data of pod owner
func GetOwnerData(ctx context.Context, name string, kind string, apiVersion string, namespace string, wh *WatchHandler) interface{} {
	switch kind {
	case "Deployment":
		options := metav1.GetOptions{}
		depDet, err := wh.RestAPIClient.AppsV1().Deployments(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			logger.L().Ctx(ctx).Error("GetOwnerData Deployments", helpers.Error(err))
			return nil
		}
		depDet.TypeMeta.Kind = kind
		depDet.TypeMeta.APIVersion = apiVersion
		depDet.ManagedFields = []metav1.ManagedFieldsEntry{}
		return depDet
	case "DaemonSet":
		options := metav1.GetOptions{}
		daemSetDet, err := wh.RestAPIClient.AppsV1().DaemonSets(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			logger.L().Ctx(ctx).Error("GetOwnerData DaemonSet", helpers.Error(err))
			return nil
		}
		daemSetDet.TypeMeta.Kind = kind
		daemSetDet.TypeMeta.APIVersion = apiVersion
		daemSetDet.ManagedFields = []metav1.ManagedFieldsEntry{}
		return daemSetDet
	case "StatefulSet":
		options := metav1.GetOptions{}
		statSetDet, err := wh.RestAPIClient.AppsV1().StatefulSets(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			logger.L().Ctx(ctx).Error("GetOwnerData StatefulSet", helpers.Error(err))
			return nil
		}
		statSetDet.TypeMeta.Kind = kind
		statSetDet.TypeMeta.APIVersion = apiVersion
		statSetDet.ManagedFields = []metav1.ManagedFieldsEntry{}
		return statSetDet
	case "Job":
		options := metav1.GetOptions{}
		jobDet, err := wh.RestAPIClient.BatchV1().Jobs(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			logger.L().Ctx(ctx).Error("GetOwnerData Job", helpers.Error(err))
			return nil
		}
		jobDet.TypeMeta.Kind = kind
		jobDet.TypeMeta.APIVersion = apiVersion
		jobDet.ManagedFields = []metav1.ManagedFieldsEntry{}
		return jobDet
	case "CronJob":
		options := metav1.GetOptions{}
		cronJobDet, err := wh.RestAPIClient.BatchV1().CronJobs(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			logger.L().Ctx(ctx).Error("GetOwnerData CronJob", helpers.Error(err))
			return nil
		}
		cronJobDet.TypeMeta.Kind = kind
		cronJobDet.TypeMeta.APIVersion = apiVersion
		cronJobDet.ManagedFields = []metav1.ManagedFieldsEntry{}
		return cronJobDet
	case "Pod":
		options := metav1.GetOptions{}
		podDet, err := wh.RestAPIClient.CoreV1().Pods(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			logger.L().Ctx(ctx).Error("GetOwnerData Pod", helpers.Error(err))
			return nil
		}
		podDet.TypeMeta.Kind = kind
		podDet.TypeMeta.APIVersion = apiVersion
		podDet.ManagedFields = []metav1.ManagedFieldsEntry{}
		return podDet

	default:
		if wh.extensionsClient == nil {
			return nil
		}
		options := metav1.ListOptions{}
		crds, err := wh.extensionsClient.CustomResourceDefinitions().List(context.Background(), options)
		if err != nil {
			logger.L().Ctx(ctx).Error("GetOwnerData CustomResourceDefinitions", helpers.Error(err))
			return nil
		}
		for crdIdx := range crds.Items {
			if crds.Items[crdIdx].Status.AcceptedNames.Kind == kind {
				return CRDOwnerData{
					metav1.TypeMeta{Kind: crds.Items[crdIdx].Kind,
						APIVersion: apiVersion,
					}}
			}
		}
	}

	return nil
}

func GetAncestorFromLocalPodsList(pod *core.Pod, wh *WatchHandler) (*OwnerDet, error) {
	for _, v := range wh.pdm {
		if v == nil || v.Front() == nil {
			continue
		}
		element := v.Front().Next()
		for element != nil {
			if strings.Compare(element.Value.(PodDataForExistMicroService).PodName, pod.ObjectMeta.Name) == 0 {
				pdm := v.Front().Value.(MicroServiceData)
				return &pdm.Owner, nil
			}
			element = element.Next()
		}
	}
	return nil, fmt.Errorf("error getting owner reference")
}

func GetAncestorOfPod(ctx context.Context, pod *core.Pod, wh *WatchHandler) (OwnerDet, error) {
	od := OwnerDet{}

	if pod.OwnerReferences != nil {
		switch pod.OwnerReferences[0].Kind {
		case "Node":
			od.Name = pod.ObjectMeta.Name
			od.Kind = "Pod"
			od.OwnerData = GetOwnerData(ctx, pod.ObjectMeta.Name, od.Kind, pod.APIVersion, pod.ObjectMeta.Namespace, wh)
			if crd, ok := od.OwnerData.(CRDOwnerData); ok {
				od.Kind = crd.Kind
			}
		case "ReplicaSet":
			repItem, err := wh.RestAPIClient.AppsV1().ReplicaSets(pod.ObjectMeta.Namespace).Get(globalHTTPContext, pod.OwnerReferences[0].Name, metav1.GetOptions{})
			if err != nil {
				if localOD, inner_err := GetAncestorFromLocalPodsList(pod, wh); inner_err == nil {
					return *localOD, nil
				}
				return od, fmt.Errorf("error getting owner reference: %s", err.Error())
			}
			if repItem.OwnerReferences != nil {
				od.Name = repItem.OwnerReferences[0].Name
				od.Kind = repItem.OwnerReferences[0].Kind
				//meanwhile owner reference must be in the same namespace, so owner reference doesn't have the namespace field(may be changed in the future)
				od.OwnerData = GetOwnerData(ctx, repItem.OwnerReferences[0].Name, repItem.OwnerReferences[0].Kind, repItem.OwnerReferences[0].APIVersion, pod.ObjectMeta.Namespace, wh)
			} else {
				depInt := wh.RestAPIClient.AppsV1().Deployments(pod.ObjectMeta.Namespace)
				selector, err := metav1.LabelSelectorAsSelector(repItem.Spec.Selector)
				if err != nil {
					return od, fmt.Errorf("error getting owner reference: %s", err.Error())
				}

				options := metav1.ListOptions{}
				depList, _ := depInt.List(globalHTTPContext, options)
				for _, item := range depList.Items {
					if selector.Empty() || !selector.Matches(labels.Set(pod.Labels)) {
						continue
					} else {
						od.Name = item.ObjectMeta.Name
						od.Kind = item.Kind
						od.OwnerData = GetOwnerData(ctx, od.Name, od.Kind, item.TypeMeta.APIVersion, pod.ObjectMeta.Namespace, wh)
						break
					}
				}
			}
		case "Job":
			od.Name = pod.OwnerReferences[0].Name
			od.Kind = pod.OwnerReferences[0].Kind
			//meanwhile owner reference must be in the same namespace, so owner reference doesn't have the namespace field(may be changed in the future)
			od.OwnerData = GetOwnerData(ctx, pod.OwnerReferences[0].Name, pod.OwnerReferences[0].Kind, pod.OwnerReferences[0].APIVersion, pod.ObjectMeta.Namespace, wh)
			jobItem, err := wh.RestAPIClient.BatchV1().Jobs(pod.ObjectMeta.Namespace).Get(globalHTTPContext, pod.OwnerReferences[0].Name, metav1.GetOptions{})
			if err != nil {
				if localOD, inner_err := GetAncestorFromLocalPodsList(pod, wh); inner_err == nil {
					return *localOD, nil
				}
				logger.L().Ctx(ctx).Error(err.Error())
				return od, err
			}
			if jobItem.OwnerReferences != nil {
				od.Name = jobItem.OwnerReferences[0].Name
				od.Kind = jobItem.OwnerReferences[0].Kind
				//meanwhile owner reference must be in the same namespace, so owner reference doesn't have the namespace field(may be changed in the future)
				od.OwnerData = GetOwnerData(ctx, jobItem.OwnerReferences[0].Name, jobItem.OwnerReferences[0].Kind, jobItem.OwnerReferences[0].APIVersion, pod.ObjectMeta.Namespace, wh)
				break
			}

			depList, _ := wh.RestAPIClient.BatchV1().CronJobs(pod.ObjectMeta.Namespace).List(globalHTTPContext, metav1.ListOptions{})
			selector, err := metav1.LabelSelectorAsSelector(jobItem.Spec.Selector)
			if err != nil {
				return od, fmt.Errorf("error getting owner reference")
			}

			for _, item := range depList.Items {
				if selector.Empty() || !selector.Matches(labels.Set(pod.Labels)) {
					continue
				} else if item.Kind != "" && item.ObjectMeta.Name != "" {
					od.Name = item.ObjectMeta.Name
					od.Kind = item.Kind
					od.OwnerData = GetOwnerData(ctx, od.Name, od.Kind, item.TypeMeta.APIVersion, pod.ObjectMeta.Namespace, wh)
					break
				}
			}

		default: // POD
			od.Name = pod.OwnerReferences[0].Name
			od.Kind = pod.OwnerReferences[0].Kind
			od.OwnerData = GetOwnerData(ctx, pod.OwnerReferences[0].Name, pod.OwnerReferences[0].Kind, pod.OwnerReferences[0].APIVersion, pod.ObjectMeta.Namespace, wh)
		}
	} else {
		od.Name = pod.ObjectMeta.Name
		od.Kind = "Pod"
		od.OwnerData = GetOwnerData(ctx, pod.ObjectMeta.Name, od.Kind, pod.APIVersion, pod.ObjectMeta.Namespace, wh)
		if crd, ok := od.OwnerData.(CRDOwnerData); ok {
			od.Kind = crd.Kind
		}
	}
	return od, nil
}

func (wh *WatchHandler) updatePod(pod *core.Pod, pdm map[int]*list.List, podStatus string) (int, PodDataForExistMicroService) {
	id := -2
	podDataForExistMicroService := PodDataForExistMicroService{}
	for _, v := range pdm {
		if v == nil || v.Front() == nil {
			continue
		}
		element := v.Front().Next()
		for element != nil {
			if strings.Compare(element.Value.(PodDataForExistMicroService).PodName, pod.ObjectMeta.Name) == 0 {
				// newOwner := GetAncestorOfPod(pod, wh)

				if reflect.DeepEqual(*v.Front().Value.(MicroServiceData).Pod, *pod) {
					if err := DeepCopy(*pod, *v.Front().Value.(MicroServiceData).Pod); err != nil {
						id = -1
					} else {
						id = v.Front().Value.(MicroServiceData).PodSpecId
					}
				} else {
					id = -1
				}
				podDataForExistMicroService = PodDataForExistMicroService{PodName: pod.ObjectMeta.Name, NodeName: pod.Spec.NodeName, PodIP: pod.Status.PodIP, Namespace: pod.ObjectMeta.Namespace, PodStatus: podStatus, CreationTimestamp: pod.CreationTimestamp.Time.UTC().Format(time.RFC3339)}

				DeepCopy(element.Value.(PodDataForExistMicroService).Owner, &podDataForExistMicroService.Owner)
				DeepCopyObj(podDataForExistMicroService, element.Value.(PodDataForExistMicroService))
				break
			}
			element = element.Next()
		}
	}
	return id, podDataForExistMicroService
}

func (wh *WatchHandler) isMicroServiceNeedToBeRemoved(ownerData interface{}, kind, namespace string) bool {
	switch kind {
	case "Deployment":
		options := metav1.GetOptions{}
		name := ownerData.(*appsv1.Deployment).ObjectMeta.Name
		_, err := wh.RestAPIClient.AppsV1().Deployments(namespace).Get(globalHTTPContext, name, options)
		if errors.IsNotFound(err) {
			return true
		}

	case "DeamonSet", "DaemonSet":
		options := metav1.GetOptions{}
		name := ownerData.(*appsv1.DaemonSet).ObjectMeta.Name
		_, err := wh.RestAPIClient.AppsV1().DaemonSets(namespace).Get(globalHTTPContext, name, options)
		if errors.IsNotFound(err) {
			return true
		}

	case "StatefulSets":
		options := metav1.GetOptions{}
		name := ownerData.(*appsv1.StatefulSet).ObjectMeta.Name
		_, err := wh.RestAPIClient.AppsV1().StatefulSets(namespace).Get(globalHTTPContext, name, options)
		if errors.IsNotFound(err) {
			return true
		}
	case "Job":
		options := metav1.GetOptions{}
		name := ownerData.(*batchv1.Job).ObjectMeta.Name
		_, err := wh.RestAPIClient.BatchV1().Jobs(namespace).Get(globalHTTPContext, name, options)
		if errors.IsNotFound(err) {
			return true
		}
	case "CronJob":
		options := metav1.GetOptions{}
		cronJob, ok := ownerData.(*v1beta1.CronJob)
		if !ok {
			return true
		}
		_, err := wh.RestAPIClient.BatchV1().CronJobs(namespace).Get(globalHTTPContext, cronJob.ObjectMeta.Name, options)
		if errors.IsNotFound(err) {
			return true
		}
	case "Pod":
		options := metav1.GetOptions{}
		name := ownerData.(*core.Pod).ObjectMeta.Name
		_, err := wh.RestAPIClient.CoreV1().Pods(namespace).Get(globalHTTPContext, name, options)
		if errors.IsNotFound(err) {
			return true
		}
	}

	return false
}

// RemovePod remove pod and check if has parents. Returns 3 elements: 1. pod spec ID, 2. is owner removed, 3. owner
func (wh *WatchHandler) RemovePod(pod *core.Pod, pdm map[int]*list.List) (int, bool, OwnerDet) {
	var owner OwnerDet
	removed := false
	podSpecID := -1
	for id, v := range pdm {
		for element := v.Front(); element != nil; element = element.Next() {
			podData, ok := element.Value.(PodDataForExistMicroService)
			if !ok {
				continue
			}
			if podData.PodName == pod.ObjectMeta.Name {
				owner = v.Front().Value.(MicroServiceData).Owner
				v.Remove(element)
				podSpecID = id
				if v.Len() <= 1 {
					msd := v.Front().Value.(MicroServiceData)
					removed = wh.isMicroServiceNeedToBeRemoved(msd.Owner.OwnerData, msd.Owner.Kind, msd.ObjectMeta.Namespace)
					if removed {
						v.Remove(v.Front())
						delete(pdm, id)
					}
				}
			}
			if element.Value.(PodDataForExistMicroService).PodName == pod.ObjectMeta.GenerateName {
				owner = v.Front().Value.(MicroServiceData).Owner
				v.Remove(element)
				if v.Len() <= 1 {
					msd := v.Front().Value.(MicroServiceData)
					removed := wh.isMicroServiceNeedToBeRemoved(msd.Owner.OwnerData, msd.Owner.Kind, msd.ObjectMeta.Namespace)
					if removed {
						v.Remove(v.Front())
						delete(pdm, id)
					}
				}
				podSpecID = v.Front().Value.(MicroServiceData).PodSpecId
			}

		}

	}
	return podSpecID, removed, owner
}
func getPodStatus(pod *core.Pod) string {
	containerStatuses := pod.Status.ContainerStatuses
	status := ""
	if len(containerStatuses) > 0 {
		for i := range containerStatuses {
			if containerStatuses[i].State.Terminated != nil {
				status = containerStatuses[i].State.Terminated.Reason
			}
			if containerStatuses[i].State.Waiting != nil {
				status = containerStatuses[i].State.Waiting.Reason
			}
			if containerStatuses[i].State.Running != nil {
				if status == "" { // if none of the containers report a error
					status = "Running"
				}
			}
		}
	}
	if status == "" {
		status = string(pod.Status.Phase)
	}
	return status
}
