package opapoliciesstore

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v2"
)

var podYAML = []byte(`
kind: Pod
message: world
metadata:
  labels:
    app: golang-inf
  name: golang-inf-79c97c9864-tzhtx
  namespace: armo-playground
  selfLink: /api/v1/namespaces/armo-playground/pods/golang-inf-79c97c9864-tzhtx
spec:
  containers:
  - image: 015253967648.dkr.ecr.eu-central-1.amazonaws.com/armo:1
    name: armo
  - image: 015253967648.dkr.ecr.eu-central-1.amazonaws.com/armo:3
    name: armo3
`)

func TestLoadFromDir(t *testing.T) {
	store := NewPoliciyStore()
	if err := store.LoadRegoPoliciesFromDir("."); err != nil {
		t.Errorf("%v", err)
	}
	if res, err := store.Eval(map[string]interface{}{
		"identity": "bob",
		"method":   "GET",
		"message":  "world",
	}); err != nil {
		t.Errorf("eval - %v", err)
	} else {
		t.Errorf("%+v", res)
	}
	input := make(map[string]interface{})

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

	if res, err := store.Eval(input); err != nil {
		t.Errorf("eval2 - %v", err)
	} else {
		t.Errorf("%+v", res)
	}
	t.Error("OK")
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
