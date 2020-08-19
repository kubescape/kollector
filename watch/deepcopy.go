package watch

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

// Clone uses "encoding/gob" package
func Clone(src, dst interface{}) error {
	buff := new(bytes.Buffer)
	enc := gob.NewEncoder(buff)
	dec := gob.NewDecoder(buff)
	enc.Encode(src)
	return dec.Decode(dst)
}

// fo
// DeepCopy uses "encoding/json" package
func DeepCopy(src, dst interface{}) error {
	buff, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(buff, dst)
}
