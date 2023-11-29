package storagesc

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/util"
)

//msgp:ignore readPoolRedeem
//go:generate msgp -io=false -tests=false -unexported=true -v

//
// client read pool (consist of allocation pools)
//

func readPoolKey(scKey, clientID string) datastore.Key {
	return scKey + ":readpool:" + clientID
}

// readPool represents new trimmed down readPool consisting of two balances,
// one for the allocations that the client (client_id) owns
// and the other for the allocations that the client (client_id) doesn't own
type readPool struct {
	Balance currency.Coin `json:"balance"`
}

type readPoolLockRequest struct {
	TargetId string `json:"target_id,omitempty"`
}

func (lr *readPoolLockRequest) decode(input []byte) (err error) {
	if err = json.Unmarshal(input, lr); err != nil {
		return
	}
	return // ok
}

// The readPoolRedeem represents part of response of read markers redeeming.
// A Blobber uses this response for internal read pools cache.
type readPoolRedeem struct {
	PoolID  string        `json:"pool_id"` // read pool ID
	Balance currency.Coin `json:"balance"` // balance reduction
}

// Encode implements util.Serializable interface.
func (rp *readPool) Encode() []byte {
	var b, err = json.Marshal(rp)
	if err != nil {
		panic(err) // must never happen
	}
	return b
}

// Decode implements util.Serializable interface.
func (rp *readPool) Decode(p []byte) error {
	return json.Unmarshal(p, rp)
}

func (rp *readPool) add(coin currency.Coin) error {
	sum, err := currency.AddCoin(rp.Balance, coin)
	if err != nil {
		return err
	}
	rp.Balance = sum
	return nil
}

func (rp *readPool) drain() (diff currency.Coin) {
	diff = rp.Balance
	rp.Balance = 0
	return
}

func (rp *readPool) save(sscKey, clientID string, balances cstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(readPoolKey(sscKey, clientID), rp)
	return
}

// getReadPool of current client
func (ssc *StorageSmartContract) getReadPool(clientID datastore.Key, balances cstate.StateContextI) (rp *readPool, err error) {
	rp = new(readPool)
	err = balances.GetTrieNode(readPoolKey(ssc.ID, clientID), rp)
	return
}

func toJson(val interface{}) string {
	var b, err = json.Marshal(val)
	if err != nil {
		panic(err) // must not happen
	}
	return string(b)
}

func (rp *readPool) moveToBlobber(sscID string, allocID, blobID string,
	sp *stakePool, value currency.Coin, balances cstate.StateContextI) (resp string, err error) {

	// all redeems to response at the end
	var redeems []readPoolRedeem
	var moved currency.Coin
	currentBalance := rp.Balance

	if value > currentBalance {
		return "", fmt.Errorf("not enough tokens in read pool for "+
			"allocation: %s, blobber: %s", allocID, blobID)
	} else {
		moved, currentBalance = value, currentBalance-value
	}

	redeems = append(redeems, readPoolRedeem{
		PoolID:  blobID,
		Balance: moved,
	})

	rp.Balance = currentBalance

	err = sp.DistributeRewards(sscID, value, blobID, spenum.Blobber, spenum.FileDownloadReward, balances, true, allocID)
	if err != nil {
		return "", fmt.Errorf("can't move tokens to blobber: %v", err)
	}

	// return the read redeems for blobbers read pools cache
	return toJson(redeems), nil // ok
}

//
// smart contract methods
//

func (ssc *StorageSmartContract) readPoolLock(txn *transaction.Transaction, input []byte, balances cstate.StateContextI) (string, error) {
	conf, err := ssc.getReadPoolConfig(balances, true)
	if err != nil {
		return "", common.NewError("read_pool_lock_failed",
			"can't get configs: "+err.Error())
	}

	if txn.Value < conf.MinLock {
		return "", common.NewError("read_pool_lock_failed",
			"insufficient amount to lock")
	}

	if txn.Value <= 0 {
		return "", common.NewError("read_pool_lock_failed",
			"invalid amount to lock [ensure token > 0].")
	}

	var req readPoolLockRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	if req.TargetId == "" {
		req.TargetId = txn.ClientID
	}

	return ssc.readPoolLockInternal(txn, txn.Value, false, req.TargetId, balances)
}

func (ssc *StorageSmartContract) readPoolLockInternal(txn *transaction.Transaction, toLock currency.Coin, mint bool, targetId string, balances cstate.StateContextI) (string, error) {
	if !mint {
		// check client balance
		if err := stakepool.CheckClientBalance(txn.ClientID, toLock, balances); err != nil {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		}
		// transfer balance from client to smart contract
		transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, currency.Coin(txn.Value))
		if err := balances.AddTransfer(transfer); err != nil {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		}
	} else {
		if err := balances.AddMint(&state.Mint{
			Minter:     ADDRESS,
			ToClientID: ADDRESS,
			Amount:     toLock,
		}); err != nil {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		}
	}

	var newReadPool = false
	rp, err := ssc.getReadPool(targetId, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		} else {
			rp = new(readPool)
			newReadPool = true
		}
	}

	//add to read pool balance
	if err = rp.add(toLock); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	// Save read pool
	if err = rp.save(ssc.ID, targetId, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	i, _ := txn.Value.Int64()

	// updates the snapshot table
	balances.EmitEvent(event.TypeStats, event.TagLockReadPool, txn.ClientID, event.ReadPoolLock{
		Client: txn.ClientID,
		PoolId: targetId,
		Amount: i,
	})

	if newReadPool {
		balances.EmitEvent(event.TypeStats, event.TagInsertReadpool, txn.ClientID, event.ReadPool{
			UserID:  txn.ClientID,
			Balance: rp.Balance,
		})
	} else {
		// updates the readpool table
		balances.EmitEvent(event.TypeStats, event.TagUpdateReadpool, txn.ClientID, event.ReadPool{
			UserID:  txn.ClientID,
			Balance: rp.Balance,
		})
	}

	return "", nil
}

// unlock tokens if expired
func (ssc *StorageSmartContract) readPoolUnlock(txn *transaction.Transaction, _ []byte, balances cstate.StateContextI) (string, error) {
	rp, err := ssc.getReadPool(txn.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("read_pool_unlock_failed", "no read pool found for clientID to unlock token: %v", err)
	}

	// adjust balance
	balance := rp.drain()
	// transfer adjusted balance to client
	transfer := state.NewTransfer(ssc.ID, txn.ClientID, balance)
	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	// Save read pool
	if err = rp.save(ssc.ID, txn.ClientID, balances); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	i, _ := balance.Int64()
	key := readPoolKey(ssc.ID, txn.ClientID)

	// updates the snapshot table
	balances.EmitEvent(event.TypeStats, event.TagUnlockReadPool, key, event.ReadPoolLock{
		Client: txn.ClientID,
		PoolId: key,
		Amount: i,
	})

	// updates the readpool table
	balances.EmitEvent(event.TypeStats, event.TagUpdateReadpool, txn.ClientID, event.ReadPool{
		UserID:  txn.ClientID,
		Balance: rp.Balance,
	})

	return "", nil
}
