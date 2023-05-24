package watch

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	v1core "k8s.io/api/core/v1"
)

var (
	wh WatchHandler
)

func TestJson(test *testing.T) {
	wh.jsonReport.AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), NODE, CREATED)
	wh.jsonReport.AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), SERVICES, DELETED)
	wh.jsonReport.AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), PODS, UPDATED)

	if !bytes.Equal(wh.jsonReport.Nodes.Created[0].([]byte), []byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs")) {
		test.Errorf("NODE")
	}
	if !bytes.Equal(wh.jsonReport.Services.Deleted[0].([]byte), []byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs")) {
		test.Errorf("SERVICES")
	}
	if !bytes.Equal(wh.jsonReport.Pods.Updated[0].([]byte), []byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs")) {
		test.Errorf("PODS")
	}
}

func TestIsEmptyFirstReport(test *testing.T) {
	jsonReport := &jsonFormat{FirstReport: true}
	jsonReportToSend, _ := json.Marshal(jsonReport)
	if !isEmptyFirstReport(jsonReportToSend) {
		test.Errorf("First report is empty")
	}
	jsonReport.CloudVendor = "aws"
	jsonReportToSend, _ = json.Marshal(jsonReport)
	if isEmptyFirstReport(jsonReportToSend) {
		test.Errorf("First report is not empty")
	}
}

func TestSetInstallationData(t *testing.T) {
	testCases := []struct {
		cfgMapData        map[string]string
		expectedError     bool
		expectedErrorMsg  string
		expectedStorage   bool
		expectedNodeAgent bool
	}{
		// Test case 1: Valid clusterData
		{
			cfgMapData: map[string]string{
				"clusterData": `{"storage": "true", "nodeAgent": "false"}`,
			},
			expectedError:     false,
			expectedStorage:   true,
			expectedNodeAgent: false,
		},
		// Test case 2: Missing clusterData
		{
			cfgMapData:        map[string]string{},
			expectedError:     true,
			expectedStorage:   false,
			expectedNodeAgent: false,
		},
		// Test case 3: Invalid clusterData JSON
		{
			cfgMapData: map[string]string{
				"clusterData": `{"storage": "true", "nodeAgent": "invalid"}`,
			},
			expectedError:     false,
			expectedStorage:   true,
			expectedNodeAgent: false,
		},
		// Test case 4: Both storage and nodeAgent are false
		{
			cfgMapData: map[string]string{
				"clusterData": `{"storage": "false", "nodeAgent": "false"}`,
			},
			expectedError:     false,
			expectedStorage:   false,
			expectedNodeAgent: false,
		},
		// Test case 5: Both storage and nodeAgent are true
		{
			cfgMapData: map[string]string{
				"clusterData": `{"storage": "true", "nodeAgent": "true"}`,
			},
			expectedError:     false,
			expectedStorage:   true,
			expectedNodeAgent: true,
		},
	}

	for _, tc := range testCases {
		cfgMap := &v1core.ConfigMap{
			Data: tc.cfgMapData,
		}

		jsonReport := &jsonFormat{}
		err := setInstallationData(cfgMap, jsonReport)
		if tc.expectedError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)

			if jsonReport.InstallationData.StorageEnabled != tc.expectedStorage {
				t.Errorf("expected StorageEnabled to be %v, got %v", tc.expectedStorage, jsonReport.InstallationData.StorageEnabled)
			}
			if jsonReport.InstallationData.NodeAgentEnabled != tc.expectedNodeAgent {
				t.Errorf("expected NodeAgentEnabled to be %v, got %v", tc.expectedNodeAgent, jsonReport.InstallationData.NodeAgentEnabled)
			}
		}
	}
}
