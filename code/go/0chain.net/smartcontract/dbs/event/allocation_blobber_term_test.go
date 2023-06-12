package event

import (
	"fmt"
	"strconv"
	"testing"

	"0chain.net/core/encryption"
	common2 "0chain.net/smartcontract/common"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestAllocationBlobberTerms(t *testing.T) {
	t.Run("test edb.addOrOverwriteAllocationBlobberTerms", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		// Create owner and allocation
		err := eventDb.Get().Model(&User{}).Create(&User{
			UserID: OwnerId,
		}).Error
		require.NoError(t, err, "owner couldn't be created")

		allocId := createMockAllocations(t, eventDb, 1)[0]
		blobber1Id := encryption.Hash("mockBlobber_" + strconv.Itoa(0))
		blobber2Id := encryption.Hash("mockBlobber_" + strconv.Itoa(1))

		// Insert the blobbers
		err = eventDb.Get().Model(&Blobber{}).Create(&Blobber{
			Provider: Provider{
				ID: blobber1Id,
			},
			BaseURL: "http://localhost:3131",
		}).Error
		require.NoError(t, err, "blobber couldn't be created")

		err = eventDb.Get().Model(&Blobber{}).Create(&Blobber{
			Provider: Provider{
				ID: blobber2Id,
			},
			BaseURL: "http://localhost:3232",
		}).Error
		require.NoError(t, err, "blobber couldn't be created")

		terms := []AllocationBlobberTerm{
			{
				AllocationID:  allocId,
				BlobberID:     blobber1Id,
				ReadPrice:     int64(currency.Coin(29)),
				WritePrice:    int64(currency.Coin(31)),
				MinLockDemand: 37.0,
			},
			{
				AllocationID:  allocId,
				BlobberID:     blobber2Id,
				ReadPrice:     int64(currency.Coin(41)),
				WritePrice:    int64(currency.Coin(43)),
				MinLockDemand: 47.0,
			},
		}

		err = eventDb.addOrOverwriteAllocationBlobberTerms(terms)
		require.NoError(t, err, "Error while inserting Allocation's Blobber's AllocationBlobberTerm to event database")

		var term *AllocationBlobberTerm
		var res []AllocationBlobberTerm
		limit := common2.Pagination{
			Offset:       0,
			Limit:        20,
			IsDescending: true,
		}
		res, err = eventDb.GetAllocationBlobberTerms(terms[0].AllocationID, limit)
		require.Equal(t, 2, len(res), "AllocationBlobberTerm not getting inserted")

		terms[1].MinLockDemand = 70.0
		err = eventDb.addOrOverwriteAllocationBlobberTerms(terms)
		require.NoError(t, err, "Error while inserting Allocation's Blobber's AllocationBlobberTerm to event database")

		term, err = eventDb.GetAllocationBlobberTerm(terms[1].AllocationID, terms[1].BlobberID)
		require.Equal(t, terms[1].MinLockDemand, term.MinLockDemand, "Error while overriding AllocationBlobberTerm in event Database")
	})

	t.Run("test edb.updateAllocationBlobberTerms", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		// Create owner and allocation
		err := eventDb.Get().Model(&User{}).Create(&User{
			UserID: OwnerId,
		}).Error
		require.NoError(t, err, "owner couldn't be created")

		allocId := createMockAllocations(t, eventDb, 1)[0]
		blobber1Id := encryption.Hash("mockBlobber_" + strconv.Itoa(0))
		blobber2Id := encryption.Hash("mockBlobber_" + strconv.Itoa(1))

		// Insert the blobbers
		err = eventDb.Get().Model(&Blobber{}).Create(&Blobber{
			Provider: Provider{
				ID: blobber1Id,
			},
			BaseURL: "http://localhost:3131",
		}).Error
		require.NoError(t, err, "blobber couldn't be created")

		err = eventDb.Get().Model(&Blobber{}).Create(&Blobber{
			Provider: Provider{
				ID: blobber2Id,
			},
			BaseURL: "http://localhost:3232",
		}).Error
		require.NoError(t, err, "blobber couldn't be created")
		
		terms := []AllocationBlobberTerm{
			{
				AllocationID:  allocId,
				BlobberID:     blobber1Id,
				ReadPrice:     int64(currency.Coin(29)),
				WritePrice:    int64(currency.Coin(31)),
				MinLockDemand: 37.0,
			},
			{
				AllocationID:  allocId,
				BlobberID:     blobber2Id,
				ReadPrice:     int64(currency.Coin(41)),
				WritePrice:    int64(currency.Coin(43)),
				MinLockDemand: 47.0,
			},
		}

		err = eventDb.addOrOverwriteAllocationBlobberTerms(terms)
		require.NoError(t, err, "Error while inserting Allocation's Blobber's AllocationBlobberTerm to event database")

		err = eventDb.updateAllocationBlobberTerms([]AllocationBlobberTerm{
			{
				AllocationID:  allocId,
				BlobberID:     blobber1Id,
				ReadPrice:     int64(currency.Coin(59)),
				WritePrice:    int64(currency.Coin(61)),
				MinLockDemand: 57.0,
			}, {
				AllocationID:  allocId,
				BlobberID:     blobber2Id,
				ReadPrice:     int64(currency.Coin(61)),
				WritePrice:    int64(currency.Coin(63)),
				MinLockDemand: 67.0,
			},
		})
		require.NoError(t, err, "Error while updating Allocation's Blobber's AllocationBlobberTerm to event database")

		term, err := eventDb.GetAllocationBlobberTerm(allocId, blobber1Id)
		require.NoError(t, err, "Error while reading Allocation Blobber Terms")

		require.Equal(t, int64(currency.Coin(59)), term.ReadPrice)
		require.Equal(t, int64(currency.Coin(61)), term.WritePrice)
		require.Equal(t, float64(57.0), term.MinLockDemand)

		term, err = eventDb.GetAllocationBlobberTerm(allocId, blobber2Id)
		require.NoError(t, err, "Error while reading Allocation Blobber Terms")

		require.Equal(t, int64(currency.Coin(61)), term.ReadPrice)
		require.Equal(t, int64(currency.Coin(63)), term.WritePrice)
		require.Equal(t, float64(67.0), term.MinLockDemand)
	})
}

