package dto

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"github.com/0chain/common/core/currency"
)

//go:generate msgp -v -io=false -tests=false

type PoolStatus int

// StakePool holds delegate information for an 0chain providers
type StakePool struct {
	Pools             map[string]*DelegatePool `json:"pools"`
	Reward            currency.Coin            `json:"rewards"`
	StakePoolSettings Settings                 `json:"settings"`
	Minter            cstate.ApprovedMinter    `json:"minter"`
	HasBeenKilled     bool                     `json:"is_dead"`
}

type DelegatePool struct {
	Balance      currency.Coin    `json:"balance"`
	Reward       currency.Coin    `json:"reward"`
	Status       *PoolStatus      `json:"status"`
	RoundCreated *int64           `json:"round_created"` // used for cool down
	DelegateID   *string          `json:"delegate_id"`
	StakedAt     common.Timestamp `json:"staked_at"`
}

type Settings struct {
	DelegateWallet     *string  `json:"delegate_wallet,omitempty"`
	MaxNumDelegates    *int     `json:"num_delegates,omitempty"`
	ServiceChargeRatio *float64 `json:"service_charge,omitempty"`
}

func NewStakePool() *StakePool {
	return &StakePool{
		Pools: make(map[string]*DelegatePool),
	}
}
