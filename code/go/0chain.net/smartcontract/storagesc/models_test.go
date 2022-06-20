package storagesc

import (
	"testing"
	"time"

	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
)

func TestStorageAllocation_validate(t *testing.T) {

	const (
		errMsg1 = "invalid read_price range"
		errMsg2 = "invalid write_price range"
		errMsg3 = "insufficient allocation size"
		errMsg4 = "insufficient allocation duration"
		errMsg5 = "invalid number of data shards"
		errMsg6 = "missing owner public key"
		errMsg7 = "missing owner id"
	)

	var (
		now   = time.Unix(150, 0)
		alloc StorageAllocation
		conf  Config
	)

	conf.MinAllocSize = 10 * 1024
	conf.MinAllocDuration = 48 * time.Hour

	alloc.ReadPriceRange = PriceRange{Min: 20, Max: 10}
	requireErrMsg(t, alloc.validate(now, &conf), errMsg1)

	alloc.ReadPriceRange = PriceRange{Min: 10, Max: 20}
	alloc.WritePriceRange = PriceRange{Min: 20, Max: 10}
	requireErrMsg(t, alloc.validate(now, &conf), errMsg2)

	alloc.WritePriceRange = PriceRange{Min: 10, Max: 20}
	alloc.Size = 5 * 1024
	requireErrMsg(t, alloc.validate(now, &conf), errMsg3)

	alloc.Size = 10 * 1024
	alloc.Expiration = 170
	requireErrMsg(t, alloc.validate(now, &conf), errMsg4)

	alloc.Expiration = 150 + toSeconds(48*time.Hour)
	alloc.DataShards = 0
	requireErrMsg(t, alloc.validate(now, &conf), errMsg5)

	alloc.DataShards = 1
	alloc.OwnerPublicKey = ""
	requireErrMsg(t, alloc.validate(now, &conf), errMsg6)

	alloc.OwnerPublicKey = "pk_hex"
	alloc.Owner = ""
	requireErrMsg(t, alloc.validate(now, &conf), errMsg7)

	alloc.Owner = "client_hex"
	assert.NoError(t, alloc.validate(now, &conf))
}

func TestStorageAllocation_filterBlobbers(t *testing.T) {

	var (
		alloc StorageAllocation
		list  []*StorageNode
		now   common.Timestamp = 150
		size  int64            = 230
	)

	list = []*StorageNode{
		&StorageNode{Terms: Terms{
			MaxOfferDuration: 8 * time.Second,
		}},
		&StorageNode{Terms: Terms{
			MaxOfferDuration: 6 * time.Second,
		}},
	}

	// 1. filter all by max offer duration

	alloc.Expiration = now + 10 // one second duration
	assert.Len(t, alloc.filterBlobbers(list, now, size), 0)

	// 2. filter all by read price range
	alloc.Expiration = now + 5
	alloc.ReadPriceRange = PriceRange{Min: 10, Max: 40}

	list[0].Terms.ReadPrice = 100
	list[1].Terms.ReadPrice = 150
	assert.Len(t, alloc.filterBlobbers(list, now, size), 0)

	// 3. filter all by write price range
	alloc.ReadPriceRange = PriceRange{Min: 10, Max: 200}

	alloc.WritePriceRange = PriceRange{Min: 10, Max: 40}
	list[0].Terms.WritePrice = 100
	list[1].Terms.WritePrice = 150
	assert.Len(t, alloc.filterBlobbers(list, now, size), 0)

	// 4. filter all by size
	alloc.WritePriceRange = PriceRange{Min: 10, Max: 200}
	list[0].Capacity, list[0].Allocated = 100, 90
	list[1].Capacity, list[1].Allocated = 100, 50
	assert.Len(t, alloc.filterBlobbers(list, now, size), 0)

	// accept one
	list[0].Capacity, list[0].Used = 330, 100
	assert.Len(t, alloc.filterBlobbers(list, now, size), 1)

	// accept all
	list[1].Capacity, list[1].Allocated = 330, 100
	assert.Len(t, alloc.filterBlobbers(list, now, size), 2)
}
