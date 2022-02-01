package stakepool

import (
	"encoding/json"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
)

type payRewardRequest struct {
	ProviderType Provider
	PoolId       string
}

func (spr *payRewardRequest) decode(p []byte) error {
	return json.Unmarshal(p, spr)
}

func PayoutReward(
	client string,
	input []byte,
	balances cstate.StateContextI,
) (state.Balance, error) {
	var prr payRewardRequest
	if err := prr.decode(input); err != nil {
		return 0, common.NewErrorf("pay_reward_failed",
			"can't decode request: %v", err)
	}

	var usp *UserStakePools
	usp, err := GetUserStakePool(prr.ProviderType, client, balances)
	if err != nil {
		return 0, common.NewErrorf("stake_pool_unlock_failed",
			"can't get related user stake pools: %v", err)
	}

	providerId := usp.find(prr.PoolId)
	if len(providerId) == 0 {
		return 0, common.NewErrorf("stake_pool_unlock_failed",
			"user %v does not own stake pool %v", client, prr.PoolId)
	}

	sp, err := GetStakePool(prr.ProviderType, providerId, balances)
	if err != nil {
		return 0, common.NewErrorf("pay_reward_failed",
			"can't get related stake pool: %v", err)
	}

	total, removed, err := sp.EmptyAccount(
		client, prr.PoolId, providerId, prr.ProviderType, balances)
	if err != nil {
		return 0, common.NewErrorf("stake_pool_unlock_failed",
			"error emptying account, %v", err)
	}
	if removed {
		usp.del(providerId, prr.PoolId)
	}

	return total, nil
}
