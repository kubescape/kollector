package config

import (
	"github.com/armosec/utils-k8s-go/armometadata"
	"github.com/kubescape/backend/pkg/utils"
)

// IConfig is an interface for all config types used in the operator
type IConfig interface {
	ClusterName() string
	AccountID() string
	AccessKey() string
	GatewayRestURL() string
	EventReceiverWebsocketURL() string
	ClusterConfig() *armometadata.ClusterConfig
}

// KollectorConfig implements IConfig
type KollectorConfig struct {
	accountID                 string
	accessKey                 string
	clusterConfig             *armometadata.ClusterConfig
	eventReceiverWebsocketURL string
}

func NewKollectorConfig(clusterConfig *armometadata.ClusterConfig, credentials utils.Credentials, eventReceiverWebsocketURL string) *KollectorConfig {
	return &KollectorConfig{
		accountID:                 credentials.Account,
		accessKey:                 credentials.AccessKey,
		clusterConfig:             clusterConfig,
		eventReceiverWebsocketURL: eventReceiverWebsocketURL,
	}
}

func (k *KollectorConfig) EventReceiverWebsocketURL() string {
	return k.eventReceiverWebsocketURL
}

func (k *KollectorConfig) ClusterName() string {
	return k.clusterConfig.ClusterName
}

func (k *KollectorConfig) AccountID() string {
	return k.accountID
}

func (k *KollectorConfig) AccessKey() string {
	return k.accessKey
}

func (k *KollectorConfig) ClusterConfig() *armometadata.ClusterConfig {
	return k.clusterConfig
}

func (k *KollectorConfig) GatewayRestURL() string {
	return k.clusterConfig.GatewayRestURL
}
