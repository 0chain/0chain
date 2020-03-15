package storagesc

import (
	"testing"
	"time"
)

//
// use:
//
//      go test -cover -coverprofile=cover.out && go tool cover -html=cover.out -o=cover.html
//
// to test and generate coverage html page
//

func TestStorageSmartContract_getAllocation(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_newAllocationRequest_storageAllocation(t *testing.T) {
	// TODO (sfxdx): implements tests
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
