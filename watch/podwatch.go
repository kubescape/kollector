package watch

import (
	"container/list"
	"log"
	"reflect"
	"strings"
	"time"

	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	v2alpha1 "k8s.io/api/batch/v2alpha1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type OwnerDet struct {
	Name      string      `json:"name"`
	Kind      string      `json:"kind"`
	OwnerData interface{} `json:"ownerData, omitempty"`
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
}

func IsPodExist(pod *core.Pod, pdm map[int]*list.List) bool {
	for _, v := range pdm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name, pod.ObjectMeta.Name) == 0 {
			return true
		}
		if strings.Compare(v.Front().Value.(MicroServiceData).Pod.ObjectMeta.GenerateName, pod.ObjectMeta.Name) == 0 {
			return true
		}
		for e := ids.Ids.Front().Next(); e != nil; e = e.Next() {
			if strings.Compare(e.Value.(PodDataForExistMicroService).PodName, pod.ObjectMeta.Name) == 0 {
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

func GetOwnerData(name string, kind string, apiVersion string, namespace string, wh *WatchHandler) interface{} {
	switch kind {
	case "Deployment":
		var options v1.GetOptions = v1.GetOptions{}
		depDet, err := wh.RestAPIClient.AppsV1beta1().Deployments(namespace).Get(name, options)
		if err != nil {
			log.Printf("GetOwnerData err %v\n", err)
			return nil
		}
		depDet.TypeMeta.Kind = kind
		depDet.TypeMeta.APIVersion = apiVersion
		return depDet
	case "DeamonSet":
		var options v1.GetOptions = v1.GetOptions{}
		daemSetDet, err := wh.RestAPIClient.AppsV1beta2().DaemonSets(namespace).Get(name, options)
		if err != nil {
			log.Printf("GetOwnerData err %v\n", err)
			return nil
		}
		daemSetDet.TypeMeta.Kind = kind
		daemSetDet.TypeMeta.APIVersion = apiVersion
		return daemSetDet
	case "StatefulSets":
		var options v1.GetOptions = v1.GetOptions{}
		statSetDet, err := wh.RestAPIClient.AppsV1beta1().StatefulSets(namespace).Get(name, options)
		if err != nil {
			log.Printf("GetOwnerData err %v\n", err)
			return nil
		}
		statSetDet.TypeMeta.Kind = kind
		statSetDet.TypeMeta.APIVersion = apiVersion
		return statSetDet
	case "Job":
		var options v1.GetOptions = v1.GetOptions{}
		jobDet, err := wh.RestAPIClient.BatchV1().Jobs(namespace).Get(name, options)
		if err != nil {
			log.Printf("GetOwnerData err %v\n", err)
			return nil
		}
		jobDet.TypeMeta.Kind = kind
		jobDet.TypeMeta.APIVersion = apiVersion
		return jobDet
	case "CronJob":
		var options v1.GetOptions = v1.GetOptions{}
		cronJobDet, err := wh.RestAPIClient.BatchV1beta1().CronJobs(namespace).Get(name, options)
		if err != nil {
			log.Printf("GetOwnerData err %v\n", err)
			return nil
		}
		cronJobDet.TypeMeta.Kind = kind
		cronJobDet.TypeMeta.APIVersion = apiVersion
		return cronJobDet
	}

	return nil
}

// GetAncestorOfPod -
func GetAncestorOfPod(pod *core.Pod, wh *WatchHandler) OwnerDet {
	od := OwnerDet{}

	if pod.OwnerReferences != nil {
		switch pod.OwnerReferences[0].Kind {
		case "ReplicaSet":
			repItem, _ := wh.RestAPIClient.AppsV1().ReplicaSets(pod.ObjectMeta.Namespace).Get(pod.OwnerReferences[0].Name, metav1.GetOptions{})
			if repItem.OwnerReferences != nil {
				od.Name = repItem.OwnerReferences[0].Name
				od.Kind = repItem.OwnerReferences[0].Kind
				//meanwhile owner refferance must be in the same namespce, so owner refferance dont have namespace field(may be changed in the future)
				od.OwnerData = GetOwnerData(repItem.OwnerReferences[0].Name, repItem.OwnerReferences[0].Kind, repItem.OwnerReferences[0].APIVersion, pod.ObjectMeta.Namespace, wh)
				return od
			} else {
				depInt := wh.RestAPIClient.AppsV1beta1().Deployments(pod.ObjectMeta.Namespace)
				selector, err := metav1.LabelSelectorAsSelector(repItem.Spec.Selector)
				if err != nil {
					log.Printf("LabelSelectorAsSelector err %v\n", err)
				}

				options := metav1.ListOptions{}
				depList, _ := depInt.List(options)
				for _, item := range depList.Items {
					if selector.Empty() || !selector.Matches(labels.Set(pod.Labels)) {
						continue
					} else {
						od.Name = item.ObjectMeta.Name
						od.Kind = item.Kind
						od.OwnerData = GetOwnerData(od.Name, od.Kind, item.TypeMeta.APIVersion, pod.ObjectMeta.Namespace, wh)
						return od
					}
				}
			}

		default:
			od.Name = pod.OwnerReferences[0].Name
			od.Kind = pod.OwnerReferences[0].Kind
			od.OwnerData = GetOwnerData(pod.OwnerReferences[0].Name, pod.OwnerReferences[0].Kind, pod.OwnerReferences[0].APIVersion, pod.ObjectMeta.Namespace, wh)
			return od
		}
	}
	od.Name = pod.ObjectMeta.Name
	od.Kind = "Pod"
	return od
}

func (wh *WatchHandler) UpdatePod(pod *core.Pod, pdm map[int]*list.List) (int, PodDataForExistMicroService) {
	id := -1
	podDataForExistMicroService := PodDataForExistMicroService{}
	for _, v := range pdm {
		element := v.Front().Next()
		for element != nil {
			if strings.Compare(element.Value.(PodDataForExistMicroService).PodName, pod.ObjectMeta.Name) == 0 {
				newOwner := GetAncestorOfPod(pod, wh)
				if reflect.DeepEqual(*v.Front().Value.(MicroServiceData).Pod, *pod) {
					err := DeepCopy(*pod, *v.Front().Value.(MicroServiceData).Pod)
					if err != nil {
						log.Printf("error in DeepCopy in UpdatePod")
					}
					err = DeepCopy(newOwner, v.Front().Value.(MicroServiceData).Owner)
					if err != nil {
						log.Printf("error in DeepCopy in UpdatePod")
					}
					id = v.Front().Value.(MicroServiceData).PodSpecId
				}
				podDataForExistMicroService = PodDataForExistMicroService{PodName: pod.ObjectMeta.Name, NumberOfRunnigPods: element.Value.(PodDataForExistMicroService).NumberOfRunnigPods, NodeName: pod.Spec.NodeName, PodIP: pod.Status.PodIP, Namespace: pod.ObjectMeta.Namespace, Owner: OwnerDetNameAndKindOnly{Name: newOwner.Name, Kind: newOwner.Kind}}

				err := DeepCopy(podDataForExistMicroService, element.Value.(PodDataForExistMicroService))
				if err != nil {
					log.Printf("error in DeepCopy in UpdatePod")
				}
				break
			}
			element = element.Next()
		}
	}
	return id, podDataForExistMicroService
}

func (wh *WatchHandler) isMicroServiceNeedToBeRemoved(ownerData interface{}, kind, namespace string) bool {
	delete := false

	switch kind {
	case "Deployment":
		options := v1.GetOptions{}
		name := ownerData.(*v1beta1.Deployment).ObjectMeta.Name
		_, err := wh.RestAPIClient.AppsV1beta1().Deployments(namespace).Get(name, options)
		if errors.IsNotFound(err) {
			delete = true
		}

	case "DeamonSet":
		options := v1.GetOptions{}
		name := ownerData.(*v1beta2.DaemonSet).ObjectMeta.Name
		_, err := wh.RestAPIClient.AppsV1beta2().DaemonSets(namespace).Get(name, options)
		if errors.IsNotFound(err) {
			delete = true
		}

	case "StatefulSets":
		options := v1.GetOptions{}
		name := ownerData.(*v1beta1.StatefulSet).ObjectMeta.Name
		_, err := wh.RestAPIClient.AppsV1beta1().StatefulSets(namespace).Get(name, options)
		if errors.IsNotFound(err) {
			delete = true
		}
	case "Job":
		options := v1.GetOptions{}
		name := ownerData.(*batchv1.Job).ObjectMeta.Name
		_, err := wh.RestAPIClient.BatchV1().Jobs(namespace).Get(name, options)
		if errors.IsNotFound(err) {
			delete = true
		}
	case "CronJob":
		options := v1.GetOptions{}
		name := ownerData.(*v2alpha1.CronJob).ObjectMeta.Name
		_, err := wh.RestAPIClient.BatchV1beta1().CronJobs(namespace).Get(name, options)
		if errors.IsNotFound(err) {
			delete = true
		}
	}

	return delete
}

// RemovePod remove pod and check if has parents
func (wh *WatchHandler) RemovePod(pod *core.Pod, pdm map[int]*list.List) (int, int, bool, OwnerDet) {
	var owner OwnerDet
	for _, v := range pdm {
		element := v.Front().Next()
		for element != nil {
			if strings.Compare(element.Value.(PodDataForExistMicroService).PodName, pod.ObjectMeta.Name) == 0 {
				//log.Printf("microservice %s removed\n", element.Value.(PodDataForExistMicroService).PodName)
				owner = v.Front().Value.(MicroServiceData).Owner
				v.Remove(element)
				removed := false
				if v.Len() == 1 {
					msd := v.Front().Value.(MicroServiceData)
					removed = wh.isMicroServiceNeedToBeRemoved(msd.Owner.OwnerData, msd.Owner.Kind, msd.ObjectMeta.Namespace)
					podSpecID := v.Front().Value.(MicroServiceData).PodSpecId
					v.Remove(v.Front())
					return podSpecID, 0, removed, owner
				}
				// remove before testing len?
				return v.Front().Value.(MicroServiceData).PodSpecId, element.Value.(PodDataForExistMicroService).NumberOfRunnigPods, removed, owner
			}
			if strings.Compare(element.Value.(PodDataForExistMicroService).PodName, pod.ObjectMeta.GenerateName) == 0 {
				//log.Printf("microservice %s removed\n", element.Value.(PodDataForExistMicroService).PodName)
				owner = v.Front().Value.(MicroServiceData).Owner
				v.Remove(element)
				if v.Len() == 1 {
					msd := v.Front().Value.(MicroServiceData)
					removed := wh.isMicroServiceNeedToBeRemoved(msd.Owner.OwnerData, msd.Owner.Kind, msd.ObjectMeta.Namespace)
					podSpecID := v.Front().Value.(MicroServiceData).PodSpecId
					v.Remove(v.Front())
					return podSpecID, 0, removed, owner
				}
				return v.Front().Value.(MicroServiceData).PodSpecId, element.Value.(PodDataForExistMicroService).NumberOfRunnigPods, false, owner
			}
			element = element.Next()
		}
	}
	return 0, 0, false, owner
}

func (wh *WatchHandler) podEnterDesiredState(pod *core.Pod) (*core.Pod, bool) {
	begin := time.Now()
	log.Printf("waiting for pod %v enter desired state\n", pod.ObjectMeta.Name)
	for {
		desiredStatePod, err := wh.RestAPIClient.CoreV1().Pods(pod.ObjectMeta.Namespace).Get(pod.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			log.Printf("podEnterDesiredState fail while we Get the pod %v\n", pod.ObjectMeta.Name)
			return nil, false
		}
		if strings.Compare(string(desiredStatePod.Status.Phase), string(core.PodRunning)) == 0 || strings.Compare(string(desiredStatePod.Status.Phase), string(core.PodSucceeded)) == 0 {
			log.Printf("pod %v enter desired state\n", pod.ObjectMeta.Name)
			return desiredStatePod, true
		} else if strings.Compare(string(desiredStatePod.Status.Phase), string(core.PodFailed)) == 0 || strings.Compare(string(desiredStatePod.Status.Phase), string(core.PodUnknown)) == 0 {
			log.Printf("pod %v State is %v\n", pod.ObjectMeta.Name, pod.Status.Phase)
			return desiredStatePod, true
		} else {
			if time.Now().Sub(begin) > 5*60*time.Second {
				log.Printf("we wait for 5 nimutes pod %v to change his state to desired state and it's too long\n", pod.ObjectMeta.Name)
				return nil, false
			}
		}
	}
}

// PodWatch - StayUpadted starts infinite loop which will observe changes in pods so we can know if they changed and acts accordinally
func (wh *WatchHandler) PodWatch() {
	log.Printf("Watching over pods starting")
	for {
		podsWatcher, err := wh.RestAPIClient.CoreV1().Pods("").Watch(metav1.ListOptions{Watch: true})
		if err != nil {
			log.Printf("Cannot watch over pods. %v", err)
			time.Sleep(time.Duration(10) * time.Second)
			continue
		}
		podsChan := podsWatcher.ResultChan()
		log.Printf("Watching over pods started")
		for event := range podsChan {
			if pod, ok := event.Object.(*core.Pod); ok {
				switch event.Type {
				case "ADDED":
					od := GetAncestorOfPod(pod, wh)
					var id int
					var runnigPodNum int
					if id, runnigPodNum = IsPodSpecAlreadyExist(pod, wh.pdm); runnigPodNum == 0 {
						wh.pdm[id] = list.New()
						nms := MicroServiceData{Pod: pod, Owner: od, PodSpecId: id}
						wh.pdm[id].PushBack(nms)
						wh.jsonReport.AddToJsonFormat(nms, MICROSERVICES, CREATED)
						runnigPodNum = 1
					}
					podName := pod.ObjectMeta.Name
					if podName == "" {
						podName = pod.ObjectMeta.GenerateName
					}

					np := PodDataForExistMicroService{PodName: podName, NumberOfRunnigPods: runnigPodNum, NodeName: pod.Spec.NodeName, PodIP: pod.Status.PodIP, Namespace: pod.ObjectMeta.Namespace, Owner: OwnerDetNameAndKindOnly{Name: od.Name, Kind: od.Kind}}
					wh.pdm[id].PushBack(np)
					wh.jsonReport.AddToJsonFormat(np, PODS, CREATED)
					informNewDataArrive(wh)
					if strings.Compare(string(pod.Status.Phase), string(core.PodPending)) == 0 {
						go func() {
							if podInDesiredState, yes := wh.podEnterDesiredState(pod); yes {
								err := DeepCopy(podInDesiredState, wh.pdm[id].Front().Value.(MicroServiceData).Pod)
								if err != nil {
									log.Printf("error while updating the microservice to desired state %v", err)
									return
								}
								od = GetAncestorOfPod(podInDesiredState, wh)
								od.Kind = wh.pdm[id].Front().Value.(MicroServiceData).Owner.Kind
								od.Name = wh.pdm[id].Front().Value.(MicroServiceData).Owner.Name
								od.OwnerData = wh.pdm[id].Front().Value.(MicroServiceData).Owner.OwnerData
								wh.jsonReport.AddToJsonFormat(wh.pdm[id].Front().Value.(MicroServiceData), MICROSERVICES, UPDATED)
								informNewDataArrive(wh)
							}
						}()
					}
				case "MODIFY":
					log.Printf("pod %s modify", pod.ObjectMeta.Name)
					podSpecID, newPodData := wh.UpdatePod(pod, wh.pdm)
					wh.jsonReport.AddToJsonFormat(newPodData, PODS, UPDATED)
					if podSpecID != -1 {
						wh.jsonReport.AddToJsonFormat(wh.pdm[podSpecID].Front().Value.(MicroServiceData), MICROSERVICES, UPDATED)
					}
					informNewDataArrive(wh)
				case "DELETED":
					log.Printf("pod %v deleted\n", pod.ObjectMeta.Name)
					podSpecID, numberOfRunningPods, removeMicroServiceAsWell, owner := wh.RemovePod(pod, wh.pdm)
					// od := GetAncestorOfPod(pod, wh)
					np := PodDataForExistMicroService{PodName: pod.ObjectMeta.Name, NumberOfRunnigPods: numberOfRunningPods - 1, NodeName: pod.Spec.NodeName, PodIP: pod.Status.PodIP, Namespace: pod.ObjectMeta.Namespace, Owner: OwnerDetNameAndKindOnly{Name: owner.Name, Kind: owner.Kind}}
					wh.jsonReport.AddToJsonFormat(np, PODS, DELETED)
					if removeMicroServiceAsWell {
						log.Printf("remove MicroService as well")
						nms := MicroServiceData{Pod: pod, Owner: owner, PodSpecId: podSpecID}
						wh.jsonReport.AddToJsonFormat(nms, MICROSERVICES, DELETED)

					}
					informNewDataArrive(wh)
				case "BOOKMARK": //only the resource version is changed but it's the same workload
					log.Printf("BOOKMARK: pod %s modify", pod.ObjectMeta.Name)
					continue
				case "ERROR":
					log.Printf("while watching over pods we got an error: ")
				}
			} else {
				log.Printf("Got unexpected pod from chan: %t, %v", event.Object, event.Object)
			}
		}
		log.Printf("Watching over pods ended - since we got timeout")
	}
	log.Printf("Wathching over pods ending")
}
