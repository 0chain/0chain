package stakepool

import (
	"encoding/json"
	"fmt"

	"github.com/0chain/common/core/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/stakepool/spenum"
)

// CollectRewardRequest uniquely defines a stake pool.
type CollectRewardRequest struct {
	ProviderId   string          `json:"provider_id"`
	ProviderType spenum.Provider `json:"provider_type"`
}

func (spr *CollectRewardRequest) Decode(p []byte) error {
	return json.Unmarshal(p, spr)
}

func (spr *CollectRewardRequest) Encode() []byte {
	bytes, _ := json.Marshal(spr)
	return bytes
}

func CollectReward(
	input []byte,
	mintTokens func(CollectRewardRequest, cstate.StateContextI) (currency.Coin, error),
	balances cstate.StateContextI,
) (currency.Coin, error) {
	var crr CollectRewardRequest
	if err := crr.Decode(input); err != nil {
		return 0, fmt.Errorf("can't decode request: %v", err)
	}

	if crr.ProviderId == "" {
		return 0, fmt.Errorf("no provider")
	}

	reward, err := mintTokens(crr, balances)
	if err != nil {
		return 0, err
	}

	return reward, nil
}
