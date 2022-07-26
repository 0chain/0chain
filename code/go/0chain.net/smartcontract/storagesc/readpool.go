package storagesc

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
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
// swagger:model readPool
type readPool struct {
	Balance currency.Coin `json:"balance"`
}

type readPoolLockRequest struct {
	TargetId   string `json:"target_id,omitempty"`
	MintTokens bool   `json:"mint_tokens,omitempty"`
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

func (rp *readPool) moveToBlobber(allocID, blobID string,
	sp *stakePool, value currency.Coin, balances cstate.StateContextI) (resp string, err error) {

	// all redeems to response at the end
	var redeems []readPoolRedeem
	var moved currency.Coin
	currentBalance := rp.Balance

	if currentBalance == 0 {
		return "", fmt.Errorf("no tokens in read pool for allocation: %s,"+
			" blobber: %s", allocID, blobID)
	}
	if value >= currentBalance {
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

	err = sp.DistributeRewards(value, blobID, spenum.Blobber, balances)
	if err != nil {
		return "", fmt.Errorf("can't move tokens to blobber: %v", err)
	}

	// return the read redeems for blobbers read pools cache
	return toJson(redeems), nil // ok
}

//
// smart contract methods
//

// lock tokens for read pool of transaction's client
func (ssc *StorageSmartContract) newReadPool(t *transaction.Transaction,
	_ []byte, balances cstate.StateContextI) (resp string, err error) {
	_, err = ssc.getReadPool(t.ClientID, balances)
	if err == nil {
		return "", common.NewError("new_read_pool_failed", "already exist")
	} else if err != util.ErrValueNotPresent {
		return "", common.NewError("new_read_pool_failed", err.Error())
	}

	rp := new(readPool)
	if err = rp.save(ssc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("new_read_pool_failed", err.Error())
	}

	return string(rp.Encode()), nil
}

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

	if !req.MintTokens {
		// check client balance
		if err = stakepool.CheckClientBalance(txn, balances); err != nil {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		}
		// transfer balance from client to smart contract
		transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, currency.Coin(txn.Value))
		if err = balances.AddTransfer(transfer); err != nil {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		}
	} else {
		if err = balances.AddMint(&state.Mint{
			Minter:     ADDRESS,
			ToClientID: ADDRESS,
			Amount:     txn.Value,
		}); err != nil {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		}
	}

	rp, err := ssc.getReadPool(req.TargetId, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return "", common.NewError("read_pool_lock_failed", err.Error())
		} else {
			rp = new(readPool)
		}
	}

	//add to read pool balance
	if err = rp.add(currency.Coin(txn.Value)); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	// save read pool
	if err = rp.save(ssc.ID, req.TargetId, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	return "", nil
}

// unlock tokens if expired
func (ssc *StorageSmartContract) readPoolUnlock(txn *transaction.Transaction, input []byte, balances cstate.StateContextI) (string, error) {
	rp, err := ssc.getReadPool(txn.ClientID, balances)
	if err != nil {
		return "", common.NewError("read_pool_unlock_failed", "no read pool found for clientID to unlock token")
	}

	// adjust balance
	balance := rp.drain()
	// transfer adjusted balance to client
	transfer := state.NewTransfer(ssc.ID, txn.ClientID, balance)
	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	// save read pool
	if err = rp.save(ssc.ID, txn.ClientID, balances); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	return "", nil
}
