package watch

import (
	"encoding/json"

	"github.com/golang/glog"
)

// InterfaceToString conver interface to string
func InterfaceToString(inter interface{}) string {
	bytObj, err := json.Marshal(inter)
	if err != nil {
		glog.Error("InterfaceToString, error: %s", err.Error())
		bytObj = []byte{}
	}
	return string(bytObj)
}
