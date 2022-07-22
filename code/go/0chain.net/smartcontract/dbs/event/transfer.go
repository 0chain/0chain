package event

import (
	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/stakepool/spenum"
)

type DelegatePoolLock struct {
	Client       string          `json:"client"`
	PoolId       string          `json:"pool_id"`
	ProviderId   string          `json:"provider_id"`
	ProviderType spenum.Provider `json:"provider_type"`
	Amount       currency.Coin   `json:"amount"`
}
