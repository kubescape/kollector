package watch

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// SecretData -
type SecretData struct {
	Secret *corev1.Secret `json:",inline"`
}

// SecretWatch watch over secrets
func (wh *WatchHandler) SecretWatch() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("RECOVER SecretWatch. error: %v\n %s", err, string(debug.Stack()))
		}
	}()
	newStateChan := make(chan bool)
	resourceMap := make(map[string]string)
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
		wh.HandleDataMismatch("secrets", resourceMap)
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
			if err := wh.SecretEventHandler(&event, resourceMap); err != nil {
				break ChanLoop
			}
		}
		glog.Infof("Watching over secrets ended - timeout")
	}
}
func (wh *WatchHandler) SecretEventHandler(event *watch.Event, resourceMap map[string]string) error {
	if secret, ok := event.Object.(*corev1.Secret); ok {
		secret.ManagedFields = []metav1.ManagedFieldsEntry{}
		removeSecretData(secret)
		switch event.Type {
		case "ADDED":
			resourceMap[string(secret.GetUID())] = secret.GetResourceVersion()
			secretdm := SecretData{Secret: secret}
			id := CreateID()
			wh.secretdm.Init(id)
			wh.secretdm.PushBack(id, secretdm)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(secret, SECRETS, CREATED)
		case "MODIFY":
			resourceMap[string(secret.GetUID())] = secret.GetResourceVersion()
			wh.UpdateSecret(secret)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(secret, SECRETS, UPDATED)
		case "DELETED":
			delete(resourceMap, string(secret.GetUID()))
			wh.RemoveSecret(secret)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(secret, SECRETS, DELETED)
		case "BOOKMARK": //only the resource version is changed but it's the same workload
			return nil
		case "ERROR":
			log.Printf("while watching over secrets we got an error ")
			return fmt.Errorf("while watching over secrets we got an error")
		}
	} else {
		log.Printf("Got unexpected secret from chan: %v", event.Object)
		return fmt.Errorf("got unexpected secret from chan")
	}
	return nil
}

// UpdateSecret update websocket when secret is updated
func (wh *WatchHandler) UpdateSecret(secret *corev1.Secret) {
	for id := range wh.secretdm.GetIDs() {
		front := wh.secretdm.Front(id)
		if front == nil || front.Value == nil {
			continue
		}
		secretData, ok := front.Value.(SecretData)
		if !ok || secretData.Secret == nil {
			continue
		}
		if strings.Compare(secretData.Secret.Namespace, secret.Namespace) != 0 {
			continue
		}
		if strings.Compare(secretData.Secret.ObjectMeta.Name, secret.ObjectMeta.Name) == 0 {
			*secretData.Secret = *secret
			log.Printf("secret %s updated", secretData.Secret.ObjectMeta.Name)
			break
		}
		if strings.Compare(secretData.Secret.ObjectMeta.GenerateName, secret.ObjectMeta.Name) == 0 {
			*secretData.Secret = *secret
			log.Printf("secret %s updated", secretData.Secret.ObjectMeta.Name)
			break
		}
	}
}

// RemoveSecret update websocket when secret is removed
func (wh *WatchHandler) RemoveSecret(secret *corev1.Secret) string {
	for id := range wh.secretdm.GetIDs() {
		front := wh.secretdm.Front(id)
		if front == nil || front.Value == nil {
			continue
		}
		secretData, ok := front.Value.(SecretData)
		if !ok || secretData.Secret == nil {
			continue
		}
		if strings.Compare(secretData.Secret.Namespace, secret.Namespace) != 0 {
			continue
		}
		if strings.Compare(secretData.Secret.ObjectMeta.Name, secret.ObjectMeta.Name) == 0 {
			name := secretData.Secret.ObjectMeta.Name
			wh.secretdm.Remove(id)
			log.Printf("secret %s removed", name)
			return name
		}
		if strings.Compare(secretData.Secret.ObjectMeta.GenerateName, secret.ObjectMeta.Name) == 0 {
			gName := secretData.Secret.ObjectMeta.Name
			wh.secretdm.Remove(id)
			log.Printf("secret %s removed", gName)
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
