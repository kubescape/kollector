package opapoliciesstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
)

// DesicionOutput defines the expected result of given policy desicion
type DesicionOutput struct {
	Alert        bool   `json:"alert"`         // Is ARMO suppose to alert on
	AlertMessage string `json:"alert-message"` // The message ARMO ought to show as alert message
	AlertScore   int    `json:"alert-score"`   // The score of the severity of the alert
	AlertObject  string `json:"alert-object"`  // The policy object which cause this alert
	// alert-object
	Prevent bool `json:"prevent"` // Is ARMO need suppose to prevent the execution of this process
}

// PoliciyStore holds set of policies to validate
type PoliciyStore struct {
	policiesList     []rego.Rego
	compiledPolicies *ast.Compiler
}

// NewPoliciyStore returns new policy store object
func NewPoliciyStore() *PoliciyStore {
	return &PoliciyStore{}
}

// LoadRegoPoliciesFromDir loads the policies list from *.rego file in given directory
// if no directory sepcified, it will load the policies from "./opapolicies" directory
func (ps *PoliciyStore) LoadRegoPoliciesFromDir(dir string) error {
	if dir == "" {
		dir = "./opapolicies"
	}
	dir, _ = filepath.Abs(dir)
	policiesMap := make(map[string]string)

	// Compile the module. The keys are used as identifiers in error messages.
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(path, ".rego") && !info.IsDir() {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				glog.Errorf("LoadRegoPoliciesFromDir, Failed to load: %s: %v", path, err)
			} else {
				glog.Infof("LoadRegoPoliciesFromDir, loaded: %s: %v", path, err)
				policiesMap[path[len(dir):]] = string(content)
			}
		}
		return nil
	})

	compiler, err := ast.CompileModules(policiesMap)
	if err != nil {
		return fmt.Errorf("In LoadRegoPoliciesFromDir: %v", err)
	}

	ps.compiledPolicies = compiler
	return err
}

// Eval eval a given input against all pre-compiled policies
func (ps *PoliciyStore) Eval(input interface{}) ([]DesicionOutput, error) {
	iputByets, err := json.Marshal(input)
	if err == nil {
		glog.Infof("iputByets: %s", string(iputByets))
	}
	result := make([]DesicionOutput, 0)
	rego := rego.New(
		rego.Query("data.armo_builtins"),
		// rego.ParsedPackage(ps.compiledPolicies.ModuleTree.Key.String())
		rego.Compiler(ps.compiledPolicies),
		rego.Input(input),
	)

	// Run evaluation.
	rs, err := rego.Eval(context.Background())
	if err != nil {
		return result, fmt.Errorf("Eval failed: %v", err)
	}
	for resultIdx := range rs {
		for desicionIdx := range rs[resultIdx].Expressions {
			if resMap, ok := rs[resultIdx].Expressions[desicionIdx].Value.(map[string]interface{}); ok {
				for objName := range resMap {
					jsonBytes, err := json.Marshal(resMap[objName])
					if err != nil {
						return result, fmt.Errorf("json.Marshal of desicion '%+v' failed: %v", resMap[objName], err)
					}
					desObj := make([]DesicionOutput, 0)
					err = json.Unmarshal(jsonBytes, &desObj)
					if err != nil {
						glog.Error("json.Unmarshal of desicion failed", resMap[objName], err)
					} else {
						result = append(result, desObj...)
					}
				}
			}
		}
	}
	return result, nil
}
