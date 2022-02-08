package stakepool

import (
	"encoding/json"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
)

type PayRewardRequest struct {
	ProviderType Provider `json:"provider_type"`
	PoolId       string   `json:"pool_id"`
}

func (spr *PayRewardRequest) Decode(p []byte) error {
	return json.Unmarshal(p, spr)
}

func PayoutReward(
	client string,
	input []byte,
	balances cstate.StateContextI,
) (state.Balance, error) {
	var prr PayRewardRequest
	if err := prr.Decode(input); err != nil {
		return 0, common.NewErrorf("pay_reward_failed",
			"can't decode request: %v", err)
	}

	var usp *UserStakePools
	usp, err := GetUserStakePool(prr.ProviderType, client, balances)
	if err != nil {
		return 0, common.NewErrorf("pay_reward_failed",
			"can't get related user stake pools: %v", err)
	}

	providerId := usp.Find(prr.PoolId)
	if len(providerId) == 0 {
		return 0, common.NewErrorf("pay_reward_failed",
			"user %v does not own stake pool %v", client, prr.PoolId)
	}

	sp, err := GetStakePool(prr.ProviderType, providerId, balances)
	if err != nil {
		return 0, common.NewErrorf("pay_reward_failed",
			"can't get related stake pool: %v", err)
	}

	total, err := sp.MintRewards(
		client, prr.PoolId, providerId, prr.ProviderType, usp, balances)
	if err != nil {
		return 0, common.NewErrorf("pay_reward_failed",
			"error emptying account, %v", err)
	}

	return total, nil
}
