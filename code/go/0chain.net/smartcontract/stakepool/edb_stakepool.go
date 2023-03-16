package stakepool

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

type StakePoolReward dbs.StakePoolReward

func NewStakePoolReward(pId string, pType spenum.Provider, rewardType spenum.Reward, options ...string) *StakePoolReward {

	logging.Logger.Debug("jayashNewStakePoolReward", zap.String("pId", pId), zap.Any("pType", pType), zap.Any("rewardType", rewardType), zap.Any("options", options))

	var spu StakePoolReward
	spu.ProviderId = pId
	spu.ProviderType = pType
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

	logging.Logger.Debug("jayashNewStakePoolReward", zap.Any("spu", spu))

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
