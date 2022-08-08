package watch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/armosec/armoapi-go/apis"
	"github.com/armosec/cluster-notifier-api-go/notificationserver"
	"github.com/armosec/utils-go/boolutils"
	"github.com/golang/glog"
)

var inClusterTriggerURL string
var defaultClientInClusterTrigger = http.DefaultClient

func createNotificationPostJson(namespace string, k8sType string, name string) (*bytes.Buffer, error) {
	glog.Info("creating new in cluster trigger notification")

	cmds := apis.Commands{}
	clusterName := os.Getenv(clusterNameEnvironmentVariable)
	wlid := "wlid://cluster-" + clusterName + "/namespace-" + namespace + "/" + k8sType + "-" + name
	cmds.Commands = append(cmds.Commands, apis.Command{CommandName: apis.TypeScanImages, Wlid: wlid})

	notification := notificationserver.Notification{
		Target: map[string]string{
			notificationserver.TargetCluster:   os.Getenv(clusterNameEnvironmentVariable),
			notificationserver.TargetCustomer:  os.Getenv(customerGuidEnvironmentVariable),
			notificationserver.TargetComponent: notificationserver.TargetComponentTriggerHandler,
			"dest":                             "trigger", // TODO: use const!
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

	url, exist := os.LookupEnv(notificationServerRestEnvironmentVariable)
	if !exist {
		glog.Warningf("%s env var is missing, vulnerability scan on new pod will not work", notificationServerRestEnvironmentVariable)
		return nil
	}
	inClusterTriggerURL = "http://" + url + notificationserver.PathRESTV1

	req, err := http.NewRequest(http.MethodPost, inClusterTriggerURL, body)
	if err != nil {
		return err
	}

	glog.Infof("send post to %s the json %v", inClusterTriggerURL, body.String())
	resp, err := defaultClientInClusterTrigger.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func notifyNewMicroServiceCreatedInTheCluster(namespace string, k8sType string, name string) error {
	trigger := os.Getenv(activateScanOnNewImageFeatureEnvironmentVariable)
	if !boolutils.StringToBool(trigger) {
		return nil
	}

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
