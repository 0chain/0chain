package util

import "encoding/json"

//go:generate msgp -v -io=false -tests=false -unexported=true

type deadNodes struct {
	Nodes map[string]int64 `json:"nodes"`
}

func (d *deadNodes) decode(data []byte, v int64) error {
	if v < 2890000 {
		return json.Unmarshal(data, d)
	}

	_, err := d.UnmarshalMsg(data)
	return err
}

func (d *deadNodes) encode(v int64) ([]byte, error) {
	if v < 2890000 {
		return json.Marshal(d)
	}

	return d.MarshalMsg(nil)
}
