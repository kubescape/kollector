package watch

import (
	"container/list"
	"context"
	"encoding/json"
	"log"
	"reflect"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v2alpha1 "k8s.io/api/batch/v2alpha1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

type OwnerDet struct {
	Name      string      `json:"name"`
	Kind      string      `json:"kind"`
	OwnerData interface{} `json:"ownerData, omitempty"`
}
type CRDOwnerData struct {
	v1.TypeMeta
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
	PodName            string                  `json:"podName"`
	NumberOfRunnigPods int                     `json:"numberOfRunnigPods"`
	NodeName           string                  `json:"nodeName"`
	PodIP              string                  `json:"podIP"`
	Namespace          string                  `json:"namespace, omitempty"`
	Owner              OwnerDetNameAndKindOnly `json:"uptreeOwner"`
	PodStatus          string                  `json:"podStatus"`
}

func NewPodDataForExistMicroService(pod *core.Pod, ownerDetNameAndKindOnly OwnerDetNameAndKindOnly, numberOfRunnigPods int, podStatus string) PodDataForExistMicroService {
	return PodDataForExistMicroService{
		PodName:            pod.ObjectMeta.Name,
		NumberOfRunnigPods: numberOfRunnigPods,
		NodeName:           pod.Spec.NodeName,
		PodIP:              pod.Status.PodIP,
		Namespace:          pod.ObjectMeta.Namespace,
		Owner:              ownerDetNameAndKindOnly,
		PodStatus:          podStatus,
	}
}

// PodWatch - StayUpadted starts infinite loop which will observe changes in pods so we can know if they changed and acts accordinally
func (wh *WatchHandler) PodWatch() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("RECOVER PodWatch. error: %v", err)
		}
	}()

	for {
		glog.Infof("Watching over pods starting")
		podsWatcher, err := wh.RestAPIClient.CoreV1().Pods("").Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			glog.Errorf("Watch error: %s", err.Error())
		}
		for event := range podsWatcher.ResultChan() {
			pod, _ := event.Object.(*core.Pod)
			podName := pod.ObjectMeta.Name
			if podName == "" {
				podName = pod.ObjectMeta.GenerateName
			}
			podStatus := getPodStatus(pod)
			switch event.Type {
			case watch.Added:
				glog.Infof("added. name: %s, status: %s", podName, podStatus)
				od := GetAncestorOfPod(pod, wh)

				first := true
				id, runnigPodNum := IsPodSpecAlreadyExist(pod, wh.pdm)
				if runnigPodNum == 0 {
					wh.pdm[id] = list.New()
					nms := MicroServiceData{Pod: pod, Owner: od, PodSpecId: id}
					wh.pdm[id].PushBack(nms)
					wh.jsonReport.AddToJsonFormat(nms, MICROSERVICES, CREATED)
					runnigPodNum = 1
				} else { // Check if pod is already reported
					if wh.pdm[id].Front() != nil {
						element := wh.pdm[id].Front().Next()
						for element != nil {
							if element.Value.(PodDataForExistMicroService).PodName == podName {
								first = false
							}
							element = element.Next()
						}
					}
				}
				if !first {
					continue
				}
				np := PodDataForExistMicroService{PodName: podName, NumberOfRunnigPods: runnigPodNum, NodeName: pod.Spec.NodeName, PodIP: pod.Status.PodIP, Namespace: pod.ObjectMeta.Namespace, Owner: OwnerDetNameAndKindOnly{Name: od.Name, Kind: od.Kind}, PodStatus: podStatus}
				wh.pdm[id].PushBack(np)
				wh.jsonReport.AddToJsonFormat(np, PODS, CREATED)
				informNewDataArrive(wh)

			case watch.Modified:
				glog.Infof("Modified. name: %s, status: %s", podName, podStatus)
				podSpecID, newPodData := wh.UpdatePod(pod, wh.pdm, podStatus)
				wh.jsonReport.AddToJsonFormat(newPodData, PODS, UPDATED)
				if podSpecID != -1 {
					wh.jsonReport.AddToJsonFormat(wh.pdm[podSpecID].Front().Value.(MicroServiceData), MICROSERVICES, UPDATED)
				}
				informNewDataArrive(wh)
			case watch.Deleted:
				podStatus = "Terminating"
				glog.Infof("Deleted. name: %s, status: %s", podName, podStatus)
				podSpecID, numberOfRunningPods, removeMicroServiceAsWell, owner := wh.RemovePod(pod, wh.pdm)
				np := PodDataForExistMicroService{PodName: pod.ObjectMeta.Name, NumberOfRunnigPods: numberOfRunningPods - 1, NodeName: pod.Spec.NodeName, PodIP: pod.Status.PodIP, Namespace: pod.ObjectMeta.Namespace, Owner: OwnerDetNameAndKindOnly{Name: owner.Name, Kind: owner.Kind}, PodStatus: podStatus}
				wh.jsonReport.AddToJsonFormat(np, PODS, DELETED)
				if removeMicroServiceAsWell {
					glog.Infof("remove %s.%s", owner.Kind, owner.Name)
					nms := MicroServiceData{Pod: pod, Owner: owner, PodSpecId: podSpecID}
					wh.jsonReport.AddToJsonFormat(nms, MICROSERVICES, DELETED)
				}
				informNewDataArrive(wh)
			case watch.Bookmark:
				glog.Infof("Bookmark. name: %s, status: %s", podName, podStatus)
			case watch.Error:
				glog.Infof("Error. name: %s, status: %s", podName, podStatus)
			}
		}

	}
}

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

