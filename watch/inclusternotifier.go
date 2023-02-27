package watch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/armosec/armoapi-go/apis"
	"github.com/armosec/cluster-notifier-api-go/notificationserver"
	"github.com/armosec/utils-go/boolutils"
	"github.com/armosec/utils-k8s-go/armometadata"
	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"github.com/kubescape/kollector/consts"
)

var defaultClientInClusterTrigger = http.DefaultClient

func newInClusterNotifier(config *armometadata.ClusterConfig) iClusterNotifier {
	trigger := os.Getenv(consts.ActivateScanOnNewImageFeatureEnvironmentVariable)
	if !boolutils.StringToBool(trigger) {
		return newSkipInClusterNotifier("", "", "")
	}
	return newClusterNotifierImpl(config.AccountID, config.ClusterName, config.GatewayRestURL)
}

type iClusterNotifier interface {
	notifyNewMicroServiceCreatedInTheCluster(namespace string, k8sType string, name string) error
}

type clusterNotifierImpl struct {
	clusterName  string
	customerGuid string
	notifierURL  *url.URL
}

func newClusterNotifierImpl(customerGuid, clusterName, notifierHost string) *clusterNotifierImpl {
	logger.L().Info("setting up cluster trigger notification")
	return &clusterNotifierImpl{
		customerGuid: customerGuid,
		clusterName:  clusterName,
		notifierURL:  generateNotifierURL(notifierHost),
	}
}

func (notifier *clusterNotifierImpl) notifyNewMicroServiceCreatedInTheCluster(namespace string, k8sType string, name string) error {

	var body *bytes.Buffer
	var err error

	if body, err = notifier.createNotificationPostJson(namespace, k8sType, name); err != nil {
		return fmt.Errorf("createNotificationPostJson: fail to create notification post json with err %v", err)
	}

	if err := notifier.executeTriggeredNotification(body); err != nil {
		return fmt.Errorf("executeTriggeredNotification: fail to execute with err %v", err)
	}

	return nil
}

func (notifier *clusterNotifierImpl) createNotificationPostJson(namespace string, k8sType string, name string) (*bytes.Buffer, error) {

	cmds := apis.Commands{}
	wlid := "wlid://cluster-" + notifier.clusterName + "/namespace-" + namespace + "/" + k8sType + "-" + name // TODO: Use a wlid generator function
	cmds.Commands = append(cmds.Commands, apis.Command{CommandName: apis.TypeScanImages, Wlid: wlid})

	notification := notificationserver.Notification{
		Target: map[string]string{
			notificationserver.TargetCustomer:  notifier.customerGuid,
			notificationserver.TargetCluster:   notifier.clusterName,
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

func (notifier *clusterNotifierImpl) executeTriggeredNotification(body *bytes.Buffer) error {

	req, err := http.NewRequest(http.MethodPost, notifier.notifierURL.String(), body)
	if err != nil {
		return err
	}

	logger.L().Info("send", helpers.String("url", notifier.notifierURL.String()))
	resp, err := defaultClientInClusterTrigger.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// generate notifier URL
func generateNotifierURL(host string) *url.URL {
	u := url.URL{}
	u.Scheme = "http"
	u.Host = host
	u.Path = notificationserver.PathRESTV1

	return &u
}

type skipInClusterNotifier struct {
}

func newSkipInClusterNotifier(customerGuid, clusterName, notifierHost string) *skipInClusterNotifier {
	logger.L().Info("skipping cluster trigger notification")
	return &skipInClusterNotifier{}
}

func (skip *skipInClusterNotifier) notifyNewMicroServiceCreatedInTheCluster(namespace string, k8sType string, name string) error {
	return nil
}
