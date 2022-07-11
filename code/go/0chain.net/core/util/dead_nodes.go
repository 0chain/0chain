package util

import (
	"encoding/json"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

//go:generate msgp -v -io=false -tests=false -unexported=true

type deadNodes struct {
	Nodes map[string]int64 `json:"nodes"`
}

func (d *deadNodes) decode(data []byte, v int64) error {
	pv := v - 1
	if pv < 2860000 {
		return json.Unmarshal(data, d)
	}

	_, err := d.UnmarshalMsg(data)
	if err != nil {
		// try decode with json again
		return json.Unmarshal(data, d)
	}
	return err
}

func (d *deadNodes) encode(v int64) ([]byte, error) {
	logging.Logger.Debug("encode dead nodes", zap.Int64("round", v))
	if v < 2860000 {
		return json.Marshal(d)
	}

	return d.MarshalMsg(nil)
}
