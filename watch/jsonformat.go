package watch

import (
	"encoding/json"
	"log"

	"k8s.io/apimachinery/pkg/version"
)

type JsonType int
type StateType int

const (
	NODE          JsonType = 1
	SERVICES      JsonType = 2
	MICROSERVICES JsonType = 3
	PODS          JsonType = 4
	SECRETS       JsonType = 5
)

const (
	CREATED StateType = 1
	DELETED StateType = 2
	UPDATED StateType = 3
)

type ObjectData struct {
	Created []interface{} `json:"create"`
	Deleted []interface{} `json:"delete"`
	Updated []interface{} `json:"update"`
}

type jsonFormat struct {
	FirstReport             bool          `json:"firstReport"`
	ClusterAPIServerVersion *version.Info `json:"clusterAPIServerVersion,omitempty"`
	Nodes                   ObjectData    `json:"node"`
	Services                ObjectData    `json:"service"`
	MicroServices           ObjectData    `json:"microservice"`
	Pods                    ObjectData    `json:"pod"`
	Secret                  ObjectData    `json:"secret"`
}

func (obj *ObjectData) AddToJsonFormatByState(NewData interface{}, stype StateType) {
	switch stype {
	case CREATED:
		obj.Created = append(obj.Created, NewData)
	case DELETED:
		obj.Deleted = append(obj.Deleted, NewData)
	case UPDATED:
		obj.Updated = append(obj.Updated, NewData)
	}
}

func (jsonReport *jsonFormat) AddToJsonFormat(data interface{}, jtype JsonType, stype StateType) {
	switch jtype {
	case NODE:
		jsonReport.Nodes.AddToJsonFormatByState(data, stype)
	case SERVICES:
		jsonReport.Services.AddToJsonFormatByState(data, stype)
	case MICROSERVICES:
		jsonReport.MicroServices.AddToJsonFormatByState(data, stype)
	case PODS:
		jsonReport.Pods.AddToJsonFormatByState(data, stype)
	case SECRETS:
		jsonReport.Secret.AddToJsonFormatByState(data, stype)
	}

}

//PrepareDataToSend -
func PrepareDataToSend(wh *WatchHandler) []byte {
	jsonReport := wh.jsonReport
	if *wh.GetAggregateFirstDataFlag() {
		jsonReport.ClusterAPIServerVersion = wh.clusterAPIServerVersion
	} else {
		jsonReport.ClusterAPIServerVersion = nil
	}
	jsonReportToSend, err := json.Marshal(jsonReport)
	if nil != err {
		log.Printf("json.Marshal %v", err)
		return nil
	}
	deleteJsonData(wh)
	wh.aggregateFirstDataFlag = false
	return jsonReportToSend
}

//WaitTillNewDataArrived -
func WaitTillNewDataArrived(wh *WatchHandler) bool {
	<-wh.informNewDataChannel
	return true
}

func informNewDataArrive(wh *WatchHandler) {
	if !wh.aggregateFirstDataFlag {
		wh.informNewDataChannel <- 1
	}
}

func deleteObjecData(l *[]interface{}) {
	*l = []interface{}{}
}

func deleteJsonData(wh *WatchHandler) {
	jsonReport := &wh.jsonReport
	deleteObjecData(&jsonReport.Nodes.Created)
	deleteObjecData(&jsonReport.Nodes.Deleted)
	deleteObjecData(&jsonReport.Nodes.Updated)

	deleteObjecData(&jsonReport.Pods.Created)
	deleteObjecData(&jsonReport.Pods.Deleted)
	deleteObjecData(&jsonReport.Pods.Updated)

	deleteObjecData(&jsonReport.Services.Created)
	deleteObjecData(&jsonReport.Services.Deleted)
	deleteObjecData(&jsonReport.Services.Updated)

	deleteObjecData(&jsonReport.MicroServices.Created)
	deleteObjecData(&jsonReport.MicroServices.Deleted)
	deleteObjecData(&jsonReport.MicroServices.Updated)

	deleteObjecData(&jsonReport.Secret.Created)
	deleteObjecData(&jsonReport.Secret.Deleted)
	deleteObjecData(&jsonReport.Secret.Updated)
}
