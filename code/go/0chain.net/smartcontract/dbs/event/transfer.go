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
	Reward       currency.Coin   `json:"reward"`
	Total        int64           `json:"total"`
}

type ReadPoolLock struct {
	Client string `json:"client"`
	PoolId string `json:"pool_id"`
	Amount int64  `json:"amount"`
	IsMint bool   `json:"mint"`
}

type WritePoolLock struct {
	Client       string `json:"client"`
	AllocationId string `json:"allocation_id"`
	Amount       int64  `json:"amount"`
	IsMint       bool   `json:"mint"`
}

type ChallengePoolLock struct {
	Client       string `json:"client"`
	FromPoolId   string `json:"from_pool_id"`
	ToPoolId     string `json:"to_pool_id"`
	AllocationId string `json:"allocation_id"`
	Amount       int64  `json:"amount"`
}

type BridgeMint struct {
	UserID    string        `json:"user_id"`
	MintNonce int64         `json:"mint_nonce"`
	Amount    currency.Coin `json:"amount"`
	Signers   []string      `json:"signers"`
}

func mergeToChallengePoolsEvents() *eventsMergerImpl[ChallengePoolLock] {
	return newEventsMerger[ChallengePoolLock](TagToChallengePool, withMergeChallengePoolLockEvents())
}

func mergeFromChallengePoolsEvents() *eventsMergerImpl[ChallengePoolLock] {
	return newEventsMerger[ChallengePoolLock](TagFromChallengePool, withMergeChallengePoolLockEvents())
}

func withMergeChallengePoolLockEvents() eventMergeMiddleware {
	return withEventMerge(func(a, b *ChallengePoolLock) (*ChallengePoolLock, error) {
		a.Amount += a.Amount
		return a, nil
	})
}
