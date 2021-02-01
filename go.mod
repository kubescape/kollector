module k8s-ca-dashboard-aggregator

go 1.14

replace asterix.cyberarmor.io/cyberarmor/capacketsgo => ./vendor/asterix.cyberarmor.io/cyberarmor/capacketsgo

require (
	asterix.cyberarmor.io/cyberarmor/capacketsgo v0.0.0 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/googleapis/gnostic v0.4.0 // indirect
	github.com/gorilla/websocket v1.4.1
	github.com/imdario/mergo v0.3.8 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
)
