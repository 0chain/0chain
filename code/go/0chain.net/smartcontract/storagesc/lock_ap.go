package storagesc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool"
)

// lock tokens for write pool of transaction's client
func (ssc *StorageSmartContract) allocationPoolLock(
	txn *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	var conf *Config
	var err error
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"can't get configs: "+err.Error())
	}

	var lr lockRequest
	if err = lr.decode(input); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	if lr.AllocationID == "" {
		return "", common.NewError("write_pool_lock_failed",
			"missing allocation ID in request")
	}

	iTxnVal, err := currency.Int64ToCoin(txn.Value)
	if err != nil {
		return "", err
	}
	if iTxnVal < conf.WritePool.MinLock || txn.Value <= 0 {
		return "", common.NewError("write_pool_lock_failed",
			"insufficient amount to lock")
	}

	if lr.Duration < conf.WritePool.MinLockPeriod {
		return "", common.NewError("write_pool_lock_failed",
			fmt.Sprintf("duration (%s) is shorter than min lock period (%s)",
				lr.Duration.String(), conf.WritePool.MinLockPeriod.String()))
	}

	if lr.Duration > conf.WritePool.MaxLockPeriod {
		return "", common.NewError("write_pool_lock_failed",
			fmt.Sprintf("duration (%s) is longer than max lock period (%v)",
				lr.Duration.String(), conf.WritePool.MaxLockPeriod.String()))
	}

	// check client balance
	if err = stakepool.CheckClientBalance(txn, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	aps, err := getAllocationPools(lr.AllocationID, balances)
	if err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"cannot find allocation pools for "+lr.AllocationID+": "+err.Error())
	}

	transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, currency.Coin(txn.Value))
	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	ap, found := aps.Pools[txn.ClientID]
	if !found {
		if len(aps.Pools) >= conf.MaxPoolsPerAllocation {
			return "", common.NewError("write_pool_lock_failed",
				fmt.Sprintf("exceeded the  maximum number of pools:  %v", conf.MaxPoolsPerAllocation))
		}
		ap = new(allocationPool)
		aps.Pools[txn.ClientID] = ap
	} else {
		if ap.ExpireAt > txn.CreationDate+toSeconds(lr.Duration) {
			return "", common.NewError("write_pool_lock_failed",
				"can only decrease the expiry date  "+ap.ExpireAt.Duration().String())
		}
	}
	ap.Balance += currency.Coin(txn.Value)
	ap.ExpireAt = txn.CreationDate + toSeconds(lr.Duration)
	if err := aps.save(lr.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}
	return "", nil
}

// unlock tokens if expired
func (ssc *StorageSmartContract) allocationPoolUnlock(
	txn *transaction.Transaction,
	input []byte, balances cstate.StateContextI,
) (string, error) {
	var err error
	var req unlockRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(req.AllocationID, balances)
	if err != nil {
		return "", common.NewError("write_pool_unlock_failed",
			"can't get related allocation: "+err.Error())
	}

	aps, err := getAllocationPools(req.AllocationID, balances)
	if err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"cannot find allocation pools for "+req.AllocationID+": "+err.Error())
	}

	ap, found := aps.Pools[txn.ClientID]
	if !found {
		return "", common.NewError("write_pool_unlock_failed",
			fmt.Sprintf("no write pool for user %s in allocation %s", txn.ClientID, req.AllocationID))
	}

	if ap.Balance < currency.Coin(txn.Value) {
		return "", common.NewError("write_pool_unlock_failed",
			fmt.Sprintf("insufficent funds %v in allocation pool", ap.Balance))

	}
	ap.Balance -= currency.Coin(txn.Value)

	// don't unlock over min lock demand left
	if !alloc.Finalized && !alloc.Canceled {
		var (
			want  = alloc.restMinLockDemand()
			unitl = alloc.Until()
			leave = aps.allocUntil(unitl) - ap.Balance
		)
		if leave < want && ap.ExpireAt >= unitl {
			return "", common.NewError("write_pool_unlock_failed",
				"can't unlock, because min lock demand is not paid yet")
		}
	}

	transfer := state.NewTransfer(ADDRESS, txn.ToClientID, currency.Coin(txn.Value))
	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	return "", nil
}
