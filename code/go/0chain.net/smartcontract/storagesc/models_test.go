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

func TestStorageAllocation_filterBlobbers(t *testing.T) {

	var (
		alloc StorageAllocation
		list  []*StorageNode
		now   common.Timestamp = 150
		size  int64            = 230
	)

	list = []*StorageNode{
		&StorageNode{Terms: Terms{
			MaxOfferDuration:        8 * time.Second,
			ChallengeCompletionTime: 10 * time.Second,
		}},
		&StorageNode{Terms: Terms{
			MaxOfferDuration:        6 * time.Second,
			ChallengeCompletionTime: 20 * time.Second,
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
	list[0].Capacity, list[0].Used = 100, 90
	list[1].Capacity, list[1].Used = 100, 50
	assert.Len(t, alloc.filterBlobbers(list, now, size), 0)

	// 5. filter all by max CCT
	alloc.MaxChallengeCompletionTime = 5 * time.Second
	assert.Len(t, alloc.filterBlobbers(list, now, size), 0)

	// accept one
	alloc.MaxChallengeCompletionTime = 30 * time.Second
	list[0].Capacity, list[0].Used = 330, 100
	assert.Len(t, alloc.filterBlobbers(list, now, size), 1)

	// accept all
	list[1].Capacity, list[1].Used = 330, 100
	assert.Len(t, alloc.filterBlobbers(list, now, size), 2)
}

func TestStorageAllocation_diversifyBlobbers(t *testing.T) {
	type args struct {
		params []int
		size int
	}

	type want struct {
		params []int
	}

	var params = []*StorageNode{
		&StorageNode{Geolocation:
			StorageNodeGeolocation{Latitude: 37.773972, Longitude: -122.431297}}, // San Francisco
		&StorageNode{Geolocation:
			StorageNodeGeolocation{Latitude: -33.918861, Longitude: 18.423300}}, // Cape Town
		&StorageNode{Geolocation:
			StorageNodeGeolocation{Latitude: 59.937500, Longitude: 30.308611}}, // St Petersburg
		&StorageNode{Geolocation:
			StorageNodeGeolocation{Latitude: 22.633333, Longitude: 120.266670}}, // Kaohsiung City
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "OK optimal geo diversification (2 of 3)",
			args: args{
				params: []int{0, 1, 2},
				size: 2,
			},
			want: want{
				params: []int{0, 1},
			},
		},
		{
			name: "OK optimal geo diversification (3 of 4)",
			args: args{
				params: []int{0, 1, 2, 3},
				size: 3,
			},
			want: want{
				params: []int{0, 1, 3},
			},
		},
		{
			name: "OK optimal geo diversification not applied",
			args: args{
				params: []int{1},
				size: 2,
			},
			want: want{
				params: []int{1},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var sa = &StorageAllocation{
				DiverseBlobbers: len(tt.args.params) > len(tt.want.params),
			}

			var argsBlobbers []*StorageNode
			for _, v := range tt.args.params {
				argsBlobbers = append(argsBlobbers, params[v])
			}

			var wantBlobbers []*StorageNode
			for _, v := range tt.want.params {
				wantBlobbers = append(wantBlobbers, params[v])
			}

			got := sa.diversifyBlobbers(argsBlobbers, tt.args.size)
			assert.Equal(t, got, wantBlobbers)
		})
	}
}
