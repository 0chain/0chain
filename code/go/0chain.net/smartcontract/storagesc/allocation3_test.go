package storagesc

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"0chain.net/core/util"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSelectBlobbers(t *testing.T) {
	const (
		randomSeed           = 1
		mockURL              = "mock_url"
		mockOwner            = "mock owner"
		mockPublicKey        = "mock public key"
		mockBlobberId        = "mock_blobber_id"
		mockPoolId           = "mock pool id"
		mockMinPrice         = 0
		confTimeUnit         = 720 * time.Hour
		confMinAllocSize     = 1024
		confMinAllocDuration = 5 * time.Minute
		mockMaxOffDuration   = 744 * time.Hour
	)
	var mockStatke = zcnToBalance(100)
	var mockBlobberCapacity int64 = 1000 * confMinAllocSize
	var mockMaxPrice = zcnToBalance(100.0)
	var mockReadPrice = zcnToBalance(0.01)
	var mockWritePrice = zcnToBalance(0.10)
	var now = common.Timestamp(1000000)

	type args struct {
		diverseBlobbers      bool
		numBlobbers          int
		numPreferredBlobbers int
		dataShards           int
		parityShards         int
		allocSize            int64
		expiration           common.Timestamp
	}
	type want struct {
		blobberIds []int
		err        bool
		errMsg     string
	}

	makeMockBlobber := func(index int) *StorageNode {
		return &StorageNode{
			ID:              mockBlobberId + strconv.Itoa(index),
			BaseURL:         mockURL + strconv.Itoa(index),
			Capacity:        mockBlobberCapacity,
			LastHealthCheck: now - blobberHealthTime + 1,
			Terms: Terms{
				ReadPrice:        mockReadPrice,
				WritePrice:       mockWritePrice,
				MaxOfferDuration: mockMaxOffDuration,
			},
		}
	}

	setup := func(
		t *testing.T, args args,
	) (StorageSmartContract, StorageAllocation, StorageNodes, chainState.StateContextI) {
		var balances = &mocks.StateContextI{}
		var ssc = StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		var sa = StorageAllocation{
			PreferredBlobbers: []string{},
			DataShards:        args.dataShards,
			ParityShards:      args.parityShards,
			Owner:             mockOwner,
			OwnerPublicKey:    mockPublicKey,
			Expiration:        args.expiration,
			Size:              args.allocSize,
			ReadPriceRange:    PriceRange{mockMinPrice, mockMaxPrice},
			WritePriceRange:   PriceRange{mockMinPrice, mockMaxPrice},
			DiverseBlobbers:   args.diverseBlobbers,
		}
		for i := 0; i < args.numPreferredBlobbers; i++ {
			sa.PreferredBlobbers = append(sa.PreferredBlobbers, mockURL+strconv.Itoa(i))
		}
		var sNodes = StorageNodes{}
		for i := 0; i < args.numBlobbers; i++ {
			sNodes.Nodes.add(makeMockBlobber(i))
			sp := stakePool{
				Pools: map[string]*delegatePool{
					mockPoolId: {},
				},
			}
			sp.Pools[mockPoolId].Balance = mockStatke
			balances.On(
				"GetTrieNode",
				stakePoolKey(ssc.ID, mockBlobberId+strconv.Itoa(i)),
			).Return(&sp, nil).Once()
		}

		var conf = &scConfig{
			TimeUnit:         confTimeUnit,
			MinAllocSize:     confMinAllocSize,
			MinAllocDuration: confMinAllocDuration,
		}
		balances.On("GetTrieNode", scConfigKey(ssc.ID)).Return(conf, nil).Once()

		return ssc, sa, sNodes, balances
	}

	testCases := []struct {
		name string
		args args
		want want
	}{
		{
			name: "test_diverse_blobbers",
			args: args{
				diverseBlobbers:      true,
				numBlobbers:          6,
				numPreferredBlobbers: 2,
				dataShards:           5,
				allocSize:            confMinAllocSize,
				expiration:           common.Timestamp(common.ToTime(now).Add(confMinAllocDuration).Unix()),
			},
			want: want{
				blobberIds: []int{0, 1, 2, 3, 4},
			},
		},
		{
			name: "test_randomised_blobbers",
			args: args{
				diverseBlobbers:      false,
				numBlobbers:          6,
				numPreferredBlobbers: 2,
				dataShards:           5,
				allocSize:            confMinAllocSize,
				expiration:           common.Timestamp(common.ToTime(now).Add(confMinAllocDuration).Unix()),
			},
			want: want{
				blobberIds: []int{0, 1, 5, 3, 2},
			},
		},
		{
			name: "test_excess_preferred_blobbers",
			args: args{
				diverseBlobbers:      false,
				numBlobbers:          6,
				numPreferredBlobbers: 8,
				dataShards:           5,
				allocSize:            confMinAllocSize,
				expiration:           common.Timestamp(common.ToTime(now).Add(confMinAllocDuration).Unix()),
			},
			want: want{
				err:    true,
				errMsg: "allocation_creation_failed: invalid preferred blobber URL",
			},
		},
		{
			name: "test_all_preferred_blobbers",
			args: args{
				diverseBlobbers:      false,
				numBlobbers:          6,
				numPreferredBlobbers: 6,
				dataShards:           4,
				allocSize:            confMinAllocSize,
				expiration:           common.Timestamp(common.ToTime(now).Add(confMinAllocDuration).Unix()),
			},
			want: want{
				blobberIds: []int{0, 1, 2, 3},
			},
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ssc, sa, blobbers, balances := setup(t, tt.args)

			outBlobbers, outSize, err := ssc.selectBlobbers(
				now, blobbers, &sa, randomSeed, balances,
			)

			require.EqualValues(t, len(tt.want.blobberIds), len(outBlobbers))
			require.EqualValues(t, tt.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errMsg, err.Error())
				return
			}

			size := int64(sa.DataShards + sa.ParityShards)
			require.EqualValues(t, int64(sa.Size+size-1)/size, outSize)

			for _, blobber := range outBlobbers {
				found := false
				for _, index := range tt.want.blobberIds {
					if mockBlobberId+strconv.Itoa(index) == blobber.ID {
						require.EqualValues(t, makeMockBlobber(index), blobber)
						found = true
						break
					}
				}
				require.True(t, found)
			}
		})
	}
}

