package interestpoolsc

import (
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"encoding/json"
)

type Setting int

const (
	MinLock Setting = iota
	Apr
	MinLockPeriod
	MaxMint
)

var (
	Settings = []string{
		"min_lock",
		"apr",
		"min_lock_period",
		"max_mint",
	}
)

type inputMap struct {
	Fields map[string]interface{} `json:"fields"`
}

func (im *inputMap) Decode(input []byte) error {
	err := json.Unmarshal(input, im)
	if err != nil {
		return err
	}
	return nil
}

func (im *inputMap) Encode() []byte {
	buff, _ := json.Marshal(im)
	return buff
}

func (ip *InterestPoolSmartContract) updateVariables(t *transaction.Transaction, gn *GlobalNode, inputData []byte, balances c_state.StateContextI) (string, error) {
	if t.ClientID != owner {
		return "", common.NewError("failed to update variables", "unauthorized access - only the owner can update the variables")
	}

	changes := &inputMap{}
	if err := changes.Decode(inputData); err != nil {
		return "", common.NewError("failed to update variables", "request not formatted correctly")
	}

	for key, value := range changes.Fields {
		if fValue, ok := value.(float64); !ok {
			return "", common.NewErrorf("failed to update variables", "new value %v not numeric", value)
		} else {
			if err := gn.set(key, fValue); err != nil {
				return "", common.NewError("failed to update variables", err.Error())
			}
		}

	}

	balances.InsertTrieNode(gn.getKey(), gn)
	return string(gn.Encode()), nil
}
