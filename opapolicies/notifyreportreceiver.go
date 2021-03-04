package opapoliciesstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/francoispqt/gojay"
	"github.com/golang/glog"
	uuid "github.com/satori/go.uuid"
)

var (
	httpClient = http.Client{}
)

// NotifyReceiver notifies reports
func NotifyReceiver(alerts []DesicionOutput) error {
	reportReceiver := os.Getenv("CA_K8S_REPORT_URL")
	reportReceiver = strings.Replace(reportReceiver, "ws", "http", 1)
	reportReceiver += "/report"
	reqBody, err := buildReportSession(alerts)
	if err != nil {
		return fmt.Errorf("buildReportSession failed: %v", err)
	}
	req, err := http.NewRequest("POST", reportReceiver, reqBody)
	if err != nil {
		return fmt.Errorf("NewRequest failed: %v", err)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("httpClient.Do failed: %v", err)
	}
	msg, err := HTTPRespToString(res)
	if err != nil {
		err = fmt.Errorf("%v:%s", err, msg)
	}
	return err
}

func buildReportSession(alerts []DesicionOutput) (io.Reader, error) {
	clusterName := os.Getenv("CA_CLUSTER_NAME")
	machineID := os.Getenv("HOSTNAME")
	wlid := fmt.Sprintf("wlid://cluster-%s/namespace-cyberarmor-system/deployment-ca-dashboard-aggregator", clusterName)
	if len(reportStructBytes) == 0 {
		reportStructBytesa, err := ioutil.ReadFile("./opapolicies/reportSkeleton.json")
		if err != nil {
			return nil, fmt.Errorf("ioutil.ReadFile failed: %v", err)
		}
		reportStructBytes = reportStructBytesa
	}
	ec := &EventsContainer{}
	dec := gojay.NewDecoder(bytes.NewReader(reportStructBytes))
	err := dec.DecodeObject(ec)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal failed: %v", err)
	}
	ec.CreatedTime = time.Now().UTC().Unix()
	ec.TransactionID = 1
	for evntIDx := range ec.EventsList {
		switch ec.EventsList[evntIDx].Type {
		case 0:
			afd := AgentFirstData{}
			err := json.Unmarshal(*ec.EventsList[evntIDx].Payload, &afd)
			if err != nil {
				return nil, fmt.Errorf("json.Unmarshal of AgentFirstData failed: %v", err)
			}
			customerGUID := os.Getenv("CA_CUSTOMER_GUID")
			afd.CustomerGUID = uuid.FromStringOrNil(customerGUID)
			afd.SolutionName = wlid
			afd.CustomerName = wlid
			afd.MachineID = machineID
			afd.SolutionDescription = wlid
			afd.ComponentDescription = wlid
			afd.SolutionOwner = "ARMO!!!"
			afdBytes, err := json.Marshal(afd)
			if err != nil {
				return nil, fmt.Errorf("json.Marshal of AgentFirstData failed: %v", err)
			}
			afdRaw := json.RawMessage(afdBytes)
			ec.EventsList[evntIDx].Payload = &afdRaw
		case 4101:
			ica := InnerComponentAttributes{}
			err := json.Unmarshal(*ec.EventsList[evntIDx].Payload, &ica)
			if err != nil {
				return nil, fmt.Errorf("json.Unmarshal of InnerComponentAttributes failed: %v", err)
			}
			ica.Cluster = clusterName
			ica.GroupingLevel0 = clusterName
			ica.Wlid = wlid
			afdBytes, err := json.Marshal(ica)
			if err != nil {
				return nil, fmt.Errorf("json.Marshal of InnerComponentAttributes failed: %v", err)
			}
			afdRaw := json.RawMessage(afdBytes)
			ec.EventsList[evntIDx].Payload = &afdRaw
		case 4097:
			ica := make(map[string]interface{})
			err := json.Unmarshal(*ec.EventsList[evntIDx].Payload, &ica)
			if err != nil {
				return nil, fmt.Errorf("json.Unmarshal of 4097 failed: %v", err)
			}
			ica["machineID"] = machineID
			afdBytes, err := json.Marshal(ica)
			if err != nil {
				return nil, fmt.Errorf("json.Marshal of 4097 failed: %v", err)
			}
			afdRaw := json.RawMessage(afdBytes)
			ec.EventsList[evntIDx].Payload = &afdRaw
		}
	}
	for alertIdx := range alerts {
		alertBody := map[string]interface{}{
			"level":     100,
			"timestamp": time.Now().UTC().Unix(),
			"details":   alerts[alertIdx],
		}
		afdBytes, err := json.Marshal(alertBody)
		if err != nil {
			return nil, fmt.Errorf("json.Marshal of alert %d failed: %v,%+v", alertIdx, err, alertBody)
		}
		afdRaw := json.RawMessage(afdBytes)
		ec.EventsList = append(ec.EventsList, LightAgentEvent{Type: 0xF071, Payload: &afdRaw})
	}
	ec.EventsList = append(ec.EventsList, LightAgentEvent{Type: 1})
	bytesSlice, err := json.Marshal(*ec)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal failed: %v", err)
	}
	glog.Infof("Reports body: %s", string(bytesSlice))
	fmt.Printf("Reports body: %s", string(bytesSlice))
	return bytes.NewReader(bytesSlice), nil
}

// HTTPRespToString parses the body as string and checks the HTTP status code, it closes the body reader at the end
func HTTPRespToString(resp *http.Response) (string, error) {
	if resp == nil || resp.Body == nil {
		return "", nil
	}
	strBuilder := strings.Builder{}
	defer resp.Body.Close()
	if resp.ContentLength > 0 {
		strBuilder.Grow(int(resp.ContentLength))
	}
	_, err := io.Copy(&strBuilder, resp.Body)
	if err != nil {
		return strBuilder.String(), err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = fmt.Errorf("Response ststus: %d. Content: %s", resp.StatusCode, strBuilder.String())
	}

	return strBuilder.String(), err
}