func TestEventDb_GetAllocationsByBlobberId(t *testing.T) {
	type args struct {
		blobberId string
		limit     common2.Pagination
	}

	edb, clean := GetTestEventDB(t)
	defer clean()

	// Add user for the allocation
	err := edb.Get().Model(&User{}).Create(&User{
		UserID: OwnerId,
	}).Error
	require.NoError(t, err, "owner couldn't be created")

	// Add 3 blobbers
	blobbers := []Blobber{
		{
			Provider: Provider{
				ID: encryption.Hash("mockBlobber_" + strconv.Itoa(0)),
			},
			BaseURL: "http://mockBlobber_0.com",
		},
		{
			Provider: Provider{
				ID: encryption.Hash("mockBlobber_" + strconv.Itoa(1)),
			},
			BaseURL: "http://mockBlobber_1.com",
		},
		{
			Provider: Provider{
				ID: encryption.Hash("mockBlobber_" + strconv.Itoa(2)),
			},
			BaseURL: "http://mockBlobber_2.com",
		},
	}
	err = edb.Get().Model(&Blobber{}).Create(&blobbers).Error
	require.NoError(t, err, "Error while inserting blobbers to event database")

	// Add 5 allocations
	allocationIds := createMockAllocations(t, edb, 5)

	for i, alloc := range allocationIds {
		fmt.Printf("Allocation %d: %s\n", i, alloc)
	}

	// Add 10 allocation blobber terms, B1 => A1, A3, A5, B2 => A2, A3, A5, B3 => A1, A2, A4, A5
	terms := []AllocationBlobberTerm{
		mockAllocationBlobberTerm(allocationIds[0], blobbers[0].ID),
		mockAllocationBlobberTerm(allocationIds[2], blobbers[0].ID),
		mockAllocationBlobberTerm(allocationIds[4], blobbers[0].ID),
		mockAllocationBlobberTerm(allocationIds[1], blobbers[1].ID),
		mockAllocationBlobberTerm(allocationIds[2], blobbers[1].ID),
		mockAllocationBlobberTerm(allocationIds[4], blobbers[1].ID),
		mockAllocationBlobberTerm(allocationIds[0], blobbers[2].ID),
		mockAllocationBlobberTerm(allocationIds[1], blobbers[2].ID),
		mockAllocationBlobberTerm(allocationIds[3], blobbers[2].ID),
		mockAllocationBlobberTerm(allocationIds[4], blobbers[2].ID),
	}
	err = edb.Get().Model(&AllocationBlobberTerm{}).Create(&terms).Error
	require.NoError(t, err, "Error while inserting Allocation's Blobber's AllocationBlobberTerm to event database")

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "get allocations by blobber id, ascending",
			args: args{
				blobberId: blobbers[0].ID,
				limit:    common2.Pagination{
					Offset: 0,
					Limit:  10,
					IsDescending: false,
				},
			},
			want: []string{allocationIds[0], allocationIds[2], allocationIds[4]},
			wantErr: false,
		},
		{
			name: "get allocations by blobber id, with pagination descending",
			args: args{
				blobberId: blobbers[2].ID,
				limit:    common2.Pagination{
					Offset: 1,
					Limit:  2,
					IsDescending: true,
				},
			},
			want: []string{allocationIds[3], allocationIds[1]},
			wantErr: false,
		},
		{
			name: "get allocations by blobber id, with pagination ascending",
			args: args{
				blobberId: blobbers[2].ID,
				limit:    common2.Pagination{
					Offset: 2,
					Limit:  2,
					IsDescending: false,
				},
			},
			want: []string{allocationIds[3], allocationIds[4]},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := edb.GetAllocationsByBlobberId(tt.args.blobberId, tt.args.limit)
			require.NoError(t, err, "Error while reading Allocation Blobber Terms")
			require.Equal(t, len(tt.want), len(got))
			gotIds := make([]string, 0, len(got))
			for _, a := range got {
				gotIds = append(gotIds, a.AllocationID)
			}
			require.Equal(t, tt.want, gotIds)
			if (err != nil) != tt.wantErr {
				t.Errorf("EventDb.GetAllocationsByBlobberId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func mockAllocationBlobberTerm(allocationId string, blobberId string) AllocationBlobberTerm {
	return AllocationBlobberTerm{
		AllocationID:  allocationId,
		BlobberID:     blobberId,
		ReadPrice:     int64(currency.Coin(41)),
		WritePrice:    int64(currency.Coin(43)),
		MinLockDemand: 47.0,
	}
}