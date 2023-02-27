package stakepool

import (
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/dbs"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

type StakePoolReward dbs.StakePoolReward

func NewStakePoolReward(pId string, pType spenum.Provider, rewardType spenum.Reward, optional ...string) *StakePoolReward {

	if len(optional) == 0 {
		optional = append(optional, "")
	}

	var spu StakePoolReward
	spu.ProviderId = pId
	spu.ProviderType = pType
	spu.DelegateRewards = make(map[string]currency.Coin)
	spu.DelegatePenalties = make(map[string]currency.Coin)
	spu.RewardType = rewardType
	spu.ChallengeID = optional[0]
	return &spu
}

func (spu StakePoolReward) Emit(
	tag event.EventTag,
	balances cstate.StateContextI,
) error {

	balances.EmitEvent(
		event.TypeStats,
		tag,
		spu.RewardType.String()+spu.ProviderId,
		stakePoolRewardToStakePoolRewardEvent(spu),
	)
	return nil
}

func stakePoolRewardToStakePoolRewardEvent(spu StakePoolReward) *dbs.StakePoolReward {
	return &dbs.StakePoolReward{
		StakePoolId:     spu.StakePoolId,
		Reward:          spu.Reward,
		DelegateRewards: spu.DelegateRewards,
		RewardType:      spu.RewardType,
		ChallengeID:     spu.ChallengeID,
	}
}
