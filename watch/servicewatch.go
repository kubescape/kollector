package watch

import (
	"container/list"
	"log"
	"strings"
	"time"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceData struct {
	Service *core.Service `json:"data"`
}

func UpdateService(service *core.Service, sdm map[int]*list.List) string {
	for _, v := range sdm {
		if strings.Compare(v.Front().Value.(ServiceData).Service.ObjectMeta.Name, service.ObjectMeta.Name) == 0 {
			*v.Front().Value.(ServiceData).Service = *service
			log.Printf("service %s updated", v.Front().Value.(ServiceData).Service.ObjectMeta.Name)
			return v.Front().Value.(ServiceData).Service.ObjectMeta.Name
		}
		if strings.Compare(v.Front().Value.(ServiceData).Service.ObjectMeta.GenerateName, service.ObjectMeta.Name) == 0 {
			*v.Front().Value.(ServiceData).Service = *service
			log.Printf("service %s updated", v.Front().Value.(ServiceData).Service.ObjectMeta.Name)
			return v.Front().Value.(ServiceData).Service.ObjectMeta.Name
		}
	}
	return ""
}

func RemoveService(service *core.Service, sdm map[int]*list.List) string {
	for _, v := range sdm {
		if strings.Compare(v.Front().Value.(ServiceData).Service.ObjectMeta.Name, service.ObjectMeta.Name) == 0 {
			v.Remove(v.Front())
			log.Printf("service %s removed", v.Front().Value.(ServiceData).Service.ObjectMeta.Name)
			return v.Front().Value.(ServiceData).Service.ObjectMeta.Name
		}
		if strings.Compare(v.Front().Value.(ServiceData).Service.ObjectMeta.GenerateName, service.ObjectMeta.Name) == 0 {
			v.Remove(v.Front())
			log.Printf("service %s removed", v.Front().Value.(ServiceData).Service.ObjectMeta.Name)
			return v.Front().Value.(ServiceData).Service.ObjectMeta.Name
		}
	}
	return ""
}

func (wh *WatchHandler) ServiceWatch(namespace string) {
	log.Printf("Watching over services starting")
	for {
		podsWatcher, err := wh.RestAPIClient.CoreV1().Services(namespace).Watch(metav1.ListOptions{Watch: true})
		if err != nil {
			log.Printf("Cannot wathching over services. %v", err)
			time.Sleep(time.Duration(10) * time.Second)
			continue
		}
		podsChan := podsWatcher.ResultChan()
		for event := range podsChan {
			if service, ok := event.Object.(*core.Service); ok {
				switch event.Type {
				case "ADDED":
					id := CreateID()
					if wh.sdm[id] == nil {
						wh.sdm[id] = list.New()
					}
					sd := ServiceData{Service: service}
					wh.sdm[id].PushBack(sd)
					wh.jsonReport.AddToJsonFormat(service, SERVICES, CREATED)
				case "MODIFY":
					UpdateService(service, wh.sdm)
					wh.jsonReport.AddToJsonFormat(service, SERVICES, UPDATED)
				case "DELETED":
					RemoveService(service, wh.sdm)
					wh.jsonReport.AddToJsonFormat(service, SERVICES, DELETED)
				}
			} else {
				log.Printf("Got unexpected pod from chan: %t, %v", event.Object, event.Object)
			}
		}
		log.Printf("Wathching over services ended")
	}
}
