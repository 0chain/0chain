package stakepool

import (
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

type StakePoolReward dbs.StakePoolReward

func NewStakePoolReward(pId string, pType spenum.Provider) *StakePoolReward {
	var spu StakePoolReward
	spu.ProviderId = pId
	spu.ProviderType = int(pType)
	spu.DelegateRewards = make(map[string]int64)
	spu.DelegatePenalties = make(map[string]int64)
	return &spu
}

func (spu StakePoolReward) Emit(
	tag event.EventTag,
	desc string,
	balances cstate.StateContextI,
) error {

	balances.EmitEvent(
		event.TypeStats,
		tag,
		spu.ProviderId,
		stakePoolRewardToStakePoolRewardEvent(spu, desc),
	)
	return nil
}

func stakePoolRewardToStakePoolRewardEvent(spu StakePoolReward, desc string) *dbs.StakePoolReward {
	return &dbs.StakePoolReward{
		StakePoolId:     spu.StakePoolId,
		Reward:          spu.Reward,
		DelegateRewards: spu.DelegateRewards,
		Desc:            []string{desc},
	}
}
