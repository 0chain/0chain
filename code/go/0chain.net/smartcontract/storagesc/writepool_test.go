package storagesc

import (
	"testing"
	"time"

	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

func Test_writePoolRequest_encode_decode(t *testing.T) {
	const allocID = "alloc_hex"
	var (
		wrd, wre writePoolRequest
		err      error
	)
	wrd.AllocationID = allocID
	if err = wre.decode(wrd.encode()); err != nil {
		t.Fatal(err)
	}
	if wre.AllocationID != allocID {
		t.Fatal("wrong")
	}
}

func Test_newWritePool(t *testing.T) {
	const clientID = "client_hex"
	var wp = newWritePool(clientID)
	if wp.ZcnLockingPool == nil {
		t.Fatal("underlying pool is nil")
	}
	if wp.ClientID != clientID {
		t.Fatal("wrong of missing client id")
	}
}

func Test_writePoolKey(t *testing.T) {
	const sscKey, allocID = "ssc_hex", "alloc_hex"
	if writePoolKey(sscKey, allocID) == "" {
		t.Fatal("missing write pool key")
	}
}

func newTestWritePool(clientID, poolID string, balance state.Balance,
	startTime common.Timestamp, duration time.Duration) (wp *writePool) {

	wp = newWritePool(clientID)
	wp.ZcnLockingPool.ID = poolID
	wp.ZcnLockingPool.Balance = balance
	wp.ZcnLockingPool.TokenLockInterface = &tokenLock{
		StartTime: startTime,
		Duration:  duration,
		Owner:     clientID,
	}
	return
}

func Test_writePool_Encode_Decode(t *testing.T) {
	const clientID, poolID, balance = "client_hex", "pool_id", 100500
	var (
		sp     = common.Now() // start point
		dur    = 20 * time.Hour
		we, wd *writePool
		err    error
	)

	we = newTestWritePool(clientID, poolID, balance, sp, dur)
	wd = newWritePool("")

	if err = wd.Decode(we.Encode()); err != nil {
		t.Fatal(err)
	}
	if wd.ClientID != clientID {
		t.Fatal("wrong", wd.ClientID, clientID)
	}
	if wd.ZcnLockingPool == nil {
		t.Fatal("nil")
	}
	if we.ZcnLockingPool.ID != poolID {
		t.Fatal("wrong pool id")
	}
	if we.ZcnLockingPool.Balance != balance {
		t.Fatal("wrong balance")
	}
	if we.ZcnLockingPool.TokenLockInterface == nil {
		t.Fatal("missing tok. lock")
	}
	var tl, ok = we.ZcnLockingPool.TokenLockInterface.(*tokenLock)
	if !ok {
		t.Fatalf("wrong type %T", we.ZcnLockingPool.TokenLockInterface)
	}
	if tl.StartTime != sp || tl.Duration != dur || tl.Owner != clientID {
		t.Fatal("something wrong")
	}
}

func Test_writePool_save(t *testing.T) {
	const sscKey, allocID = "ssc_key", "alloc_hex"
	var (
		wp       = newWritePool("")
		balances = newTestBalances()
		err      error
	)
	if err = wp.save(sscKey, allocID, balances); err != nil {
		t.Fatal(err)
	}
	var seri util.Serializable
	seri, err = balances.GetTrieNode(writePoolKey(sscKey, allocID))
	if err != nil {
		t.Fatal(err)
	}
	var got, ok = seri.(*writePool)
	if !ok {
		t.Fatal("wrong type")
	}
	if got != wp {
		t.Fatal("wrong")
	}
}

func Test_writePool_fill(t *testing.T) {
	const sscKey, clientID, txHash = "ssc_hex", "client_hex", "tx_hash"
	var (
		wp       = newWritePool("")
		balances = newTestBalances()
		tx       = &transaction.Transaction{
			ClientID:   clientID,
			ToClientID: sscKey,
			Value:      90,
		}
	)
	tx.Hash = txHash
	balances.txn = tx
	var tr, resp, err = wp.fill(tx, balances)
	if err != nil {
		t.Fatal(err)
	}
	if tr == nil || resp == "" {
		t.Fatal("missing")
	}
	if wp.Balance != 90 {
		t.Fatal("wrong")
	}
}

func Test_writePool_setExpiration(t *testing.T) {
	const clientID = "client_hex"
	var (
		tp  = common.Now()
		dur = 60 * time.Second
		wp  = newWritePool(clientID)
		err error
	)
	wp.ZcnLockingPool.TokenLockInterface = &tokenLock{
		StartTime: tp,
	}
	if err = wp.setExpiration(tp + toSeconds(dur)); err != nil {
		t.Fatal(err)
	}
	var tl, ok = wp.ZcnLockingPool.TokenLockInterface.(*tokenLock)
	if !ok {
		t.Fatalf("wrong type %T", wp.ZcnLockingPool.TokenLockInterface)
	}
	if tl.Duration != dur {
		t.Fatal("wrong")
	}
}

func Test_writePoolStat_encode_decode(t *testing.T) {
	var (
		state, statd writePoolStat
		err          error
	)
	state.ID = "pool_hex"
	state.StartTime = common.Now()
	state.Duartion = 60 * time.Second
	state.TimeLeft = 90 * time.Second
	state.Locked = true
	state.Balance = 150
	if err = statd.decode(state.encode()); err != nil {
		t.Fatal(err)
	}
	if state != statd {
		t.Fatal("wrong")
	}
}

func newTestStorageSC() (ssc *StorageSmartContract) {
	ssc = new(StorageSmartContract)
	ssc.SmartContract = new(smartcontractinterface.SmartContract)
	ssc.ID = ADDRESS
	return
}

func TestStorageSmartContract_getWritePoolBytes(t *testing.T) {
	const allocID, clientID = "alloc_hex", "client_hex"
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		b, err   = ssc.getWritePoolBytes(allocID, balances)
	)
	if err == nil {
		t.Fatal("missing error")
	}
	if err != util.ErrValueNotPresent {
		t.Fatal("wrong error:", err)
	}
	var wp = newWritePool(clientID)
	_, err = balances.InsertTrieNode(writePoolKey(ssc.ID, allocID), wp)
	if err != nil {
		t.Fatal(err)
	}
	if b, err = ssc.getWritePoolBytes(allocID, balances); err != nil {
		t.Fatal(err)
	}
	if string(b) != string(wp.Encode()) {
		t.Fatal("wrong")
	}
}