func TestExtendAllocation(t *testing.T) {
	const (
		randomSeed                  = 1
		mockURL                     = "mock_url"
		mockOwner                   = "mock owner"
		mockNotTheOwner             = "mock not the owner"
		mockWpOwner                 = "mock write pool owner"
		mockPublicKey               = "mock public key"
		mockBlobberId               = "mock_blobber_id"
		mockPoolId                  = "mock pool id"
		mockAllocationId            = "mock allocation id"
		mockMinPrice                = 0
		confTimeUnit                = 720 * time.Hour
		confMinAllocSize            = 1024
		confMinAllocDuration        = 5 * time.Minute
		mockMaxOffDuration          = 744 * time.Hour
		mocksSize                   = 10000000000
		mockDataShards              = 2
		mockParityShards            = 2
		mockNumAllBlobbers          = 2 + mockDataShards + mockParityShards
		mockExpiration              = common.Timestamp(17000)
		mockStake                   = 3
		mockChallengeCompletionTime = 1 * time.Hour
		mockMinLockDemmand          = 0.1
		mockTimeUnit                = 1 * time.Hour
		mockBlobberBalance          = 11
		mockHash                    = "mock hash"
	)
	var mockBlobberCapacity int64 = 3700000000 * confMinAllocSize
	var mockMaxPrice = zcnToBalance(100.0)
	var mockReadPrice = zcnToBalance(0.01)
	var mockWritePrice = zcnToBalance(0.10)
	var now = common.Timestamp(1000000)

	type args struct {
		request    updateAllocationRequest
		expiration common.Timestamp
		value      float64
		poolFunds  []float64
		poolCount  []int
	}
	type want struct {
		blobberIds []int
		err        bool
		errMsg     string
	}

	makeMockBlobber := func(index int) *StorageNode {
		return &StorageNode{
			ID:              mockBlobberId + strconv.Itoa(index),
			BaseURL:         mockURL + strconv.Itoa(index),
			Capacity:        mockBlobberCapacity,
			LastHealthCheck: now - blobberHealthTime + 1,
			Terms: Terms{
				ReadPrice:        mockReadPrice,
				WritePrice:       mockWritePrice,
				MaxOfferDuration: mockMaxOffDuration,
			},
		}
	}

	setup := func(
		t *testing.T, args args,
	) (
		StorageSmartContract,
		transaction.Transaction,
		StorageAllocation,
		[]*StorageNode,
		chainState.StateContextI,
	) {
		var balances = &mocks.StateContextI{}
		var ssc = StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		var txn = transaction.Transaction{
			ClientID:     mockOwner,
			ToClientID:   ADDRESS,
			CreationDate: now,
			Value:        zcnToInt64(args.value),
		}
		txn.Hash = mockHash
		if txn.Value > 0 {
			balances.On(
				"GetClientBalance", txn.ClientID,
			).Return(state.Balance(txn.Value+1), nil).Once()
			balances.On(
				"AddTransfer", &state.Transfer{
					ClientID:   txn.ClientID,
					ToClientID: txn.ToClientID,
					Amount:     state.Balance(txn.Value),
				},
			).Return(nil).Once()
		}

		var sa = StorageAllocation{
			ID:                      mockAllocationId,
			DataShards:              mockDataShards,
			ParityShards:            mockParityShards,
			Owner:                   mockOwner,
			OwnerPublicKey:          mockPublicKey,
			Expiration:              now + mockExpiration,
			Size:                    mocksSize,
			ReadPriceRange:          PriceRange{mockMinPrice, mockMaxPrice},
			WritePriceRange:         PriceRange{mockMinPrice, mockMaxPrice},
			ChallengeCompletionTime: mockChallengeCompletionTime,
			TimeUnit:                mockTimeUnit,
		}
		require.True(t, len(args.poolFunds) > 0)

		bCount := sa.DataShards + sa.ParityShards
		var blobbers []*StorageNode
		for i := 0; i < mockNumAllBlobbers; i++ {
			mockBlobber := makeMockBlobber(i)

			if i < sa.DataShards+sa.ParityShards {
				blobbers = append(blobbers, mockBlobber)
				sa.BlobberDetails = append(sa.BlobberDetails, &BlobberAllocation{
					BlobberID:     mockBlobber.ID,
					MinLockDemand: zcnToBalance(mockMinLockDemmand),
					Terms: Terms{
						ChallengeCompletionTime: mockChallengeCompletionTime,
						WritePrice:              mockWritePrice,
					},
					Stats: &StorageAllocationStats{
						UsedSize: sa.Size / int64(bCount),
					},
				})
				sp := stakePool{
					Pools: map[string]*delegatePool{
						mockPoolId: {},
					},
					Offers: map[string]*offerPool{
						mockAllocationId: {},
					},
				}
				sp.Pools[mockPoolId].Balance = zcnToBalance(mockStake)
				balances.On(
					"GetTrieNode", stakePoolKey(ssc.ID, mockBlobber.ID),
				).Return(&sp, nil).Once()
				balances.On(
					"InsertTrieNode",
					stakePoolKey(ssc.ID, mockBlobber.ID),
					mock.Anything,
				).Return("", nil).Once()
			}
		}

		require.EqualValues(t, len(args.poolFunds), len(args.poolCount))
		for i, funds := range args.poolFunds {
			var wp writePool
			for j := 0; j < args.poolCount[i]; j++ {
				expiresAt := sa.Expiration + toSeconds(sa.ChallengeCompletionTime) + args.request.Expiration + 1
				ap := allocationPool{
					AllocationID: sa.ID,
					ExpireAt:     expiresAt,
				}
				ap.Balance = zcnToBalance(funds)
				for _, blobber := range blobbers {
					ap.Blobbers.add(&blobberPool{
						BlobberID: blobber.ID,
						Balance:   ap.Balance / state.Balance(bCount*args.poolCount[i]),
					})
				}
				wp.Pools.add(&ap)
			}

			if i == 0 {
				sa.addWritePoolOwner(sa.Owner)
				balances.On(
					"GetTrieNode", writePoolKey(ssc.ID, sa.Owner),
				).Return(&wp, nil).Once()
				balances.On(
					"InsertTrieNode", writePoolKey(ssc.ID, sa.Owner), mock.Anything,
				).Return("", nil).Once()
			} else {
				wpOwner := mockWpOwner + strconv.Itoa(i)
				sa.addWritePoolOwner(wpOwner)
				balances.On(
					"GetTrieNode", writePoolKey(ssc.ID, wpOwner),
				).Return(&wp, nil).Once()
				balances.On(
					"InsertTrieNode", writePoolKey(ssc.ID, wpOwner), mock.Anything,
				).Return("", nil).Once()
			}
		}

		balances.On(
			"GetTrieNode", challengePoolKey(ssc.ID, sa.ID),
		).Return(newChallengePool(), nil).Once()
		balances.On(
			"InsertTrieNode", challengePoolKey(ssc.ID, sa.ID),
			mock.MatchedBy(func(cp *challengePool) bool {
				var size int64
				for _, blobber := range sa.BlobberDetails {
					size += blobber.Stats.UsedSize
				}
				newFunds := sizeInGB(size) *
					float64(mockWritePrice) *
					float64(sa.durationInTimeUnits(args.request.Expiration))
				return cp.Balance/10 == state.Balance(newFunds/10) // ignore type cast errors
			}),
		).Return("", nil).Once()

		return ssc, txn, sa, blobbers, balances
	}

	testCases := []struct {
		name string
		args args
		want want
	}{
		{
			name: "ok_multiple_users",
			args: args{
				request: updateAllocationRequest{
					ID:           mockAllocationId,
					OwnerID:      mockOwner,
					Size:         zcnToInt64(31),
					Expiration:   7000,
					SetImmutable: false,
				},
				expiration: mockExpiration,
				value:      0.1,
				poolFunds:  []float64{0.0, 5.0, 5.0},
				poolCount:  []int{1, 3, 4},
			},
		},
		{
			name: "ok_multiple_allocation_pools",
			args: args{
				request: updateAllocationRequest{
					ID:           mockAllocationId,
					OwnerID:      mockOwner,
					Size:         zcnToInt64(31),
					Expiration:   7000,
					SetImmutable: false,
				},
				expiration: mockExpiration,
				value:      0.1,
				poolFunds:  []float64{7},
				poolCount:  []int{5},
			},
		},
		{
			name: "ok_multiple_users",
			args: args{
				request: updateAllocationRequest{
					ID:           mockAllocationId,
					OwnerID:      mockOwner,
					Size:         zcnToInt64(31),
					Expiration:   7000,
					SetImmutable: false,
				},
				expiration: mockExpiration,
				value:      0.1,
				poolFunds:  []float64{0.0, 0.0},
				poolCount:  []int{1, 3},
			},
			want: want{
				err:    true,
				errMsg: "allocation_extending_failed: not enough tokens in write pool to extend allocation",
			},
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ssc, txn, sa, aBlobbers, balances := setup(t, tt.args)

			err := ssc.extendAllocation(
				&txn,
				&sa,
				aBlobbers,
				&tt.args.request,
				false,
				balances,
			)
			require.EqualValues(t, tt.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errMsg, err.Error())
			} else {
				mock.AssertExpectationsForObjects(t, balances)
			}
		})
	}
}

