package storagesc

import (
	"testing"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
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
	startTime, duration common.Timestamp) (wp *writePool) {

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
		tp     = common.Now()
		we, wd *writePool
		err    error
	)

	we = newTestWritePool(clientID, poolID, balance, tp, tp)
	wd = newWritePool("")

	if err = wd.Decode(we.Encode()); err != nil {
		t.Fatal(err)
	}
	if wd.ClientID != clientID {
		t.Fatal("wrong")
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
	if tl.StartTime != tp || tl.Duration != tp || tl.Owner != clientID {
		t.Fatal("something wrong")
	}
}

func Test_writePool_save(t *testing.T) {
	//
}

func Test_writePool_fill(t *testing.T) {
	//
}

func Test_writePool_extend(t *testing.T) {
	//
}

func Test_writePoolStat_encode_decode(t *testing.T) {
	//
}

func newTestStorageSC() (ssc *StorageSmartContract) {
	ssc = new(StorageSmartContract)
	ssc.ID = ADDRESS
	return
}

func TestStorageSmartContract_getWritePoolBytes(t *testing.T) {
	//
}

func TestStorageSmartContract_getWritePool(t *testing.T) {
	//
}

func TestStorageSmartContract_newWritePool(t *testing.T) {
	//
}

func TestStorageSmartContract_createWritePool(t *testing.T) {
	//
}

func TestStorageSmartContract_writePoolLock(t *testing.T) {
	//
}

func TestStorageSmartContract_writePoolUnlock(t *testing.T) {
	//
}

func TestStorageSmartContract_getWritePoolStat(t *testing.T) {
	//
}

func TestStorageSmartContract_getWritePoolStatHandler(t *testing.T) {
	//
}
