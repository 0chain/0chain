package dbs

import "0chain.net/chaincore/currency"

type DbUpdates struct {
	Id      string                 `json:"id"`
	Updates map[string]interface{} `json:"updates"`
}

func NewDbUpdates(id string) *DbUpdates {
	return &DbUpdates{
		Id:      id,
		Updates: make(map[string]interface{}),
	}
}

type StakePoolId struct {
	ProviderId   string `json:"provider_id"`
	ProviderType int    `json:"provider_type"`
}

type StakePoolReward struct {
	StakePoolId
	Reward          currency.Coin    `json:"reward"`
	DelegateRewards map[string]int64 `json:"delegate_rewards"`
}

type StakePoolUpdate struct {
	StakePoolId
	Updates         map[string]interface{} `json:"updates"`
	DelegateUpdates map[string]map[string]interface{}
}

type DelegatePoolId struct {
	StakePoolId
	PoolId string `json:"pool_id"`
}

type DelegatePoolUpdate struct {
	DelegatePoolId
	Updates map[string]interface{} `json:"updates"`
}

func NewDelegatePoolUpdate(pool, provider string, pType int) *DelegatePoolUpdate {
	var dpu DelegatePoolUpdate
	dpu.PoolId = pool
	dpu.ProviderId = provider
	dpu.ProviderType = pType
	dpu.Updates = make(map[string]interface{})
	return &dpu
}

type SpBalance struct {
	StakePoolId
	Balance         int64            `json:"sp_reward"`
	DelegateBalance map[string]int64 `json:"delegate_reward"`
}

type SpReward struct {
	StakePoolId
	SpReward       int64            `json:"sp_reward"`
	DelegateReward map[string]int64 `json:"delegate_reward"`
}

type ChallengeResult struct {
	BlobberId string `json:"blobberId"`
	Passed    bool   `json:"passed"`
}
