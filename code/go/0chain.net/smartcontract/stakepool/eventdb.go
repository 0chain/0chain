package stakepool

import (
	"encoding/json"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs/event"
)

type DbUpdates struct {
	Id      string                 `json:"id"`
	Updates map[string]interface{} `json:"updates"`
}

type StakePoolId struct {
	ProviderId   string `json:"provider_id"`
	ProviderType int    `json:"provider_type"`
}

type DelegatePoolId struct {
	StakePoolId
	PoolId string `json:"pool_id"`
}

func (dp *DelegatePoolId) emit(tag event.EventTag, balances cstate.StateContextI) error {
	data, err := json.Marshal(dp)
	if err != nil {
		return err
	}
	balances.EmitEvent(event.TypeStats, tag, dp.PoolId, string(data))
	return nil
}

type SpReward struct {
	StakePoolId
	SpReward       int64            `json:"sp_reward"`
	DelegateReward map[string]int64 `json:"delegate_reward"`
}

func (spr *SpReward) emit(balances cstate.StateContextI) error {
	edbData, err := json.Marshal(spr)
	if err != nil {
		return err
	}
	balances.EmitEvent(event.TypeStats, event.TagStakePoolReward, spr.ProviderId, string(edbData))
	return nil
}

func emitDeleteSp(
	providerId string,
	providerType int,
) error {
	return nil
}

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

type EdbDelegatePool struct {
	PoolId       string `json:"pool_id"`
	DelegateId   string `json:"delegate_id"`
	ProviderId   string `json:"provider_id"`
	ProviderType int    `json:"provider_type"`

	Reward       int64 `json:"reward"`
	TotalReward  int64 `json:"total_reward"`
	TotalPenalty int64 `json:"total_penalty"`
	Status       int   `json:"status"`
	RoundCreated int64 `json:"round_created"`
}

func (dp DelegatePool) updates(poolId string, providerType Provider) DbUpdates {
	return DbUpdates{
		Id: poolId,
		Updates: map[string]interface{}{
			"pool_id":       poolId,
			"provider_type": providerType,
			"status":        dp.Status,
		},
	}
}

func (sp StakePool) updates(id string, providerType int) (DbUpdates, []DbUpdates) {
	spUpdates := DbUpdates{
		Id: id,
		Updates: map[string]interface{}{
			"provider_id":    id,
			"provider_type":  providerType,
			"balance":        sp.stake(),
			"min_stake":      sp.Settings.MinStake,
			"max_stake":      sp.Settings.MaxStake,
			"num_delegates":  sp.Settings.MaxNumDelegates,
			"service_charge": sp.Settings.ServiceCharge,
		},
	}
	var dpUpdates []DbUpdates

	return spUpdates, dpUpdates
}

func (sp StakePool) emit(id string, providerType int, balances cstate.StateContextI) {

}
