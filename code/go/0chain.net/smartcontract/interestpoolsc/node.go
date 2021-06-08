package interestpoolsc

import (
	"encoding/json"
	"fmt"
	"time"

	"0chain.net/core/common"
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

func (gn *GlobalNode) validate() error {
	switch {
	case gn.APR < 0:
		return common.NewError("failed to validate global node", fmt.Sprintf("apr(%v) is too low", gn.APR))
	case gn.MinLockPeriod < 0:
		return common.NewError("failed to validate global node", fmt.Sprintf("min lock period(%v) is too short", gn.MinLockPeriod))
	case gn.MinLock < 0:
		return common.NewError("failed to validate global node", fmt.Sprintf("min lock(%v) is too low", gn.MinLock))
	case gn.MaxMint < 0:
		return common.NewError("failed to validate global node", fmt.Sprintf("max mint(%v) is too low", gn.MaxMint))
	}
	return nil
}
