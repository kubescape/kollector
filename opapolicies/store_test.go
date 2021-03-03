package opapoliciesstore

import (
	"encoding/json"
	"io/ioutil"
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

	if _, err := store.Eval(input); err != nil {
		t.Errorf("eval2 - %v", err)
	}
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
