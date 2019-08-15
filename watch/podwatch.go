package watch

import (
	"container/list"
	"log"
	"reflect"
	"strings"
	"time"

	core "k8s.io/api/core/v1"
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
						od.OwnerData = GetOwnerData(item.OwnerReferences[0].Name, item.OwnerReferences[0].Kind, item.OwnerReferences[0].APIVersion, pod.ObjectMeta.Namespace, wh)
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

func UpdatePod(pod *core.Pod, pdm map[int]*list.List) string {
	for _, v := range pdm {
		if strings.Compare(v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name, pod.ObjectMeta.Name) == 0 {
			*v.Front().Value.(MicroServiceData).Pod = *pod
			log.Printf("microservice %s updated\n", v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name)
			return v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name
		}
		if strings.Compare(v.Front().Value.(MicroServiceData).Pod.ObjectMeta.GenerateName, pod.ObjectMeta.Name) == 0 {
			*v.Front().Value.(MicroServiceData).Pod = *pod
			log.Printf("microservice %s updated\n", v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name)
			return v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name
		}
		element := v.Front().Next()
		for element != nil {
			if strings.Compare(element.Value.(PodDataForExistMicroService).PodName, pod.ObjectMeta.Name) == 0 {
				log.Printf("pod %s removed\n", element.Value.(PodDataForExistMicroService).PodName)
				return element.Value.(PodDataForExistMicroService).PodName
			}
			if strings.Compare(element.Value.(PodDataForExistMicroService).PodName, pod.ObjectMeta.Name) == 0 {
				log.Printf("pod %s removed\n", element.Value.(PodDataForExistMicroService).PodName)
				return element.Value.(PodDataForExistMicroService).PodName
			}
			element = element.Next()
		}
	}
	return ""
}

func RemovePod(pod *core.Pod, pdm map[int]*list.List) string {
	for _, v := range pdm {
		if strings.Compare(v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name, pod.ObjectMeta.Name) == 0 {
			log.Printf("microservice %s removed\n", v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name)
			name := v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name
			v.Remove(v.Front())
			return name
		}
		if strings.Compare(v.Front().Value.(MicroServiceData).Pod.ObjectMeta.GenerateName, pod.ObjectMeta.Name) == 0 {
			log.Printf("microservice %s removed\n", v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name)
			name := v.Front().Value.(MicroServiceData).Pod.ObjectMeta.Name
			v.Remove(v.Front())
			return name
		}
		element := v.Front().Next()
		for element != nil {
			if strings.Compare(element.Value.(PodDataForExistMicroService).PodName, pod.ObjectMeta.Name) == 0 {
				log.Printf("microservice %s removed\n", element.Value.(PodDataForExistMicroService).PodName)
				name := element.Value.(PodDataForExistMicroService).PodName
				v.Remove(element)
				return name
			}
			if strings.Compare(element.Value.(PodDataForExistMicroService).PodName, pod.ObjectMeta.Name) == 0 {
				log.Printf("microservice %s removed\n", element.Value.(PodDataForExistMicroService).PodName)
				name := element.Value.(PodDataForExistMicroService).PodName
				v.Remove(element)
				return name
			}
			element = element.Next()
		}
	}
	return ""
}

// StayUpadted starts infinite loop which will observe changes in pods so we can know if they changed and acts accordinally
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
					var podName string
					if pod.ObjectMeta.Name == "" {
						podName = pod.ObjectMeta.GenerateName
					} else {
						podName = pod.ObjectMeta.Name
					}
					np := PodDataForExistMicroService{PodName: podName, NumberOfRunnigPods: runnigPodNum, NodeName: pod.Spec.NodeName, PodIP: pod.Status.PodIP, Namespace: pod.ObjectMeta.Namespace, Owner: OwnerDetNameAndKindOnly{Name: od.Name, Kind: od.Kind}}
					wh.pdm[id].PushBack(np)
					wh.jsonReport.AddToJsonFormat(np, PODS, CREATED)
				case "MODIFY":
					name := UpdatePod(pod, wh.pdm)
					wh.jsonReport.AddToJsonFormat(name, PODS, UPDATED)
				case "DELETED":
					name := RemovePod(pod, wh.pdm)
					wh.jsonReport.AddToJsonFormat(name, PODS, DELETED)
				}
			} else {
				log.Printf("Got unexpected pod from chan: %t, %v", event.Object, event.Object)
			}
		}
		log.Printf("Wathching over pods ended")
	}
}
