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
		now   common.Timestamp = 150
		alloc StorageAllocation
		conf  scConfig
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
