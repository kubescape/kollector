package watch

import (
	"github.com/kubescape/k8s-interface/cloudsupport"
	"github.com/kubescape/k8s-interface/k8sinterface"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setCloudProvider(k8sApi *k8sinterface.KubernetesApi) error {
	nodeList, err := k8sApi.KubernetesClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	cloudProvider = cloudsupport.GetCloudProvider(nodeList)
	return nil
}
