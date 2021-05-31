package watch

import (
	"log"
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
			log.Printf("RECOVER SecretWatch. error: %v", err)
		}
	}()
	newStateChan := make(chan bool)
	wh.newStateReportChans = append(wh.newStateReportChans, newStateChan)
WatchLoop:
	for {
		log.Printf("Watching over secrets starting")
		secretsWatcher, err := wh.RestAPIClient.CoreV1().Secrets("").Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			glog.Errorf("Failed watching over secrets. %s", err.Error())
			time.Sleep(time.Duration(3) * time.Second)
			continue
		}
		secretsChan := secretsWatcher.ResultChan()
		log.Printf("Watching over secrets started")
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
			log.Printf("secret event: %v", event)
			wh.SecretEventHandler(&event)
		}
		log.Printf("Watching over secrets ended - timeout")
	}
}
func (wh *WatchHandler) SecretEventHandler(event *watch.Event) {
	if secret, ok := event.Object.(*corev1.Secret); ok {
		removeSecretData(secret)
		switch event.Type {
		case "ADDED":
			secretdm := SecretData{Secret: secret}
			id := CreateID()
			wh.secretdm.Init(id)
			wh.secretdm.PushBack(id, secretdm)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(secret, SECRETS, CREATED)
		case "MODIFY":
			wh.UpdateSecret(secret)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(secret, SECRETS, UPDATED)
		case "DELETED":
			wh.RemoveSecret(secret)
			informNewDataArrive(wh)
			wh.jsonReport.AddToJsonFormat(secret, SECRETS, DELETED)
		case "BOOKMARK": //only the resource version is changed but it's the same workload
			return
		case "ERROR":
			log.Printf("while watching over secrets we got an error: ")
		}
	} else {
		log.Printf("Got unexpected secret from chan: %t, %v", event.Object, event.Object)
	}
}

// UpdateSecret update websocket when secret is updated
func (wh *WatchHandler) UpdateSecret(secret *corev1.Secret) {
	for id := range wh.secretdm.GetIDs() {
		front := wh.secretdm.Front(id)
		if front == nil {
			continue
		}
		if strings.Compare(front.Value.(SecretData).Secret.ObjectMeta.Name, secret.ObjectMeta.Name) == 0 {
			*front.Value.(SecretData).Secret = *secret
			log.Printf("secret %s updated", front.Value.(SecretData).Secret.ObjectMeta.Name)
			break
		}
		if strings.Compare(front.Value.(SecretData).Secret.ObjectMeta.GenerateName, secret.ObjectMeta.Name) == 0 {
			*front.Value.(SecretData).Secret = *secret
			log.Printf("secret %s updated", front.Value.(SecretData).Secret.ObjectMeta.Name)
			break
		}
	}
}

// RemoveSecret update websocket when secret is removed
func (wh *WatchHandler) RemoveSecret(secret *corev1.Secret) string {
	for id := range wh.secretdm.GetIDs() {
		front := wh.secretdm.Front(id)
		if front == nil {
			continue
		}
		if strings.Compare(front.Value.(SecretData).Secret.ObjectMeta.Name, secret.ObjectMeta.Name) == 0 {
			name := front.Value.(SecretData).Secret.ObjectMeta.Name
			wh.secretdm.Remove(id)
			log.Printf("secret %s removed", name)
			return name
		}
		if strings.Compare(front.Value.(SecretData).Secret.ObjectMeta.GenerateName, secret.ObjectMeta.Name) == 0 {
			gName := front.Value.(SecretData).Secret.ObjectMeta.Name
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
