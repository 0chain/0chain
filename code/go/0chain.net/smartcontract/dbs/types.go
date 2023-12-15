package dbs

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

type DbHealthCheck struct {
	ID              string           `json:"id"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	Downtime        uint64           `json:"downtime"`
}

type DbUpdates struct {
	Id      string                 `json:"id"`
	Updates map[string]interface{} `json:"updates"`
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

type ProviderID struct {
	ID   string          `json:"provider_id"`
	Type spenum.Provider `json:"provider_type"`
}

func (p *ProviderID) GetID() string {
	return p.ID
}

type StakePoolReward struct {
	ProviderID
	Reward     currency.Coin `json:"reward"`
	RewardType spenum.Reward `json:"reward_type"`
	// rewards delegate pools
	DelegateRewards map[string]currency.Coin `json:"delegate_rewards"`
	// penalties delegate pools
	DelegatePenalties map[string]currency.Coin `json:"delegate_penalties"`
	// allocation id
	AllocationID string `json:"allocation_id"`

	DelegateWallet string `json:"delegate_wallet"`
}

func (sp *StakePoolReward) TotalReward() currency.Coin {
	totalReward := sp.Reward
	for _, reward := range sp.DelegateRewards {
		totalReward += reward
	}

	return totalReward
}

func (sp *StakePoolReward) TotalDelegateReward() currency.Coin {
	logging.Logger.Info("TotalDelegateReward", zap.Any("sp", sp))
	totalReward := sp.Reward
	for id, reward := range sp.DelegateRewards {
		if id == sp.DelegateWallet {
			logging.Logger.Info("TotalDelegateReward", zap.Any("id", id), zap.Any("reward", reward))
			totalReward += reward
			break
		}
	}

	return totalReward
}

func (sp *StakePoolReward) TotalPenalty() currency.Coin {
	totalPenalty := currency.Coin(0)
	for _, penalty := range sp.DelegatePenalties {
		totalPenalty += penalty
	}

	return totalPenalty
}

func (sp *StakePoolReward) TotalDelegatePenalty() currency.Coin {
	totalPenalty := currency.Coin(0)
	for id, penalty := range sp.DelegatePenalties {
		if id == sp.DelegateWallet {
			totalPenalty += penalty
			break
		}
	}

	return totalPenalty
}

type DelegatePoolId struct {
	ProviderID
	PoolId string `json:"pool_id"`
}

type DelegatePoolUpdate struct {
	DelegatePoolId
	Updates map[string]interface{} `json:"updates"`
}

func NewDelegatePoolUpdate(pool, provider string, pType spenum.Provider) *DelegatePoolUpdate {
	var dpu DelegatePoolUpdate
	dpu.PoolId = pool
	dpu.ID = provider
	dpu.Type = pType
	dpu.Updates = make(map[string]interface{})
	return &dpu
}

type SpBalance struct {
	ProviderID
	Balance         int64            `json:"sp_reward"`
	DelegateBalance map[string]int64 `json:"delegate_reward"`
}

type SpReward struct {
	ProviderID
	SpReward       int64            `json:"sp_reward"`
	DelegateReward map[string]int64 `json:"delegate_reward"`
}

type ChallengeResult struct {
	BlobberId string `json:"blobberId"`
	Passed    bool   `json:"passed"`
}
