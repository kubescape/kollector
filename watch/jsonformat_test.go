package watch

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/armosec/utils-k8s-go/armometadata"
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
		name   string
		config armometadata.ClusterConfig
	}{

		{
			name: "all true",
			config: armometadata.ClusterConfig{
				Namespace:                           "test-namespace",
				RelevantImageVulnerabilitiesEnabled: true,
				StorageEnabled:                      true,
				ImageVulnerabilitiesScanningEnabled: true,
				PostureScanEnabled:                  true,
				OtelCollectorEnabled:                true,
				ClusterName:                         "test-cluster",
			},
		},
		{
			name: "all false",
			config: armometadata.ClusterConfig{
				Namespace:                           "test-namespace",
				RelevantImageVulnerabilitiesEnabled: false,
				StorageEnabled:                      false,
				ImageVulnerabilitiesScanningEnabled: false,
				PostureScanEnabled:                  false,
				OtelCollectorEnabled:                false,
				ClusterName:                         "test-cluster",
			},
		},
		{
			name: "half true half false",
			config: armometadata.ClusterConfig{
				Namespace:                           "test-namespace",
				RelevantImageVulnerabilitiesEnabled: false,
				StorageEnabled:                      true,
				ImageVulnerabilitiesScanningEnabled: false,
				PostureScanEnabled:                  true,
				OtelCollectorEnabled:                false,
				ClusterName:                         "test-cluster",
			},
		},
		{
			name: "empty",
			config: armometadata.ClusterConfig{
				Namespace:   "test-namespace",
				ClusterName: "test-cluster",
			},
		},
	}
	for _, tc := range testCases {
		jsonReport := &jsonFormat{}
		setInstallationData(jsonReport, tc.config)
		if jsonReport.InstallationData.Namespace != tc.config.Namespace {
			t.Errorf("Namespace is not equal")
		}
		if jsonReport.InstallationData.RelevantImageVulnerabilitiesEnabled != tc.config.RelevantImageVulnerabilitiesEnabled {
			t.Errorf("RelevantImageVulnerabilitiesEnabled is not equal")
		}
		if jsonReport.InstallationData.StorageEnabled != tc.config.StorageEnabled {
			t.Errorf("StorageEnabled is not equal")
		}
		if jsonReport.InstallationData.ImageVulnerabilitiesScanningEnabled != tc.config.ImageVulnerabilitiesScanningEnabled {
			t.Errorf("ImageVulnerabilitiesScanningEnabled is not equal")
		}
		if jsonReport.InstallationData.PostureScanEnabled != tc.config.PostureScanEnabled {
			t.Errorf("PostureScanEnabled is not equal")
		}
		if jsonReport.InstallationData.OtelCollectorEnabled != tc.config.OtelCollectorEnabled {
			t.Errorf("OtelCollectorEnabled is not equal")
		}
		if jsonReport.InstallationData.ClusterName != tc.config.ClusterName {
			t.Errorf("ClusterName is not equal")
		}
	}
}
