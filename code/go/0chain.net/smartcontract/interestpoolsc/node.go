package interestpoolsc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"0chain.net/chaincore/state"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

type GlobalNode struct {
	*SimpleGlobalNode `json:"simple_global_node"`
	ID                string
	MinLockPeriod     int64 `json:"min_lock_period"`
}

func newGlobalNode() *GlobalNode {
	return &GlobalNode{
		ID:               ADDRESS,
		SimpleGlobalNode: &SimpleGlobalNode{},
		MinLockPeriod:    0,
	}
}

func (gn *GlobalNode) Encode() []byte {
	rawMessage := make(map[string]*json.RawMessage)
	// encoding SimpleGlobalNode to json.RawMessage
	simpleNodeEnc := json.RawMessage(gn.SimpleGlobalNode.Encode())
	rawMessage["simple_global_node"] = &simpleNodeEnc
	// encoding simple_global_node to json.RawMeesage
	dur, _ := json.Marshal(time.Duration(gn.MinLockPeriod).String())
	durEnc := json.RawMessage(dur)
	rawMessage["min_lock_period"] = &durEnc
	b, _ := json.Marshal(rawMessage)
	return b
}

func (gn *GlobalNode) Decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	sgn, ok := objMap["simple_global_node"]
	if ok {
		err = gn.SimpleGlobalNode.Decode(*sgn)
		if err != nil {
			return err
		}
	}
	var min string
	minlp, ok := objMap["min_lock_period"]
	if ok {
		err = json.Unmarshal(*minlp, &min)
		if err != nil {
			return err
		}
		dur, err := time.ParseDuration(min)
		if err != nil {
			return err
		}
		gn.MinLockPeriod = int64(dur)
	}
	return nil
}

func (gn *GlobalNode) set(key string, value string) error {
	const pfx = "smart_contracts.interestpoolsc."
	var err error
	switch key {
	case Settings[MinLock]:
		fValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("cannot conver key %s, value %s into state.balane; %v", key, value, err)
		}
		gn.MinLock = state.Balance(fValue * 1e10)
	case Settings[Apr]:
		gn.APR, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("cannot conver key %s, value %s into float64e; %v", key, value, err)
		}
	case Settings[MinLockPeriod]:
		mlp, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("cannot conver key %s, value %s into time.duration; %v", key, value, err)
		}

		gn.MinLockPeriod = int64(mlp)
	case Settings[MaxMint]:
		fValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("cannot conver key %s, value %s into state.balane; %v", key, value, err)
		}
		gn.MaxMint = state.Balance(fValue * 1e10)
	default:
		return fmt.Errorf("config setting %s not found", key)
	}
	return nil
}

func (gn *GlobalNode) GetHash() string {
	return util.ToHex(gn.GetHashBytes())
}

func (gn *GlobalNode) GetHashBytes() []byte {
	return encryption.RawHash(gn.Encode())
}

func (gn *GlobalNode) getKey() datastore.Key {
	return datastore.Key(gn.ID + gn.ID)
}

// canMint more tokens
func (gn *GlobalNode) canMint() bool {
	return gn.SimpleGlobalNode.TotalMinted < gn.SimpleGlobalNode.MaxMint
}
