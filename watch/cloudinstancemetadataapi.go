package watch

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	httpClient = http.Client{Timeout: 5 * time.Second}
)

const (
	awsInstanceMetadataUrl   = "http://169.254.169.254/latest/meta-data/local-hostname"
	gcpInstanceMetadataUrl   = "http://169.254.169.254/computeMetadata/v1/?alt=json&recursive=true"
	azureInstanceMetadataUrl = "http://169.254.169.254/metadata/instance?api-version=2020-09-01"

	awsVendorName   = "AWS"
	gcpVendorName   = "GCP"
	azureVendorName = "Azure"
)

func getAWSInstanceMetadata() (string, error) {
	resp, err := httpClient.Get(awsInstanceMetadataUrl)
	if err != nil {
		return "", err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http error: %s", resp.Status)
	}
	return awsVendorName, nil
}

func getGCPInstanceMetadata() (string, error) {
	req, err := http.NewRequest(http.MethodGet, gcpInstanceMetadataUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http error: %s", resp.Status)
	}
	return gcpVendorName, nil
}

func getAzureInstanceMetadata() (string, error) {
	req, err := http.NewRequest(http.MethodGet, azureInstanceMetadataUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Metadata", "true")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http error: %s", resp.Status)
	}
	return azureVendorName, nil
}

func getInstanceMetadata() (string, error) {
	if resp, err := getAzureInstanceMetadata(); err == nil {
		return resp, nil
	}
	if resp, err := getGCPInstanceMetadata(); err == nil {
		return resp, nil
	}
	if resp, err := getAWSInstanceMetadata(); err == nil {
		return resp, nil
	}
	return "", nil
}
