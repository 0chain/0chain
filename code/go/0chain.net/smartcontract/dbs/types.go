package dbs

import (
	"time"

	"0chain.net/core/common"

	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/stakepool/spenum"
)

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

type Provider struct {
	ProviderId   string          `json:"provider_id"`
	ProviderType spenum.Provider `json:"provider_type"`
}

type StakePoolReward struct {
	Provider
	Reward          currency.Coin    `json:"reward"`
	DelegateRewards map[string]int64 `json:"delegate_rewards"`
}

type StakePoolUpdate struct {
	Provider
	Updates         map[string]interface{} `json:"updates"`
	DelegateUpdates map[string]map[string]interface{}
}

type DelegatePoolId struct {
	Provider
	PoolId string `json:"pool_id"`
}

type DelegatePoolUpdate struct {
	DelegatePoolId
	Updates map[string]interface{} `json:"updates"`
}

func NewDelegatePoolUpdate(pool, provider string, pType spenum.Provider) *DelegatePoolUpdate {
	var dpu DelegatePoolUpdate
	dpu.PoolId = pool
	dpu.ProviderId = provider
	dpu.ProviderType = pType
	dpu.Updates = make(map[string]interface{})
	return &dpu
}

type SpBalance struct {
	Provider
	Balance         int64            `json:"sp_reward"`
	DelegateBalance map[string]int64 `json:"delegate_reward"`
}

type SpReward struct {
	Provider
	SpReward       int64            `json:"sp_reward"`
	DelegateReward map[string]int64 `json:"delegate_reward"`
}

type ChallengeResult struct {
	BlobberId string `json:"blobberId"`
	Passed    bool   `json:"passed"`
}

type HealthCheck struct {
	Provider
	LastHealthCheck   common.Timestamp `json:"last_heath_check"`
	Now               common.Timestamp `json:"now"`
	HealthCheckPeriod time.Duration    `json:"healch_check_period"`
}
