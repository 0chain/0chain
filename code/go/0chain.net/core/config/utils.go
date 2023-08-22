package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0chain/common/core/currency"
)

type ConfigType int

const (
	Int ConfigType = iota
	Int64
	Int32
	Duration
	Float64
	Boolean
	String
	CurrencyCoin
	Key
	Cost
	Strings
)

//go:generate msgp -io=false -tests=false -v

var ConfigTypeName = []string{
	"int",
	"int64",
	"int32",
	"time.duration",
	"float64",
	"bool",
	"string",
	"currency.Coin",
	"datastore.Key",
	"Cost",
	"[]string",
}

// swagger:model StringMap
type StringMap struct {
	Fields map[string]string `json:"fields"`
}

func NewStringMap() *StringMap {
	return &StringMap{
		Fields: make(map[string]string),
	}
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
	case Int32:
		v64, err := strconv.ParseInt(input, 10, 32)
		return int32(v64), err
	case Int64:
		return strconv.ParseInt(input, 10, 64)
	case Duration:
		return time.ParseDuration(input)
	case Float64:
		return strconv.ParseFloat(input, 64)
	case Boolean:
		return strconv.ParseBool(input)
	case String:
		return input, nil
	case CurrencyCoin:
		value, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return nil, err
		}
		return currency.Int64ToCoin(value)
	case Strings:
		return strings.Split(input, ","), nil
	default:
		panic(fmt.Sprintf("StringToInterface input %s unsupported type %v", input, iType))
	}
}
