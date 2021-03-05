package interestpoolsc

import (
	"encoding/json"
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
	return &GlobalNode{ID: ADDRESS, SimpleGlobalNode: &SimpleGlobalNode{}}
}

func (gn *GlobalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
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
