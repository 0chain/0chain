package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"0chain.net/chaincore/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool"
)

//msgp:ignore lockRequest unlockRequest
//go:generate msgp -io=false -tests=false -unexported=true -v

//
// SC / API requests
//

// lock request

// request to lock tokens creating a read pool;
// the allocation_id is required, if blobber_id provided, then
// it locks tokens for allocation -> {blobber}, otherwise
// all tokens divided for all blobbers of the allocation
// automatically
type lockRequest struct {
	Duration     time.Duration `json:"duration"`
	AllocationID string        `json:"allocation_id"`
}

func (lr *lockRequest) decode(input []byte) (err error) {
	if err = json.Unmarshal(input, lr); err != nil {
		return
	}
	return // ok
}

// unlock request used to unlock all tokens of a read pool
type unlockRequest struct {
	AllocationID string `json:"allocation_id"`
}

func (ur *unlockRequest) decode(input []byte) error {
	return json.Unmarshal(input, ur)
}

//
// allocation read/write pool
//

// allocation read/write pool represents tokens locked for an allocation;
type allocationPool struct {
	Balance  currency.Coin    `json:"balance"`
	ExpireAt common.Timestamp `json:"expire_at"` // inclusive
}

func newAllocationPool(
	txn *transaction.Transaction,
	until common.Timestamp,
	mintNewTokens bool,
	balances cstate.StateContextI,
) (*allocationPool, error) {
	var err error
	if !mintNewTokens {
		if err = stakepool.CheckClientBalance(txn, balances); err != nil {
			return nil, err
		}
	}

	var ap allocationPool

	if mintNewTokens {
		if err := balances.AddMint(&state.Mint{
			Minter:     ADDRESS,
			ToClientID: ADDRESS,
			Amount:     currency.Coin(txn.Value),
		}); err != nil {
			return nil, fmt.Errorf("minting tokens for write pool: %v", err)
		}
	} else {
		transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, currency.Coin(txn.Value))
		if err = balances.AddTransfer(transfer); err != nil {
			return nil, fmt.Errorf("adding transfer to allocation pool: %v", err)
		}
	}
	ap.Balance = currency.Coin(txn.Value)
	ap.ExpireAt = until
	return &ap, nil
}

func isInTOMRList(torm []*allocationPool, ax *allocationPool) bool {
	for _, tr := range torm {
		if tr == ax {
			return true
		}
	}
	return false
}

func sortExpireAt(cut []*allocationPool) {
	sort.Slice(cut, func(i, j int) bool {
		return cut[i].ExpireAt < cut[j].ExpireAt
	})
}

//
// stat
//

// allocation read/write pool represents tokens locked for an allocation;
type allocationPoolStat struct {
	ID           string           `json:"id"`
	Balance      currency.Coin    `json:"balance"`
	ExpireAt     common.Timestamp `json:"expire_at"`
	AllocationID string           `json:"allocation_id"`
	Locked       bool             `json:"locked"`
}

func (ap *allocationPool) stat(now common.Timestamp) (stat allocationPoolStat) {
	stat.Balance = ap.Balance
	stat.ExpireAt = ap.ExpireAt
	stat.Locked = ap.ExpireAt >= now

	return
}

func (ap *allocationPool) moveToAllocationPool(
	cp *challengePool,
	value currency.Coin,
) error {
	if value == 0 {
		return nil
	}

	if cp == nil {
		return errors.New("invalid challenge pool")
	}

	if cp.Balance < value {
		return fmt.Errorf("not enough tokens in challenge pool %s: %d < %d",
			cp.ID, cp.Balance, value)
	}
	cp.Balance -= value
	ap.Balance += value
	return nil
}

// swagger:model allocationPoolsStat
type allocationPoolsStat struct {
	Pools []allocationPoolStat `json:"pools"`
}

func (aps allocationPools) stat(now common.Timestamp) (
	stat allocationPoolsStat) {

	stat.Pools = make([]allocationPoolStat, 0, len(aps.Pools))
	for _, ap := range aps.Pools {
		stat.Pools = append(stat.Pools, ap.stat(now))
	}
	return
}

//
// until stat
//
// swagger:model untilStat
type untilStat struct {
	PoolID   string           `json:"pool_id"`
	Balance  currency.Coin    `json:"balance"`
	ExpireAt common.Timestamp `json:"expire_at"`
}
