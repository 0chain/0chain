package stakepool

import (
	"encoding/json"

	"0chain.net/smartcontract/dbs"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

type StakePoolReward dbs.StakePoolReward

func NewStakePoolReward(pId string, pType Provider) *StakePoolReward {
	var spu StakePoolReward
	spu.ProviderId = pId
	spu.ProviderType = int(pType)
	spu.DelegateRewards = make(map[string]int64)
	return &spu
}

/*
type EdbStakePool struct {
	ProviderId   string `json:"provider_id"`
	ProviderType int    `json:"provider_type"`
	Balance      int64  `json:"balance"`
	Unstake      int64  `json:"unstake"`

	TotalOffers  int64 `json:"total_offers"`
	TotalUnStake int64 `json:"total_un_stake"`

	Reward int64 `json:"reward"`

	TotalRewards int64 `json:"total_rewards"`
	TotalPenalty int64 `json:"total_penalty"`

	MinStake        state.Balance `json:"min_stake"`
	MaxStake        state.Balance `json:"max_stake"`
	MaxNumDelegates int           `json:"num_delegates"`
	ServiceCharge   float64       `json:"service_charge"`
}

func (sp StakePool) EmitNew(
	providerId string,
	providerType Provider,
	balances cstate.StateContextI,
) error {
	data, err := json.Marshal(&EdbStakePool{
		ProviderId:      providerId,
		ProviderType:    int(providerType),
		MinStake:        sp.Settings.MinStake,
		MaxStake:        sp.Settings.MaxStake,
		MaxNumDelegates: sp.Settings.MaxNumDelegates,
		ServiceCharge:   sp.Settings.ServiceCharge,
	})
	if err != nil {
		return err
	}
	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteStakePool, providerId, string(data))
	return nil
}
*/
func (spu StakePoolReward) Emit(
	tag event.EventTag,
	balances cstate.StateContextI,
) error {
	data, err := json.Marshal(&spu)
	if err != nil {
		return err
	}
	balances.EmitEvent(
		event.TypeStats,
		tag,
		spu.ProviderId,
		string(data),
	)
	return nil
}
