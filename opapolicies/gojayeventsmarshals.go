package opapoliciesstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/francoispqt/gojay"
)

// InnerComponentAttributes are ARMO inner attributes of this component
type InnerComponentAttributes struct {
	Wlid                     string                 `json:"wlid"`
	GroupingLevel0           string                 `json:"groupingLevel0"`
	GroupingLevel1           string                 `json:"groupingLevel1"`
	Project                  string                 `json:"project,omitempty"`
	Namespace                string                 `json:"namespace,omitempty"`
	Kind                     string                 `json:"kind"`
	Name                     string                 `json:"name"`
	Datacentere              string                 `json:"datacenter,omitempty"`
	Cluster                  string                 `json:"cluster,omitempty"`
	ProcessName              string                 `json:"process_name,omitempty"`
	ComponentLevel           string                 `json:"component_level"`
	ImageHash                string                 `json:"imageHash,omitempty"`
	ImageTag                 string                 `json:"imageTag,omitempty"`
	AgentType                string                 `json:"agentType,omitempty"`
	ContainerName            string                 `json:"container_name,omitempty"`
	WorkloadKind             string                 `json:"workloadKind,omitempty"`
	SigningProfileName       string                 `json:"signingProfileName,omitempty"`
	NetworkConfigurationGUID string                 `json:"networkConfigurationGUID,omitempty"`
	AutoAccessTokenUpdate    bool                   `json:"autoAccessTokenUpdate"`
	MetaInfo                 map[string]interface{} `json:"metainfo"`
	Categories               []string               `json:"categories"`
}

// AgentFirstData represents the first data the agent send when it asking for rid
type AgentFirstData struct {
	CustomerGUID         uuid.UUID   `json:"customerGUID"`
	SolutionGUID         uuid.UUID   `json:"solutionGUID"`
	ComponentGUID        uuid.UUID   `json:"componentGUID"`
	CustomerName         string      `json:"customerName"`
	SolutionName         string      `json:"solutionName"`
	ComponentName        string      `json:"componentName"`
	SolutionDescription  string      `json:"solutionDescription"`
	ComponentDescription string      `json:"componentDescription"`
	SolutionOwner        string      `json:"solutionOwner"`
	MachineID            string      `json:"machineID"`
	ExternalIP           string      `json:"externalIP"`
	OutboundPort         string      `json:"outboundPort"`
	AgentVersion         interface{} `json:"agentVersion"`
	Trace                interface{} `json:"trace"`
	BuildTime            interface{} `json:"buildTime"`
}

// AgentEvent represent event whose came from some agent
type AgentEvent struct {
	RID          uuid.UUID   `json:"sessionID"` // the agent instance ID (Report ID)
	Version      int32       `json:"version"`
	CreatedTime  interface{} `json:"createdTime"`
	ReceivedTime time.Time   `json:"receivedTime"`
	Type         int32       `json:"type"`
	Payload      interface{} `json:"payload"`
}

// LightEventList holds slice of LightAgentEvent
type LightEventList []LightAgentEvent

// LightAgentEvent holds the type + payload of single report
type LightAgentEvent struct {
	Type    int32            `json:"type"`
	Payload *json.RawMessage `json:"payload"`
}

// EventsContainer holds list of reports from agent
type EventsContainer struct {
	SessionID      uuid.UUID      `json:"sessionID"` // the agent instance ID (Report ID)
	CreatedTime    interface{}    `json:"createdTime"`
	ResponseFormat string         `json:"responseFormat"`
	TransactionID  uint64         `json:"transactionID"`
	EventsList     LightEventList `json:"eventsList"`
}

