package opapoliciesstore

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestLoadFromDir(t *testing.T) {
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
	if err := yaml.Unmarshal(podYAML, &input); err != nil {
		t.Errorf("yaml.Unmarshal - %v", err)

	}

	inputa := convert(input)
	if jsonBytes, err := json.Marshal(inputa); err != nil {
		t.Errorf("json.Marshal - %v", err)
	} else {
		if err := json.Unmarshal(jsonBytes, &input); err != nil {
			t.Errorf("json.Unmarshal - %v", err)
		}
	}
	res, err := store.Eval(input)
	if err != nil {
		t.Errorf("eval2 - %v", err)
	}
	os.Setenv("CA_K8S_REPORT_URL", "wss://report.eudev3.cyberarmorsoft.com")
	// os.Setenv("CA_K8S_REPORT_URL", "ws://localhost:7555")
	os.Setenv("CA_CUSTOMER_GUID", "5d817063-096f-4d91-b39b-8665240080af")
	os.Setenv("CA_CLUSTER_NAME", "collector_test_dummy")
	if err := NotifyReceiver(res); err != nil {
		t.Errorf("NotifyReceiver - %v", err)
	}
	// t.Error("OK")
}

func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case string:
		return i.(string)
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	case map[string]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k] = convert(v)
		}
		return m2
	}
	return i
}
