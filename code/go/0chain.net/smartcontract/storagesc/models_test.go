package storagesc

import (
	"testing"
	"time"

	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	list[0].Capacity, list[0].Allocated = 330, 100
	assert.Len(t, alloc.filterBlobbers(list, now, size), 1)

	// accept all
	list[1].Capacity, list[1].Allocated = 330, 100
	assert.Len(t, alloc.filterBlobbers(list, now, size), 2)
}

func TestVerifyClientID(t *testing.T) {
	type input struct {
		name     string
		clientID string
		pk       string
		wantErr  bool
	}

	tests := []input{
		{
			name:     "Client id verification ok #1",
			clientID: "7c7a336dc27306b76044e9492402e840768a8abb4347a2c5eef0afdf26d0350f",
			pk: "563351287055d5d1b37cc36ce2c46e5a7c9ccc7325db44df01ef63713bd53013a673" +
				"4b2ebb76f97366ad42a1f06abeef4376f67a37814b5a5aa64584e882a112",
		},
		{
			name:     "Client id verification ok #2",
			clientID: "f07d2f9551cfc4d2e45411106732127e7d4f3ee5b068bb8a511f7d6d288a5dba",
			pk: "a05f35f954e7a96e36c6ea970225c98561434da80c2c7e2a35a213fa7a38310c" +
				"53df6f04284e30a54b99e6090d7f892b869e3e166e0194cc5d4779ac2cab5810",
		},
		{
			name:     "Client id verification ok #3",
			clientID: "5b3ac53beeb2a6e678395683fb7fa04ffb94104c3829207364badfb632134dd7",
			pk: "ff9a3ce40caceacbd937be3318d0cb1599781d11e4f202050c15646332395401" +
				"2946f0287c181ec3068a6611f0db40bd6f95b02679d3933c9a10c26a355a238a",
		},
		{
			name:    "Invalid public key",
			pk:      "invalid public key",
			wantErr: true,
		},
		{
			name:     "Invalid client id",
			clientID: "5b3ac53beeb2a6e678395683fb7fa04ffb94104c3829207364badfb632134dd7",
			pk: "a05f35f954e7a96e36c6ea970225c98561434da80c2c7e2a35a213fa7a38310c" +
				"53df6f04284e30a54b99e6090d7f892b869e3e166e0194cc5d4779ac2cab5810",
			wantErr: true,
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			rm := ReadMarker{
				ClientID:        test.clientID,
				ClientPublicKey: test.pk,
			}

			err := rm.VerifyClientID()
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
