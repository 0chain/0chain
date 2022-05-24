package interestpoolsc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"0chain.net/chaincore/state"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -v

// swagger:model InterestPoolGlobalNode
type GlobalNode struct {
	*SimpleGlobalNode `json:"simple_global_node"`
	ID                string
	MinLockPeriod     time.Duration `json:"min_lock_period"`
}

// swagger:model InterestPoolGlobalNode
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

func (gn *GlobalNode) set(key string, value string) error {
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

		gn.MinLockPeriod = mlp
	case Settings[MaxMint]:
		fValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("cannot conver key %s, value %s into state.balane; %v", key, value, err)
		}
		gn.MaxMint = state.Balance(fValue * 1e10)
	case Settings[OwnerId]:
		if _, err := hex.DecodeString(value); err != nil {
			return fmt.Errorf("%s must be a hes string: %v", key, err)
		}
		gn.OwnerId = value
	default:
		return gn.setCostValue(key, value)
	}
	return nil
}

func (gn *GlobalNode) setCostValue(key, value string) error {
	if !strings.HasPrefix(key, Settings[Cost]) {
		return fmt.Errorf("config setting %q not found", key)
	}

	costKey := strings.ToLower(strings.TrimPrefix(key, Settings[Cost]+"."))
	for _, costFunction := range costFunctions {
		if costKey != strings.ToLower(costFunction) {
			continue
		}
		costValue, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("key %s, unable to convert %v to integer", key, value)
		}

		if costValue < 0 {
			return fmt.Errorf("cost.%s contains invalid value %s", key, value)
		}

		gn.Cost[costKey] = costValue

		return nil
	}

	return fmt.Errorf("cost config setting %s not found", costKey)
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
