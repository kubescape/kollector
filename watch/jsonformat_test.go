package watch

import (
	"bytes"
	"testing"
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
