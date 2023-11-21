package watch

import (
	"context"
	"encoding/json"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/armosec/utils-k8s-go/armometadata"
	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
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
	NAMESPACES    JsonType = 6
)

const (
	CREATED StateType = 1
	DELETED StateType = 2
	UPDATED StateType = 3
)

var (
	FirstReportEmptyBytes  = []byte("{\"firstReport\":true}")
	FirstReportEmptyLength = len(FirstReportEmptyBytes)
)

type ObjectData struct {
	Created []interface{} `json:"create,omitempty"`
	Deleted []interface{} `json:"delete,omitempty"`
	Updated []interface{} `json:"update,omitempty"`
}

type jsonFormat struct {
	FirstReport             bool                        `json:"firstReport"`
	ClusterAPIServerVersion *version.Info               `json:"clusterAPIServerVersion,omitempty"`
	CloudVendor             string                      `json:"cloudVendor,omitempty"`
	Nodes                   *ObjectData                 `json:"node,omitempty"`
	Services                *ObjectData                 `json:"service,omitempty"`
	MicroServices           *ObjectData                 `json:"microservice,omitempty"`
	Pods                    *ObjectData                 `json:"pod,omitempty"`
	Secret                  *ObjectData                 `json:"secret,omitempty"`
	Namespace               *ObjectData                 `json:"namespace,omitempty"`
	InstallationData        *armotypes.InstallationData `json:"installationData,omitempty"`
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

func (obj *ObjectData) Len() int {
	sum := 0
	if obj == nil {
		return sum
	}
	if obj.Created != nil {
		sum += len(obj.Created)
	}
	if obj.Deleted != nil {
		sum += len(obj.Deleted)
	}
	if obj.Updated != nil {
		sum += len(obj.Updated)
	}
	return sum
}

func (jsonReport *jsonFormat) AddToJsonFormat(data interface{}, jtype JsonType, stype StateType) {
	switch jtype {
	case NODE:
		if jsonReport.Nodes == nil {
			jsonReport.Nodes = &ObjectData{}
		}
		jsonReport.Nodes.AddToJsonFormatByState(data, stype)
	case SERVICES:
		if jsonReport.Services == nil {
			jsonReport.Services = &ObjectData{}
		}
		jsonReport.Services.AddToJsonFormatByState(data, stype)
	case MICROSERVICES:
		if jsonReport.MicroServices == nil {
			jsonReport.MicroServices = &ObjectData{}
		}
		jsonReport.MicroServices.AddToJsonFormatByState(data, stype)
	case PODS:
		if jsonReport.Pods == nil {
			jsonReport.Pods = &ObjectData{}
		}
		jsonReport.Pods.AddToJsonFormatByState(data, stype)
	case SECRETS:
		if jsonReport.Secret == nil {
			jsonReport.Secret = &ObjectData{}
		}
		jsonReport.Secret.AddToJsonFormatByState(data, stype)
	case NAMESPACES:
		if jsonReport.Namespace == nil {
			jsonReport.Namespace = &ObjectData{}
		}
		jsonReport.Namespace.AddToJsonFormatByState(data, stype)
	}

}

func prepareDataToSend(ctx context.Context, wh *WatchHandler) []byte {
	jsonReport := wh.jsonReport
	if wh.clusterAPIServerVersion == nil {
		return nil
	}
	if *wh.getAggregateFirstDataFlag() {
		setInstallationData(&jsonReport, *wh.config.ClusterConfig())

		jsonReport.ClusterAPIServerVersion = wh.clusterAPIServerVersion
		jsonReport.CloudVendor = wh.cloudVendor
	} else {
		jsonReport.ClusterAPIServerVersion = nil
		jsonReport.CloudVendor = ""
	}
	if jsonReport.Nodes.Len() == 0 {
		jsonReport.Nodes = nil
	}
	if jsonReport.Services.Len() == 0 {
		jsonReport.Services = nil
	}
	if jsonReport.Secret.Len() == 0 {
		jsonReport.Secret = nil
	}
	if jsonReport.Pods.Len() == 0 {
		jsonReport.Pods = nil
	}
	if jsonReport.MicroServices.Len() == 0 {
		jsonReport.MicroServices = nil
	}
	if jsonReport.Namespace.Len() == 0 {
		jsonReport.Namespace = nil
	}
	jsonReportToSend, err := json.Marshal(jsonReport)
	if nil != err {
		logger.L().Ctx(ctx).Error("In PrepareDataToSend json.Marshal", helpers.Error(err))
		return nil
	}
	deleteJsonData(wh)
	if *wh.getAggregateFirstDataFlag() && !isEmptyFirstReport(jsonReportToSend) {
		wh.aggregateFirstDataFlag = false
	}
	return jsonReportToSend
}

func isEmptyFirstReport(jsonReportToSend []byte) bool {
	// len==0 is for empty json, len==2 is for "{}"
	if len(jsonReportToSend) == 0 || len(jsonReportToSend) == 2 || len(jsonReportToSend) == FirstReportEmptyLength {
		return true
	}

	return false
}

// WaitTillNewDataArrived -
func WaitTillNewDataArrived(wh *WatchHandler) bool {
	<-wh.informNewDataChannel
	return true
}

func informNewDataArrive(wh *WatchHandler) {
	if !wh.aggregateFirstDataFlag || wh.clusterAPIServerVersion != nil {
		wh.informNewDataChannel <- 1
	}
}

func deleteObjectData(l *[]interface{}) {
	*l = []interface{}{}
}

func deleteJsonData(wh *WatchHandler) {
	jsonReport := &wh.jsonReport
	// DO NOT DELETE jsonReport.ClusterAPIServerVersion data. it's not a subject to change

	if jsonReport.Nodes != nil {
		deleteObjectData(&jsonReport.Nodes.Created)
		deleteObjectData(&jsonReport.Nodes.Deleted)
		deleteObjectData(&jsonReport.Nodes.Updated)
	}

	if jsonReport.Pods != nil {
		deleteObjectData(&jsonReport.Pods.Created)
		deleteObjectData(&jsonReport.Pods.Deleted)
		deleteObjectData(&jsonReport.Pods.Updated)
	}

	if jsonReport.Services != nil {
		deleteObjectData(&jsonReport.Services.Created)
		deleteObjectData(&jsonReport.Services.Deleted)
		deleteObjectData(&jsonReport.Services.Updated)
	}

	if jsonReport.MicroServices != nil {
		deleteObjectData(&jsonReport.MicroServices.Created)
		deleteObjectData(&jsonReport.MicroServices.Deleted)
		deleteObjectData(&jsonReport.MicroServices.Updated)
	}

	if jsonReport.Secret != nil {
		deleteObjectData(&jsonReport.Secret.Created)
		deleteObjectData(&jsonReport.Secret.Deleted)
		deleteObjectData(&jsonReport.Secret.Updated)
	}

	if jsonReport.Namespace != nil {
		deleteObjectData(&jsonReport.Namespace.Created)
		deleteObjectData(&jsonReport.Namespace.Deleted)
		deleteObjectData(&jsonReport.Namespace.Updated)
	}
}

func setInstallationData(jsonReport *jsonFormat, config armometadata.ClusterConfig) {
	jsonReport.InstallationData = &armotypes.InstallationData{}
	jsonReport.InstallationData.Namespace = config.Namespace
	jsonReport.InstallationData.RelevantImageVulnerabilitiesEnabled = config.RelevantImageVulnerabilitiesEnabled
	jsonReport.InstallationData.StorageEnabled = config.StorageEnabled
	jsonReport.InstallationData.ImageVulnerabilitiesScanningEnabled = config.ImageVulnerabilitiesScanningEnabled
	jsonReport.InstallationData.PostureScanEnabled = config.PostureScanEnabled
	jsonReport.InstallationData.OtelCollectorEnabled = config.OtelCollectorEnabled
	jsonReport.InstallationData.ClusterName = config.ClusterName
	jsonReport.InstallationData.ClusterProvider = cloudProvider
	jsonReport.InstallationData.RelevantImageVulnerabilitiesConfiguration = config.RelevantImageVulnerabilitiesConfiguration

	logger.L().Debug("setting installation data", helpers.Interface("installation data", jsonReport.InstallationData))
}
