package watch

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// secretData -
type secretData struct {
	Secret *corev1.Secret `json:",inline"`
}

// SecretWatch watch over secrets
func (wh *WatchHandler) SecretWatch() {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorf("RECOVER SecretWatch. error: %v\n %s", err, string(debug.Stack()))
		}
	}()
	var lastWatchEventCreationTime time.Time
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
WatchLoop:
	for {
		glog.Infof("Watching over secrets starting")
		secretsWatcher, err := wh.RestAPIClient.CoreV1().Secrets("").Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			glog.Errorf("Failed watching over secrets. %s", err.Error())
			time.Sleep(3 * time.Second)
			continue
		}
		secretsChan := secretsWatcher.ResultChan()
		glog.Infof("Watching over secrets started")
	ChanLoop:
		for {
			var event watch.Event
			select {
			case event = <-secretsChan:
			case <-newStateChan:
				secretsWatcher.Stop()
				glog.Errorf("Secrets watch - newStateChan signal")
				continue WatchLoop
			}

			if event.Type == watch.Error {
				glog.Errorf("Secrets watch chan loop error: %v", event.Object)
				secretsWatcher.Stop()
				break ChanLoop
			}
			if err := wh.secretEventHandler(&event, lastWatchEventCreationTime); err != nil {
				break ChanLoop
			}
		}
		lastWatchEventCreationTime = time.Now()
		glog.Infof("Watching over secrets ended - timeout")
	}
}
func (wh *WatchHandler) secretEventHandler(event *watch.Event, lastWatchEventCreationTime time.Time) error {
	if secret, ok := event.Object.(*corev1.Secret); ok {
		if !wh.isNamespaceWatched(secret.Namespace) {
			return nil
		}
		secret.ManagedFields = []metav1.ManagedFieldsEntry{}
		removeSecretData(secret)
		switch event.Type {
		case "ADDED":
			if secret.CreationTimestamp.Time.Before(lastWatchEventCreationTime) {
				glog.Infof("secret %s already exist, will not be reported", secret.ObjectMeta.Name)
				return nil
			}
			secretdm := secretData{Secret: secret}
			id := CreateID()
			wh.secretdm.init(id)
			wh.secretdm.pushBack(id, secretdm)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(secret, SECRETS, CREATED)
		case "MODIFY":
			wh.updateSecret(secret)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(secret, SECRETS, UPDATED)
		case "DELETED":
			wh.removeSecret(secret)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(secret, SECRETS, DELETED)
		case "BOOKMARK": //only the resource version is changed but it's the same workload
			return nil
		case "ERROR":
			glog.Errorf("while watching over secrets we got an error: %v", event)
			return fmt.Errorf("while watching over secrets we got an error")
		}
	} else {
		return fmt.Errorf("got unexpected secret from chan")
	}
	return nil
}

// UpdateSecret update websocket when secret is updated
func (wh *WatchHandler) updateSecret(secret *corev1.Secret) {
	for _, id := range wh.secretdm.getIDs() {
		front := wh.secretdm.front(id)
		if front == nil || front.Value == nil {
			continue
		}
		secretData, ok := front.Value.(secretData)
		if !ok || secretData.Secret == nil {
			continue
		}
		if strings.Compare(secretData.Secret.Namespace, secret.Namespace) != 0 {
			continue
		}
		if strings.Compare(secretData.Secret.ObjectMeta.Name, secret.ObjectMeta.Name) == 0 {
			*secretData.Secret = *secret
			glog.Infof("secret %s updated", secretData.Secret.ObjectMeta.Name)
			break
		}
		if strings.Compare(secretData.Secret.ObjectMeta.GenerateName, secret.ObjectMeta.Name) == 0 {
			*secretData.Secret = *secret
			glog.Infof("secret %s updated", secretData.Secret.ObjectMeta.Name)
			break
		}
	}
}

// RemoveSecret update websocket when secret is removed
func (wh *WatchHandler) removeSecret(secret *corev1.Secret) string {
	for _, id := range wh.secretdm.getIDs() {
		front := wh.secretdm.front(id)
		if front == nil || front.Value == nil {
			continue
		}
		secretData, ok := front.Value.(secretData)
		if !ok || secretData.Secret == nil {
			continue
		}
		if strings.Compare(secretData.Secret.Namespace, secret.Namespace) != 0 {
			continue
		}
		if strings.Compare(secretData.Secret.ObjectMeta.Name, secret.ObjectMeta.Name) == 0 {
			name := secretData.Secret.ObjectMeta.Name
			wh.secretdm.remove(id)
			glog.Infof("secret %s removed", name)
			return name
		}
		if strings.Compare(secretData.Secret.ObjectMeta.GenerateName, secret.ObjectMeta.Name) == 0 {
			gName := secretData.Secret.ObjectMeta.Name
			wh.secretdm.remove(id)
			glog.Infof("secret %s removed", gName)
			return gName
		}
	}
	return ""
}
func removeSecretData(secret *corev1.Secret) {
	secret.Data = nil
	if secret.Annotations != nil {
		delete(secret.Annotations, "data")
		delete(secret.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}
}
