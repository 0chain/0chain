package stakepool

import (
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

type StakePoolReward dbs.StakePoolReward

func (sp *StakePool) EmitStakePoolBalanceUpdate(
	pId string,
	pType spenum.Provider,
	balances cstate.StateContextI,
) {
	for id, dp := range sp.Pools {
		dpu := dbs.NewDelegatePoolUpdate(id, pId, pType)
		dpu.Updates["balance"] = dp.Balance
		balances.EmitEvent(event.TypeStats, event.TagUpdateDelegatePool, id, *dpu)
	}
}

func NewStakePoolReward(pId string, pType spenum.Provider) *StakePoolReward {
	var spu StakePoolReward
	spu.ProviderId = pId
	spu.ProviderType = pType
	spu.DelegateRewards = make(map[string]int64)
	return &spu
}

func (spu StakePoolReward) Emit(
	tag event.EventTag,
	balances cstate.StateContextI,
) error {
	balances.EmitEvent(
		event.TypeStats,
		tag,
		spu.ProviderId,
		stakePoolRewardToStakePoolRewardEvent(spu),
	)
	return nil
}

func stakePoolRewardToStakePoolRewardEvent(spu StakePoolReward) *dbs.StakePoolReward {
	return &dbs.StakePoolReward{
		Provider:        spu.Provider,
		Reward:          spu.Reward,
		DelegateRewards: spu.DelegateRewards,
	}
}
