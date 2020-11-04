package watch

import (
	"container/list"
	"log"
	"strings"
	"time"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretData -
type SecretData struct {
	Secret *core.Secret `json:",inline"`
}

// UpdateSecret update websocket when secret is updated
func UpdateSecret(secret *core.Secret, secretdm map[int]*list.List) string {
	for _, v := range secretdm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(SecretData).Secret.ObjectMeta.Name, secret.ObjectMeta.Name) == 0 {
			*v.Front().Value.(SecretData).Secret = *secret
			log.Printf("secret %s updated", v.Front().Value.(SecretData).Secret.ObjectMeta.Name)
			return v.Front().Value.(SecretData).Secret.ObjectMeta.Name
		}
		if strings.Compare(v.Front().Value.(SecretData).Secret.ObjectMeta.GenerateName, secret.ObjectMeta.Name) == 0 {
			*v.Front().Value.(SecretData).Secret = *secret
			log.Printf("secret %s updated", v.Front().Value.(SecretData).Secret.ObjectMeta.Name)
			return v.Front().Value.(SecretData).Secret.ObjectMeta.Name
		}
	}
	return ""
}

// RemoveSecret update websocket when secret is removed
func RemoveSecret(secret *core.Secret, secretdm map[int]*list.List) string {
	for _, v := range secretdm {
		if v == nil || v.Len() == 0 {
			continue
		}
		if strings.Compare(v.Front().Value.(SecretData).Secret.ObjectMeta.Name, secret.ObjectMeta.Name) == 0 {
			name := v.Front().Value.(SecretData).Secret.ObjectMeta.Name
			v.Remove(v.Front())
			log.Printf("secret %s removed", name)
			return name
		}
		if strings.Compare(v.Front().Value.(SecretData).Secret.ObjectMeta.GenerateName, secret.ObjectMeta.Name) == 0 {
			gName := v.Front().Value.(SecretData).Secret.ObjectMeta.Name
			v.Remove(v.Front())
			log.Printf("secret %s removed", gName)
			return gName
		}
	}
	return ""
}

// SecretWatch watch over secrets
func (wh *WatchHandler) SecretWatch() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("RECOVER SecretWatch. error: %v", err)
		}
	}()
	log.Printf("Watching over secrets starting")
	for {
		secretsWatcher, err := wh.RestAPIClient.CoreV1().Secrets("").Watch(globalHTTPContext, metav1.ListOptions{Watch: true})
		if err != nil {
			log.Printf("Cannot wathching over secrets. %v", err)
			time.Sleep(time.Duration(10) * time.Second)
			continue
		}
		secretsChan := secretsWatcher.ResultChan()
		log.Printf("Watching over secrets started")
		for event := range secretsChan {
			if secret, ok := event.Object.(*core.Secret); ok {
				removeSecretData(secret)
				switch event.Type {
				case "ADDED":
					if !newSecret(wh.secretdm, secret.SelfLink) {
						break
					}
					id := CreateID()
					if wh.secretdm[id] == nil {
						wh.secretdm[id] = list.New()
					}
					secretdm := SecretData{Secret: secret}
					wh.secretdm[id].PushBack(secretdm)
					informNewDataArrive(wh)
					wh.jsonReport.AddToJsonFormat(secret, SECRETS, CREATED)
				case "MODIFY":
					UpdateSecret(secret, wh.secretdm)
					informNewDataArrive(wh)
					wh.jsonReport.AddToJsonFormat(secret, SECRETS, UPDATED)
				case "DELETED":
					RemoveSecret(secret, wh.secretdm)
					informNewDataArrive(wh)
					wh.jsonReport.AddToJsonFormat(secret, SECRETS, DELETED)
				case "BOOKMARK": //only the resource version is changed but it's the same workload
					continue
				case "ERROR":
					log.Printf("while watching over secrets we got an error: ")
				}
			} else {
				log.Printf("Got unexpected secret from chan: %t, %v", event.Object, event.Object)
			}
		}
		log.Printf("Wathching over secrets ended - since we got timeout")
	}
}

func removeSecretData(secret *core.Secret) {
	secret.Data = nil
	if secret.Annotations != nil {
		delete(secret.Annotations, "data")
		delete(secret.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}
}

func newSecret(secretdm map[int]*list.List, selfLink string) bool {
	if secretdm == nil || selfLink == "" {
		return true
	}
	for _, secret := range secretdm {
		if secret.Front() != nil {
			element := secret.Front().Next()
			for element != nil {
				if element.Value.(SecretData).Secret.SelfLink == selfLink {
					return false
				}
				element = element.Next()
			}
		}
	}
	return true
}
