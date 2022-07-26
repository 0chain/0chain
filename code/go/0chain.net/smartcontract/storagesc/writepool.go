package storagesc

import (
	"encoding/json"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool"
)

//
// SC / API requests
//

// lock request
type lockRequest struct {
	AllocationID string `json:"allocation_id"`
}

func (lr *lockRequest) decode(input []byte) (err error) {
	if err = json.Unmarshal(input, lr); err != nil {
		return
	}
	return // ok
}

type unlockRequest struct {
	AllocationID string `json:"allocation_id"`
}

func (ur *unlockRequest) decode(input []byte) error {
	return json.Unmarshal(input, ur)
}

func (ssc *StorageSmartContract) writePoolLock(
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

	if txn.Value < conf.WritePool.MinLock {
		return "", common.NewError("write_pool_lock_failed",
			"insufficient amount to lock")
	}

	// check client balance
	if err = stakepool.CheckClientBalance(txn, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, txn.Value)
	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	allocation, err := ssc.getAllocation(lr.AllocationID, balances)
	if err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"cannot find allocation pools for "+lr.AllocationID+": "+err.Error())
	}

	if allocation.Finalized || allocation.Canceled {
		return "", common.NewError("write_pool_unlock_failed",
			"can't lock tokens with a finalized or cancelled allocation")

	}

	allocation.WritePool, err = currency.AddCoin(allocation.WritePool, txn.Value)
	if err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}
	if err := allocation.saveUpdatedAllocation(nil, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	return "", nil
}

// unlock tokens if expired
func (ssc *StorageSmartContract) writePoolUnlock(
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

	if alloc.Owner != txn.ClientID {
		return "", common.NewError("write_pool_unlock_failed",
			"only owner can unlock tokens")
	}

	if !alloc.Finalized && !alloc.Canceled {
		return "", common.NewError("write_pool_unlock_failed",
			"can't unlock until the allocation is finalized or cancelled")
	}

	if alloc.WritePool == 0 {
		return "", common.NewError("write_pool_unlock_failed",
			"no tokens to unlock")
	}

	transfer := state.NewTransfer(ssc.ID, txn.ClientID, alloc.WritePool)
	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}
	alloc.WritePool = 0
	if err = alloc.saveUpdatedAllocation(nil, balances); err != nil {
		return "", common.NewError("write_pool_unlock_failed",
			"saving allocation pools: "+err.Error())
	}
	return "", nil
}