func (ae *AgentEvent) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey("sessionID", ae.RID.String())
	enc.Int32Key("version", ae.Version)
	enc.Int32Key("type", ae.Type)
	switch creTime := ae.CreatedTime.(type) {
	case time.Time:
		enc.TimeKey("createdTime", &creTime, time.RFC3339)
		break
	}
	enc.TimeKey("receivedTime", &ae.ReceivedTime, time.RFC3339)
	switch eventPayload := ae.Payload.(type) {
	case []byte:
		tmpEJ := gojay.EmbeddedJSON(eventPayload)
		enc.AddEmbeddedJSONKey("payload", &tmpEJ)
		break
	case *gojay.EmbeddedJSON:
		if eventPayload != nil {
			enc.AddEmbeddedJSONKey("payload", eventPayload)
		} else if ae.Type == 0x1 { // bye-bye packet have no payload
			tmpEJ := gojay.EmbeddedJSON([]byte{123, 125})
			enc.AddEmbeddedJSONKey("payload", &tmpEJ)
		}

		break
	}
}

func (ae *AgentEvent) IsNil() bool {
	return false
}
func (ae *AgentEvent) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	switch key {
	case "sessionID":
		sessionIDString := ""
		err := dec.String(&sessionIDString)
		if err == nil {
			ae.RID, err = uuid.FromString(sessionIDString)
		}
		return err
	case "version":
		return dec.Int32(&ae.Version)
	case "createdTime":
		caTime := int64(0)
		err := dec.Int64(&caTime)
		ae.CreatedTime = caTime
		return err
	case "receivedTime":
		return dec.Time(&ae.ReceivedTime, time.RFC3339)
	case "type":
		return dec.Int32(&ae.Type)
	case "payload":
		ae.Payload = &gojay.EmbeddedJSON{}
		return dec.EmbeddedJSON(ae.Payload.(*gojay.EmbeddedJSON))
	}
	return nil
}

func (ae *AgentEvent) NKeys() int {
	return 6
}

func (t *LightEventList) UnmarshalJSONArray(dec *gojay.Decoder) error {
	lae := LightAgentEvent{}
	if err := dec.Object(&lae); err != nil {
		return err
	}
	if lae.Payload == nil && lae.Type != 0x1 { // only bye packet allowed to send report without payload
		return fmt.Errorf("Empty payload for report type 0x%X", lae.Type)
	}
	*t = append(*t, lae)
	return nil
}

func (lae *LightAgentEvent) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	switch key {
	case "type":
		return dec.Int32(&lae.Type)
	case "payload":
		v := &gojay.EmbeddedJSON{}
		if err := dec.AddEmbeddedJSON(v); err != nil {
			return err
		}
		k := json.RawMessage(*v)
		lae.Payload = &k
		if lae.Payload == nil && lae.Type > 0x1 {
			return fmt.Errorf("Empty payload for report type 0x%X", lae.Type)
		}
		return ValidateJSONTokens(*lae.Payload)
	}
	return nil
}

func (lae *LightAgentEvent) NKeys() int {
	return 2
}

func (container *EventsContainer) UnmarshalJSONObject(dec *gojay.Decoder, key string) (err error) {

	switch key {
	case "sessionID":
		seStr := ""
		err = dec.String(&seStr)
		if err == nil {
			container.SessionID, err = uuid.FromString(seStr)
		}
	case "createdTime":
		err = dec.Interface(&(container.CreatedTime))
	case "responseFormat":
		err = dec.String(&(container.ResponseFormat))
	case "transactionID":
		err = dec.Uint64(&(container.TransactionID))
	case "eventsList":
		err = dec.Array(&(container.EventsList))
	}
	return err
}

func (lae *EventsContainer) NKeys() int {
	return 5
}

// ValidateJSONTokens validates that a given bytes slice contains a valid JSON
func ValidateJSONTokens(jsonBytes []byte) error {
	// The build-in json.Valid not doing the work since it returns only
	// true or false, without explantion of what the error is
	// https://github.com/valyala/fastjson#benchmarks requires the input to be converted into
	//  string, which can lead onto performane issue
	jsonDec := json.NewDecoder(bytes.NewReader(jsonBytes))
	_, err := jsonDec.Token()
	for ; err == nil; _, err = jsonDec.Token() {
	}
	if err != nil && err.Error() != io.EOF.Error() {
		return err
	}
	return nil
}
