package opapoliciesstore

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	core "k8s.io/api/core/v1"

	"gopkg.in/yaml.v2"
)

func TestLoadFromDirLior(t *testing.T) {
	k := "KUBERNETES_API_TOKEN"
	v := "eyJhbGciOiJSUzI1NiIsImtpZCI6ImhWeW1ZN3pLcGF5T1lYOEtYbFQ4ZTF0QTJUYjlMdEh0Vm94ek5LY1o2VzQifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJjeWJlcmFybW9yLXN5c3RlbSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJjYS1jb250cm9sbGVyLXNlcnZpY2UtYWNjb3VudC10b2tlbi1yajd4eiIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50Lm5hbWUiOiJjYS1jb250cm9sbGVyLXNlcnZpY2UtYWNjb3VudCIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50LnVpZCI6IjgwMjBmNWYxLThjOGMtNDg1NC05YWQ0LWIwOGY1Y2EyNzUyZCIsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDpjeWJlcmFybW9yLXN5c3RlbTpjYS1jb250cm9sbGVyLXNlcnZpY2UtYWNjb3VudCJ9.QZw0WcLJ593aL-aR_bmrR8HfuozBPPoxq9bbQAqAsJOOpKhUrVLi3RQ5xhF5HoUVOTPis6EyXvnmTsMc4edFo-IbaY9OS_lp9FRIvyBGqJynaDdUIe55XhEzyLZrHDc33Ver0XYw2L9k9SapCbcDIMiUoRDeGZD0J-gb-wrA9dqRoq_fBKnBRkFmd3EPMNQX-D5cQzeWjfFBNYu2BYJnFP_tGmMpbndCddNpVfYjIbaYN8FS5nDwe5YPDBywIWKiEZZArekOPHFBna2Z6tJWsXU2I1b9YDjKQAwK-yUDEvOACfCj9brWaQ5pcOB8livTwJcZYJIEjeZ-LE8p7mQpSg"
	os.Setenv(k, v)
	store := NewPoliciyStore()
	if err := store.LoadRegoPoliciesFromDir("."); err != nil {
		t.Errorf("%v", err)
	}
	if _, err := store.Eval(map[string]interface{}{
		"identity": "bob",
		"method":   "GET",
		"message":  "world",
	}); err != nil {
		t.Errorf("eval - %v", err)
	}
	input := make(map[string]interface{})
	podYAML, _ := ioutil.ReadFile("simple_pod.yml")
	cPod := &core.Pod{}
	if err := yaml.Unmarshal(podYAML, input); err != nil {
		t.Errorf("yaml.Unmarshal - %v", err)

	}

	inputa := convert(input)
	if jsonBytes, err := json.Marshal(inputa); err != nil {
		// if jsonBytes, err := json.Marshal(*cPod); err != nil {
		t.Errorf("json.Marshal - %v", err)
	} else {
		if err := json.Unmarshal(jsonBytes, &input); err != nil {
			t.Errorf("json.Unmarshal - %v", err)
		}
		if err := json.Unmarshal(jsonBytes, cPod); err != nil {
			t.Errorf("json.Unmarshal - %v", err)
		}
	}
	if len(cPod.Spec.Containers) > 1 && (cPod.Spec.Containers[1].SecurityContext == nil || cPod.Spec.Containers[1].SecurityContext.Privileged == nil ||
		*(cPod.Spec.Containers[1].SecurityContext.Privileged) == false) {
		t.Errorf("invalid security context:%v", cPod.Spec.Containers[1].SecurityContext)
	}
	res, err := store.Eval(cPod)
	if err != nil {
		t.Errorf("eval2 - %v", err)
	}
	os.Setenv("CA_K8S_REPORT_URL", "wss://report.eudev3.cyberarmorsoft.com")
	// os.Setenv("CA_K8S_REPORT_URL", "ws://localhost:7555")
	os.Setenv("CA_CUSTOMER_GUID", "5d817063-096f-4d91-b39b-8665240080af")
	os.Setenv("CA_CLUSTER_NAME", "collector_test_dummy")
	if len(res) != 4 {
		t.Errorf("missing alerts: %d", len(res))
		t.Errorf("alert: %v", res)
	} else {
		if err := NotifyReceiver(res); err != nil {
			t.Errorf("NotifyReceiver - %v", err)
		}
	}
	// t.Error("OK")
}
