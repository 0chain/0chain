package storagesc

import (
	"testing"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/util"
)

//
// use:
//
//      go test -cover -coverprofile=cover.out && go tool cover -html=cover.out -o=cover.html
//
// to test and generate coverage html page
//

func TestStorageSmartContract_getAllocation(t *testing.T) {
	const allocID, clientID, clientPk = "alloc_hex", "client_hex", "pk"
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		alloc    *StorageAllocation
		err      error
	)
	if alloc, err = ssc.getAllocation(allocID, balances); err == nil {
		t.Fatal("missing erorr")
	}
	if err != util.ErrValueNotPresent {
		t.Fatal("unexpectede error:", err)
	}
	alloc = new(StorageAllocation)
	alloc.ID = allocID
	alloc.DataShards = 1
	alloc.ParityShards = 1
	alloc.Size = 1024
	alloc.Expiration = 1050
	alloc.Owner = clientID
	alloc.OwnerPublicKey = clientPk
	_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), alloc)
	if err != nil {
		t.Fatal(err)
	}
	var got *StorageAllocation
	if got, err = ssc.getAllocation(allocID, balances); err != nil {
		t.Fatal(err)
	}
	if string(got.Encode()) != string(alloc.Encode()) {
		t.Fatal("wrong")
	}
}

func isEqualStrings(a, b []string) (eq bool) {
	if len(a) != len(b) {
		return
	}
	for i, ax := range a {
		if b[i] != ax {
			return false
		}
	}
	return true
}

func Test_newAllocationRequest_storageAllocation(t *testing.T) {
	const allocID, clientID, clientPk = "alloc_hex", "client_hex", "pk"
	var nar newAllocationRequest
	nar.DataShards = 2
	nar.ParityShards = 3
	nar.Size = 1024
	nar.Expiration = common.Now()
	nar.Owner = clientID
	nar.OwnerPublicKey = clientPk
	nar.PreferredBlobbers = []string{"one", "two"}
	nar.ReadPriceRange = PriceRange{Min: 10, Max: 20}
	nar.WritePriceRange = PriceRange{Min: 100, Max: 200}
	var alloc = nar.storageAllocation()
	if alloc.DataShards != nar.DataShards {
		t.Error("wrong")
	}
	if alloc.ParityShards != nar.ParityShards {
		t.Error("wrong")
	}
	if alloc.Size != nar.Size {
		t.Error("wrong")
	}
	if alloc.Expiration != nar.Expiration {
		t.Error("wrong")
	}
	if alloc.Owner != nar.Owner {
		t.Error("wrong")
	}
	if alloc.OwnerPublicKey != nar.OwnerPublicKey {
		t.Error("wrong")
	}
	if !isEqualStrings(alloc.PreferredBlobbers, nar.PreferredBlobbers) {
		t.Error("wrong")
	}
	if alloc.ReadPriceRange != nar.ReadPriceRange {
		t.Error("wrong")
	}
	if alloc.WritePriceRange != nar.WritePriceRange {
		t.Error("wrong")
	}
}

func Test_newAllocationRequest_decode(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_addBlobbersOffers(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_updateBlobbersInAll(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_toSeconds(t *testing.T) {
	if toSeconds(time.Second*60+time.Millisecond*90) != 60 {
		t.Fatal("wrong")
	}
}

func Test_sizeInGB(t *testing.T) {
	if sizeInGB(12345*1024*1024*1024) != 12345.0 {
		t.Error("wrong")
	}
}

func TestStorageSmartContract_newAllocationRequest(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_updateAllocationRequest_decode(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_updateAllocationRequest_validate(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_updateAllocationRequest_getBlobbersSizeDiff(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_updateAllocationRequest_getNewBlobbersSize(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_getAllocationBlobbers(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_closeAllocation(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_saveUpdatedAllocation(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_extendAllocation(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_reduceAllocation(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_updateAllocationRequest(t *testing.T) {
	// TODO (sfxdx): implements tests
}
