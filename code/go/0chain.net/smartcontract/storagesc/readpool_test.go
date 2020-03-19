package storagesc

import (
	"encoding/json"
	"testing"
	"time"

	"0chain.net/core/common"
)

func mustEncode(t *testing.T, val interface{}) []byte {
	var err error
	b, err := json.Marshal(val)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func Test_lockRequest_decode(t *testing.T) {
	var (
		lre, lrd lockRequest
		err      error
	)
	lre.Duration = time.Second * 60
	if err = lrd.decode(mustEncode(t, lre)); err != nil {
		t.Fatal(err)
	}
	if !isEqualStrings(lrd.Blobbers, lre.Blobbers) {
		t.Fatal("wrong blobbers list")
	}
	if lrd.AllocationID != lre.AllocationID || lrd.Duration != lre.Duration {
		t.Fatal("wrong")
	}
}

func Test_lockRequest_validate(t *testing.T) {
	// TODO (sfxdx): implement
}

func Test_readPoolLock_copy(t *testing.T) {
	// TODO (sfxdx): implement
}

func Test_readPoolBlobbers_copy(t *testing.T) {
	// TODO (sfxdx): implement
}

func Test_readPoolAllocs_update(t *testing.T) {
	// TODO (sfxdx): implement
}

func Test_readPoolAllocs_copy(t *testing.T) {
	// TODO (sfxdx): implement
}

func Test_newReadPool(t *testing.T) {
	var rp = newReadPool()
	if rp.Locked == nil {
		t.Fatal("missing locked pool")
	}
	if rp.Unlocked == nil {
		t.Fatal("missing unlocked pool")
	}
	if rp.Locks == nil {
		t.Fatal("map is not created")
	}
}

func Test_readPool_Encode_Decode(t *testing.T) {
	const allocID, dur, value = "alloc_hex", 10 * time.Second, 150
	var (
		now      = common.Now()
		rpe, rpd = newReadPool(), newReadPool()
		blobbers = []string{"b1_hex", "b2_hex", "b3_hex"}
		lr       lockRequest
		err      error
	)

	lr.AllocationID = "all"
	lr.Blobbers = blobbers
	lr.Duration = dur

	if err = rpe.addLocks(now, value, &lr); err != nil {
		t.Fatal(err)
	}

	if err = rpd.Decode(rpe.Encode()); err != nil {
		t.Fatal(err)
	}
	if string(rpe.Encode()) != string(rpd.Encode()) {
		t.Fatal("wrong")
	}
}

func Test_readPoolsKey(t *testing.T) {
	if readPoolKey("scKey", "clientID") == "" {
		t.Fatal("missing")
	}
}

func Test_readPool_fill(t *testing.T) {
	//
}

func Test_readPool_addLocks(t *testing.T) {
	//
}

func Test_readPool_update(t *testing.T) {
	//
}

func Test_readPool_save(t *testing.T) {
	//
}

func Test_readPool_stat(t *testing.T) {
	//
}

func TestStorageSmartContract_getReadPoolBytes(t *testing.T) {
	// TODO (sfxdx): implement
}

func TestStorageSmartContract_getReadPool(t *testing.T) {
	// TODO (sfxdx): implement
}

func TestStorageSmartContract_newReadPool(t *testing.T) {
	// TODO (sfxdx): implement
}

func TestStorageSmartContract_checkFill(t *testing.T) {
	// TODO (sfxdx): implement
}

func TestStorageSmartContract_readPoolLock(t *testing.T) {
	// TODO (sfxdx): implement
}

func TestStorageSmartContract_readPoolUnlock(t *testing.T) {
	// TODO (sfxdx): implement
}

func TestStorageSmartContract_getReadPoolStatHandler(t *testing.T) {
	// TODO (sfxdx): implement
}

func TestStorageSmartContract_getReadPoolBlobberHandler(t *testing.T) {
	// TODO (sfxdx): implement
}
