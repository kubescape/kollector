package watch

import (
	"encoding/json"

	"github.com/golang/glog"
)

// InterfaceToString convert interface to string
func InterfaceToString(inter interface{}) string {
	byteObj, err := json.Marshal(inter)
	if err != nil {
		glog.Errorf("InterfaceToString, error: %s", err.Error())
		byteObj = []byte{}
	}
	return string(byteObj)
}
