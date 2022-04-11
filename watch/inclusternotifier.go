package watch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/armosec/armoapi-go/apis"
	"github.com/armosec/cluster-notifier-api-go/notificationserver"
	"github.com/golang/glog"
)

var inClusterTriggerURL string
var defaultClientInClusterTrigger = http.DefaultClient

func createNotificationPostJson(namespace string, k8sType string, name string) (*bytes.Buffer, error) {
	glog.Info("creating new in cluster trigger notification")

	cmds := apis.Commands{}
	clusterName := os.Getenv("CA_CLUSTER_NAME")
	wlid := "wlid://cluster-" + clusterName + "/namespace-" + namespace + "/" + k8sType + "-" + name
	cmds.Commands = append(cmds.Commands, apis.Command{CommandName: apis.SCAN, Wlid: wlid})

	notification := notificationserver.Notification{
		Target: map[string]string{
			notificationserver.TargetCluster:   os.Getenv("CA_CLUSTER_NAME"),
			notificationserver.TargetCustomer:  os.Getenv("CA_CUSTOMER_GUID"),
			notificationserver.TargetComponent: notificationserver.TargetComponentTriggerHandler,
			"dest":                             "trigger",
		},
		Notification: cmds,
	}

	body, err := json.Marshal(notification)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(body), nil
}

func executeTriggeredNotification(body *bytes.Buffer) error {

	url, exist := os.LookupEnv("CA_NOTIFICATION_SERVER_REST")
	if !exist {
		glog.Warning("CA_NOTIFICATION_SERVER_REST env var is missing, vuln scan on new pod will not work")
	}
	inClusterTriggerURL = "http://" + url + notificationserver.PathRESTV1

	glog.Info("post in cluster trigger notification")
	req, err := http.NewRequest("POST", inClusterTriggerURL, body)
	if err != nil {
		return err
	}

	glog.Infof("send post to %s the json %v", inClusterTriggerURL, string(body.Bytes()))
	resp, err := defaultClientInClusterTrigger.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	glog.Info("post in cluster trigger notification successfully")
	return nil
}

func NotifyNewMicroServiceCreatedInTheCluster(namespace string, k8sType string, name string) error {
	var body *bytes.Buffer
	var err error

	if body, err = createNotificationPostJson(namespace, k8sType, name); err != nil {
		return fmt.Errorf("createNotificationPostJson: fail to create notification post json with err %v", err)
	}

	if err := executeTriggeredNotification(body); err != nil {
		return fmt.Errorf("executeTriggeredNotification: fail to execute with err %v", err)
	}

	return nil
}
