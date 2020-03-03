package storagesc

/*
import (
	"encoding/json"
	"time"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

// write pool (a locked tokens for an allocation)

type writePool struct {
	*tokenpool.ZcnLockingPool `json:"pool"`
}

func newWritePool() *writePool {
	return &writePool{ZcnLockingPool: &tokenpool.ZcnLockingPool{}}
}

func (wp *writePool) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(rp); err != nil {
		panic(err) // must never happens
	}
	return
}

func (wp *writePool) decode(input []byte) (err error) {

	type writePoolJSON struct {
		Pool json.RawMessage `json:"pool"`
	}

	var writePoolVal writePoolJSON
	if err = json.Unmarshal(input, &writePoolVal); err != nil {
		return
	}

	if len(writePoolVal.Pool) == 0 {
		return // no data given
	}

	err = rp.ZcnLockingPool.Decode(writePoolVal.Pool, &tokenLock{})
	return
}


// transfer tokens

type transferResponses struct {
	Responses []string `json:"responses"`
}

func (tr *transferResponses) addResponse(response string) {
	tr.Responses = append(tr.Responses, response)
}

func (tr *transferResponses) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(tr); err != nil {
		panic(err) // must never happens
	}
	return
}

func (tr *transferResponses) decode(input []byte) error {
	return json.Unmarshal(input, tr)
}

type readPoolStats struct {
	Stats []*readPoolStat `json:"stats"`
}

func (stats *readPoolStats) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(stats); err != nil {
		panic(err) // must never happens
	}
	return
}

func (stats *readPoolStats) decode(input []byte) error {
	return json.Unmarshal(input, stats)
}

func (stats *readPoolStats) addStat(stat *readPoolStat) {
	stats.Stats = append(stats.Stats, stat)
}

type readPoolStat struct {
	ID        datastore.Key    `json:"pool_id"`
	StartTime common.Timestamp `json:"start_time"`
	Duartion  time.Duration    `json:"duration"`
	TimeLeft  time.Duration    `json:"time_left"`
	Locked    bool             `json:"locked"`
	Balance   state.Balance    `json:"balance"`
}

func (stat *readPoolStat) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(stat); err != nil {
		panic(err) // must never happens
	}
	return
}

func (stat *readPoolStat) decode(input []byte) error {
	return json.Unmarshal(input, stat)
}

type tokenLock struct {
	StartTime common.Timestamp `json:"start_time"`
	Duration  time.Duration    `json:"duration"`
	Owner     datastore.Key    `json:"owner"`
}

func (tl tokenLock) IsLocked(entity interface{}) bool {
	if tm, ok := entity.(time.Time); ok {
		return tm.Sub(common.ToTime(tl.StartTime)) < tl.Duration
	}
	return true
}

func (tl tokenLock) LockStats(entity interface{}) []byte {
	if tm, ok := entity.(time.Time); ok {
		var stat readPoolStat
		stat.StartTime = tl.StartTime
		stat.Duartion = tl.Duration
		stat.TimeLeft = (tl.Duration - tm.Sub(common.ToTime(tl.StartTime)))
		stat.Locked = tl.IsLocked(tm)
		return stat.encode()
	}
	return nil
}

//
// smart contract methods
//

// getReadPoolsBytes of a client
func (ssc *StorageSmartContract) getReadPoolsBytes(t *transaction.Transaction,
	balances chainState.StateContextI) ([]byte, error) {

	return balances.GetTrieNode(readPoolsKey(ssc.ID, t.ClientID))
}

// getReadPools of current client
func (ssc *StorageSmartContract) getReadPools(t *transaction.Transaction,
	balances chainState.StateContextI) (wps *writePools, err error) {

	var poolb []byte
	if poolb, err = ssc.getReadPoolsBytes(t, balances); err != nil {
		return
	}
	rps = new(writePools)
	err = rps.Decode(poolb)
	return
}

// newWritePool SC function creates new read pool for a client.
func (ssc *StorageSmartContract) newWritePool(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	_, err = ssc.getReadPoolsBytes(t, balances)

	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("new_read_pool_failed", err.Error())
	}

	if err == nil {
		return "", common.NewError("new_read_pool_failed", "already exist")
	}

	var rps = newWritePools(t.ClientID)
	if _, err = balances.InsertTrieNode(rps.getKey(scKey), rps); err != nil {
		return "", common.NewError("new_read_pool_failed", err.Error())
	}

	return string(rps.Encode()), nil
}

// lock tokens for read pool of transaction's client
func (ssc *StorageSmartContract) readPoolLock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// user read pools

	var wps *writePools
	if rps, err = ssc.getReadPools(t, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	// lock request & user balance

	var lr lockRequest
	if err = lr.decode(inpu); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}
	var balance state.Balance
	balance, err = balances.GetClientBalance(t.ClientID)

	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	if err == util.ErrValueNotPresent {
		return "", common.NewError("read_pool_lock_failed", "no tokens to lock")
	}

	if state.Balance(t.Value) > balance {
		return "", common.NewError("read_pool_lock_failed",
			"lock amount is greater than balance")
	}

	if lp.Duration <= 0 {
		return "", common.NewError("read_pool_lock_failed",
			"invalid locking period")
	}

	// lock

	var rp = newWritePool()
	rp.TokenLockInterface = &tokenLock{
		StartTime: t.CreationDate,
		Duration:  lr.Duration,
		Owner:     t.ClientID,
	}

	var transfer *state.Transfer
	if transfer, resp, err = rp.DigPool(t.Hash, t); err != nil {
		return "", common.NewError("read_pool_lock_failed",
			err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	if err = rps.addPool(rp); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	_, err = balances.InsertTrieNode(rps.getKey(ssc.ID), rps)
	if err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	return
}

// unlock tokens if expired
func (ssc *StorageSmartContract) readPoolUnlock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// user read pools

	var wps *writePools
	if rps, err = ssc.getReadPools(t, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	// the request

	var (
		transfer *state.Transfer
		req      unlockRequest
	)

	if err = req.decode(input); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	var pool, ok = rps.Pools[req.PoolID]
	if !ok {
		return "", common.NewError("read_pool_unlock_failed", "pool not found")
	}

	transfer, resp, err = pool.EmptyPool(pool.ID, t.ClientID,
		common.ToTime(t.CreationDate))
	if err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}
	rps.delPool(pool.ID)
	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	// save pools
	if _, err = balances.InsertTrieNode(rps.getKey(ssc.ID), rps); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	return
}
*/
