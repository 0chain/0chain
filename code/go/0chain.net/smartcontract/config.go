package smartcontract

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"0chain.net/chaincore/state"
)

type ConfigType int

const (
	Int ConfigType = iota
	Int64
	Int32
	Duration
	Boolean
	String
	StateBalance
	NumberOfTypes
)

var ConfigTypeName = []string{
	"int", "int64", "int32", "time.duration", "bool", "string", "state.Balance",
}

type StringMap struct {
	Fields map[string]string `json:"fields"`
}

func (im *StringMap) Decode(input []byte) error {
	err := json.Unmarshal(input, im)
	if err != nil {
		return err
	}
	return nil
}

func (im *StringMap) Encode() []byte {
	buff, _ := json.Marshal(im)
	return buff
}

func InterfaceMapToStringMap(in map[string]interface{}) map[string]string {
	out := make(map[string]string)
	for key, value := range in {
		out[key] = fmt.Sprintf("%v", value)
	}
	return out
}

func StringToInterface(input string, iType ConfigType) (interface{}, error) {
	switch iType {
	case Int:
		return strconv.Atoi(input)
	case Int64:
		return strconv.ParseInt(input, 10, 64)
	case Int32:
		return strconv.ParseInt(input, 10, 32)
	case Duration:
		return time.ParseDuration(input)
	case Boolean:
		return strconv.ParseBool(input)
	case String:
		return input, nil
	case StateBalance:
		value, err := strconv.ParseInt(input, 10, 64)
		return state.Balance(value), err
	default:
		panic("unsupported type")
	}
}
