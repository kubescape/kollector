module k8s-armo-collector

go 1.13

require (
	github.com/armosec/armoapi-go v0.0.2
	github.com/armosec/cluster-notifier-api-go v0.0.2
	github.com/armosec/k8s-interface v0.0.70
	github.com/armosec/utils-k8s-go v0.0.7
	github.com/francoispqt/gojay v1.2.13
	github.com/golang/glog v1.0.0
	github.com/gorilla/websocket v1.5.0
	github.com/open-policy-agent/opa v0.39.0
	github.com/satori/go.uuid v1.2.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.23.5
	k8s.io/apiextensions-apiserver v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v0.23.5
)

require (
	github.com/armosec/armoapi-go v0.0.68 // indirect
	github.com/armosec/utils-go v0.0.5 // indirect
)
