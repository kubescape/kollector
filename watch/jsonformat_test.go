package watch

import (
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
}
