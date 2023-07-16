package dto

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/provider"
)

type ValidationDtoNode struct {
	provider.Provider
	BaseURL           *string           `json:"url"`
	StakePoolSettings *Settings         `json:"stake_pool_settings"`
	LastHealthCheck   *common.Timestamp `json:"last_health_check"`
}
