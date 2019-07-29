package watch

import (
	"encoding/json"
)

type JsonType int
type StateType int

const (
	NODE          JsonType = 1
	SERVICES      JsonType = 2
	MICROSERVICES JsonType = 3
	PODS          JsonType = 4
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
	Nodes         ObjectData `json:"node"`
	Services      ObjectData `json:"service"`
	MicroServices ObjectData `json:"microservice"`
	Pods          ObjectData `json:"pod"`
}

//var newdataarrive chan int = make(chan int, 1)

//var jsonReport jsonFormat = jsonFormat{Nodes: ObjectData{Created: make(map[string][]interface{}), Deleted: make(map[string][]interface{}), Updated: make(map[string][]interface{})}, Services: ObjectData{Created: make(map[string][]interface{}), Deleted: make(map[string][]interface{}), Updated: make(map[string][]interface{})}, MicroServices: ObjectData{Created: make(map[string][]interface{}), Deleted: make(map[string][]interface{}), Updated: make(map[string][]interface{})}, Pods: ObjectData{Created: make(map[string][]interface{}), Deleted: make(map[string][]interface{}), Updated: make(map[string][]interface{})}}
func (obj *ObjectData) AddToJsonFormatByState(NewData interface{}, stype StateType) {
	switch stype {
	case CREATED:
		obj.Created = append(obj.Created, NewData)
	case DELETED:
		obj.Created = append(obj.Created, NewData)
	case UPDATED:
		obj.Created = append(obj.Created, NewData)
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
	}

}

func PrepareDataToSend(wh *WatchHandler) []byte {
	jsonReport := wh.jsonReport
	jsonReportToSend, _ := json.Marshal(jsonReport)

	return jsonReportToSend
}

func deleteObjecData(l *[]interface{}) {
	*l = []interface{}{}
}

func DeleteJsonData(wh *WatchHandler) {
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
}
