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
	if len(wh.jsonReport.Nodes.Created) != 0 || len(wh.jsonReport.Nodes.Updated) != 0 || len(wh.jsonReport.Nodes.Deleted) != 0 || len(wh.jsonReport.MicroServices.Created) != 0 || len(wh.jsonReport.MicroServices.Updated) != 0 || len(wh.jsonReport.MicroServices.Deleted) != 0 || len(wh.jsonReport.Pods.Created) != 0 || len(wh.jsonReport.Pods.Updated) != 0 || len(wh.jsonReport.Pods.Deleted) != 0 || len(wh.jsonReport.Services.Created) != 0 || len(wh.jsonReport.Services.Updated) != 0 || len(wh.jsonReport.Services.Deleted) != 0 {
		jsonReport := wh.jsonReport
		jsonReportToSend, _ := json.Marshal(jsonReport)

		return jsonReportToSend
	}

	return nil
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
