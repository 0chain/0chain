package dbs

import (
	"time"

	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
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


type DbUpdateProvider struct {
	DbUpdates
	Type spenum.Provider `json:"type"`
}

func NewDbUpdateProvider(id string, typ spenum.Provider) *DbUpdateProvider {
	return &DbUpdateProvider{
		DbUpdates: DbUpdates{
			Id:      id,
			Updates: make(map[string]interface{}),
		},
		Type: typ,
	}

}

type Provider struct {
	ProviderId   string          `json:"provider_id"`
	ProviderType spenum.Provider `json:"provider_type"`
}
type StakePoolReward struct {
	Provider
	Reward     currency.Coin `json:"reward"`
	RewardType spenum.Reward `json:"reward_type"`
	// rewards delegate pools
	DelegateRewards map[string]currency.Coin `json:"delegate_rewards"`
	// penalties delegate pools
	DelegatePenalties map[string]currency.Coin `json:"delegate_penalties"`
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