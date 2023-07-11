package dto

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/provider"
	"github.com/0chain/common/core/currency"
)

// AuthorizerDtoNode used in `UpdateAuthorizerConfig` functions
type AuthorizerDtoNode struct {
	provider.Provider
	URL             *string              `json:"url"`
	Config          *AuthorizerDtoConfig `json:"config"`
	LastHealthCheck common.Timestamp     `json:"last_health_check"`
}

type AuthorizerDtoConfig struct {
	Fee *currency.Coin `json:"fee"`
}
