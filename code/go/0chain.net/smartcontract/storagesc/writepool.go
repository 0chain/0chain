package storagesc

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"github.com/0chain/common/core/currency"
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

//nolint:unused
type unlockRequest struct {
	AllocationID string `json:"allocation_id"`
}

//nolint:unused
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
	if err = stakepool.CheckClientBalance(txn.ClientID, txn.Value, balances); err != nil {
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

	if allocation.mustBase().Finalized || allocation.mustBase().Canceled {
		return "", common.NewError("write_pool_lock_failed",
			"can't lock tokens with a finalized or cancelled allocation")

	}

	err = allocation.mustUpdateBase(func(ab *storageAllocationBase) error {
		ab.WritePool, err = currency.AddCoin(ab.WritePool, txn.Value)
		return err
	})
	if err != nil {
		return "", common.NewError("write_pool_lock_failed", fmt.Sprintf("write pool token overflow: %v", err))
	}

	i, err := txn.Value.Int64()
	if err != nil {
		return "", common.NewError("write_pool_lock_failed", fmt.Sprintf("invalid lock value: %v", err))
	}

	balances.EmitEvent(event.TypeStats, event.TagLockWritePool, allocation.mustBase().ID, event.WritePoolLock{
		Client:       txn.ClientID,
		AllocationId: allocation.mustBase().ID,
		Amount:       i,
	})

	if err := allocation.saveUpdatedStakes(balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	return "", nil
}
