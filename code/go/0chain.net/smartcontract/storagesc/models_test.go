package storagesc

import (
	"testing"
	"time"

	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAllocationRequest_validate(t *testing.T) {

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
		nar  newAllocationRequest
		conf Config
	)

	conf.MinAllocSize = 10 * 1024
	conf.TimeUnit = 48 * time.Hour
	nar.DataShards = 1
	nar.ParityShards = 1
	nar.Blobbers = []string{"1", "2"}

	nar.ReadPriceRange = PriceRange{Min: 20, Max: 10}
	requireErrMsg(t, nar.validate(&conf), errMsg1)

	nar.ReadPriceRange = PriceRange{Min: 10, Max: 20}
	nar.WritePriceRange = PriceRange{Min: 20, Max: 10}
	requireErrMsg(t, nar.validate(&conf), errMsg2)

	nar.WritePriceRange = PriceRange{Min: 10, Max: 20}
	nar.Size = 5 * 1024
	requireErrMsg(t, nar.validate(&conf), errMsg3)

	nar.DataShards = 0
	requireErrMsg(t, nar.validate(&conf), errMsg5)
}

func TestStorageAllocation_filterBlobbers(t *testing.T) {

	var (
		alloc storageAllocationBase
		list  []*StorageNode
		now   common.Timestamp = 150
		size  int64            = 230
	)

	newEmptyStorageNode := func() *StorageNode {
		b := &StorageNode{}
		b.SetEntity(&storageNodeV3{})
		return b
	}

	list = []*StorageNode{
		newEmptyStorageNode(),
		newEmptyStorageNode(),
	}

	// 1. filter all by max offer duration

	alloc.Expiration = now + 10 // one second duration
	bs, err := alloc.filterBlobbers(list, now, size)
	require.NoError(t, err)
	assert.Len(t, bs, 0)

	// 2. filter all by read price range
	alloc.Expiration = now + 5
	alloc.ReadPriceRange = PriceRange{Min: 10, Max: 40}

	list[0].mustUpdateBase(func(b *storageNodeBase) error {
		b.Terms.ReadPrice = 100
		return nil
	})

	list[1].mustUpdateBase(func(b *storageNodeBase) error {
		b.Terms.ReadPrice = 150
		return nil
	})
	bs, err = alloc.filterBlobbers(list, now, size)
	require.NoError(t, err)
	assert.Len(t, bs, 0)

	// 3. filter all by write price range
	alloc.ReadPriceRange = PriceRange{Min: 10, Max: 200}

	alloc.WritePriceRange = PriceRange{Min: 10, Max: 40}
	list[0].mustUpdateBase(func(b *storageNodeBase) error {
		b.Terms.WritePrice = 100
		return nil
	})
	list[1].mustUpdateBase(func(snb *storageNodeBase) error {
		snb.Terms.WritePrice = 150
		return nil
	})
	bs, err = alloc.filterBlobbers(list, now, size)
	require.NoError(t, err)
	assert.Len(t, bs, 0)

	// 4. filter all by size
	alloc.WritePriceRange = PriceRange{Min: 10, Max: 200}
	list[0].mustUpdateBase(func(b *storageNodeBase) error {
		b.Capacity = 100
		b.Allocated = 90
		return nil
	})

	list[1].mustUpdateBase(func(b *storageNodeBase) error {
		b.Capacity = 100
		b.Allocated = 50
		return nil
	})
	bs, err = alloc.filterBlobbers(list, now, size)
	require.NoError(t, err)
	assert.Len(t, bs, 0)

	// accept one
	list[0].mustUpdateBase(func(b *storageNodeBase) error {
		b.Capacity = 330
		b.Allocated = 100
		return nil
	})
	bs, err = alloc.filterBlobbers(list, now, size)
	assert.Len(t, bs, 1)

	// accept all
	list[1].mustUpdateBase(func(b *storageNodeBase) error {
		b.Capacity = 330
		b.Allocated = 100
		return nil
	})
	bs, err = alloc.filterBlobbers(list, now, size)
	assert.Len(t, bs, 2)
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

//func TestDTU(t *testing.T) {
//	timeUnit := 720 * time.Hour
//	sa := StorageAllocation{}
//
//}
//
//func TestRDTU(t *testing.T) {
//
//	timeUnit := 720 * time.Hour
//	sa := StorageAllocation{}
//
//	t.Run("Allocation Expiry is less than now", func(t *testing.T) {
//		sa.Expiration = 50
//		rdtu, err := sa.restDurationInTimeUnits(100, timeUnit)
//		assert.Equal(t, int64(0), rdtu)
//		require.Error(t, err)
//		require.EqualValues(t, "rest duration time overflow, timestamp is beyond alloc expiration", err.Error())
//	})
//
//	t.Run("Allocation Expiry is greater than now", func(t *testing.T) {
//		sa.Expiration = 100
//		rdtu, err := sa.restDurationInTimeUnits(50, timeUnit)
//		assert.Equal(t, int64(1), rdtu)
//		require.NoError(t, err)
//	})
//}
