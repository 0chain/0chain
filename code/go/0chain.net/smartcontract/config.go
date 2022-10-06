package smartcontract

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"0chain.net/smartcontract/zbig"

	"0chain.net/chaincore/currency"
)

type ConfigType int

const (
	Int ConfigType = iota
	Int64
	Int32
	Duration
	Boolean
	String
	CurrencyCoin
	Key
	Cost
	Strings
	BigRational
)

//go:generate msgp -io=false -tests=false -v

var ConfigTypeName = []string{
	"int",
	"int64",
	"int32",
	"time.duration",
	"bool",
	"string",
	"currency.Coin",
	"datastore.Key",
	"Cost",
	"[]string",
	"BigRat",
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
	case BigRational:
		r := new(big.Rat)
		_, ok := r.SetString(input)
		if !ok {
			return nil, fmt.Errorf("failed to convert %s to big.rat", input)
		}
		return zbig.BigRat{r}, nil
	default:
		panic(fmt.Sprintf("StringToInterface input %s unsupported type %v", input, iType))
	}
}
