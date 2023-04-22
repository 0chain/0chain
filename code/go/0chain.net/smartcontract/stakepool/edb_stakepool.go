package stakepool

import (
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

type StakePoolReward dbs.StakePoolReward

func (sp *StakePool) EmitStakePoolBalanceUpdate(
	pId string,
	pType spenum.Provider,
	balances cstate.StateContextI,
) {
	orderedPoolIds := sp.OrderedPoolIds()
	for _, id := range orderedPoolIds {
		dp := sp.Pools[id]
		dpu := dbs.NewDelegatePoolUpdate(id, pId, pType)
		dpu.Updates["balance"] = dp.Balance
		balances.EmitEvent(event.TypeStats, event.TagUpdateDelegatePool, id, *dpu)
	}
}

func NewStakePoolReward(pId string, pType spenum.Provider, rewardType spenum.Reward, options ...string) *StakePoolReward {
	var spu StakePoolReward
	spu.ID = pId
	spu.Type = pType
	spu.DelegateRewards = make(map[string]currency.Coin)
	spu.DelegatePenalties = make(map[string]currency.Coin)
	spu.RewardType = rewardType

	var challengeID string
	if len(options) > 0 {
		challengeID = options[0]
	} else {
		challengeID = ""
	}
	spu.ChallengeID = challengeID

	return &spu
}

func (spu StakePoolReward) Emit(
	tag event.EventTag,
	balances cstate.StateContextI,
) error {

	balances.EmitEvent(
		event.TypeStats,
		tag,
		spu.RewardType.String()+spu.ID,
		stakePoolRewardToStakePoolRewardEvent(spu),
	)
	return nil
}

func stakePoolRewardToStakePoolRewardEvent(spu StakePoolReward) *dbs.StakePoolReward {
	return &dbs.StakePoolReward{
		ProviderID:      spu.ProviderID,
		Reward:          spu.Reward,
		DelegateRewards: spu.DelegateRewards,
		RewardType:      spu.RewardType,
		ChallengeID:     spu.ChallengeID,
	}
}