func TestStorageSmartContract_getWritePool(t *testing.T) {
	const allocID, clientID, txHash = "alloc_hex", "client_hex", "tx_hash"
	var (
		ssc      = newTestStorageSC()
		wp       = newWritePool(clientID)
		balances = newTestBalances()
		tx       = &transaction.Transaction{
			ClientID:   clientID,
			ToClientID: ssc.ID,
			Value:      90,
		}
		err error
	)
	tx.Hash = txHash
	balances.txn = tx
	if _, _, err = wp.fill(tx, balances); err != nil {
		t.Fatal(err)
	}
	_, err = ssc.getWritePool(allocID, balances)
	if err == nil {
		t.Fatal("missing")
	}
	if err != util.ErrValueNotPresent {
		t.Fatal("wrong error")
	}
	if err = wp.save(ssc.ID, allocID, balances); err != nil {
		t.Fatal(err)
	}
	var wd *writePool
	if wd, err = ssc.getWritePool(allocID, balances); err != nil {
		t.Fatal(err)
	}
	if string(wd.Encode()) != string(wp.Encode()) {
		t.Fatal("wrong")
	}
}

func TestStorageSmartContract_newWritePool(t *testing.T) {
	const allocID, clientID = "alloc_hex", "client_hex"
	var (
		tp       = common.Now()
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		wp, err  = ssc.newWritePool(allocID, clientID, tp, tp, balances)
	)
	if err != nil {
		t.Fatal(err)
	}
	if wp.TokenPool.ID != writePoolKey(ssc.ID, allocID) {
		t.Fatal("wrong pool key")
	}
}

func TestStorageSmartContract_createWritePool(t *testing.T) {
	const (
		allocID, clientID, txHash = "alloc_hex", "client_hex", "tx_hash"

		errMsg1 = "not enough tokens to create allocation: 90 < 150"
		errMsg2 = "can't fill write pool: no tokens to lock"
		errMsg3 = "can't fill write pool: lock amount is greater than balance"
	)
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		tx       = &transaction.Transaction{
			ClientID:   clientID,
			ToClientID: ssc.ID,
			Value:      90,
		}
		sa = &StorageAllocation{
			ID:            allocID,
			MinLockDemand: 150,
		}
		wp  *writePool
		err error
	)
	tx.Hash = txHash
	balances.txn = tx
	if err = ssc.createWritePool(tx, sa, balances); err == nil {
		t.Fatal("missing error")
	} else if err.Error() != errMsg1 {
		t.Fatal("unexpected error:", err)
	}
	sa.MinLockDemand = 90
	if err = ssc.createWritePool(tx, sa, balances); err == nil {
		t.Fatal("missing")
	} else if err.Error() != errMsg2 {
		t.Fatal("unexpected error:", err)
	}
	balances.balances[clientID] = 85
	if err = ssc.createWritePool(tx, sa, balances); err == nil {
		t.Fatal("missing")
	} else if err.Error() != errMsg3 {
		t.Fatal("unexpected error:", err)
	}
	balances.balances[clientID] = 120
	if err = ssc.createWritePool(tx, sa, balances); err != nil {
		t.Fatal(err)
	}
	if wp, err = ssc.getWritePool(allocID, balances); err != nil {
		t.Fatal(err)
	}
	if wp == nil {
		t.Fatal("missing")
	}
	if wp.Balance != 90 {
		t.Fatal("wrong pool balance:", wp.Balance, ", expected 90")
	}
	// check transfer
	if len(balances.transfers) != 1 {
		t.Fatal("unexpected length of transfers:", len(balances.transfers))
	}
	var transfer = balances.transfers[0]
	if transfer.ClientID != clientID {
		t.Fatal("invalid transfer.ClientID:", transfer.ClientID)
	}
	if transfer.ToClientID != ADDRESS {
		t.Fatal("invalid transfer.ToClientID:", transfer.ToClientID)
	}
	if transfer.Amount != 90 {
		t.Fatal("invalid transfer amount:", transfer.Amount)
	}
}

func TestStorageSmartContract_writePoolLock(t *testing.T) {
	// TODO (sfxdx): implement tests
}

func TestStorageSmartContract_writePoolUnlock(t *testing.T) {
	// TODO (sfxdx): implement tests
}

func TestStorageSmartContract_getWritePoolStat(t *testing.T) {
	// TODO (sfxdx): implement tests
}

func TestStorageSmartContract_getWritePoolStatHandler(t *testing.T) {
	// TODO (sfxdx): implement tests
}
