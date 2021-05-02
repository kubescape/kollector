package watch

import (
	"io"
	"net/http"
	"time"
)

var (
	httpClient = http.Client{Timeout: 5 * time.Second}
)

func getAWSInstanceMetatadata() (string, error) {
	resp, err := httpClient.Get("http://169.254.169.254/latest/meta-data/local-hostname")
	if err != nil {
		return "", err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	io.Copy(io.Discard, resp.Body)
	return "AWS", nil
}

func getGCPInstanceMetatadata() (string, error) {
	req, err := http.NewRequest("GET", "http://169.254.169.254/computeMetadata/v1/?alt=json&recursive=true", nil)
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
	return "AWS", nil
}

func getAzureInstanceMetatadata() (string, error) {
	req, err := http.NewRequest("GET", "http://169.254.169.254/metadata/instance?api-version=2020-09-01", nil)
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
	return "AWS", nil
}

func getInstanceMetadata() (string, error) {
	if resp, err := getAzureInstanceMetatadata(); err == nil {
		return resp, nil
	}
	if resp, err := getGCPInstanceMetatadata(); err == nil {
		return resp, nil
	}
	if resp, err := getAWSInstanceMetatadata(); err == nil {
		return resp, nil
	}
	return "", nil
}
