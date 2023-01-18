package event

import (
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
)

type DelegatePoolLock struct {
	Client       string          `json:"client"`
	ProviderId   string          `json:"provider_id"`
	ProviderType spenum.Provider `json:"provider_type"`
	Amount       int64           `json:"amount"`
	Reward		 currency.Coin   `json:"reward"`
}

type ReadPoolLock struct {
	Client string `json:"client"`
	PoolId string `json:"pool_id"`
	Amount int64  `json:"amount"`
}

type WritePoolLock struct {
	Client       string `json:"client"`
	AllocationId string `json:"allocation_id"`
	Amount       int64  `json:"amount"`
}

type ChallengePoolLock struct {
	Client       string `json:"client"`
	FromPoolId   string `json:"from_pool_id"`
	ToPoolId     string `json:"to_pool_id"`
	AllocationId string `json:"allocation_id"`
	Amount       int64  `json:"amount"`
}
