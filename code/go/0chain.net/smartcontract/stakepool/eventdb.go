package stakepool

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
)

type DbUpdates struct {
	Id      string                 `json:"id"`
	Updates map[string]interface{} `json:"updates"`
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
