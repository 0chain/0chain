package interestpoolsc

import (
	"encoding/json"
	"time"
)

type newPoolRequest struct {
	Duration time.Duration `json:"duration"`
}

func (npr *newPoolRequest) encode() []byte {
	buff, _ := json.Marshal(npr)
	return buff
}

func (npr *newPoolRequest) decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	var d string
	duration, ok := objMap["duration"]

	if ok {
		err = json.Unmarshal(*duration, &d)
		if err != nil {
			return err
		}
		dur, err := time.ParseDuration(d)
		if err != nil {
			return err
		}
		npr.Duration = dur
	}
	return nil
}
