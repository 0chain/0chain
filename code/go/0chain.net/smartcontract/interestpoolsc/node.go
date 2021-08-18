package interestpoolsc

import (
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"encoding/json"
	"fmt"
	"time"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

type GlobalNode struct {
	ID                datastore.Key
	*SimpleGlobalNode `json:"simple_global_node"`
	MinLockPeriod     time.Duration `json:"min_lock_period"`
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
	dur, _ := json.Marshal(gn.MinLockPeriod.String())
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
		gn.MinLockPeriod = dur
	}
	return nil
}

func (gn *GlobalNode) set(key string, value float64) error {
	const pfx = "smart_contracts.interestpoolsc."
	switch key {
	case Settings[MinLock]:
		gn.MinLock = state.Balance(value)
		config.SmartContractConfig.Set(pfx+key, gn.MinLock)
	case Settings[Apr]:
		gn.APR = value
		config.SmartContractConfig.Set(pfx+key, gn.APR)
	case Settings[MinLockPeriod]:
		gn.MinLockPeriod = time.Duration(value)
		config.SmartContractConfig.Set(pfx+key, gn.MinLockPeriod)
	case Settings[MaxMint]:
		gn.MaxMint = state.Balance(value)
		config.SmartContractConfig.Set(pfx+key, gn.MaxMint)
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
