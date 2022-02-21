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
	ProviderId   string   `json:"provider_id"`
	ProviderType Provider `json:"provider_type"`
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
	data, err := json.Marshal(spr)
	if err != nil {
		return err
	}
	balances.EmitEvent(event.TypeStats, event.TagStakePoolReward, spr.ProviderId, string(data))
	return nil
}

type SpBalance struct {
	StakePoolId
	Balance         int64            `json:"sp_reward"`
	DelegateBalance map[string]int64 `json:"delegate_reward"`
}

func (spr *SpBalance) emit(balances cstate.StateContextI) error {
	data, err := json.Marshal(spr)
	if err != nil {
		return err
	}
	balances.EmitEvent(event.TypeStats, event.TagStakePoolBalance, spr.ProviderId, string(data))
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

func (dp DelegatePool) emitNew(
	clientId, poolId, providerId string,
	providerType Provider,
	status PoolStatus,
	balances cstate.StateContextI,
) error {
	data, err := json.Marshal(&EdbDelegatePool{
		PoolId:       poolId,
		DelegateId:   clientId,
		ProviderId:   providerId,
		ProviderType: int(providerType),
		Status:       int(status),
		RoundCreated: balances.GetBlock().Round,
	})
	if err != nil {
		return err
	}
	balances.EmitEvent(
		event.TypeStats,
		event.TagAddOrOverwriteDelegatePool,
		providerId,
		string(data),
	)
	return nil
}

func (dp DelegatePool) Updates(poolId string, providerType Provider) DbUpdates {
	return DbUpdates{
		Id: poolId,
		Updates: map[string]interface{}{
			"pool_id":       poolId,
			"provider_type": providerType,
			"status":        dp.Status,
		},
	}
}

func (sp *StakePool) EmitUpdate(id string, providerType Provider, balances cstate.StateContextI) error {
	data, err := json.Marshal(sp.updates(id, providerType))
	if err != nil {
		return err
	}
	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteStakePool, id, string(data))
	return nil
}

func (sp StakePool) updates(id string, providerType Provider) *DbUpdates {
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
	return &spUpdates
}

func (sp StakePool) emit(id string, providerType int, balances cstate.StateContextI) {

}