func IsPodSpecAlreadyExist(pod *core.Pod, pdm map[int]*list.List) (int, int) {
	for _, v := range pdm {
		if v == nil || v.Len() == 0 {
			continue
		}
		p := v.Front().Value.(MicroServiceData)
		if reflect.DeepEqual(pod.Spec.Containers, p.Pod.Spec.Containers) {
			return v.Front().Value.(MicroServiceData).PodSpecId, v.Len()
		}
	}

	return CreateID(), 0
}

// GetOwnerData - get the data of pod owner
func GetOwnerData(name string, kind string, apiVersion string, namespace string, wh *WatchHandler) interface{} {
	switch kind {
	case "Deployment":
		options := v1.GetOptions{}
		depDet, err := wh.RestAPIClient.AppsV1().Deployments(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			glog.Errorf("GetOwnerData Deployments: %s", err.Error())
			return nil
		}
		depDet.TypeMeta.Kind = kind
		depDet.TypeMeta.APIVersion = apiVersion
		return depDet
	case "DeamonSet", "DaemonSet":
		options := v1.GetOptions{}
		daemSetDet, err := wh.RestAPIClient.AppsV1().DaemonSets(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			glog.Errorf("GetOwnerData DaemonSets: %s", err.Error())
			return nil
		}
		daemSetDet.TypeMeta.Kind = kind
		daemSetDet.TypeMeta.APIVersion = apiVersion
		return daemSetDet
	case "StatefulSet":
		options := v1.GetOptions{}
		statSetDet, err := wh.RestAPIClient.AppsV1().StatefulSets(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			glog.Errorf("GetOwnerData StatefulSets: %s", err.Error())
			return nil
		}
		statSetDet.TypeMeta.Kind = kind
		statSetDet.TypeMeta.APIVersion = apiVersion
		return statSetDet
	case "Job":
		options := v1.GetOptions{}
		jobDet, err := wh.RestAPIClient.BatchV1().Jobs(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			glog.Errorf("GetOwnerData Jobs: %s", err.Error())
			return nil
		}
		jobDet.TypeMeta.Kind = kind
		jobDet.TypeMeta.APIVersion = apiVersion
		return jobDet
	case "CronJob":
		options := v1.GetOptions{}
		cronJobDet, err := wh.RestAPIClient.BatchV1beta1().CronJobs(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			glog.Errorf("GetOwnerData CronJobs: %s", err.Error())
			return nil
		}
		cronJobDet.TypeMeta.Kind = kind
		cronJobDet.TypeMeta.APIVersion = apiVersion
		return cronJobDet
	case "Pod":
		options := v1.GetOptions{}
		podDet, err := wh.RestAPIClient.CoreV1().Pods(namespace).Get(globalHTTPContext, name, options)
		if err != nil {
			glog.Errorf("GetOwnerData Pods: %s", err.Error())
			return nil
		}
		podDet.TypeMeta.Kind = kind
		podDet.TypeMeta.APIVersion = apiVersion
		return podDet

	default:
		if wh.extensionsClient == nil {
			return nil
		}
		options := v1.ListOptions{}
		crds, err := wh.extensionsClient.CustomResourceDefinitions().List(context.Background(), options)
		if err != nil {
			glog.Errorf("GetOwnerData CustomResourceDefinitions: %s", err.Error())
			return nil
		}
		for crdIdx := range crds.Items {
			if crds.Items[crdIdx].Status.AcceptedNames.Kind == kind {
				return CRDOwnerData{
					v1.TypeMeta{Kind: crds.Items[crdIdx].Kind,
						APIVersion: apiVersion,
					}}
			}
		}
	}

	return nil
}

