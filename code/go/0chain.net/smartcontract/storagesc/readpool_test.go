package storagesc

import (
	"testing"
	"time"
)

func Test_lockRequest_encode_decode(t *testing.T) {
	var (
		lre, lrd lockRequest
		err      error
	)
	lre.Duration = time.Second * 60
	if err = lrd.decode(lre.encode()); err != nil {
		t.Fatal(err)
	}
	if lrd != lre {
		t.Fatal("wrong")
	}
}

func Test_unlockRequest_encode_decode(t *testing.T) {
	var (
		ure, urd unlockRequest
		err      error
	)
	ure.PoolID = "pool_hex"
	if err = urd.decode(ure.encode()); err != nil {
		t.Fatal(err)
	}
	if urd != ure {
		t.Fatal("wrong")
	}
}

func Test_newReadPool(t *testing.T) {
	var rp = newReadPool()
	if rp.ZcnLockingPool == nil {
		t.Fatal("missing")
	}
}

func Test_readPool_encode_decode(t *testing.T) {
	var (
		rpe, rpd = newReadPool(), newReadPool()
		err      error
	)
	if err = rpd.decode(rpe.encode()); err != nil {
		t.Fatal(err)
	}
	if string(rpe.encode()) != string(rpd.encode()) {
		t.Fatal("wrong")
	}
}

func Test_newReadPools(t *testing.T) {
	var rps = newReadPools()
	if rps.Pools == nil {
		t.Fatal("map not created")
	}
}

func Test_readPools_Encode_Decode(t *testing.T) {
	var (
		rpse, rpsd = newReadPools(), newReadPools()
		rp         = newReadPool()
		err        error
	)
	if err = rpse.addPool(rp); err != nil {
		t.Fatal(err)
	}
	if err = rpsd.Decode(rpse.Encode()); err != nil {
		t.Fatal(err)
	}
	if string(rpse.Encode()) != string(rpsd.Encode()) {
		t.Fatal("wrong")
	}
}

func Test_readPoolsKey(t *testing.T) {
	if readPoolsKey("scKey", "clientID") == "" {
		t.Fatal("missing")
	}
}

func Test_readPools_addPool_delPool(t *testing.T) {
	var (
		rps = newReadPools()
		rp  = newReadPool()
		err error
	)
	if err = rps.addPool(rp); err != nil {
		t.Fatal(err)
	}
	if err = rps.addPool(rp); err == nil {
		t.Fatal("missing error")
	}
	rps.delPool(rp.ID)
	if len(rps.Pools) != 0 {
		t.Fatal("not deleted")
	}
}

func Test_readPoolStats_encode_decode(t *testing.T) {
	// TODO (sfxdx): implement
}

func Test_readPoolStats_addStat(t *testing.T) {
	// TODO (sfxdx): implement
}

func Test_readPoolStat_encode_decode(t *testing.T) {
	// TODO (sfxdx): implement
}

func Test_tokenLock_IsLocked(t *testing.T) {
	// TODO (sfxdx): implement
}

func Test_tokenLock_LockStats(t *testing.T) {
	// TODO (sfxdx): implement
}

func TestStorageSmartContract_getReadPoolsBytes(t *testing.T) {
	//
}

func TestStorageSmartContract_getReadPools(t *testing.T) {
	//
}

func TestStorageSmartContract_newReadPool(t *testing.T) {
	//
}

func TestStorageSmartContract_readPoolLock(t *testing.T) {
	//
}

func TestStorageSmartContract_readPoolUnlock(t *testing.T) {
	//
}

func TestStorageSmartContract_getReadPoolStat(t *testing.T) {
	//
}

func TestStorageSmartContract_getReadPoolsStatsHandler(t *testing.T) {
	//
}