func TestTransferAllocation(t *testing.T) {
	t.Parallel()
	const (
		mockNewOwnerId        = "mock new owner id"
		mockNewOwnerPublicKey = "mock new owner public key"
		mockOldOwner          = "mock old owner"
		mockCuratorId         = "mock curator id"
		mockAllocationId      = "mock allocation id"
		mockNotAllocationId   = "mock not allocation id"
	)
	type args struct {
		ssc      *StorageSmartContract
		txn      *transaction.Transaction
		input    []byte
		balances chainState.StateContextI
	}
	type parameters struct {
		curator                 string
		info                    transferAllocationInput
		existingCurators        []string
		existingWPForAllocation bool
		existingNoiseWPools     int
	}
	type want struct {
		err    bool
		errMsg string
	}
	var setExpectations = func(t *testing.T, name string, p parameters, want want) args {
		var balances = &mocks.StateContextI{}
		var txn = &transaction.Transaction{
			ClientID: p.curator,
		}
		var ssc = &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		input, err := json.Marshal(p.info)
		require.NoError(t, err)

		var sa = StorageAllocation{
			Owner: mockOldOwner,
			ID:    p.info.AllocationId,
		}
		for _, curator := range p.existingCurators {
			sa.Curators = append(sa.Curators, curator)
		}
		balances.On("GetTrieNode", sa.GetKey(ssc.ID)).Return(&sa, nil).Once()

		var wp writePool
		if p.existingWPForAllocation || p.existingNoiseWPools > 0 {
			for i := 0; i < p.existingNoiseWPools; i++ {
				wp.Pools.add(&allocationPool{AllocationID: mockNotAllocationId + strconv.Itoa(i)})
			}
			if p.existingWPForAllocation {
				wp.Pools.add(&allocationPool{AllocationID: p.info.AllocationId})
			}
			balances.On("GetTrieNode", writePoolKey(ssc.ID, p.info.NewOwnerId)).Return(&wp, nil).Twice()
		} else {
			balances.On("GetTrieNode", writePoolKey(ssc.ID, p.info.NewOwnerId)).Return(
				nil, util.ErrValueNotPresent).Twice()
		}

		balances.On(
			"InsertTrieNode",
			writePoolKey(ssc.ID, p.info.NewOwnerId),
			mock.MatchedBy(func(wp *writePool) bool {
				if p.existingNoiseWPools+1 != len(wp.Pools) {
					return false
				}
				_, ok := wp.Pools.get(p.info.AllocationId)
				return ok

			})).Return("", nil).Once()

		balances.On(
			"InsertTrieNode",
			sa.GetKey(ssc.ID),
			mock.MatchedBy(func(sa *StorageAllocation) bool {
				for i, curator := range p.existingCurators {
					if sa.Curators[i] != curator {
						return false
					}
				}
				return sa.ID == p.info.AllocationId &&
					sa.Owner == p.info.NewOwnerId &&
					sa.OwnerPublicKey == p.info.NewOwnerPublicKey
			})).Return("", nil).Once()

		var oldClientAlloc = ClientAllocation{
			ClientID: mockOldOwner,
			Allocations: &Allocations{
				List: sortedList{},
			},
		}
		oldClientAlloc.Allocations.List.add(p.info.AllocationId)
		balances.On(
			"GetTrieNode", oldClientAlloc.GetKey(ssc.ID),
		).Return(&oldClientAlloc, nil).Once()
		balances.On(
			"InsertTrieNode",
			oldClientAlloc.GetKey(ssc.ID),
			mock.MatchedBy(func(ca *ClientAllocation) bool {
				return ca.ClientID == mockOldOwner &&
					len(ca.Allocations.List) == 0
			})).Return("", nil).Once()

		var newClientAlloc = ClientAllocation{
			ClientID:    p.info.NewOwnerId,
			Allocations: &Allocations{},
		}
		balances.On(
			"GetTrieNode", newClientAlloc.GetKey(ssc.ID),
		).Return(&newClientAlloc, nil).Once()
		balances.On(
			"InsertTrieNode",
			newClientAlloc.GetKey(ssc.ID),
			mock.MatchedBy(func(ca *ClientAllocation) bool {
				_, ok := ca.Allocations.List.getIndex(p.info.AllocationId)
				return ca.ClientID == p.info.NewOwnerId &&
					len(ca.Allocations.List) == 1 && ok
			})).Return("", nil).Once()

		return args{ssc, txn, input, balances}
	}

	testCases := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok",
			parameters: parameters{
				curator: mockCuratorId,
				info: transferAllocationInput{
					AllocationId:      mockAllocationId,
					NewOwnerId:        mockNewOwnerId,
					NewOwnerPublicKey: mockNewOwnerPublicKey,
				},
				existingCurators:        []string{mockCuratorId, "another", "and another"},
				existingNoiseWPools:     3,
				existingWPForAllocation: false,
			},
		},
		{
			name: "ok",
			parameters: parameters{
				curator: mockCuratorId,
				info: transferAllocationInput{
					AllocationId:      mockAllocationId,
					NewOwnerId:        mockNewOwnerId,
					NewOwnerPublicKey: mockNewOwnerPublicKey,
				},
				existingCurators:        []string{mockCuratorId, "another", "and another"},
				existingNoiseWPools:     0,
				existingWPForAllocation: false,
			},
		},
		{
			name: "Err_not_curator",
			parameters: parameters{
				curator: mockCuratorId,
				info: transferAllocationInput{
					AllocationId:      mockAllocationId,
					NewOwnerId:        mockNewOwnerId,
					NewOwnerPublicKey: mockNewOwnerPublicKey,
				},
				existingCurators: []string{"not mock curator"},
			},
			want: want{
				err:    true,
				errMsg: "curator_transfer_allocation_failed: only curators can transfer allocations; mock curator id is not a curator",
			},
		},
	}
	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			args := setExpectations(t, test.name, test.parameters, test.want)

			resp, err := args.ssc.curatorTransferAllocation(args.txn, args.input, args.balances)

			require.EqualValues(t, test.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.errMsg, err.Error())
				return
			}
			require.EqualValues(t, args.txn.Hash, resp)
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}
