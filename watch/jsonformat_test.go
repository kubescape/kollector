package watch

import (
	"bytes"
	"fmt"
	"testing"
)

var (
	wh WatchHandler
)

func TestJson(test *testing.T) {
	wh.jsonReport.AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), NODE, CREATED)
	wh.jsonReport.AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), SERVICES, DELETED)
	wh.jsonReport.AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), PODS, UPDATED)

	fmt.Printf("json %s\n", string(PrepareDataToSend(&wh)))
	if bytes.Compare(wh.jsonReport.Nodes.Created[0].([]byte), []byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs")) != 0 {
		test.Errorf("NODE")
	}
	if bytes.Compare(wh.jsonReport.Services.Deleted[0].([]byte), []byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs")) != 0 {
		test.Errorf("SERVICES")
	}
	if bytes.Compare(wh.jsonReport.Pods.Updated[0].([]byte), []byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs")) != 0 {
		test.Errorf("PODS")
	}
}
