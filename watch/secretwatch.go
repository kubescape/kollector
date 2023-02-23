package watch

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	logger "github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
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
			logger.L().Error("RECOVER SecretWatch", helpers.Interface("error", err), helpers.String("stack", string(debug.Stack())))
		}
	}()
	var lastWatchEventCreationTime time.Time
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
WatchLoop:
	for {
		logger.L().Info("Watching over secrets starting")
		secretsWatcher, err := wh.RestAPIClient.CoreV1().Secrets("").Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		secretsChan := secretsWatcher.ResultChan()
		logger.L().Info("Watching over secrets started")
	ChanLoop:
		for {
			var event watch.Event
			select {
			case event = <-secretsChan:
			case <-newStateChan:
				secretsWatcher.Stop()
				continue WatchLoop
			}

			if event.Type == watch.Error {
				secretsWatcher.Stop()
				break ChanLoop
			}
			if err := wh.secretEventHandler(&event, lastWatchEventCreationTime); err != nil {
				break ChanLoop
			}
		}
		lastWatchEventCreationTime = time.Now()
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
			break
		}
		if strings.Compare(secretData.Secret.ObjectMeta.GenerateName, secret.ObjectMeta.Name) == 0 {
			*secretData.Secret = *secret
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
			return name
		}
		if strings.Compare(secretData.Secret.ObjectMeta.GenerateName, secret.ObjectMeta.Name) == 0 {
			gName := secretData.Secret.ObjectMeta.Name
			wh.secretdm.remove(id)
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