// GetAncestorOfPod -
func GetAncestorOfPod(pod *core.Pod, wh *WatchHandler) OwnerDet {
	od := OwnerDet{}

	if pod.OwnerReferences != nil {
		switch pod.OwnerReferences[0].Kind {
		case "ReplicaSet":
			repItem, err := wh.RestAPIClient.AppsV1().ReplicaSets(pod.ObjectMeta.Namespace).Get(globalHTTPContext, pod.OwnerReferences[0].Name, metav1.GetOptions{})
			if err != nil {
				glog.Errorf("ReplicaSets get: %s", err.Error())
				break
			}
			if repItem.OwnerReferences != nil {
				od.Name = repItem.OwnerReferences[0].Name
				od.Kind = repItem.OwnerReferences[0].Kind
				//meanwhile owner refferance must be in the same namespce, so owner refferance dont have namespace field(may be changed in the future)
				od.OwnerData = GetOwnerData(repItem.OwnerReferences[0].Name, repItem.OwnerReferences[0].Kind, repItem.OwnerReferences[0].APIVersion, pod.ObjectMeta.Namespace, wh)
			} else {
				depInt := wh.RestAPIClient.AppsV1().Deployments(pod.ObjectMeta.Namespace)
				selector, err := metav1.LabelSelectorAsSelector(repItem.Spec.Selector)
				if err != nil {
					glog.Errorf("LabelSelectorAsSelector: %s", err.Error())
					break
				}

				options := metav1.ListOptions{}
				depList, _ := depInt.List(globalHTTPContext, options)
				for _, item := range depList.Items {
					if selector.Empty() || !selector.Matches(labels.Set(pod.Labels)) {
						continue
					} else {
						od.Name = item.ObjectMeta.Name
						od.Kind = item.Kind
						od.OwnerData = GetOwnerData(od.Name, od.Kind, item.TypeMeta.APIVersion, pod.ObjectMeta.Namespace, wh)
						break
					}
				}
			}
		case "Job":
			jobItem, err := wh.RestAPIClient.BatchV1().Jobs(pod.ObjectMeta.Namespace).Get(globalHTTPContext, pod.OwnerReferences[0].Name, metav1.GetOptions{})
			if err != nil {
				glog.Error(err)
				break
			}
			if jobItem.OwnerReferences != nil {
				od.Name = jobItem.OwnerReferences[0].Name
				od.Kind = jobItem.OwnerReferences[0].Kind
				//meanwhile owner refferance must be in the same namespce, so owner refferance dont have namespace field(may be changed in the future)
				od.OwnerData = GetOwnerData(jobItem.OwnerReferences[0].Name, jobItem.OwnerReferences[0].Kind, jobItem.OwnerReferences[0].APIVersion, pod.ObjectMeta.Namespace, wh)
				break
			}

			depList, _ := wh.RestAPIClient.BatchV1beta1().CronJobs(pod.ObjectMeta.Namespace).List(globalHTTPContext, metav1.ListOptions{})
			selector, err := metav1.LabelSelectorAsSelector(jobItem.Spec.Selector)
			if err != nil {
				glog.Errorf("LabelSelectorAsSelector: %s", err.Error())
				break
			}

			for _, item := range depList.Items {
				if selector.Empty() || !selector.Matches(labels.Set(pod.Labels)) {
					continue
				} else {
					od.Name = item.ObjectMeta.Name
					od.Kind = item.Kind
					od.OwnerData = GetOwnerData(od.Name, od.Kind, item.TypeMeta.APIVersion, pod.ObjectMeta.Namespace, wh)
					break
				}
			}

		default: // POD
			od.Name = pod.OwnerReferences[0].Name
			od.Kind = pod.OwnerReferences[0].Kind
			od.OwnerData = GetOwnerData(pod.OwnerReferences[0].Name, pod.OwnerReferences[0].Kind, pod.OwnerReferences[0].APIVersion, pod.ObjectMeta.Namespace, wh)
		}
	} else {
		od.Name = pod.ObjectMeta.Name
		od.Kind = "Pod"
		od.OwnerData = GetOwnerData(pod.ObjectMeta.Name, od.Kind, pod.APIVersion, pod.ObjectMeta.Namespace, wh)
		if crd, ok := od.OwnerData.(CRDOwnerData); ok {
			od.Kind = crd.Kind
		}
	}
	return od
}

