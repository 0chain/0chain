package dto

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/provider"
	"github.com/0chain/common/core/currency"
)

// StorageDtoNode represents Blobber configurations used as DTO.
// This is just a DTO model which should be used in passing data to other service for ex, via HTTP call.
// This should not be used in the application for any business logic.
// The corresponding model is storagesc.StorageNode.
type StorageDtoNode struct {
	provider.Provider
	BaseURL                 *string      `json:"url,omitempty"`
	Terms                   *Terms       `json:"terms,omitempty"`
	Capacity                *int64       `json:"capacity,omitempty"`
	Allocated               *int64       `json:"allocated,omitempty"`
	SavedData               *int64       `json:"saved_data,omitempty"`
	DataReadLastRewardRound *float64     `json:"data_read_last_reward_round,omitempty"`
	LastRewardDataReadRound *int64       `json:"last_reward_data_read_round,omitempty"`
	StakePoolSettings       *Settings    `json:"stake_pool_settings,omitempty"`
	RewardRound             *RewardRound `json:"reward_round,omitempty"`
	NotAvailable            *bool        `json:"not_available,omitempty"`
}

type RewardRound struct {
	StartRound *int64            `json:"start_round,omitempty"`
	Timestamp  *common.Timestamp `json:"timestamp,omitempty"`
}

type Terms struct {
	ReadPrice  *currency.Coin `json:"read_price,omitempty"`
	WritePrice *currency.Coin `json:"write_price,omitempty"`
}
