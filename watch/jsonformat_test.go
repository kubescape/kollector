package watch

import (
<<<<<<< HEAD
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
=======
	"testing"
)

// func TestJson(test *testing.T) {
// 	AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), NODE, CREATED)
// 	AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), SERVICES, DELETED)
// 	AddToJsonFormat([]byte("12343589thfgnvdfklbnvklbnmdfk'lbgfbhs"), PODS, UPDATED)

// 	fmt.Printf("json %s\n", string(PrepareDataToSend()))
// 	test.Errorf("123")
// }

type abc struct {
	A int `json:"a"`
}

func TestJson1(test *testing.T) {
	// var q abc = abc{A: 1}

	// fmt.Printf("123456 %d\n", q.A)
	// w, _ := json.Marshal(q)
	// fmt.Printf("123456 %s\n", string(w))
	// test.Errorf("123")
>>>>>>> 4585080cdfe5a3fd4568c5d02ff0ca7782f593a6
}