// UpdatePod -
func (wh *WatchHandler) UpdatePod(pod *core.Pod, pdm map[int]*list.List, podStatus string) (int, PodDataForExistMicroService) {
	id := -1
	podDataForExistMicroService := PodDataForExistMicroService{}
	for _, v := range pdm {
		if v == nil || v.Front() == nil {
			continue
		}
		element := v.Front().Next()
		for element != nil {
			if element.Value.(PodDataForExistMicroService).PodName == pod.ObjectMeta.Name {
				// newOwner := GetAncestorOfPod(pod, wh)
				// if reflect.DeepEqual(*v.Front().Value.(MicroServiceData).Pod, *pod) {
				// 	err := DeepCopy(*pod, *v.Front().Value.(MicroServiceData).Pod)
				// 	if err != nil {
				// 		glog.Errorf("error in A DeepCopy in UpdatePod, err: %s", err.Error())
				// 	}
				// 	// if v.Front().Value.(MicroServiceData).Owner.Kind == "" {
				// 	err = DeepCopy(newOwner, v.Front().Value.(MicroServiceData).Owner)
				// 	if err != nil {
				// 		glog.Errorf("error in B DeepCopy in UpdatePod, err: %s", err.Error())
				// 	}
				// 	// }
				// }
				// if pod.Namespace == "default" || pod.Namespace == "" {
				// 	glog.Infof("----------------------------------------------------------------------------------------------------")
				// 	oldd, _ := json.Marshal(element.Value.(PodDataForExistMicroService))
				// 	glog.Infof("dwertent, old: %s", string(oldd))
				// }

				podDataForExistMicroService = element.Value.(PodDataForExistMicroService)
				podDataForExistMicroService.PodIP = pod.Status.PodIP
				podDataForExistMicroService.PodStatus = podStatus
				podDataForExistMicroService.NodeName = pod.Spec.NodeName
				podDataForExistMicroService.Namespace = pod.ObjectMeta.Namespace
				if podDataForExistMicroService.Owner.Kind != "" {
					id = v.Front().Value.(MicroServiceData).PodSpecId
				}

				element.Value = podDataForExistMicroService
				// if err := DeepCopy(podDataForExistMicroService, element.Value.(PodDataForExistMicroService)); err != nil {
				// 	glog.Errorf("error in C DeepCopy in UpdatePod, err: %s", err.Error())
				// }
				// if pod.Namespace == "default" || pod.Namespace == "" {
				// 	neww, _ := json.Marshal(element.Value.(PodDataForExistMicroService))
				// 	glog.Infof("dwertent, new: %s", string(neww))
				// 	glog.Infof("----------------------------------------------------------------------------------------------------")
				// }
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
		options := v1.GetOptions{}
		name := ownerData.(*appsv1.Deployment).ObjectMeta.Name
		mic, err := wh.RestAPIClient.AppsV1().Deployments(namespace).Get(globalHTTPContext, name, options)
		if errors.IsNotFound(err) {
			return true
		}
		v, _ := json.Marshal(mic)
		glog.Infof("Removing pod but not Deployment: %s", string(v))

	case "DeamonSet", "DaemonSet":
		options := v1.GetOptions{}
		name := ownerData.(*appsv1.DaemonSet).ObjectMeta.Name
		mic, err := wh.RestAPIClient.AppsV1().DaemonSets(namespace).Get(globalHTTPContext, name, options)
		if errors.IsNotFound(err) {
			return true
		}
		v, _ := json.Marshal(mic)
		glog.Infof("Removing pod but not DaemonSet: %s", string(v))

	case "StatefulSets":
		options := v1.GetOptions{}
		name := ownerData.(*appsv1.StatefulSet).ObjectMeta.Name
		mic, err := wh.RestAPIClient.AppsV1().StatefulSets(namespace).Get(globalHTTPContext, name, options)
		if errors.IsNotFound(err) {
			return true
		}
		v, _ := json.Marshal(mic)
		glog.Infof("Removing pod but not StatefulSet: %s", string(v))
	case "Job":
		options := v1.GetOptions{}
		name := ownerData.(*batchv1.Job).ObjectMeta.Name
		mic, err := wh.RestAPIClient.BatchV1().Jobs(namespace).Get(globalHTTPContext, name, options)
		if errors.IsNotFound(err) {
			return true
		}
		v, _ := json.Marshal(mic)
		glog.Infof("Removing pod but not Job: %s", string(v))
	case "CronJob":
		options := v1.GetOptions{}
		name := ownerData.(*v2alpha1.CronJob).ObjectMeta.Name
		mic, err := wh.RestAPIClient.BatchV1beta1().CronJobs(namespace).Get(globalHTTPContext, name, options)
		if errors.IsNotFound(err) {
			return true
		}
		v, _ := json.Marshal(mic)
		glog.Infof("Removing pod but not CronJob: %s", string(v))
	case "Pod":
		options := v1.GetOptions{}
		name := ownerData.(*core.Pod).ObjectMeta.Name
		mic, err := wh.RestAPIClient.CoreV1().Pods(namespace).Get(globalHTTPContext, name, options)
		if errors.IsNotFound(err) {
			return true
		}
		v, _ := json.Marshal(mic)
		glog.Infof("Removing pod but not Pod: %s", string(v))
	}

	return false
}

// RemovePod remove pod and check if has parents
func (wh *WatchHandler) RemovePod(pod *core.Pod, pdm map[int]*list.List) (int, int, bool, OwnerDet) {
	var owner OwnerDet
	for _, v := range pdm {
		if v.Front() != nil {
			element := v.Front().Next()
			for element != nil {
				if element.Value.(PodDataForExistMicroService).PodName == pod.ObjectMeta.Name {
					//log.Printf("microservice %s removed\n", element.Value.(PodDataForExistMicroService).PodName)
					owner = v.Front().Value.(MicroServiceData).Owner
					v.Remove(element)
					removed := false
					if v.Len() == 1 {
						msd := v.Front().Value.(MicroServiceData)
						removed = wh.isMicroServiceNeedToBeRemoved(msd.Owner.OwnerData, msd.Owner.Kind, msd.ObjectMeta.Namespace)
						podSpecID := v.Front().Value.(MicroServiceData).PodSpecId
						numberOfRunningPods := element.Value.(PodDataForExistMicroService).NumberOfRunnigPods
						if removed {
							v.Remove(v.Front())
						}
						return podSpecID, numberOfRunningPods, removed, owner
					}
					// remove before testing len?
					return v.Front().Value.(MicroServiceData).PodSpecId, element.Value.(PodDataForExistMicroService).NumberOfRunnigPods, removed, owner
				}
				if element.Value.(PodDataForExistMicroService).PodName == pod.ObjectMeta.GenerateName {
					//log.Printf("microservice %s removed\n", element.Value.(PodDataForExistMicroService).PodName)
					owner = v.Front().Value.(MicroServiceData).Owner
					removed := false
					v.Remove(element)
					if v.Len() == 1 {
						msd := v.Front().Value.(MicroServiceData)
						removed := wh.isMicroServiceNeedToBeRemoved(msd.Owner.OwnerData, msd.Owner.Kind, msd.ObjectMeta.Namespace)
						podSpecID := v.Front().Value.(MicroServiceData).PodSpecId
						numberOfRunningPods := element.Value.(PodDataForExistMicroService).NumberOfRunnigPods
						if removed {
							v.Remove(v.Front())
						}
						return podSpecID, numberOfRunningPods, removed, owner
					}
					return v.Front().Value.(MicroServiceData).PodSpecId, element.Value.(PodDataForExistMicroService).NumberOfRunnigPods, removed, owner
				}
				element = element.Next()
			}
		}
	}
	return 0, 0, false, owner
}

// func (wh *WatchHandler) AddPod(pod *core.Pod, pdm map[int]*list.List) (int, int, bool, OwnerDet) {

// }
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
				if status == "" { // if none of the conatainers report a error
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

// func (wh *WatchHandler) waitPodStateUpdate(pod *core.Pod) *core.Pod {
// 	// begin := time.Now()
// 	// log.Printf("waiting for pod %v enter desired state\n", pod.ObjectMeta.Name)
// 	latestPodState := pod.Status.Phase

// 	for {
// 		desiredStatePod, err := wh.RestAPIClient.CoreV1().Pods(pod.ObjectMeta.Namespace).Get(globalHTTPContext, pod.ObjectMeta.Name, metav1.GetOptions{})
// 		if err != nil {
// 			log.Printf("podEnterDesiredState fail while we Get the pod %v\n", pod.ObjectMeta.Name)
// 			return nil
// 		}
// 		if desiredStatePod.Status.Phase != latestPodState {
// 			return desiredStatePod
// 		}
// 		// if desiredStatePod.Namespace == "default" || desiredStatePod.Namespace == "" {
// 		// 	podd, _ := json.Marshal(desiredStatePod)
// 		// 	glog.Infof("dwertent, Status: %s, desiredStatePod: %s", string(desiredStatePod.Status.Phase), string(podd))
// 		// }
// 		// if desiredStatePod.Status.Phase == core.PodRunning || strings.Compare(string(desiredStatePod.Status.Phase), string(core.PodSucceeded)) == 0 {
// 		// 	log.Printf("pod %v enter desired state\n", pod.ObjectMeta.Name)
// 		// 	return desiredStatePod, true
// 		// } else if strings.Compare(string(desiredStatePod.Status.Phase), string(core.PodFailed)) == 0 || strings.Compare(string(desiredStatePod.Status.Phase), string(core.PodUnknown)) == 0 {
// 		// 	log.Printf("pod %v State is %v\n", pod.ObjectMeta.Name, pod.Status.Phase)
// 		// 	return desiredStatePod, true
// 		// } else {
// 		// 	if time.Now().Sub(begin) > 5*time.Minute {
// 		// 		log.Printf("we wait for 5 nimutes pod %v to change his state to desired state and it's too long\n", pod.ObjectMeta.Name)
// 		// 		return nil, false
// 		// 	}
// 		// }
// 	}
// }
