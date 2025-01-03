package storagesc

import (
	"0chain.net/core/util/entitywrapper"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"math"
	"strconv"
	"testing"
	"time"

	"0chain.net/chaincore/tokenpool"

	"0chain.net/smartcontract/provider"

	"0chain.net/chaincore/block"
	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/partitions"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/stakepool"

	"0chain.net/chaincore/chain/state/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"github.com/stretchr/testify/mock"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const blobberHealthTime = 60 * 60 // 1 Hour

func TestSelectBlobbers(t *testing.T) {
	const (
		randomSeed       = 1
		mockURL          = "mock_url"
		mockOwner        = "mock owner"
		mockPublicKey    = "mock public key"
		mockBlobberId    = "mock_blobber_id"
		mockPoolId       = "mock pool id"
		mockMinPrice     = 0
		confTimeUnit     = 720 * time.Hour
		confMinAllocSize = 800
	)
	var mockStatke = zcnToBalance(100)
	var mockBlobberCapacity int64 = 1000 * confMinAllocSize
	var mockMaxPrice = zcnToBalance(100.0)
	var mockReadPrice = zcnToBalance(0.01)
	var mockWritePrice = zcnToBalance(0.10)
	var now = time.Unix(1000000, 0)

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
		sn := &StorageNode{}
		sn.SetEntity(&storageNodeV4{
			Provider: provider.Provider{
				ID:              mockBlobberId + strconv.Itoa(index),
				ProviderType:    spenum.Blobber,
				LastHealthCheck: common.Timestamp(now.Unix()),
			},
			BaseURL:  mockURL + strconv.Itoa(index),
			Capacity: mockBlobberCapacity,
			Terms: Terms{
				ReadPrice:  mockReadPrice,
				WritePrice: mockWritePrice,
			},
		})
		return sn
	}

	setup := func(t *testing.T, args args) (
		StorageSmartContract, storageAllocationBase, StorageNodes, chainState.StateContextI) {
		var balances = &mocks.StateContextI{}
		var ssc = StorageSmartContract{

			SmartContract: sci.NewSC(ADDRESS),
		}
		var sa = storageAllocationBase{
			DataShards:      args.dataShards,
			ParityShards:    args.parityShards,
			Owner:           mockOwner,
			OwnerPublicKey:  mockPublicKey,
			Expiration:      args.expiration,
			Size:            args.allocSize,
			ReadPriceRange:  PriceRange{mockMinPrice, mockMaxPrice},
			WritePriceRange: PriceRange{mockMinPrice, mockMaxPrice},
			DiverseBlobbers: args.diverseBlobbers,
		}

		var sNodes = StorageNodes{}
		for i := 0; i < args.numBlobbers; i++ {
			sNodes.Nodes.add(makeMockBlobber(i))
			sp := stakePool{
				StakePool: &stakepool.StakePool{
					Pools: map[string]*stakepool.DelegatePool{
						mockPoolId: {},
					},
				},
			}
			sp.Pools[mockPoolId].Balance = mockStatke
			balances.On("GetTrieNode",
				stakePoolKey(spenum.Blobber, mockBlobberId+strconv.Itoa(i)),
				mock.MatchedBy(func(s *stakePool) bool {
					*s = sp
					return true
				})).Return(nil).Once()
		}

		var conf = &Config{
			TimeUnit:     confTimeUnit,
			MinAllocSize: confMinAllocSize,
			OwnerId:      owner,
		}
		balances.On("GetTrieNode", scConfigKey(ADDRESS), mock.MatchedBy(func(c *Config) bool {
			*c = *conf
			return true
		})).Return(nil).Once()

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
				expiration:           common.Timestamp(now.Add(confTimeUnit).Unix()),
			},
			want: want{
				blobberIds: []int{0, 1, 2, 3, 5},
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
				expiration:           common.Timestamp(now.Add(confTimeUnit).Unix()),
			},
			want: want{
				blobberIds: []int{0, 1, 5, 3, 2},
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
				expiration:           common.Timestamp(now.Add(confTimeUnit).Unix()),
			},
			want: want{
				blobberIds: []int{0, 1, 3, 5},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			//	t.Parallel()
			ssc, sa, blobbers, balances := setup(t, tt.args)

			outBlobbers, outSize, err := ssc.selectBlobbers(
				now, blobbers, &sa, randomSeed, balances,
			)
			for _, b := range outBlobbers {
				t.Log(b)
			}
			require.EqualValues(t, len(tt.want.blobberIds), len(outBlobbers))
			require.EqualValues(t, tt.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errMsg, err.Error())
				return
			}

			size := int64(sa.DataShards + sa.ParityShards)
			require.EqualValues(t, (sa.Size+size-1)/size, outSize)

			for _, blobber := range outBlobbers {
				bb := blobber.mustBase()
				t.Log(bb)
				found := false
				for _, index := range tt.want.blobberIds {
					if mockBlobberId+strconv.Itoa(index) == bb.ID {
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

func (sc *StorageSmartContract) selectBlobbers(
	creationDate time.Time,
	allBlobbersList StorageNodes,
	sa *storageAllocationBase,
	randomSeed int64,
	balances chainState.CommonStateContextI,
) ([]*StorageNode, int64, error) {
	var err error
	var conf *Config
	if conf, err = getConfig(balances); err != nil {
		return nil, 0, fmt.Errorf("can't get config: %v", err)
	}

	sa.TimeUnit = conf.TimeUnit // keep the initial time unit

	// number of blobbers required
	var size = sa.DataShards + sa.ParityShards
	// size of allocation for a blobber
	var bSize = sa.bSize()
	timestamp := common.Timestamp(creationDate.Unix())

	list, err := sa.filterBlobbers(allBlobbersList.Nodes.copy(), timestamp,
		bSize, filterHealthyBlobbers(timestamp),
		sc.filterBlobbersByFreeSpace(timestamp, bSize, balances))
	if err != nil {
		return nil, 0, fmt.Errorf("could not filter blobbers: %v", err)
	}

	if len(list) < size {
		return nil, 0, errors.New("not enough blobbers to honor the allocation")
	}

	sa.BlobberAllocs = make([]*BlobberAllocation, 0)
	sa.Stats = &StorageAllocationStats{}

	var blobberNodes []*StorageNode
	if len(sa.PreferredBlobbers) > 0 {
		blobberNodes, err = getPreferredBlobbers(sa.PreferredBlobbers, list)
		if err != nil {
			return nil, 0, common.NewError("allocation_creation_failed",
				err.Error())
		}
	}

	if len(blobberNodes) < size {
		blobberNodes = randomizeNodes(list, blobberNodes, size, randomSeed)
	}

	return blobberNodes[:size], bSize, nil
}

func filterHealthyBlobbers(now common.Timestamp) filterBlobberFunc {
	return filterBlobberFunc(func(b *StorageNode) (kick bool, err error) {
		return b.mustBase().LastHealthCheck <= (now - blobberHealthTime), nil
	})
}

func TestChangeBlobbers(t *testing.T) {

	const (
		confMinAllocSize    = 1024
		mockOwner           = "mock owner"
		mockHash            = "mock hash"
		mockAllocationID    = "mock_allocation_id"
		mockAllocationName  = "mock_allocation"
		mockPoolId          = "mock pool id"
		confTimeUnit        = 720 * time.Hour
		mockMaxOffDuration  = 744 * time.Hour
		mockBlobberCapacity = 200000000 * confMinAllocSize
		mockMinPrice        = 0
	)

	type args struct {
		numBlobbers                   int
		blobbersInAllocation          int
		blobberInChallenge            int
		blobbersAllocationInChallenge []int
		addBlobberID                  string
		removeBlobberID               string
		dataShards                    int
		parityShards                  int
	}
	type want struct {
		err                           bool
		errMsg                        string
		challengeEnabled              bool
		blobberInChallenge            int
		blobbersAllocationInChallenge []int
	}

	setup := func(arg args) (
		[]*StorageNode,
		string,
		string,
		*StorageSmartContract,
		*storageAllocationBase,
		common.Timestamp,
		chainState.StateContextI) {
		var (
			blobbers             []*StorageNode
			sc                   = newTestStorageSC()
			balances             = newTestBalances(t, false)
			now                  = common.Timestamp(1000000)
			mockAllocationExpiry = common.Timestamp(2000000)
			blobberAllocation    []*BlobberAllocation
			blobberMap           = make(map[string]*BlobberAllocation)
			mockState            = zcnToBalance(100)
			mockReadPrice        = zcnToBalance(0.01)
			mockWritePrice       = zcnToBalance(0.10)
			mockMaxPrice         = zcnToBalance(100.0)
			bcPart               *partitions.Partitions
			err                  error
		)

		var txn = transaction.Transaction{
			ClientID:     mockOwner,
			ToClientID:   ADDRESS,
			CreationDate: now,
		}
		txn.Hash = mockHash

		if arg.blobberInChallenge > 0 {
			bcPart, _, err = partitionsChallengeReadyBlobbers(balances)
			require.NoError(t, err)
			defer func() {
				err = bcPart.Save(balances)
				require.NoError(t, err)
			}()
		}

		for i := 0; i < arg.numBlobbers; i++ {
			ba := &BlobberAllocation{
				BlobberID:    "blobber_" + strconv.Itoa(i),
				Size:         mockBlobberCapacity,
				AllocationID: mockAllocationID,
				Terms: Terms{
					ReadPrice:  mockReadPrice,
					WritePrice: mockWritePrice,
				},
				Stats: &StorageAllocationStats{
					UsedSize:          mockBlobberCapacity / 2,
					SuccessChallenges: 100,
					FailedChallenges:  2,
					TotalChallenges:   102,
					OpenChallenges:    0,
				},
				LatestFinalizedChallCreatedAt: now - 200,
				ChallengePoolIntegralValue:    0,
			}
			if i < arg.blobberInChallenge {
				err := bcPart.Add(balances, &ChallengeReadyBlobber{BlobberID: ba.BlobberID})
				require.NoError(t, err)

				bcAllocations := arg.blobbersAllocationInChallenge[i]
				bcAllocPart, err := partitionsBlobberAllocations(ba.BlobberID, balances)
				require.NoError(t, err)

				for j := 0; j < bcAllocations; j++ {
					allocID := mockAllocationID
					if j > 0 {
						allocID += "_" + strconv.Itoa(j)
					}
					err := bcAllocPart.Add(balances, &BlobberAllocationNode{ID: allocID})
					require.NoError(t, err)
				}
				err = bcAllocPart.Save(balances)
				require.NoError(t, err)
			}
			if i < arg.blobbersInAllocation {
				blobberAllocation = append(blobberAllocation, ba)
				blobberMap[ba.BlobberID] = ba
			}

			blobber := &StorageNode{}
			blobber.SetEntity(&storageNodeV4{
				Provider: provider.Provider{
					ID:              ba.BlobberID,
					ProviderType:    spenum.Blobber,
					LastHealthCheck: now,
				},
				Capacity: mockBlobberCapacity,
				Terms: Terms{
					ReadPrice:  mockReadPrice,
					WritePrice: mockWritePrice,
				},
				NotAvailable: false,
				Allocated:    49268107,
				SavedData:    298934,
			})

			var id = strconv.Itoa(i)
			var sp = newStakePool()
			sp.Settings.ServiceChargeRatio = blobberYaml.serviceCharge
			sp.TotalOffers = currency.Coin(200000000000)
			var delegatePool = &stakepool.DelegatePool{}
			delegatePool.Balance = zcnToBalance(10000000000.0)
			delegatePool.DelegateID = encryption.Hash("delegate " + id)
			//delegatePool.MintAt = stake.MintAt
			sp.Pools["paula "+id] = delegatePool
			sp.Pools["paula "+id] = delegatePool
			sp.Settings.DelegateWallet = blobberId + " " + id + " wallet"
			require.NoError(t, sp.Save(spenum.Blobber, blobber.Id(), balances))

			_, err := balances.InsertTrieNode(blobber.GetKey(), blobber)
			require.NoError(t, err)
			blobbers = append(blobbers, blobber)
		}

		alloc := &storageAllocationBase{
			ID:               mockAllocationID,
			Owner:            mockOwner,
			BlobberAllocs:    blobberAllocation,
			BlobberAllocsMap: blobberMap,
			Size:             confMinAllocSize,
			Expiration:       mockAllocationExpiry,
			ReadPriceRange:   PriceRange{mockMinPrice, mockMaxPrice},
			WritePriceRange:  PriceRange{mockMinPrice, mockMaxPrice},
			DataShards:       arg.dataShards,
			ParityShards:     arg.parityShards,
			WritePool:        100000000000,
			Stats: &StorageAllocationStats{
				UsedSize:          int64(arg.dataShards+arg.parityShards) * mockBlobberCapacity / 2,
				SuccessChallenges: int64(arg.dataShards+arg.parityShards) * 100,
				FailedChallenges:  int64(arg.dataShards+arg.parityShards) * 2,
				TotalChallenges:   int64(arg.dataShards+arg.parityShards) * 102,
				OpenChallenges:    0,
			},
		}

		if len(arg.addBlobberID) > 0 {

			sp := stakePool{
				StakePool: &stakepool.StakePool{
					Pools: map[string]*stakepool.DelegatePool{
						mockPoolId: {},
					},
				},
			}
			sp.Pools[mockPoolId].Balance = mockState
			_, err := balances.InsertTrieNode(stakePoolKey(spenum.Blobber, arg.addBlobberID), &sp)
			require.NoError(t, err)
		}

		var cPool = challengePool{
			ZcnPool: &tokenpool.ZcnPool{
				TokenPool: tokenpool.TokenPool{
					ID:      alloc.ID,
					Balance: 100000000,
				},
			},
		}
		require.NoError(t, cPool.save(sc.ID, alloc, balances))

		return blobbers, arg.addBlobberID, arg.removeBlobberID, sc, alloc, now, balances

	}

	validate := func(want want, arg args, sa *storageAllocationBase, sc *StorageSmartContract, balances chainState.StateContextI) {
		totalBlobbers := arg.blobbersInAllocation
		if arg.addBlobberID != "" {
			totalBlobbers++
		}
		if arg.removeBlobberID != "" {
			totalBlobbers--
		}
		blobberNameMap := make(map[string]struct{}, totalBlobbers)
		require.Equal(t, totalBlobbers, len(sa.BlobberAllocs))

		for _, ba := range sa.BlobberAllocs {
			blobberNameMap[ba.BlobberID] = struct{}{}
		}

		if arg.addBlobberID != "" {
			_, ok := blobberNameMap[arg.addBlobberID]
			require.EqualValues(t, true, ok)
			if arg.removeBlobberID == "" {
				require.EqualValues(t, 1, sa.ParityShards)
			}
		}

		if arg.removeBlobberID != "" {
			_, ok := blobberNameMap[arg.removeBlobberID]
			require.EqualValues(t, false, ok)
		}

		if want.challengeEnabled {
			for i := 0; i < arg.blobberInChallenge; i++ {
				bcPart, _, err := partitionsChallengeReadyBlobbers(balances)
				require.NoError(t, err)

				bcSize, err := bcPart.Size(balances)
				require.NoError(t, err)
				require.EqualValues(t, want.blobberInChallenge, bcSize)

				blobberID := "blobber_" + strconv.Itoa(i)

				allocLoc, err := partitionsBlobberAllocations(blobberID, balances)
				require.NoError(t, err)

				size, err := allocLoc.Size(balances)
				require.NoError(t, err)
				require.EqualValues(t, want.blobbersAllocationInChallenge[i], size)
			}
		}
	}

	testCases := []struct {
		name string
		args args
		want want
	}{
		{
			name: "remove_blobber_doesnt_exist",
			args: args{
				numBlobbers:          6,
				blobbersInAllocation: 4,
				addBlobberID:         "blobber_5",
				removeBlobberID:      "blobber_non_existent",
				dataShards:           5,
			},
			want: want{
				err:    true,
				errMsg: "cannot find blobber blobber_non_existent in allocation",
			}},
		{
			name: "add_remove_valid_blobber_with_1_challenge_allocation",
			args: args{
				numBlobbers:                   6,
				blobbersInAllocation:          5,
				blobberInChallenge:            1,
				blobbersAllocationInChallenge: []int{1},
				addBlobberID:                  "blobber_5",
				removeBlobberID:               "blobber_0",
				dataShards:                    5,
			},
			want: want{
				err:                           false,
				challengeEnabled:              true,
				blobberInChallenge:            0,
				blobbersAllocationInChallenge: []int{0},
			}},
		{
			name: "add_remove_valid_blobber_with_more_than_1_challenge_allocation",
			args: args{
				numBlobbers:                   6,
				blobbersInAllocation:          5,
				blobberInChallenge:            1,
				blobbersAllocationInChallenge: []int{4},
				addBlobberID:                  "blobber_5",
				removeBlobberID:               "blobber_0",
				dataShards:                    5,
			},
			want: want{
				err:                           false,
				challengeEnabled:              true,
				blobberInChallenge:            1,
				blobbersAllocationInChallenge: []int{3},
			}},
		{
			name: "add_valid_blobber",
			args: args{
				numBlobbers:          6,
				blobbersInAllocation: 5,
				addBlobberID:         "blobber_5",
				dataShards:           5,
			},
			want: want{
				err: false,
			}},
		{
			name: "add_duplicate_blobber",
			args: args{
				numBlobbers:          6,
				blobbersInAllocation: 6,
				addBlobberID:         "blobber_1",
				dataShards:           5,
			},
			want: want{
				err:    true,
				errMsg: "allocation already has blobber blobber_1",
			}},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			blobbers, addID, removeID, sc, sa, now, balances := setup(tt.args)
			_, err := sa.changeBlobbers(&Config{TimeUnit: confTimeUnit}, blobbers, addID, "", removeID, now, balances, sc, &transaction.Transaction{
				ClientID:     clientId,
				CreationDate: now,
			}, false, 0)
			require.EqualValues(t, tt.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errMsg, err.Error())
				return
			}
			validate(tt.want, tt.args, sa, sc, balances)
		})
	}
}

func TestExtendAllocation(t *testing.T) {
	const (
		mockURL            = "mock_url"
		mockOwner          = "mock owner"
		mockPublicKey      = "mock public key"
		mockBlobberId      = "mock_blobber_id"
		mockPoolId         = "mock pool id"
		mockAllocationId   = "mock allocation id"
		mockMinPrice       = 0
		confTimeUnit       = 720 * time.Hour
		confMinAllocSize   = 1024
		mockMaxOffDuration = 744 * time.Hour
		mocksSize          = 10000000000
		mockDataShards     = 2
		mockParityShards   = 2
		mockNumAllBlobbers = 2 + mockDataShards + mockParityShards
		mockStake          = 3
		mockTimeUnit       = 1 * time.Hour
		mockHash           = "mock hash"
	)
	var mockBlobberCapacity int64 = 3700000000 * confMinAllocSize
	var mockMaxPrice = zcnToBalance(100.0)
	var mockReadPrice = zcnToBalance(0.01)
	var mockWritePrice = zcnToBalance(0.10)
	var now = common.Timestamp(1000)

	type args struct {
		request    updateAllocationRequest
		expiration common.Timestamp
		value      currency.Coin
		poolFunds  currency.Coin
	}
	type want struct {
		err    bool
		errMsg string
	}

	makeMockBlobber := func(index int) *StorageNode {
		sn := &StorageNode{}
		sn.SetEntity(&storageNodeV4{
			Provider: provider.Provider{
				ID:              mockBlobberId + strconv.Itoa(index),
				ProviderType:    spenum.Blobber,
				LastHealthCheck: now - blobberHealthTime + 1,
			},
			BaseURL:  mockURL + strconv.Itoa(index),
			Capacity: mockBlobberCapacity,
			Terms: Terms{
				ReadPrice:  mockReadPrice,
				WritePrice: mockWritePrice,
			},
		})
		return sn
	}

	setup := func(
		t *testing.T, args args,
	) (
		StorageSmartContract,
		*transaction.Transaction,
		*storageAllocationBase,
		[]*StorageNode,
		chainState.StateContextI,
	) {
		var balances = &mocks.StateContextI{}

		h := chainState.NewHardFork("electra", 0)
		balances.On("GetTrieNode", h.GetKey(),
			mock.MatchedBy(func(s *chainState.HardFork) bool {
				s = h
				return true
			})).Return(nil).Twice()

		mockBlock := &block.Block{}
		mockBlock.Round = 0
		balances.On("GetBlock").Return(mockBlock).Twice()

		var ssc = StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		var txn = transaction.Transaction{
			ClientID:     mockOwner,
			ToClientID:   ADDRESS,
			CreationDate: now * 10,
			Value:        args.value,
		}
		txn.Hash = mockHash

		var sa = storageAllocationBase{
			ID:              mockAllocationId,
			DataShards:      mockDataShards,
			ParityShards:    mockParityShards,
			Owner:           mockOwner,
			OwnerPublicKey:  mockPublicKey,
			Expiration:      now + common.Timestamp(mockTimeUnit),
			Size:            mocksSize,
			ReadPriceRange:  PriceRange{mockMinPrice, mockMaxPrice},
			WritePriceRange: PriceRange{mockMinPrice, mockMaxPrice},
			TimeUnit:        mockTimeUnit,
			WritePool:       args.poolFunds * 1e10,
			Stats: &StorageAllocationStats{
				UsedSize:          0,
				SuccessChallenges: int64(mockDataShards+mockParityShards) * 100,
				FailedChallenges:  int64(mockDataShards+mockParityShards) * 2,
				TotalChallenges:   int64(mockDataShards+mockParityShards) * 102,
				OpenChallenges:    0,
			},
		}

		bCount := sa.DataShards + sa.ParityShards
		var blobbers []*StorageNode
		for i := 0; i < mockNumAllBlobbers; i++ {
			mockBlobber := makeMockBlobber(i)

			if i < sa.DataShards+sa.ParityShards {
				blobbers = append(blobbers, mockBlobber)
				sa.BlobberAllocs = append(sa.BlobberAllocs, &BlobberAllocation{
					BlobberID: mockBlobber.Id(),
					Terms: Terms{
						WritePrice: mockWritePrice,
					},
					Stats: &StorageAllocationStats{
						UsedSize: sa.Size / int64(bCount),
					},
				})
				sp := stakePool{
					StakePool: &stakepool.StakePool{
						Pools: map[string]*stakepool.DelegatePool{
							mockPoolId: {
								DelegateID: "32q498e2de",
								Balance:    1e15,
							},
						},
					},
				}
				sp.Pools[mockPoolId].Balance = zcnToBalance(mockStake)
				balances.On("GetTrieNode", stakePoolKey(spenum.Blobber, mockBlobber.Id()),
					mock.MatchedBy(func(s *stakePool) bool {
						*s = sp
						return true
					})).Return(nil).Once()
				balances.On("InsertTrieNode", stakePoolKey(spenum.Blobber, mockBlobber.Id()),
					mock.Anything).Return("", nil).Once()
				balances.On("EmitEvent", event.TypeStats,
					event.TagToChallengePool, mock.Anything, mock.Anything).Return().Maybe()
				balances.On("EmitEvent", event.TypeStats,
					event.TagAddOrUpdateChallengePool, mock.Anything, mock.Anything).Return().Maybe()
				balances.On("EmitEvent", event.TypeStats,
					event.TagUpdateBlobberTotalOffers, mock.Anything, mock.Anything).Return().Maybe()
			}
		}
		balances.On(
			"EmitEvent",
			event.TypeStats,
			event.TagLockWritePool,
			mock.Anything,
			mock.Anything).Return().Maybe()
		balances.On(
			"GetTrieNode", challengePoolKey(ssc.ID, sa.ID),
			mock.MatchedBy(func(p *challengePool) bool {
				*p = *(newChallengePool())
				return true
			})).Return(nil).Once()
		balances.On(
			"InsertTrieNode", challengePoolKey(ssc.ID, sa.ID),
			mock.MatchedBy(func(cp *challengePool) bool {
				var size int64
				for _, blobber := range sa.BlobberAllocs {
					size += blobber.Stats.UsedSize
				}

				ordtu, err := sa.durationInTimeUnits(sa.Expiration-txn.CreationDate, 2592000000000000)
				if err != nil {
					require.NoError(t, err)
				}
				nrdtu := 1.0

				newFunds := sizeInGB(size) * float64(mockWritePrice) * (nrdtu - ordtu)
				return cp.Balance/10 == currency.Coin(newFunds/10) // ignore type cast errors
			}),
		).Return("", nil).Once()

		return ssc, &txn, &sa, blobbers, balances
	}

	testCases := []struct {
		name string
		args args
		want want
	}{
		{
			name: "ok_funded",
			args: args{
				request: updateAllocationRequest{
					ID:          mockAllocationId,
					OwnerID:     mockOwner,
					Size:        10 * MB,
					FileOptions: 63,
					Extend:      true,
				},
				expiration: now + common.Timestamp(confTimeUnit),
				value:      195312500,
				poolFunds:  19,
			},
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ssc, txn, sa, aBlobbers, balances := setup(t, tt.args)
			conf := &Config{
				TimeUnit:      confTimeUnit,
				MaxWritePrice: currency.Coin(11e14),
				MinWritePrice: currency.Coin(1),
				MaxReadPrice:  currency.Coin(11e14),
			}

			err := ssc.extendAllocation(
				txn,
				conf,
				false,
				sa,
				aBlobbers,
				&tt.args.request,
				balances,
			)
			if tt.want.err != (err != nil) {
				require.EqualValues(t, tt.want.err, err != nil, err)
			}
			if err != nil {
				if tt.want.errMsg != err.Error() {
					require.EqualValues(t, tt.want.errMsg, err.Error())
				}
			} else {
				//mock.AssertExpectationsForObjects(t, balances)
			}
		})
	}
}

func enableHardForks(t *testing.T, tb chainState.StateContextI) {
	h := chainState.NewHardFork("apollo", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = chainState.NewHardFork("ares", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = chainState.NewHardFork("artemis", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = chainState.NewHardFork("athena", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = chainState.NewHardFork("demeter", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = chainState.NewHardFork("electra", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = chainState.NewHardFork("hercules", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}
}

func TestStorageSmartContract_getAllocation(t *testing.T) {

	const allocID, clientID, clientPk = "alloc_hex", "client_hex", "pk"
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)

		err error
	)
	if _, err = ssc.getAllocation(allocID, balances); err == nil {
		t.Fatal("missing error")
	}
	if err != util.ErrValueNotPresent {
		t.Fatal("unexpected error:", err)
	}

	alloc := StorageAllocation{}

	alloc.SetEntity(&storageAllocationV2{
		ID:             allocID,
		DataShards:     1,
		ParityShards:   1,
		Size:           1024,
		Expiration:     1050,
		Owner:          clientID,
		OwnerPublicKey: clientPk,
	})

	_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), &alloc)
	require.NoError(t, err)
	var got *StorageAllocation
	got, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	assert.Equal(t, alloc.Encode(), got.Encode())
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
	const clientID, clientPk = "client_hex", "pk"
	var nar newAllocationRequest
	nar.DataShards = 2
	nar.ParityShards = 3
	nar.Size = 1024
	nar.Owner = clientID
	nar.OwnerPublicKey = clientPk
	nar.Blobbers = []string{"one", "two"}
	nar.ReadPriceRange = PriceRange{Min: 10, Max: 20}
	nar.WritePriceRange = PriceRange{Min: 100, Max: 200}
	balances := newTestBalances(t, false)
	conf := setConfig(t, balances)
	now := common.Timestamp(time.Now().Unix())
	var sa, err = nar.storageAllocation(balances, conf, now)
	require.NoError(t, err)
	alloc := sa.mustBase()
	require.Equal(t, alloc.DataShards, nar.DataShards)
	require.Equal(t, alloc.ParityShards, nar.ParityShards)
	require.Equal(t, alloc.Size, nar.Size)
	require.Equal(t, alloc.Owner, nar.Owner)
	require.Equal(t, alloc.Expiration, common.Timestamp(common.ToTime(now).Add(conf.TimeUnit).Unix()))
	require.Equal(t, alloc.OwnerPublicKey, nar.OwnerPublicKey)
	require.True(t, isEqualStrings(alloc.PreferredBlobbers,
		nar.Blobbers))
	require.Equal(t, alloc.ReadPriceRange, nar.ReadPriceRange)
	require.Equal(t, alloc.WritePriceRange, nar.WritePriceRange)
}

func Test_newAllocationRequest_decode(t *testing.T) {
	const clientID, clientPk = "client_id_hex", "client_pk_hex"
	var ne, nd newAllocationRequest
	ne.DataShards = 1
	ne.ParityShards = 1
	ne.Size = 2 * GB
	ne.Owner = clientID
	ne.OwnerPublicKey = clientPk
	ne.Blobbers = []string{"b1", "b2"}
	ne.ReadPriceRange = PriceRange{1, 2}
	ne.WritePriceRange = PriceRange{2, 3}
	require.NoError(t, nd.decode(mustEncode(t, &ne)))
	assert.EqualValues(t, &ne, &nd)
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

func newTestAllBlobbers(options ...map[string]interface{}) (all *StorageNodes) {
	numBlobbers := 2
	notAvailable := false
	isRestricted := false
	publicKeys := []string{}
	var isEnterprisees = []bool{}
	var storageVersions = []int{}

	if len(options) > 0 {
		option := options[0]

		if v, ok := option["num_blobbers"]; ok {
			numBlobbers = v.(int)
		}

		if v, ok := option["not_available"]; ok {
			notAvailable = v.(bool)
		}

		if v, ok := option["is_restricted"]; ok {
			isRestricted = v.(bool)
		}

		if v, ok := option["public_keys"]; ok {
			publicKeys = v.([]string)
		}

		if v, ok := option["is_enterprise"]; ok {
			isEnterprisees = v.([]bool)
		}

		if v, ok := option["storage_versions"]; ok {
			storageVersions = v.([]int)
		}
	}

	if len(publicKeys) == 0 {
		for i := 1; i <= numBlobbers; i++ {
			publicKeys = append(publicKeys, "pk"+strconv.Itoa(i))
		}
	}

	all = new(StorageNodes)

	for i := 1; i <= numBlobbers; i++ {
		isEnterprise := new(bool)
		*isEnterprise = false
		if len(isEnterprisees) > 0 {
			*isEnterprise = isEnterprisees[i-1]
		}

		storageVersion := new(int)
		*storageVersion = 0
		if len(storageVersions) > 0 {
			*storageVersion = storageVersions[i-1]
		}

		sn := &StorageNode{}
		sn.SetEntity(&storageNodeV4{
			Provider: provider.Provider{
				ID:              "b" + strconv.Itoa(i),
				ProviderType:    spenum.Blobber,
				LastHealthCheck: 0,
			},
			Version:   "v4",
			PublicKey: publicKeys[i-1],
			BaseURL:   "http://blobber" + strconv.Itoa(i) + ".test.ru:9100/api",
			Terms: Terms{
				ReadPrice:  20,
				WritePrice: 200,
			},
			Capacity:       50 * GB, // 50 GB
			Allocated:      5 * GB,  //  5 GB
			NotAvailable:   notAvailable,
			IsRestricted:   &isRestricted,
			IsEnterprise:   isEnterprise,
			StorageVersion: storageVersion,
		})
		all.Nodes = append(all.Nodes, sn)
	}
	return
}

func TestStorageSmartContract_newAllocationRequest(t *testing.T) {
	const (
		txHash, clientID, pubKey = "a5f4c3d2_tx_hex", "client_hex",
			"pub_key_hex"

		errMsg1 = "allocation_creation_failed: " +
			"malformed request: unexpected end of JSON input"
		errMsg2 = "allocation_creation_failed: " + "invalid request: invalid number of data shards"
		errMsg4 = "allocation_creation_failed: malformed request: " +
			"invalid character '}' looking for beginning of value"
		errMsg6 = "allocation_creation_failed: " +
			"invalid request: blobbers provided are not enough to honour the allocation"
		errMsg7 = "allocation_creation_failed: " + "getting stake pools: could not get item \"b1\": value not present"
		errMsg8 = "allocation_creation_failed: " + "not enough tokens to honor the allocation cost 0 < 4000"
		errMsg9 = "allocation_creation_failed: " +
			"no tokens to lock"
	)
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)

		tx   transaction.Transaction
		conf *Config

		resp string
		err  error
	)

	tx.Hash = txHash
	tx.Value = 100
	tx.ClientID = clientID
	tx.CreationDate = toSeconds(2 * time.Hour)

	balances.setTransaction(t, &tx)

	conf = setConfig(t, balances)
	conf.MaxChallengeCompletionRounds = 720
	conf.MinAllocSize = 10 * GB
	conf.TimeUnit = 720 * time.Hour

	_, err = balances.InsertTrieNode(scConfigKey(ADDRESS), conf)
	require.NoError(t, err)

	// 1.
	t.Run("unexpected end of JSON input", func(t *testing.T) {
		_, err = ssc.newAllocationRequest(&tx, nil, balances, nil)
		requireErrMsg(t, err, errMsg1)
	})
	t.Run("invalid character", func(t *testing.T) {
		tx.ClientID = clientID
		_, err = ssc.newAllocationRequest(&tx, []byte("} malformed {"), balances, nil)
		requireErrMsg(t, err, errMsg4)
	})

	// 4.
	t.Run("empty request", func(t *testing.T) {
		var nar newAllocationRequest

		_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, errMsg2)
	})

	t.Run("Blobbers provided are not enough to honour the allocation", func(t *testing.T) {
		var nar newAllocationRequest
		nar.ReadPriceRange = PriceRange{20, 10}
		nar.Owner = clientID
		nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
		nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
		nar.Size = 20 * GB
		nar.DataShards = 1
		nar.ParityShards = 1
		nar.Owner = "" // not set
		nar.OwnerPublicKey = pubKey
		nar.Blobbers = nil // not set

		_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, errMsg6)
	})

	t.Run("Blobbers provided are not enough to honour the allocation", func(t *testing.T) {
		var nar newAllocationRequest
		nar.ReadPriceRange = PriceRange{20, 10}
		nar.Owner = clientID
		nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
		nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
		nar.Size = 20 * GB
		nar.DataShards = 1
		nar.ParityShards = 1
		nar.Owner = "" // not set
		nar.OwnerPublicKey = pubKey
		nar.Blobbers = nil // not set
		nar.Owner = clientID
		_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, errMsg6)
	})

	t.Run("Blobbers provided are not enough to honour the allocation 2", func(t *testing.T) {
		var nar newAllocationRequest
		nar.ReadPriceRange = PriceRange{20, 10}
		nar.Owner = clientID
		nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
		nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
		nar.Size = 20 * GB
		nar.DataShards = 1
		nar.ParityShards = 1
		nar.Owner = "" // not set
		nar.OwnerPublicKey = pubKey
		nar.Blobbers = nil // not set
		nar.Owner = clientID

		_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, errMsg6)
	})

	t.Run("Blobbers provided are not enough to honour the allocation no pools", func(t *testing.T) {
		var nar newAllocationRequest
		nar.ReadPriceRange = PriceRange{20, 10}
		nar.Owner = clientID
		nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
		nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
		nar.Size = 20 * GB
		nar.DataShards = 1
		nar.ParityShards = 1
		nar.Owner = "" // not set
		nar.OwnerPublicKey = pubKey
		nar.Blobbers = nil // not set
		nar.Owner = clientID
		// 7. missing stake pools (not enough blobbers)
		var allBlobbers = newTestAllBlobbers()
		// make the blobbers health
		b0 := allBlobbers.Nodes[0]
		b0.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			return nil
		})

		b1 := allBlobbers.Nodes[1]
		b1.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			return nil
		})

		nar.Blobbers = append(nar.Blobbers, b0.Id())
		nar.BlobberAuthTickets = append(nar.BlobberAuthTickets, "")
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		nar.Blobbers = append(nar.Blobbers, b1.Id())
		nar.BlobberAuthTickets = append(nar.BlobberAuthTickets, "")
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, errMsg7)
	})

	t.Run("Blobbers provided are restricted blobbers", func(t *testing.T) {

		wallet := newClient(1000*x10, balances)
		b0Wallet := newClient(1000*x10, balances)
		b1Wallet := newClient(1000*x10, balances)

		var nar newAllocationRequest
		nar.ReadPriceRange = PriceRange{20, 10}
		nar.Owner = clientID
		nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
		nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
		nar.Size = 20 * GB
		nar.DataShards = 1
		nar.ParityShards = 1
		nar.Blobbers = nil // not set
		nar.Owner = wallet.id
		nar.OwnerPublicKey = wallet.pk

		var tempTxn transaction.Transaction
		tempTxn.Hash = txHash
		tempTxn.Value = 10000
		tempTxn.ClientID = wallet.id
		tempTxn.CreationDate = toSeconds(2 * time.Hour)

		balances.setTransaction(t, &tempTxn)

		var conditions map[string]interface{}
		conditions = make(map[string]interface{})
		conditions["is_restricted"] = true
		conditions["public_keys"] = []string{b0Wallet.pk, b1Wallet.pk}
		var allBlobbers = newTestAllBlobbers(conditions)

		b0 := allBlobbers.Nodes[0]
		b0.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 5 * GB
			return nil
		})
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		require.NoError(t, err)

		b1 := allBlobbers.Nodes[1]
		b1.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 10 * GB
			return nil
		})
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		var (
			sp1, sp2 = newStakePool(), newStakePool()
			dp1, dp2 = new(stakepool.DelegatePool), new(stakepool.DelegatePool)
		)
		dp1.Balance, dp2.Balance = 20e10, 20e10
		sp1.Pools["hash1"], sp2.Pools["hash2"] = dp1, dp2
		require.NoError(t, sp1.Save(spenum.Blobber, "b1", balances))
		require.NoError(t, sp2.Save(spenum.Blobber, "b2", balances))

		nar.Blobbers = append(nar.Blobbers, b0.Id())
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		nar.Blobbers = append(nar.Blobbers, b1.Id())
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		nar.BlobberAuthTickets = []string{"", ""}
		_, err = ssc.newAllocationRequest(&tempTxn, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, "allocation_creation_failed: Not enough blobbers to honor the allocation: blobber b1 auth ticket verification failed: invalid_auth_ticket: empty auth ticket, blobber b2 auth ticket verification failed: invalid_auth_ticket: empty auth ticket")

		blobber0AuthTicket, err := b0Wallet.scheme.Sign(wallet.id)
		require.NoError(t, err)
		blobber1AuthTicket, err := b1Wallet.scheme.Sign(wallet.id)
		require.NoError(t, err)

		nar.BlobberAuthTickets = []string{blobber0AuthTicket, blobber1AuthTicket}
		_, err = ssc.newAllocationRequest(&tempTxn, mustEncode(t, &nar), balances, nil)
		require.NoError(t, err)
	})

	// 8. not enough tokens
	t.Run("not enough tokens to honor the min lock demand (0 < 270)", func(t *testing.T) {

		var nar newAllocationRequest
		nar.ReadPriceRange = PriceRange{20, 10}
		nar.Owner = clientID
		nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
		nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
		nar.Size = 10 * GB
		nar.DataShards = 1
		nar.ParityShards = 1
		nar.Owner = "" // not set
		nar.OwnerPublicKey = pubKey
		nar.Blobbers = nil // not set
		nar.Owner = clientID
		var allBlobbers = newTestAllBlobbers()
		// make the blobbers health
		b0 := allBlobbers.Nodes[0]
		b0.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			return nil
		})

		b1 := allBlobbers.Nodes[1]
		b1.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			return nil
		})

		nar.Blobbers = append(nar.Blobbers, b0.Id())
		nar.BlobberAuthTickets = append(nar.BlobberAuthTickets, "")
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		nar.Blobbers = append(nar.Blobbers, b1.Id())
		nar.BlobberAuthTickets = append(nar.BlobberAuthTickets, "")
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		var (
			sp1, sp2 = newStakePool(), newStakePool()
			dp1, dp2 = new(stakepool.DelegatePool), new(stakepool.DelegatePool)
		)
		dp1.Balance, dp2.Balance = 20e10, 20e10
		sp1.Pools["hash1"], sp2.Pools["hash2"] = dp1, dp2
		require.NoError(t, sp1.Save(spenum.Blobber, "b1", balances))
		require.NoError(t, sp2.Save(spenum.Blobber, "b2", balances))

		tx.Value = 0
		_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, errMsg8)
	})
	// 9. no tokens to lock (client balance check)
	t.Run("Blobbers provided are not enough to honour the allocation no pools", func(t *testing.T) {

		var nar newAllocationRequest
		nar.ReadPriceRange = PriceRange{20, 10}
		nar.Owner = clientID
		nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
		nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
		nar.Size = 10 * GB
		nar.DataShards = 1
		nar.ParityShards = 1
		nar.Owner = "" // not set
		nar.OwnerPublicKey = pubKey
		nar.Blobbers = nil // not set
		nar.Owner = clientID
		var allBlobbers = newTestAllBlobbers()
		// make the blobbers health
		b0 := allBlobbers.Nodes[0]
		b0.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 5 * GB
			return nil
		})
		b1 := allBlobbers.Nodes[1]
		b1.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 10 * GB
			return nil
		})

		nar.Blobbers = append(nar.Blobbers, b0.Id())
		nar.BlobberAuthTickets = append(nar.BlobberAuthTickets, "")
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		nar.Blobbers = append(nar.Blobbers, b1.Id())
		nar.BlobberAuthTickets = append(nar.BlobberAuthTickets, "")
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		var (
			sp1, sp2 = newStakePool(), newStakePool()
			dp1, dp2 = new(stakepool.DelegatePool), new(stakepool.DelegatePool)
		)
		dp1.Balance, dp2.Balance = 20e10, 20e10
		sp1.Pools["hash1"], sp2.Pools["hash2"] = dp1, dp2
		require.NoError(t, sp1.Save(spenum.Blobber, "b1", balances))
		require.NoError(t, sp2.Save(spenum.Blobber, "b2", balances))

		tx.Hash = encryption.Hash("blobber_not_enough_to_honour_allocation_no_pools")
		tx.Value = 400
		_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, errMsg9)

	})
	// 10. ok
	t.Run("ok", func(t *testing.T) {
		wallet := newClient(1000*x10, balances)
		var nar newAllocationRequest
		nar.ReadPriceRange = PriceRange{20, 10}
		nar.Owner = clientID
		nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
		nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
		nar.Size = 10 * GB
		nar.DataShards = 1
		nar.ParityShards = 1
		nar.Owner = "" // not set
		nar.OwnerPublicKey = pubKey
		nar.Blobbers = nil // not set
		nar.Owner = clientID

		var allBlobbers = newTestAllBlobbers()
		// make the blobbers health
		b0 := allBlobbers.Nodes[0]
		b0.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 5 * GB
			return nil
		})
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		require.NoError(t, err)
		b1 := allBlobbers.Nodes[1]
		b1.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 10 * GB
			return nil
		})
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		nar.Blobbers = append(nar.Blobbers, b0.Id())
		nar.BlobberAuthTickets = append(nar.BlobberAuthTickets, "")
		nar.Blobbers = append(nar.Blobbers, b1.Id())
		nar.BlobberAuthTickets = append(nar.BlobberAuthTickets, "")

		var (
			sp1, sp2 = newStakePool(), newStakePool()
			dp1, dp2 = new(stakepool.DelegatePool), new(stakepool.DelegatePool)
		)
		dp1.Balance, dp2.Balance = 20e10, 20e10
		sp1.Pools["hash1"], sp2.Pools["hash2"] = dp1, dp2
		require.NoError(t, sp1.Save(spenum.Blobber, "b1", balances))
		require.NoError(t, sp2.Save(spenum.Blobber, "b2", balances))

		var tempTxn transaction.Transaction
		tempTxn.Hash = encryption.Hash("ok")
		tempTxn.Value = 10000
		tempTxn.ClientID = wallet.id
		tempTxn.CreationDate = toSeconds(2 * time.Hour)

		balances.setTransaction(t, &tempTxn)

		resp, err = ssc.newAllocationRequest(&tempTxn, mustEncode(t, &nar), balances, nil)
		require.NoError(t, err)

		// check response
		var aresp NewAllocationTxnOutput
		require.NoError(t, aresp.Decode([]byte(resp)))

		assert.Equal(t, tempTxn.Hash, aresp.ID)
		assert.Equal(t, len(aresp.Blobber_ids), 2)

		allBlobbers.Nodes[0].mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tempTxn.CreationDate
			b.Allocated += 10 * GB
			return nil
		})
		allBlobbers.Nodes[1].mustUpdateBase(func(b *storageNodeBase) error {
			b.Allocated += 10 * GB
			return nil
		})

		// blobbers saved in all blobbers list
		var ab []*StorageNode
		loaded0, err := ssc.getBlobber(b0.Id(), balances)
		loaded1, err := ssc.getBlobber(b1.Id(), balances)
		ab = append(ab, loaded0)
		ab = append(ab, loaded1)
		require.NoError(t, err)
		for i, sbn := range allBlobbers.Nodes {
			sbnEntity := sbn.Entity().(*storageNodeV4)
			abEntitity := ab[i].Entity().(*storageNodeV4)

			assert.Equal(t, *sbnEntity, *abEntitity)
		}
		assert.EqualValues(t, allBlobbers.Nodes, ab)
		// independent saved blobbers
		var blob1, blob2 *StorageNode
		blob1, err = ssc.getBlobber("b1", balances)
		require.NoError(t, err)
		assert.EqualValues(t, allBlobbers.Nodes[0], blob1)
		blob2, err = ssc.getBlobber("b2", balances)
		require.NoError(t, err)
		assert.EqualValues(t, allBlobbers.Nodes[1], blob2)

		_, err = ssc.getStakePool(spenum.Blobber, "b1", balances)
		require.NoError(t, err)

		_, err = ssc.getStakePool(spenum.Blobber, "b2", balances)
		require.NoError(t, err)

		// 3. challenge pool existence
		var cp *challengePool
		cp, err = ssc.getChallengePool(aresp.ID, balances)
		require.NoError(t, err)

		assert.Zero(t, cp.Balance)
	})

	// Enterprise Allocation Tests

	t.Run("Enterprise : All blobbers provided are enterprise blobbers for enterprise allocation should work", func(t *testing.T) {

		wallet := newClient(1000*x10, balances)
		b0Wallet := newClient(1000*x10, balances)
		b1Wallet := newClient(1000*x10, balances)

		var nar newAllocationRequest
		nar.ReadPriceRange = PriceRange{20, 10}
		nar.Owner = clientID
		nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
		nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
		nar.Size = 20 * GB
		nar.DataShards = 1
		nar.ParityShards = 1
		nar.Blobbers = nil // not set
		nar.Owner = wallet.id
		nar.OwnerPublicKey = wallet.pk

		nar.IsEnterprise = true

		var tempTxn transaction.Transaction
		tempTxn.Hash = uuid.New().String()
		tempTxn.Value = 10000
		tempTxn.ClientID = wallet.id
		tempTxn.CreationDate = toSeconds(2 * time.Hour)

		balances.setTransaction(t, &tempTxn)

		var conditions map[string]interface{}
		conditions = make(map[string]interface{})
		conditions["is_restricted"] = true
		conditions["is_enterprise"] = []bool{true, true}
		conditions["public_keys"] = []string{b0Wallet.pk, b1Wallet.pk}
		var allBlobbers = newTestAllBlobbers(conditions)

		b0 := allBlobbers.Nodes[0]
		b0.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 5 * GB
			return nil
		})
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		require.NoError(t, err)

		b1 := allBlobbers.Nodes[1]
		b1.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 10 * GB
			return nil
		})
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		var (
			sp1, sp2 = newStakePool(), newStakePool()
			dp1, dp2 = new(stakepool.DelegatePool), new(stakepool.DelegatePool)
		)
		dp1.Balance, dp2.Balance = 20e10, 20e10
		sp1.Pools["hash1"], sp2.Pools["hash2"] = dp1, dp2
		require.NoError(t, sp1.Save(spenum.Blobber, "b1", balances))
		require.NoError(t, sp2.Save(spenum.Blobber, "b2", balances))

		nar.Blobbers = append(nar.Blobbers, b0.Id())
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		nar.Blobbers = append(nar.Blobbers, b1.Id())
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		nar.BlobberAuthTickets = []string{"", ""}
		_, err = ssc.newAllocationRequest(&tempTxn, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, "allocation_creation_failed: Not enough blobbers to honor the allocation: blobber b1 auth ticket verification failed: invalid_auth_ticket: empty auth ticket, blobber b2 auth ticket verification failed: invalid_auth_ticket: empty auth ticket")

		blobber0AuthTicket, err := b0Wallet.scheme.Sign(wallet.id)
		require.NoError(t, err)
		blobber1AuthTicket, err := b1Wallet.scheme.Sign(wallet.id)
		require.NoError(t, err)

		nar.BlobberAuthTickets = []string{blobber0AuthTicket, blobber1AuthTicket}
		_, err = ssc.newAllocationRequest(&tempTxn, mustEncode(t, &nar), balances, nil)
		require.NoError(t, err)
	})

	t.Run("Enterprise : One blobber enterprise and 2nd non-enterprise for non-enterprise allocation should fail", func(t *testing.T) {
		wallet := newClient(1000*x10, balances)
		b0Wallet := newClient(1000*x10, balances)
		b1Wallet := newClient(1000*x10, balances)

		var nar newAllocationRequest
		nar.ReadPriceRange = PriceRange{20, 10}
		nar.Owner = clientID
		nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
		nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
		nar.Size = 20 * GB
		nar.DataShards = 1
		nar.ParityShards = 1
		nar.Blobbers = nil // not set
		nar.Owner = wallet.id
		nar.OwnerPublicKey = wallet.pk

		nar.IsEnterprise = true

		var tempTxn transaction.Transaction
		tempTxn.Hash = uuid.New().String()
		tempTxn.Value = 10000
		tempTxn.ClientID = wallet.id
		tempTxn.CreationDate = toSeconds(2 * time.Hour)

		balances.setTransaction(t, &tempTxn)

		var conditions map[string]interface{}
		conditions = make(map[string]interface{})
		conditions["is_restricted"] = true
		conditions["is_enterprise"] = []bool{true, false}
		conditions["public_keys"] = []string{b0Wallet.pk, b1Wallet.pk}
		var allBlobbers = newTestAllBlobbers(conditions)

		b0 := allBlobbers.Nodes[0]
		b0.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 5 * GB
			return nil
		})
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		require.NoError(t, err)

		b1 := allBlobbers.Nodes[1]
		b1.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 10 * GB
			return nil
		})
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		var (
			sp1, sp2 = newStakePool(), newStakePool()
			dp1, dp2 = new(stakepool.DelegatePool), new(stakepool.DelegatePool)
		)
		dp1.Balance, dp2.Balance = 20e10, 20e10
		sp1.Pools["hash1"], sp2.Pools["hash2"] = dp1, dp2
		require.NoError(t, sp1.Save(spenum.Blobber, "b1", balances))
		require.NoError(t, sp2.Save(spenum.Blobber, "b2", balances))

		nar.Blobbers = append(nar.Blobbers, b0.Id())
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		nar.Blobbers = append(nar.Blobbers, b1.Id())
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		blobber0AuthTicket, err := b0Wallet.scheme.Sign(wallet.id)
		require.NoError(t, err)
		blobber1AuthTicket, err := b1Wallet.scheme.Sign(wallet.id)
		require.NoError(t, err)

		nar.BlobberAuthTickets = []string{blobber0AuthTicket, blobber1AuthTicket}

		_, err = ssc.newAllocationRequest(&tempTxn, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, "allocation_creation_failed: Not enough blobbers to honor the allocation: blobber b2 is not enterprise")

		_ = b1.Update(&storageNodeV4{}, func(e entitywrapper.EntityI) error {
			b := e.(*storageNodeV4)
			b.IsEnterprise = new(bool)
			*b.IsEnterprise = true
			return nil
		})
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		_, err = ssc.newAllocationRequest(&tempTxn, mustEncode(t, &nar), balances, nil)
		require.NoError(t, err)
	})

	t.Run("StorageVersion : All blobbers should be storagev2", func(t *testing.T) {

		wallet := newClient(1000*x10, balances)
		b0Wallet := newClient(1000*x10, balances)
		b1Wallet := newClient(1000*x10, balances)

		var nar newAllocationRequest
		nar.ReadPriceRange = PriceRange{20, 10}
		nar.Owner = clientID
		nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
		nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
		nar.Size = 20 * GB
		nar.DataShards = 1
		nar.ParityShards = 1
		nar.Blobbers = nil // not set
		nar.Owner = wallet.id
		nar.OwnerPublicKey = wallet.pk
		nar.StorageVersion = 1

		nar.IsEnterprise = true

		var tempTxn transaction.Transaction
		tempTxn.Hash = uuid.New().String()
		tempTxn.Value = 10000
		tempTxn.ClientID = wallet.id
		tempTxn.CreationDate = toSeconds(2 * time.Hour)

		balances.setTransaction(t, &tempTxn)

		var conditions map[string]interface{}
		conditions = make(map[string]interface{})
		conditions["is_restricted"] = true
		conditions["is_enterprise"] = []bool{true, true}
		conditions["public_keys"] = []string{b0Wallet.pk, b1Wallet.pk}
		conditions["storage_versions"] = []int{1, 1}
		var allBlobbers = newTestAllBlobbers(conditions)

		b0 := allBlobbers.Nodes[0]
		b0.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 5 * GB
			return nil
		})
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		require.NoError(t, err)

		b1 := allBlobbers.Nodes[1]
		b1.mustUpdateBase(func(b *storageNodeBase) error {
			b.LastHealthCheck = tx.CreationDate
			b.Allocated = 10 * GB
			return nil
		})
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		var (
			sp1, sp2 = newStakePool(), newStakePool()
			dp1, dp2 = new(stakepool.DelegatePool), new(stakepool.DelegatePool)
		)
		dp1.Balance, dp2.Balance = 20e10, 20e10
		sp1.Pools["hash1"], sp2.Pools["hash2"] = dp1, dp2
		require.NoError(t, sp1.Save(spenum.Blobber, "b1", balances))
		require.NoError(t, sp2.Save(spenum.Blobber, "b2", balances))

		nar.Blobbers = append(nar.Blobbers, b0.Id())
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		nar.Blobbers = append(nar.Blobbers, b1.Id())
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		nar.BlobberAuthTickets = []string{"", ""}
		_, err = ssc.newAllocationRequest(&tempTxn, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, "allocation_creation_failed: Not enough blobbers to honor the allocation: blobber b1 auth ticket verification failed: invalid_auth_ticket: empty auth ticket, blobber b2 auth ticket verification failed: invalid_auth_ticket: empty auth ticket")

		blobber0AuthTicket, err := b0Wallet.scheme.Sign(wallet.id)
		require.NoError(t, err)
		blobber1AuthTicket, err := b1Wallet.scheme.Sign(wallet.id)
		require.NoError(t, err)

		nar.BlobberAuthTickets = []string{blobber0AuthTicket, blobber1AuthTicket}
		_, err = ssc.newAllocationRequest(&tempTxn, mustEncode(t, &nar), balances, nil)
		require.NoError(t, err)

		alloc, err := ssc.getAllocation(tempTxn.Hash, balances)
		require.NoError(t, err)

		allocV3 := alloc.Entity().(*storageAllocationV3)
		assert.Equal(t, 1, *allocV3.StorageVersion)
	})
}

func Test_updateAllocationRequest_decode(t *testing.T) {
	var ud, ue updateAllocationRequest
	ue.Size = -200
	require.NoError(t, ud.decode(mustEncode(t, &ue)))
	assert.EqualValues(t, ue, ud)
}

func Test_updateAllocationRequest_validate(t *testing.T) {
	alloc := &storageAllocationBase{
		BlobberAllocsMap: make(map[string]*BlobberAllocation),
		Owner:            "owner123",
		FileOptions:      32,
	}
	alloc.BlobberAllocsMap["blobber1"] = &BlobberAllocation{}

	var sub = 9.01 * GB

	tests := []struct {
		name      string
		uar       *updateAllocationRequest
		expectErr bool
	}{
		{
			name: "Zero size",
			uar: &updateAllocationRequest{
				OwnerID: "owner123",
			},
			expectErr: true,
		},
		{
			name: "Becomes too small",
			uar: &updateAllocationRequest{
				Size:                    -int64(sub),
				SetThirdPartyExtendable: true,
				OwnerID:                 "owner123",
			},
			expectErr: true,
		},
		{
			name: "No blobbers (invalid allocation)",
			uar: &updateAllocationRequest{
				Size:    1 * GB,
				OwnerID: "owner123",
			},
			expectErr: true,
		},
		{
			name: "Positive case",
			uar: &updateAllocationRequest{
				Size:    1 * GB,
				OwnerID: "owner123",
			},
			expectErr: false,
		},
		{
			name: "No changes",
			uar: &updateAllocationRequest{
				OwnerID: "owner123",
			},
			expectErr: true,
		},
		{
			name: "Negative size",
			uar: &updateAllocationRequest{
				Size:    -1,
				OwnerID: "owner123",
			},
			expectErr: true,
		},
		{
			name: "Invalid allocation (no blobbers)",
			uar: &updateAllocationRequest{
				OwnerID: "owner123",
			},
			expectErr: true,
		},
		{
			name: "Add existing blobber",
			uar: &updateAllocationRequest{
				AddBlobberId: "blobber1",
				OwnerID:      "owner123",
			},
			expectErr: true,
		},
		{
			name: "Remove non-existing blobber",
			uar: &updateAllocationRequest{
				RemoveBlobberId: "nonexistent_blobber",
				OwnerID:         "owner123",
			},
			expectErr: true,
		},
		{
			name: "FileOptions out of range",
			uar: &updateAllocationRequest{
				FileOptions: 64,
				OwnerID:     "owner123",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Positive case" {
				alloc.BlobberAllocs = []*BlobberAllocation{{}}
			}
			err := tt.uar.validate(clientId, nil, alloc)

			if tt.expectErr && err == nil {
				t.Error("Expected an error, but got nil")
			} else if !tt.expectErr && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
		})
	}
}

func Test_updateAllocationRequest_getBlobbersSizeDiff(t *testing.T) {
	var (
		uar   updateAllocationRequest
		alloc storageAllocationBase
	)

	alloc.Size = 10 * GB
	alloc.DataShards = 2
	alloc.ParityShards = 2

	uar.Size = 1 * GB // add 1 GB
	assert.Equal(t, int64(512*MB), uar.getBlobbersSizeDiff(&alloc))

	uar.Size = -1 * GB // sub 1 GB
	assert.Equal(t, -int64(512*MB), uar.getBlobbersSizeDiff(&alloc))

	uar.Size = 0 // no changes
	assert.Zero(t, uar.getBlobbersSizeDiff(&alloc))
}

// create allocation with blobbers, configurations, stake pools
func createNewTestAllocation(t *testing.T, ssc *StorageSmartContract,
	txHash, clientID, pubKey string, balances chainState.StateContextI) {

	var (
		tx          transaction.Transaction
		nar         newAllocationRequest
		allBlobbers *StorageNodes
		conf        Config
		err         error
	)

	tx.Hash = txHash
	tx.Value = 400
	tx.ClientID = clientID
	tx.CreationDate = toSeconds(2 * time.Hour)

	balances.(*testBalances).setTransaction(t, &tx)

	conf.MaxChallengeCompletionRounds = 720
	conf.MinAllocSize = 10 * GB
	conf.MaxBlobbersPerAllocation = 4
	conf.TimeUnit = 20 * time.Second

	_, err = balances.InsertTrieNode(scConfigKey(ADDRESS), &conf)
	require.NoError(t, err)

	allBlobbers = newTestAllBlobbers()
	// make the blobbers health
	b0 := allBlobbers.Nodes[0]
	b0.mustUpdateBase(func(b *storageNodeBase) error {
		b.LastHealthCheck = tx.CreationDate
		b.Allocated = 5 * GB
		return nil
	})
	b1 := allBlobbers.Nodes[1]
	b1.mustUpdateBase(func(b *storageNodeBase) error {
		b.LastHealthCheck = tx.CreationDate
		b.Allocated = 10 * GB
		return nil
	})

	nar.Blobbers = append(nar.Blobbers, b0.Id())
	_, err = balances.InsertTrieNode(b0.GetKey(), b0)
	nar.Blobbers = append(nar.Blobbers, b1.Id())
	_, err = balances.InsertTrieNode(b1.GetKey(), b1)
	require.NoError(t, err)

	nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
	nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
	nar.Size = 10 * GB
	nar.DataShards = 1
	nar.ParityShards = 1
	nar.Owner = clientID
	nar.OwnerPublicKey = pubKey
	nar.Blobbers = []string{"b1", "b2"}
	nar.BlobberAuthTickets = []string{"", ""}

	var (
		sp1, sp2 = newStakePool(), newStakePool()
		dp1, dp2 = new(stakepool.DelegatePool), new(stakepool.DelegatePool)
	)
	dp1.Balance, dp2.Balance = 20e10, 20e10
	sp1.Pools["hash1"], sp2.Pools["hash2"] = dp1, dp2
	require.NoError(t, sp1.Save(spenum.Blobber, "b1", balances))
	require.NoError(t, sp2.Save(spenum.Blobber, "b2", balances))

	balances.(*testBalances).balances[clientID] = 1100 + 4500

	tx.Value = 4500
	_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances, nil)
	require.NoError(t, err)
}

func Test_updateAllocationRequest_getNewBlobbersSize(t *testing.T) {

	const allocTxHash, clientID, pubKey = "a5f4c3d2_tx_hex", "client_hex",
		"pub_key_hex"

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)

		uar updateAllocationRequest
		err error
	)

	createNewTestAllocation(t, ssc, allocTxHash, clientID, pubKey, balances)

	sa, err := ssc.getAllocation(allocTxHash, balances)
	require.NoError(t, err)

	alloc := sa.mustBase()

	alloc.Size = 5 * GB
	alloc.DataShards = 2
	alloc.ParityShards = 2

	uar.Size = 1 * GB // add 1 GB
	assert.Less(t, math.Abs(1-float64(10*GB+256*MB)/float64(uar.getNewBlobbersSize(alloc))), 0.05)

	uar.Size = -1 * GB // sub 1 GB
	assert.Less(t, math.Abs(1-float64(10*GB-256*MB)/float64(uar.getNewBlobbersSize(alloc))), 0.05)

	uar.Size = 0 // no changes
	assert.Less(t, math.Abs(1-float64(10*GB)/float64(uar.getNewBlobbersSize(alloc))), 0.05)
}

func TestStorageSmartContract_getAllocationBlobbers(t *testing.T) {

	const allocTxHash, clientID, pubKey = "a5f4c3d2_tx_hex", "client_hex",
		"pub_key_hex"

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)

		alloc *StorageAllocation
		err   error
	)

	createNewTestAllocation(t, ssc, allocTxHash, clientID, pubKey, balances)

	alloc, err = ssc.getAllocation(allocTxHash, balances)
	require.NoError(t, err)

	var blobbers []*StorageNode
	blobbers, err = ssc.getAllocationBlobbers(alloc.mustBase(), balances)
	require.NoError(t, err)

	assert.Len(t, blobbers, 2)
}

func (sa *StorageAllocation) deepCopy(t *testing.T) (cp *StorageAllocation) {
	cp = new(StorageAllocation)
	require.NoError(t, cp.Decode(mustEncode(t, sa)))
	return
}

func TestRemoveBlobberAllocation(t *testing.T) {

	type args struct {
		numBlobbers         int
		numAllocInChallenge int
		removeBlobberID     string
		allocationID        string
	}

	type want struct {
		numBlobberChallenge              int
		numAllocationChallengePerBlobber []int
		err                              bool
		errMsg                           string
	}

	setup := func(arg args) (*StorageSmartContract, chainState.StateContextI, string, string) {
		var (
			ssc          = newTestStorageSC()
			balances     = newTestBalances(t, false)
			removeID     = arg.removeBlobberID
			allocationID = arg.allocationID
		)

		for i := 0; i < arg.numBlobbers; i++ {
			blobberID := "blobber_" + strconv.Itoa(i)
			err := PartitionsChallengeReadyBlobberAddOrUpdate(balances, blobberID, currency.Coin(1e12), uint64(1e6))
			require.NoError(t, err)

			bcAllocPartition, err := partitionsBlobberAllocations(blobberID, balances)
			require.NoError(t, err)
			for j := 0; j < arg.numAllocInChallenge; j++ {
				allocID := "allocation_" + strconv.Itoa(j)
				err := bcAllocPartition.Add(balances, &BlobberAllocationNode{ID: allocID})
				require.NoError(t, err)
			}
			err = bcAllocPartition.Save(balances)
			require.NoError(t, err)
		}

		return ssc, balances, removeID, allocationID
	}

	validate := func(want want, balances chainState.StateContextI) {
		bcPart, _, err := partitionsChallengeReadyBlobbers(balances)
		require.NoError(t, err)

		bcPartSize, err := bcPart.Size(balances)
		require.NoError(t, err)
		require.Equal(t, bcPartSize, want.numBlobberChallenge)

		for i, numAlloc := range want.numAllocationChallengePerBlobber {
			blobber := "blobber_" + strconv.Itoa(i)
			bcAllocChallenge, err := partitionsBlobberAllocations(blobber, balances)
			require.NoError(t, err)
			bcAllocSize, err := bcAllocChallenge.Size(balances)
			require.NoError(t, err)

			require.Equal(t, numAlloc, bcAllocSize)
		}

	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "1_allocation_challenge",
			args: args{
				numBlobbers:         6,
				numAllocInChallenge: 1,
				removeBlobberID:     "blobber_0",
				allocationID:        "allocation_0",
			},
			want: want{
				numBlobberChallenge:              5,
				numAllocationChallengePerBlobber: []int{0, 1, 1, 1, 1, 1},
				err:                              false,
			},
		},
		{
			name: "2_allocation_challenge",
			args: args{
				numBlobbers:         6,
				numAllocInChallenge: 2,
				removeBlobberID:     "blobber_0",
				allocationID:        "allocation_0",
			},
			want: want{
				numBlobberChallenge:              6,
				numAllocationChallengePerBlobber: []int{1, 2, 2, 2, 2, 2},
				err:                              false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, balances, removeBlobberID, allocationID := setup(tt.args)
			err := removeAllocationFromBlobberPartitions(balances,
				removeBlobberID, allocationID)
			require.NoError(t, err)
			validate(tt.want, balances)
		})

		break
	}
}

func setupAllocationWithMockStats(t *testing.T, ssc *StorageSmartContract, client *Client, tp int64, balances *testBalances, zeroStats, isRestricted, IsEnterpriseAllocation bool) (alloc *storageAllocationBase, blobbers []*Client) {
	var err error

	tokensToLock := 100 * x10

	var enterpriseTestTerms []Terms

	if IsEnterpriseAllocation {
		enterpriseTestTerms = append(enterpriseTestTerms, Terms{
			ReadPrice:  1 * x10,
			WritePrice: 1 * x10,
		})
		tokensToLock = 20 * x10
	}

	allocID, blobbers := addAllocation(t, ssc, client, tp, 10*GB, 200*GB, 5000*x10, currency.Coin(tokensToLock), 20, balances, true, isRestricted, IsEnterpriseAllocation, enterpriseTestTerms...)
	sa, err := ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	alloc = sa.mustBase()

	if !zeroStats {
		for _, ba := range alloc.BlobberAllocs {
			const allocRoot = "alloc-root-1"
			var cc = &BlobberCloseConnection{
				AllocationRoot:     allocRoot,
				PrevAllocationRoot: "",
				WriteMarker:        &WriteMarker{},
			}
			wm1 := &writeMarkerV1{
				AllocationRoot:         allocRoot,
				PreviousAllocationRoot: "",
				AllocationID:           allocID,
				Size:                   ba.Size, // 100 MB
				BlobberID:              ba.BlobberID,
				Timestamp:              common.Timestamp(tp),
				ClientID:               client.id,
			}
			wm1.Signature, err = client.scheme.Sign(
				encryption.Hash(wm1.GetHashData()))
			require.NoError(t, err)
			cc.WriteMarker.SetEntity(wm1)
			var tx = newTransaction(ba.BlobberID, ssc.ID, 0, tp)
			balances.setTransaction(t, tx)
			resp, err := ssc.commitBlobberConnection(tx, mustEncode(t, &cc), balances)
			require.NoError(t, err)
			require.NotZero(t, resp)
		}
	}

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	return sa.mustBase(), blobbers
}

func compareAllocationData(t *testing.T, beforeAlloc, afterAlloc storageAllocationBase) {
	beforeAllocJson, err := json.Marshal(beforeAlloc)
	require.NoError(t, err)

	afterAllocJson, err := json.Marshal(afterAlloc)
	require.NoError(t, err)

	beforeAllocString := string(beforeAllocJson)
	afterAllocString := string(afterAllocJson)

	assert.JSONEq(t, beforeAllocString, afterAllocString, "Allocation data should be same")
}

func checkStakesRewardsAre0ForAlloc(beforeAlloc *storageAllocationBase, ssc *StorageSmartContract, t *testing.T, balances *testBalances) {
	for _, ba := range beforeAlloc.BlobberAllocs {
		sp, err := ssc.getStakePool(spenum.Blobber, ba.BlobberID, balances)
		require.NoError(t, err)

		require.Equal(t, 0, int(sp.Reward), "30% service charge to blobber should be updated")
		require.Len(t, sp.Pools, 1, "Single delegate pool")
		// get key of the delegate pool
		var dpKey string
		for k := range sp.Pools {
			dpKey = k
			break
		}
		require.Equal(t, 0, int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")
	}
}

func checkStakesRewardsAre0ForBlobber(blobberID string, ssc *StorageSmartContract, t *testing.T, balances *testBalances) {
	sp, err := ssc.getStakePool(spenum.Blobber, blobberID, balances)
	require.NoError(t, err)

	require.Equal(t, 0, int(sp.Reward), "30% service charge to blobber should be updated")
	require.Len(t, sp.Pools, 1, "Single delegate pool")
	// get key of the delegate pool
	var dpKey string
	for k := range sp.Pools {
		dpKey = k
		break
	}
	require.Equal(t, 0, int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")
}

func TestUpdateAllocationRequest(t *testing.T) {

	// Tests :
	// 1. Update single operation and check the stats and tag events
	// 2. Update all operations at once and check the stats and events at the end

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
	)

	t.Run("Extend unused allocation duration should work without adding extra payment", func(t *testing.T) {
		var (
			tp     = int64(0)
			client = newClient(2000*x10, balances)

			// Allocation
			beforeAlloc, _ = setupAllocationWithMockStats(t, ssc, client, tp, balances, true, false, false)
			allocID        = beforeAlloc.ID
		)

		// extend
		var uar updateAllocationRequest
		uar.ID = allocID
		uar.Extend = true
		tp += int64(360 * time.Hour / 1e9)
		resp, err := uar.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
		require.NoError(t, err)

		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		require.EqualValues(t, afterAlloc, &deco, "Response and allocation in MPT should be same")
		assert.NotEqual(t, beforeAlloc.Tx, afterAlloc.mustBase().Tx, "Transaction should be updated")
		assert.Equal(t, common.Timestamp(tp+int64(720*time.Hour/1e9)), afterAlloc.mustBase().Expiration, "Allocation expiration should be increased")

		expectedAlloc := beforeAlloc
		expectedAlloc.Tx = afterAlloc.mustBase().Tx
		expectedAlloc.Expiration = afterAlloc.mustBase().Expiration
		compareAllocationData(t, *expectedAlloc, *afterAlloc.mustBase())
	})

	t.Run("Extend used allocation duration should work with adding extra payment", func(t *testing.T) {
		var (
			tp     = int64(10)
			client = newClient(200000*x10, balances)

			// Allocation
			beforeAlloc, _ = setupAllocationWithMockStats(t, ssc, client, tp, balances, false, false, false)
			allocID        = beforeAlloc.ID
		)

		require.Equal(t, 0, int(beforeAlloc.WritePool), "Write pool should be zero")

		// extend
		var uar updateAllocationRequest
		uar.ID = allocID
		uar.Extend = true
		tp += int64(360 * time.Hour / 1e9)

		resp, err := uar.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
		require.Error(t, err)
		resp, err = uar.callUpdateAllocReq(t, client.id, 50*x10, tp, ssc, balances)
		require.NoError(t, err)

		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		afterAllocBase := afterAlloc.mustBase()

		require.EqualValues(t, afterAlloc, &deco, "Response and allocation in MPT should be same")
		assert.NotEqual(t, beforeAlloc.Tx, afterAllocBase.Tx, "Transaction should be updated")

		assert.Equal(t, common.Timestamp(tp+int64(720*time.Hour/1e9)), afterAllocBase.Expiration, "Allocation expiration should be increased")
		require.Equal(t, 0, int(afterAllocBase.WritePool), "Write pool should be updated")

		cp, err := ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)
		require.Equal(t, 150*x10, int(cp.Balance), "Write pool should be updated")

		expectedAlloc := beforeAlloc
		expectedAlloc.Tx = afterAllocBase.Tx
		expectedAlloc.Expiration = afterAllocBase.Expiration
		expectedAlloc.WritePool = afterAllocBase.WritePool
		expectedAlloc.MovedToChallenge = afterAllocBase.MovedToChallenge
		for _, ba := range expectedAlloc.BlobberAllocs {
			ba.ChallengePoolIntegralValue += ba.ChallengePoolIntegralValue / 2
		}
		compareAllocationData(t, *expectedAlloc, *afterAllocBase)
	})

	t.Run("Upgrade size in unused allocation should work", func(t *testing.T) {
		var (
			tp     = int64(0)
			client = newClient(2000*x10, balances)

			// Allocation
			beforeAlloc, _ = setupAllocationWithMockStats(t, ssc, client, tp, balances, true, false, false)
			allocID        = beforeAlloc.ID
		)

		// upgrade
		var uar updateAllocationRequest
		uar.ID = allocID
		uar.Size = 10 * GB
		tp += int64(360 * time.Hour / 1e9)
		resp, err := uar.callUpdateAllocReq(t, client.id, 100*x10, tp, ssc, balances)
		require.NoError(t, err)

		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		afterAllocBase := afterAlloc.mustBase()

		require.EqualValues(t, afterAlloc, &deco, "Response and allocation in MPT should be same")
		assert.NotEqual(t, beforeAlloc.Tx, afterAllocBase.Tx, "Transaction should be updated")

		assert.Equal(t, int64(20*GB), afterAllocBase.Size, "Allocation size should be increased")
		require.Equal(t, 200*x10, int(afterAllocBase.WritePool), "Write pool should be updated")
		assert.Equal(t, common.Timestamp(tp+int64(720*time.Hour/1e9)), afterAllocBase.Expiration, "Allocation expiration should be increased")

		expectedAlloc := beforeAlloc
		expectedAlloc.Tx = afterAllocBase.Tx
		expectedAlloc.Expiration = afterAllocBase.Expiration
		expectedAlloc.WritePool = afterAllocBase.WritePool
		expectedAlloc.Size = afterAllocBase.Size
		for _, ba := range expectedAlloc.BlobberAllocs {
			ba.Size += uar.Size / int64(afterAllocBase.DataShards)
		}
		compareAllocationData(t, *expectedAlloc, *afterAllocBase)
	})

	t.Run("Upgrade size in used allocation should work", func(t *testing.T) {
		var (
			tp     = int64(10)
			client = newClient(200000*x10, balances)

			// Allocation
			beforeAlloc, _ = setupAllocationWithMockStats(t, ssc, client, tp, balances, false, false, false)
			allocID        = beforeAlloc.ID
		)

		require.Equal(t, 0, int(beforeAlloc.WritePool), "Write pool should be zero")

		// upgrade
		var uar updateAllocationRequest
		uar.ID = allocID
		uar.Size = 10 * GB
		tp += int64(360 * time.Hour / 1e9)

		resp, err := uar.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
		require.Error(t, err)
		resp, err = uar.callUpdateAllocReq(t, client.id, 150*x10, tp, ssc, balances)
		require.NoError(t, err)

		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		afterAllocBase := afterAlloc.mustBase()

		require.EqualValues(t, afterAlloc, &deco, "Response and allocation in MPT should be same")
		assert.NotEqual(t, beforeAlloc.Tx, afterAllocBase.Tx, "Transaction should be updated")

		assert.Equal(t, int64(20*GB), afterAllocBase.Size, "Allocation size should be increased")
		require.Equal(t, 100*x10, int(afterAllocBase.WritePool), "Write pool should be updated")
		assert.Equal(t, common.Timestamp(tp+int64(720*time.Hour/1e9)), afterAllocBase.Expiration, "Allocation expiration should be increased")

		cp, err := ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)
		require.Equal(t, 150*x10, int(cp.Balance), "Write pool should be updated")

		expectedAlloc := beforeAlloc
		expectedAlloc.Tx = afterAllocBase.Tx
		expectedAlloc.Expiration = afterAllocBase.Expiration
		expectedAlloc.WritePool = afterAllocBase.WritePool
		expectedAlloc.Size = afterAllocBase.Size
		expectedAlloc.MovedToChallenge = afterAllocBase.MovedToChallenge
		for _, ba := range expectedAlloc.BlobberAllocs {
			ba.ChallengePoolIntegralValue += ba.ChallengePoolIntegralValue / 2
			ba.Size += uar.Size / int64(afterAllocBase.DataShards)
		}
		compareAllocationData(t, *expectedAlloc, *afterAllocBase)
	})

	t.Run("Add blobber to unused allocation should work", func(t *testing.T) {
		var (
			tp     = int64(0)
			client = newClient(2000*x10, balances)

			// Allocation
			beforeAlloc, _ = setupAllocationWithMockStats(t, ssc, client, tp, balances, true, false, false)
			allocID        = beforeAlloc.ID
		)

		nb3 := addBlobber(t, ssc, 3*GB, tp, avgTerms, 50*x10, balances, false, false)

		// add blobber
		var uar updateAllocationRequest
		uar.ID = allocID
		uar.AddBlobberId = nb3.id
		uar.AddBlobberAuthTicket = ""
		resp, err := uar.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
		require.Error(t, err)

		resp, err = uar.callUpdateAllocReq(t, client.id, 5*x10, tp, ssc, balances)
		require.NoError(t, err)

		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		afterAllocBase := afterAlloc.mustBase()

		require.EqualValues(t, afterAlloc, &deco, "Response and allocation in MPT should be same")
		assert.NotEqual(t, beforeAlloc.Tx, afterAllocBase.Tx, "Transaction should be updated")
		assert.Equal(t, 21, len(afterAllocBase.BlobberAllocs), "Blobber should be added to the allocation")

		expectedAlloc := beforeAlloc
		expectedAlloc.Tx = afterAllocBase.Tx
		expectedAlloc.ParityShards = 11
		expectedAlloc.WritePool = afterAllocBase.WritePool
		randAllocDeepCopy := afterAlloc.deepCopy(t)
		expectedAlloc.BlobberAllocs = append(expectedAlloc.BlobberAllocs, randAllocDeepCopy.mustBase().BlobberAllocs[0])
		expectedAlloc.BlobberAllocs[len(expectedAlloc.BlobberAllocs)-1].BlobberID = nb3.id
		compareAllocationData(t, *expectedAlloc, *afterAllocBase)
	})

	// Enterprise Allocation Tests

	t.Run("Enterprise : Extend unused allocation duration should work", func(t *testing.T) {
		var (
			tp     = int64(0)
			client = newClient(2000*x10, balances)

			// Allocation
			beforeAlloc, _ = setupAllocationWithMockStats(t, ssc, client, tp, balances, true, true, true)
			allocID        = beforeAlloc.ID
		)

		allocSizePerBlobber := int64(1)
		expectedRewardPerBlobber := float64(0)

		allocWpBalance := 20 * allocSizePerBlobber * x10
		require.Equal(t, allocWpBalance, int64(beforeAlloc.WritePool), "Write pool should be 20")

		checkStakesRewardsAre0ForAlloc(beforeAlloc, ssc, t, balances)

		// upgrade
		var uar updateAllocationRequest
		uar.ID = allocID
		uar.Extend = true
		tp += int64(360 * time.Hour / 1e9)

		expectedRewardPerBlobber += 0.5 * x10
		allocWpBalance /= 2                                               // Update alloc after using 50% of time
		requiredWpBalance := 20*allocSizePerBlobber*x10 + -allocWpBalance // One blobber has double write price

		resp, err := uar.callUpdateAllocReq(t, client.id, currency.Coin(requiredWpBalance), tp, ssc, balances) // 10 is paid as reward and new alloc cost is 200 with 50 already in WP.
		require.NoError(t, err)
		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		allocWpBalance += requiredWpBalance
		require.Equal(t, allocWpBalance, int64(deco.mustBase().WritePool), "Write pool should be updated")

		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		for _, ba := range afterAlloc.mustBase().BlobberAllocs {
			sp, err := ssc.getStakePool(spenum.Blobber, ba.BlobberID, balances)
			require.NoError(t, err)

			require.Equal(t, int(expectedRewardPerBlobber*0.3), int(sp.Reward), "30% service charge to blobber should be updated")
			require.Len(t, sp.Pools, 1, "Single delegate pool")
			// get key of the delegate pool
			var dpKey string
			for k := range sp.Pools {
				dpKey = k
				break
			}
			require.Equal(t, int(expectedRewardPerBlobber*0.7), int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")
		}

		afterAlloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		afterAllocBase := afterAlloc.mustBase()

		require.EqualValues(t, afterAlloc, &deco, "Response and allocation in MPT should be same")
		assert.NotEqual(t, beforeAlloc.Tx, afterAllocBase.Tx, "Transaction should be updated")

		assert.Equal(t, true, *afterAlloc.Entity().(*storageAllocationV3).IsEnterprise, "enterprise should be true")
		assert.Equal(t, int64(10*GB), afterAllocBase.Size, "Allocation size should be increased")
		require.Equal(t, allocWpBalance, int64(afterAllocBase.WritePool), "Write pool should be updated")
		assert.Equal(t, common.Timestamp(tp+int64(720*time.Hour/1e9)), afterAllocBase.Expiration, "Allocation expiration should be increased")

		expectedAlloc := beforeAlloc
		expectedAlloc.Tx = afterAllocBase.Tx
		expectedAlloc.Expiration = afterAllocBase.Expiration
		expectedAlloc.WritePool = afterAllocBase.WritePool
		expectedAlloc.Size = afterAllocBase.Size
		for _, ba := range expectedAlloc.BlobberAllocs {
			ba.Size += (uar.Size * 2) / int64(afterAllocBase.DataShards)
		}
		expectedAlloc.WritePool = afterAlloc.mustBase().WritePool
		compareAllocationData(t, *expectedAlloc, *afterAllocBase)
	})

	t.Run("Enterprise : Price Change : Extend unused allocation duration should work", func(t *testing.T) {
		var (
			tp     = int64(0)
			client = newClient(2000*x10, balances)

			// Allocation
			beforeAlloc, _ = setupAllocationWithMockStats(t, ssc, client, tp, balances, true, true, true)
			allocID        = beforeAlloc.ID
		)

		allocSizePerBlobber := int64(1)
		expectedRewardPerBlobber := float64(0)

		allocWpBalance := 20 * allocSizePerBlobber * x10
		require.Equal(t, allocWpBalance, int64(beforeAlloc.WritePool), "Write pool should be 20")

		// change price
		b1, err := getBlobber(beforeAlloc.BlobberAllocs[0].BlobberID, balances)
		require.NoError(t, err)
		b1.Update(&storageNodeV4{}, func(e entitywrapper.EntityI) error {
			b := e.(*storageNodeV4)
			b.Terms.WritePrice *= 2
			return nil
		})
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		checkStakesRewardsAre0ForAlloc(beforeAlloc, ssc, t, balances)

		// upgrade
		var uar updateAllocationRequest
		uar.ID = allocID
		uar.Extend = true
		tp += int64(360 * time.Hour / 1e9)

		expectedRewardPerBlobber += 0.5 * x10
		allocWpBalance /= 2                                                                            // Update alloc after using 50% of time
		requiredWpBalance := 19*allocSizePerBlobber*x10 + 1*2*allocSizePerBlobber*x10 - allocWpBalance // One blobber has double write price

		resp, err := uar.callUpdateAllocReq(t, client.id, currency.Coin(requiredWpBalance), tp, ssc, balances) // 10 is paid as reward and new alloc cost is 200 with 50 already in WP.
		require.NoError(t, err)
		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		allocWpBalance += requiredWpBalance
		require.Equal(t, allocWpBalance, int64(deco.mustBase().WritePool), "Write pool should be updated")

		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		for _, ba := range afterAlloc.mustBase().BlobberAllocs {
			sp, err := ssc.getStakePool(spenum.Blobber, ba.BlobberID, balances)
			require.NoError(t, err)

			require.Equal(t, int(expectedRewardPerBlobber*0.3), int(sp.Reward), "30% service charge to blobber should be updated")
			require.Len(t, sp.Pools, 1, "Single delegate pool")
			// get key of the delegate pool
			var dpKey string
			for k := range sp.Pools {
				dpKey = k
				break
			}
			require.Equal(t, int(expectedRewardPerBlobber*0.7), int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")
		}

		tp += int64(360 * time.Hour / 1e9)

		expectedRewardPerBlobber += 0.5 * x10
		allocWpBalance /= 2
		requiredWpBalance = 19*allocSizePerBlobber*x10 + 1*2*allocSizePerBlobber*x10 - allocWpBalance // One blobber has double write price

		resp, err = uar.callUpdateAllocReq(t, client.id, currency.Coin(requiredWpBalance), tp, ssc, balances) // 50 is paid as reward and new alloc cost is 200 with 50 already in WP.
		require.NoError(t, err)
		require.NoError(t, deco.Decode([]byte(resp)))

		allocWpBalance += requiredWpBalance
		require.Equal(t, allocWpBalance, int64(deco.mustBase().WritePool), "Write pool should be updated")

		afterAlloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		for _, ba := range afterAlloc.mustBase().BlobberAllocs {
			sp, err := ssc.getStakePool(spenum.Blobber, ba.BlobberID, balances)
			require.NoError(t, err)

			expectedReward := expectedRewardPerBlobber

			if ba.BlobberID == b1.Id() {
				expectedReward += 0.5 * x10
			}

			require.Equal(t, int(expectedReward*0.3), int(sp.Reward), "30% service charge to blobber should be updated")
			require.Len(t, sp.Pools, 1, "Single delegate pool")
			// get key of the delegate pool
			var dpKey string
			for k := range sp.Pools {
				dpKey = k
				break
			}
			require.Equal(t, int(expectedReward*0.7), int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")
		}

		afterAllocBase := afterAlloc.mustBase()

		require.EqualValues(t, afterAlloc, &deco, "Response and allocation in MPT should be same")
		assert.NotEqual(t, beforeAlloc.Tx, afterAllocBase.Tx, "Transaction should be updated")

		assert.Equal(t, true, *afterAlloc.Entity().(*storageAllocationV3).IsEnterprise, "enterprise should be true")
		assert.Equal(t, int64(10*GB), afterAllocBase.Size, "Allocation size should be increased")
		require.Equal(t, allocWpBalance, int64(afterAllocBase.WritePool), "Write pool should be updated")
		assert.Equal(t, common.Timestamp(tp+int64(720*time.Hour/1e9)), afterAllocBase.Expiration, "Allocation expiration should be increased")

		expectedAlloc := beforeAlloc
		expectedAlloc.Tx = afterAllocBase.Tx
		expectedAlloc.Expiration = afterAllocBase.Expiration
		expectedAlloc.WritePool = afterAllocBase.WritePool
		expectedAlloc.Size = afterAllocBase.Size
		for _, ba := range expectedAlloc.BlobberAllocs {
			ba.Size += (uar.Size * 2) / int64(afterAllocBase.DataShards)
		}
		expectedAlloc.BlobberAllocs[0].Terms.WritePrice = b1.mustBase().Terms.WritePrice
		expectedAlloc.BlobberAllocsMap[b1.Id()].Terms.WritePrice = b1.mustBase().Terms.WritePrice
		expectedAlloc.WritePool = afterAlloc.mustBase().WritePool
		compareAllocationData(t, *expectedAlloc, *afterAllocBase)
	})

	t.Run("Enterprise : Upgrade size in unused allocation should work", func(t *testing.T) {
		var (
			tp     = int64(0)
			client = newClient(2000*x10, balances)

			// Allocation
			beforeAlloc, _ = setupAllocationWithMockStats(t, ssc, client, tp, balances, true, true, true)
			allocID        = beforeAlloc.ID
		)

		allocSizePerBlobber := int64(1)
		expectedRewardPerBlobber := float64(0)

		allocWpBalance := 20 * allocSizePerBlobber * x10
		require.Equal(t, allocWpBalance, int64(beforeAlloc.WritePool), "Write pool should be 20")

		checkStakesRewardsAre0ForAlloc(beforeAlloc, ssc, t, balances)

		// upgrade
		var uar updateAllocationRequest
		uar.ID = allocID
		uar.Size = 10 * GB
		tp += int64(360 * time.Hour / 1e9)

		allocSizePerBlobber += 1
		expectedRewardPerBlobber += 0.5 * x10
		allocWpBalance /= 2                                               // Update alloc after using 50% of time
		requiredWpBalance := 20*allocSizePerBlobber*x10 + -allocWpBalance // One blobber has double write price

		resp, err := uar.callUpdateAllocReq(t, client.id, currency.Coin(requiredWpBalance), tp, ssc, balances) // 10 is paid as reward and new alloc cost is 200 with 50 already in WP.
		require.NoError(t, err)
		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		allocWpBalance += requiredWpBalance
		require.Equal(t, allocWpBalance, int64(deco.mustBase().WritePool), "Write pool should be updated")

		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		for _, ba := range afterAlloc.mustBase().BlobberAllocs {
			sp, err := ssc.getStakePool(spenum.Blobber, ba.BlobberID, balances)
			require.NoError(t, err)

			require.Equal(t, int(expectedRewardPerBlobber*0.3), int(sp.Reward), "30% service charge to blobber should be updated")
			require.Len(t, sp.Pools, 1, "Single delegate pool")
			// get key of the delegate pool
			var dpKey string
			for k := range sp.Pools {
				dpKey = k
				break
			}
			require.Equal(t, int(expectedRewardPerBlobber*0.7), int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")
		}

		afterAllocBase := afterAlloc.mustBase()

		require.EqualValues(t, afterAlloc, &deco, "Response and allocation in MPT should be same")
		assert.NotEqual(t, beforeAlloc.Tx, afterAllocBase.Tx, "Transaction should be updated")

		assert.Equal(t, true, *afterAlloc.Entity().(*storageAllocationV3).IsEnterprise, "enterprise should be true")
		assert.Equal(t, int64(20*GB), afterAllocBase.Size, "Allocation size should be increased")
		require.Equal(t, allocWpBalance, int64(afterAllocBase.WritePool), "Write pool should be updated")
		assert.Equal(t, common.Timestamp(tp+int64(720*time.Hour/1e9)), afterAllocBase.Expiration, "Allocation expiration should be increased")

		expectedAlloc := beforeAlloc
		expectedAlloc.Tx = afterAllocBase.Tx
		expectedAlloc.Expiration = afterAllocBase.Expiration
		expectedAlloc.WritePool = afterAllocBase.WritePool
		expectedAlloc.Size = afterAllocBase.Size
		for _, ba := range expectedAlloc.BlobberAllocs {
			ba.Size = (afterAllocBase.Size) / int64(afterAllocBase.DataShards)
		}
		expectedAlloc.WritePool = afterAlloc.mustBase().WritePool
		compareAllocationData(t, *expectedAlloc, *afterAllocBase)
	})

	t.Run("Enterprise : Price Change : Upgrade size in unused allocation should work", func(t *testing.T) {
		var (
			tp     = int64(0)
			client = newClient(2000*x10, balances)

			// Allocation
			beforeAlloc, _ = setupAllocationWithMockStats(t, ssc, client, tp, balances, true, true, true)
			allocID        = beforeAlloc.ID
		)

		allocSizePerBlobber := int64(1)
		expectedRewardPerBlobber := float64(0)

		allocWpBalance := 20 * allocSizePerBlobber * x10
		require.Equal(t, allocWpBalance, int64(beforeAlloc.WritePool), "Write pool should be 20")

		// change price
		b1, err := getBlobber(beforeAlloc.BlobberAllocs[0].BlobberID, balances)
		require.NoError(t, err)
		b1.Update(&storageNodeV4{}, func(e entitywrapper.EntityI) error {
			b := e.(*storageNodeV4)
			b.Terms.WritePrice *= 2
			return nil
		})
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		checkStakesRewardsAre0ForAlloc(beforeAlloc, ssc, t, balances)

		// upgrade
		var uar updateAllocationRequest
		uar.ID = allocID
		uar.Size = 10 * GB
		tp += int64(360 * time.Hour / 1e9)

		allocSizePerBlobber += 1
		expectedRewardPerBlobber += 0.5 * x10
		allocWpBalance /= 2                                                                            // Update alloc after using 50% of time
		requiredWpBalance := 19*allocSizePerBlobber*x10 + 1*2*allocSizePerBlobber*x10 - allocWpBalance // One blobber has double write price

		resp, err := uar.callUpdateAllocReq(t, client.id, currency.Coin(requiredWpBalance), tp, ssc, balances) // 10 is paid as reward and new alloc cost is 200 with 50 already in WP.
		require.NoError(t, err)
		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		allocWpBalance += requiredWpBalance
		require.Equal(t, allocWpBalance, int64(deco.mustBase().WritePool), "Write pool should be updated")

		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		for _, ba := range afterAlloc.mustBase().BlobberAllocs {
			sp, err := ssc.getStakePool(spenum.Blobber, ba.BlobberID, balances)
			require.NoError(t, err)

			require.Equal(t, int(expectedRewardPerBlobber*0.3), int(sp.Reward), "30% service charge to blobber should be updated")
			require.Len(t, sp.Pools, 1, "Single delegate pool")
			// get key of the delegate pool
			var dpKey string
			for k := range sp.Pools {
				dpKey = k
				break
			}
			require.Equal(t, int(expectedRewardPerBlobber*0.7), int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")
		}

		uar.Size = 10 * GB
		tp += int64(360 * time.Hour / 1e9)

		allocSizePerBlobber += 1
		expectedRewardPerBlobber += 1 * x10
		allocWpBalance /= 2
		requiredWpBalance = 19*allocSizePerBlobber*x10 + 1*2*allocSizePerBlobber*x10 - allocWpBalance // One blobber has double write price

		resp, err = uar.callUpdateAllocReq(t, client.id, currency.Coin(requiredWpBalance), tp, ssc, balances) // 50 is paid as reward and new alloc cost is 200 with 50 already in WP.
		require.NoError(t, err)
		require.NoError(t, deco.Decode([]byte(resp)))

		allocWpBalance += requiredWpBalance
		require.Equal(t, allocWpBalance, int64(deco.mustBase().WritePool), "Write pool should be updated")

		afterAlloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		for _, ba := range afterAlloc.mustBase().BlobberAllocs {
			sp, err := ssc.getStakePool(spenum.Blobber, ba.BlobberID, balances)
			require.NoError(t, err)

			expectedReward := expectedRewardPerBlobber

			if ba.BlobberID == b1.Id() {
				expectedReward += 1 * x10
			}

			require.Equal(t, int(expectedReward*0.3), int(sp.Reward), "30% service charge to blobber should be updated")
			require.Len(t, sp.Pools, 1, "Single delegate pool")
			// get key of the delegate pool
			var dpKey string
			for k := range sp.Pools {
				dpKey = k
				break
			}
			require.Equal(t, int(expectedReward*0.7), int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")
		}

		afterAllocBase := afterAlloc.mustBase()

		require.EqualValues(t, afterAlloc, &deco, "Response and allocation in MPT should be same")
		assert.NotEqual(t, beforeAlloc.Tx, afterAllocBase.Tx, "Transaction should be updated")

		assert.Equal(t, true, *afterAlloc.Entity().(*storageAllocationV3).IsEnterprise, "enterprise should be true")
		assert.Equal(t, int64(30*GB), afterAllocBase.Size, "Allocation size should be increased")
		require.Equal(t, allocWpBalance, int64(afterAllocBase.WritePool), "Write pool should be updated")
		assert.Equal(t, common.Timestamp(tp+int64(720*time.Hour/1e9)), afterAllocBase.Expiration, "Allocation expiration should be increased")

		expectedAlloc := beforeAlloc
		expectedAlloc.Tx = afterAllocBase.Tx
		expectedAlloc.Expiration = afterAllocBase.Expiration
		expectedAlloc.WritePool = afterAllocBase.WritePool
		expectedAlloc.Size = afterAllocBase.Size
		for _, ba := range expectedAlloc.BlobberAllocs {
			ba.Size += (uar.Size * 2) / int64(afterAllocBase.DataShards)
		}
		expectedAlloc.BlobberAllocs[0].Terms.WritePrice = b1.mustBase().Terms.WritePrice
		expectedAlloc.BlobberAllocsMap[b1.Id()].Terms.WritePrice = b1.mustBase().Terms.WritePrice
		expectedAlloc.WritePool = afterAlloc.mustBase().WritePool
		compareAllocationData(t, *expectedAlloc, *afterAllocBase)
	})

	t.Run("Enterprise : Add blobber to unused allocation should work with extra payment", func(t *testing.T) {
		var (
			tp     = int64(0)
			client = newClient(2000*x10, balances)

			// Allocation
			beforeAlloc, _ = setupAllocationWithMockStats(t, ssc, client, tp, balances, true, true, true)
			allocID        = beforeAlloc.ID
		)
		checkStakesRewardsAre0ForAlloc(beforeAlloc, ssc, t, balances)

		nb3 := addBlobber(t, ssc, 3*GB, tp, Terms{
			WritePrice: 1 * x10,
			ReadPrice:  1 * x10,
		}, 50*x10, balances, false, false)

		checkStakesRewardsAre0ForBlobber(nb3.id, ssc, t, balances)

		// add blobber
		var uar updateAllocationRequest
		uar.ID = allocID
		uar.AddBlobberId = nb3.id
		uar.AddBlobberAuthTicket = ""
		resp, err := uar.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
		expectedErr := common.NewError("allocation_updating_failed", fmt.Sprintf("blobber %s is not enterprise", nb3.id))
		require.ErrorIs(t, expectedErr, err)

		resp, err = uar.callUpdateAllocReq(t, client.id, 5*x10, tp, ssc, balances)
		expectedErr = common.NewError("allocation_updating_failed", fmt.Sprintf("blobber %s is not enterprise", nb3.id))
		require.ErrorIs(t, expectedErr, err)

		blobber3, err := ssc.getBlobber(nb3.id, balances)
		require.NoError(t, err)
		blobber3.Update(&storageNodeV4{}, func(e entitywrapper.EntityI) error {
			b := e.(*storageNodeV4)
			b.IsEnterprise = new(bool)
			*b.IsEnterprise = true
			return nil
		})
		_, err = balances.InsertTrieNode(blobber3.GetKey(), blobber3)
		require.NoError(t, err)

		resp, err = uar.callUpdateAllocReq(t, client.id, 5*x10, tp, ssc, balances)
		expectedErr = common.NewError("allocation_updating_failed", fmt.Sprintf("blobber %s auth ticket verification failed: invalid_auth_ticket: empty auth ticket", nb3.id))
		require.ErrorIs(t, expectedErr, err)

		b3AuthTicket, err := nb3.scheme.Sign(client.id)
		require.NoError(t, err)
		uar.AddBlobberAuthTicket = b3AuthTicket

		resp, err = uar.callUpdateAllocReq(t, client.id, 5*x10, tp, ssc, balances)
		require.NoError(t, err)

		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		for _, ba := range afterAlloc.mustBase().BlobberAllocs {
			sp, err := ssc.getStakePool(spenum.Blobber, ba.BlobberID, balances)
			require.NoError(t, err)

			require.Equal(t, 0, int(sp.Reward), "30% service charge to blobber should be updated")
			require.Len(t, sp.Pools, 1, "Single delegate pool")
			// get key of the delegate pool
			var dpKey string
			for k := range sp.Pools {
				dpKey = k
				break
			}
			require.Equal(t, 0, int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")
		}

		// Added blobber should not get any rewards
		sp, err := ssc.getStakePool(spenum.Blobber, nb3.id, balances)
		require.NoError(t, err)

		require.Equal(t, 0, int(sp.Reward), "30% service charge to blobber should be updated")
		require.Len(t, sp.Pools, 1, "Single delegate pool")
		// get key of the delegate pool
		var dpKey string
		for k := range sp.Pools {
			dpKey = k
			break
		}
		require.Equal(t, 0, int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")

		afterAllocBase := afterAlloc.mustBase()

		assert.Equal(t, true, *afterAlloc.Entity().(*storageAllocationV3).IsEnterprise, "enterprise should be true")
		require.EqualValues(t, afterAlloc, &deco, "Response and allocation in MPT should be same")
		assert.NotEqual(t, beforeAlloc.Tx, afterAllocBase.Tx, "Transaction should be updated")
		assert.Equal(t, 21, len(afterAllocBase.BlobberAllocs), "Blobber should be added to the allocation")

		expectedAlloc := beforeAlloc
		expectedAlloc.Tx = afterAllocBase.Tx
		expectedAlloc.ParityShards = 11
		expectedAlloc.WritePool = afterAllocBase.WritePool
		randAllocDeepCopy := afterAlloc.deepCopy(t)
		expectedAlloc.BlobberAllocs = append(expectedAlloc.BlobberAllocs, randAllocDeepCopy.mustBase().BlobberAllocs[0])
		expectedAlloc.BlobberAllocs[len(expectedAlloc.BlobberAllocs)-1].BlobberID = nb3.id
		compareAllocationData(t, *expectedAlloc, *afterAllocBase)
	})

	t.Run("Enterprise : Replace blobber to unused allocation should work without extra payment", func(t *testing.T) {
		var (
			tp     = int64(0)
			client = newClient(2000*x10, balances)

			// Allocation
			beforeAlloc, _ = setupAllocationWithMockStats(t, ssc, client, tp, balances, true, true, true)
			allocID        = beforeAlloc.ID
		)
		checkStakesRewardsAre0ForAlloc(beforeAlloc, ssc, t, balances)

		nb3 := addBlobber(t, ssc, 3*GB, tp, Terms{
			WritePrice: 1 * x10,
			ReadPrice:  1 * x10,
		}, 50*x10, balances, true, false)
		checkStakesRewardsAre0ForBlobber(nb3.id, ssc, t, balances)

		// add blobber
		var uar updateAllocationRequest
		uar.ID = allocID
		uar.AddBlobberId = nb3.id
		uar.AddBlobberAuthTicket = ""
		uar.RemoveBlobberId = beforeAlloc.BlobberAllocs[0].BlobberID
		resp, err := uar.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
		expectedErr := common.NewError("allocation_updating_failed", fmt.Sprintf("blobber %s is not enterprise", nb3.id))
		require.ErrorIs(t, expectedErr, err)

		blobber3, err := ssc.getBlobber(nb3.id, balances)
		require.NoError(t, err)
		blobber3.Update(&storageNodeV4{}, func(e entitywrapper.EntityI) error {
			b := e.(*storageNodeV4)
			b.IsEnterprise = new(bool)
			*b.IsEnterprise = true
			return nil
		})
		_, err = balances.InsertTrieNode(blobber3.GetKey(), blobber3)
		require.NoError(t, err)

		resp, err = uar.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
		expectedErr = common.NewError("allocation_updating_failed", fmt.Sprintf("blobber %s auth ticket verification failed: invalid_auth_ticket: empty auth ticket", nb3.id))
		require.ErrorIs(t, expectedErr, err)

		b3AuthTicket, err := nb3.scheme.Sign(client.id)
		require.NoError(t, err)
		uar.AddBlobberAuthTicket = b3AuthTicket

		tp = int64(beforeAlloc.Expiration) / 2
		resp, err = uar.callUpdateAllocReq(t, client.id, 5*x10, tp, ssc, balances)
		require.NoError(t, err)

		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		for _, ba := range afterAlloc.mustBase().BlobberAllocs {
			sp, err := ssc.getStakePool(spenum.Blobber, ba.BlobberID, balances)
			require.NoError(t, err)

			require.Equal(t, 0, int(sp.Reward), "30% service charge to blobber should be updated")
			require.Len(t, sp.Pools, 1, "Single delegate pool")
			// get key of the delegate pool
			var dpKey string
			for k := range sp.Pools {
				dpKey = k
				break
			}
			require.Equal(t, 0, int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")
		}

		// Replaced blobber should get rewards
		sp, err := ssc.getStakePool(spenum.Blobber, uar.RemoveBlobberId, balances)
		require.NoError(t, err)

		require.Equal(t, int(0.15*x10), int(sp.Reward), "30% service charge to blobber should be updated") // Half of reward as half used alloc
		require.Len(t, sp.Pools, 1, "Single delegate pool")
		// get key of the delegate pool
		var dpKey string
		for k := range sp.Pools {
			dpKey = k
			break
		}
		require.Equal(t, int(0.35*x10), int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated") // Half of rewards as half used alloc

		// Added blobber should not get any rewards
		sp, err = ssc.getStakePool(spenum.Blobber, nb3.id, balances)
		require.NoError(t, err)

		require.Equal(t, 0, int(sp.Reward), "30% service charge to blobber should be updated")
		require.Len(t, sp.Pools, 1, "Single delegate pool")
		// get key of the delegate pool
		for k := range sp.Pools {
			dpKey = k
			break
		}
		require.Equal(t, 0, int(sp.Pools[dpKey].Reward), "70% reward to delegate pool should be updated")

		afterAllocBase := afterAlloc.mustBase()

		assert.Equal(t, true, *afterAlloc.Entity().(*storageAllocationV3).IsEnterprise, "enterprise should be true")
		require.EqualValues(t, afterAlloc, &deco, "Response and allocation in MPT should be same")
		assert.NotEqual(t, beforeAlloc.Tx, afterAllocBase.Tx, "Transaction should be updated")
		assert.Equal(t, 20, len(afterAllocBase.BlobberAllocs), "Blobber should be added to the allocation")

		expectedAlloc := beforeAlloc
		expectedAlloc.Tx = afterAllocBase.Tx
		expectedAlloc.ParityShards = 10
		expectedAlloc.WritePool = afterAllocBase.WritePool
		expectedAlloc.BlobberAllocs[0].BlobberID = nb3.id
		expectedAlloc.BlobberAllocs[0].LatestSuccessfulChallCreatedAt = afterAlloc.mustBase().BlobberAllocs[0].LatestSuccessfulChallCreatedAt
		expectedAlloc.BlobberAllocs[0].LatestFinalizedChallCreatedAt = afterAlloc.mustBase().BlobberAllocs[0].LatestFinalizedChallCreatedAt
		compareAllocationData(t, *expectedAlloc, *afterAllocBase)
	})
}

func TestStorageSmartContract_updateAllocationRequest(t *testing.T) {

	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(t, false)
		client         = newClient(2000*x10, balances)
		otherClient    = newClient(50*x10, balances)
		tp             = int64(0)
		allocID, blobs = addAllocation(t, ssc, client, tp, 0, 0, 0, 0, 0, balances, false, false, false)
		resp           string
		err            error
	)

	confMinAllocSize := 1024
	mockBlobberCapacity := 2000 * confMinAllocSize

	sa, err := ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	alloc := sa.mustBase()

	alloc.Stats = &StorageAllocationStats{
		UsedSize:          int64(alloc.DataShards+alloc.ParityShards) * int64(mockBlobberCapacity) / 2,
		SuccessChallenges: int64(alloc.DataShards+alloc.ParityShards) * 100,
		FailedChallenges:  int64(alloc.DataShards+alloc.ParityShards) * 2,
		TotalChallenges:   int64(alloc.DataShards+alloc.ParityShards) * 102,
		OpenChallenges:    0,
	}

	for _, ba := range alloc.BlobberAllocs {
		ba.Stats = &StorageAllocationStats{
			UsedSize:          int64(mockBlobberCapacity) / 2,
			SuccessChallenges: 100,
			FailedChallenges:  2,
			TotalChallenges:   102,
			OpenChallenges:    0,
		}

		ba.LatestFinalizedChallCreatedAt = alloc.Expiration / 2
		ba.ChallengePoolIntegralValue = 0

		blobber, err := ssc.getBlobber(ba.BlobberID, balances)
		require.NoError(t, err)

		bb := blobber.mustBase()

		bb.SavedData = int64(mockBlobberCapacity) / 2

		_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
		require.NoError(t, err)
	}

	_, err = balances.InsertTrieNode(sa.GetKey(ADDRESS), sa)
	if err != nil {
		return
	}

	cp := &StorageAllocation{}
	err = cp.Decode(sa.Encode())
	require.NoError(t, err)

	// change terms
	tp += 1000
	for _, b := range blobs {
		var blob *StorageNode
		blob, err = ssc.getBlobber(b.id, balances)
		require.NoError(t, err)
		blob.mustUpdateBase(func(b *storageNodeBase) error {
			b.Terms.WritePrice = currency.Coin(5 * x10)
			b.Terms.ReadPrice = currency.Coin(0.8 * x10)
			return nil
		})
		_, err = updateBlobber(t, blob, 0, tp, ssc, balances)
		require.NoError(t, err)
	}

	//
	// extend
	//

	var uar updateAllocationRequest
	uar.ID = alloc.ID
	uar.Extend = true
	uar.Size = alloc.Size
	tp += 1000
	resp, err = uar.callUpdateAllocReq(t, client.id, 300*x10, tp, ssc, balances)
	require.NoError(t, err)

	var deco StorageAllocation
	require.NoError(t, deco.Decode([]byte(resp)))

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	alloc = sa.mustBase()

	require.EqualValues(t, alloc, deco.mustBase())

	assert.Equal(t, cp.mustBase().Size*2, alloc.Size)
	assert.Equal(t, common.Timestamp(tp+int64(720*time.Hour/1e9)), alloc.Expiration)

	var tbs int64
	for i, d := range alloc.BlobberAllocs {
		if i == alloc.DataShards {
			break
		}
		tbs += d.Size
	}
	var (
		numb  = int64(alloc.DataShards)
		bsize = int64(math.Ceil(float64(alloc.Size) / float64(numb)))
	)

	assert.True(t, math.Abs(float64(bsize*numb-tbs)) < 100)

	// Owner can extend regardless of the value of `third_party_extendable`
	req := updateAllocationRequest{
		ID:     alloc.ID,
		Size:   100,
		Extend: true,
	}
	tp += 1000
	resp, err = req.callUpdateAllocReq(t, client.id, 20*x10, tp, ssc, balances)
	require.NoError(t, err)

	// Others cannot extend the allocation if `third_party_extendable` = false
	req = updateAllocationRequest{
		ID:     alloc.ID,
		Size:   100,
		Extend: true,
	}
	tp += 1000
	resp, err = req.callUpdateAllocReq(t, otherClient.id, 20*x10, tp, ssc, balances)
	require.Error(t, err)
	assert.Equal(t, "allocation_updating_failed: only owner can update the allocation", err.Error())

	// Owner can change `third_party_extendable`
	req = updateAllocationRequest{
		ID:                      alloc.ID,
		SetThirdPartyExtendable: true,
	}
	tp += 1000
	resp, err = req.callUpdateAllocReq(t, client.id, 20*x10, tp, ssc, balances)
	require.NoError(t, err)

	// Others can extend the allocation if `third_party_extendable` = true
	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()

	req = updateAllocationRequest{
		ID:     alloc.ID,
		Size:   100,
		Extend: true,
	}
	tp += 1000
	expectedSize := alloc.Size + 100
	resp, err = req.callUpdateAllocReq(t, otherClient.id, 20*x10, tp, ssc, balances)
	require.NoError(t, err)
	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()
	assert.Equal(t, expectedSize, alloc.Size)
	assert.Equal(t, common.Timestamp(tp+int64(720*time.Hour/1e9)), alloc.Expiration)

	// Other cannot perform any other action than extending.
	req = updateAllocationRequest{
		ID:                 alloc.ID,
		FileOptions:        61,
		FileOptionsChanged: true,
	}
	tp += 1000
	expectedFileOptions := alloc.FileOptions
	resp, err = req.callUpdateAllocReq(t, otherClient.id, 20*x10, tp, ssc, balances)
	require.Error(t, err)

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()
	assert.Equal(t, expectedFileOptions, alloc.FileOptions)

	//
	// add blobber
	//
	tp += 1000
	nb := addBlobber(t, ssc, 2*GB, tp, avgTerms, 50*x10, balances, false, false)
	tp += 1000
	req = updateAllocationRequest{
		ID:           alloc.ID,
		AddBlobberId: nb.id,
	}
	resp, err = req.callUpdateAllocReq(t, client.id, 10000000987, tp, ssc, balances)
	require.NoError(t, err)

	// assert that the new blobber offer is updated
	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()
	nblobAlloc, ok := alloc.BlobberAllocsMap[nb.id]
	require.True(t, ok)

	alloc.BlobberAllocsMap[nb.id].Stats = &StorageAllocationStats{
		UsedSize:          int64(mockBlobberCapacity) / 2,
		SuccessChallenges: 100,
		FailedChallenges:  2,
		TotalChallenges:   102,
		OpenChallenges:    0,
	}

	alloc.BlobberAllocsMap[nb.id].LatestFinalizedChallCreatedAt = common.Timestamp(tp)

	alloc.BlobberAllocsMap[nb.id].ChallengePoolIntegralValue = 0

	_, err = balances.InsertTrieNode(sa.GetKey(ADDRESS), sa)
	if err != nil {
		return
	}

	nsp, err := ssc.getStakePool(spenum.Blobber, nb.id, balances)
	require.NoError(t, err)
	require.Equal(t, nsp.TotalOffers, nblobAlloc.Offer())

	//
	// remove blobber
	//

	tp += 1000
	nb2 := addBlobber(t, ssc, 2*GB, tp, avgTerms, 50*x10, balances, false, false)
	tp += 1000

	req = updateAllocationRequest{
		ID:              alloc.ID,
		AddBlobberId:    nb2.id,
		RemoveBlobberId: nb.id,
	}
	resp, err = req.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
	require.NoError(t, err)

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()

	// assert blobber is removed from allocation
	_, ok = alloc.BlobberAllocsMap[nb.id]
	require.False(t, ok)

	// assert allocation is removed from blobber
	baParts, err := partitionsBlobberAllocations(nb.id, balances)
	require.NoError(t, err)
	var noneIt BlobberAllocationNode
	_, err = baParts.Get(balances, alloc.ID, &noneIt)
	require.True(t, partitions.ErrItemNotFound(err))

	// commit connection to get update challenge ready partition
	// assert there's no challenge ready partition before commit connection
	challengeReadyParts, _, err := partitionsChallengeReadyBlobbers(balances)
	require.NoError(t, err)
	var cit ChallengeReadyBlobber
	_, err = challengeReadyParts.Get(balances, nb2.id, &cit)
	require.True(t, partitions.ErrItemNotFound(err))

	tp += 1000
	// write
	const allocRoot = "alloc-root-1"
	var cc = &BlobberCloseConnection{
		AllocationRoot:     allocRoot,
		PrevAllocationRoot: "",
		WriteMarker:        &WriteMarker{},
	}
	wm1 := &writeMarkerV1{
		AllocationRoot:         allocRoot,
		PreviousAllocationRoot: "",
		AllocationID:           allocID,
		Size:                   10 * 1024 * 1024, // 100 MB
		BlobberID:              nb2.id,
		Timestamp:              common.Timestamp(tp),
		ClientID:               client.id,
	}
	wm1.Signature, err = client.scheme.Sign(
		encryption.Hash(wm1.GetHashData()))
	require.NoError(t, err)
	cc.WriteMarker.SetEntity(wm1)
	var tx = newTransaction(nb2.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc), balances)
	require.NoError(t, err)
	require.NotZero(t, resp)

	// assert nb2 is challenge ready
	challengeReadyParts, _, err = partitionsChallengeReadyBlobbers(balances)
	require.NoError(t, err)
	_, err = challengeReadyParts.Get(balances, nb2.id, &cit)
	require.NoError(t, err)
	require.Equal(t, cit.BlobberID, nb2.id)

	//
	// remove blobber nb2, assert it self is removed from challenge ready partition
	//

	tp += 1000
	nb3 := addBlobber(t, ssc, 3*GB, tp, avgTerms, 50*x10, balances, false, false)

	tp += 1000
	req = updateAllocationRequest{
		ID:              alloc.ID,
		AddBlobberId:    nb3.id,
		RemoveBlobberId: nb2.id,
	}

	alloc.BlobberAllocsMap[nb2.id].Stats = &StorageAllocationStats{
		UsedSize:          int64(mockBlobberCapacity) / 2,
		SuccessChallenges: 100,
		FailedChallenges:  2,
		TotalChallenges:   102,
		OpenChallenges:    0,
	}

	alloc.BlobberAllocsMap[nb2.id].LatestFinalizedChallCreatedAt = common.Timestamp(tp)

	alloc.BlobberAllocsMap[nb2.id].ChallengePoolIntegralValue = 0
	_, err = balances.InsertTrieNode(sa.GetKey(ADDRESS), sa)
	if err != nil {
		return
	}

	resp, err = req.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
	require.NoError(t, err)

	// assert blobber nb2 is removed from challenge ready partition
	challengeReadyParts, _, err = partitionsChallengeReadyBlobbers(balances)
	require.NoError(t, err)
	_, err = challengeReadyParts.Get(balances, nb2.id, &cit)
	require.True(t, partitions.ErrItemNotFound(err))

	//
	// increase duration
	//

	cp = &StorageAllocation{}
	err = cp.Decode(sa.Encode())
	require.NoError(t, err)

	uar.ID = alloc.ID
	uar.Extend = true

	tp += 1000
	resp, err = uar.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
	require.NoError(t, err)
	require.NoError(t, deco.Decode([]byte(resp)))

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()

	require.EqualValues(t, alloc, deco.mustBase())

	assert.Equal(t, common.Timestamp(tp+int64(720*time.Hour/1e9)), alloc.Expiration)

	tbs = 0
	for i, detail := range alloc.BlobberAllocs {
		if i == alloc.DataShards {
			break
		}
		tbs += detail.Size
	}
	numb = int64(alloc.DataShards + alloc.ParityShards)
	bsize = (alloc.Size + (numb - 1)) / numb
	assert.True(t, math.Abs(float64(bsize*numb-tbs)) < 100)

	//
	// change owner and owner public key
	//

	cp = &StorageAllocation{}
	err = cp.Decode(sa.Encode())
	require.NoError(t, err)

	var uarOwnerUpdate = updateAllocationRequest{
		ID:             alloc.ID,
		OwnerID:        otherClient.id,
		OwnerPublicKey: otherClient.pk,
	}

	tp += 1000
	resp, err = uarOwnerUpdate.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
	require.NoError(t, err)
	require.NoError(t, deco.Decode([]byte(resp)))

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()
	require.EqualValues(t, alloc.Owner, otherClient.id)
	require.EqualValues(t, alloc.OwnerPublicKey, otherClient.pk)
	require.EqualValues(t, alloc, deco.mustBase())

	//
	// reduce
	//

	cp = sa.deepCopy(t)

	uar.ID = alloc.ID
	uar.Size = -(alloc.Size / 2)

	tp += 1000
	resp, err = uar.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
	require.Error(t, err)

}

// - finalize allocation
func Test_finalize_allocation(t *testing.T) {

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		client   = newClient(1000*x10, balances)
		tp       = int64(0)
		err      error
	)
	confMinAllocSize := 1024
	mockBlobberCapacity := 2000 * confMinAllocSize

	setConfig(t, balances)

	tp += 1000
	var allocID, blobs = addAllocation(t, ssc, client, tp, 0, 0, 0, 0, 0, balances, false, false, false)

	// blobbers: stake 10k, balance 40k

	sa, err := ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	alloc.Stats = &StorageAllocationStats{
		UsedSize:          int64(alloc.DataShards+alloc.ParityShards) * int64(mockBlobberCapacity) / 2,
		SuccessChallenges: int64(alloc.DataShards+alloc.ParityShards) * 100,
		FailedChallenges:  int64(alloc.DataShards+alloc.ParityShards) * 2,
		TotalChallenges:   int64(alloc.DataShards+alloc.ParityShards) * 102,
		OpenChallenges:    0,
	}

	for _, ba := range alloc.BlobberAllocs {
		ba.Stats = &StorageAllocationStats{
			UsedSize:          int64(mockBlobberCapacity) / 2,
			SuccessChallenges: 100,
			FailedChallenges:  2,
			TotalChallenges:   102,
			OpenChallenges:    0,
		}

		ba.LatestFinalizedChallCreatedAt = 0
		ba.ChallengePoolIntegralValue = 0

		blobber, err := ssc.getBlobber(ba.BlobberID, balances)
		require.NoError(t, err)
		//nolint:errcheck
		blobber.mustUpdateBase(func(b *storageNodeBase) error {
			b.SavedData = int64(mockBlobberCapacity) / 2
			return nil
		})
		_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
		require.NoError(t, err)
	}

	_, err = balances.InsertTrieNode(sa.GetKey(ADDRESS), sa)
	if err != nil {
		return
	}

	var b1 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[0].BlobberID {
			b1 = b
			break
		}
	}
	require.NotNil(t, b1)

	// add 10 validators
	var valids []*Client
	tp += 1000
	for i := 0; i < 10; i++ {
		valids = append(valids, addValidator(t, ssc, tp, balances))
	}

	// generate some challenges to fill challenge pool

	const allocRoot = "alloc-root-1"

	// write 100 MB
	tp += 1000
	var cc = &BlobberCloseConnection{
		AllocationRoot:     allocRoot,
		PrevAllocationRoot: "",
		WriteMarker:        &WriteMarker{},
	}
	wm1 := &writeMarkerV1{
		AllocationRoot:         allocRoot,
		PreviousAllocationRoot: "",
		AllocationID:           allocID,
		Size:                   10 * 1024 * 1024, // 100 MB
		BlobberID:              b1.id,
		Timestamp:              common.Timestamp(tp),
		ClientID:               client.id,
	}
	wm1.Signature, err = client.scheme.Sign(
		encryption.Hash(wm1.GetHashData()))
	require.NoError(t, err)
	cc.WriteMarker.SetEntity(wm1)

	// write
	tp += 1000
	var tx = newTransaction(b1.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	var resp string
	resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc), balances)
	require.NoError(t, err)
	require.NotZero(t, resp)

	// until the end
	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()

	// load validators
	validators, err := getValidatorsList(balances)
	require.NoError(t, err)

	// load blobber
	var blobber *StorageNode
	blobber, err = ssc.getBlobber(b1.id, balances)
	require.NoError(t, err)

	//
	var (
		step    = (int64(alloc.Expiration) - tp) / 10
		challID string
	)

	// expire the allocation challenging it (+ last challenge)
	for i := int64(0); i < 2; i++ {
		tp += step / 2

		challID = fmt.Sprintf("chall-%d", i)

		currentRound := balances.GetBlock().Round
		genChall(t, ssc, tp, currentRound-200*(i-2), challID, i, validators, alloc.ID, blobber, balances)

		var chall = new(ChallengeResponse)
		chall.ID = challID

		for _, val := range valids {
			chall.ValidationTickets = append(chall.ValidationTickets,
				val.validTicket(t, chall.ID, b1.id, true, tp))
		}

		tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		b := &block.Block{}
		b.Round = 100 + i
		balances.setBlock(t, b)

		_, err = ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
		require.NoError(t, err)
	}

	// balances
	_, err = ssc.getChallengePool(allocID, balances)
	require.NoError(t, err)

	// expire the allocation
	tp += int64(alloc.Until(time.Duration(tp)))

	// finalize it

	var req lockRequest
	req.AllocationID = allocID

	tx = newTransaction(client.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)

	tx.CreationDate = alloc.Expiration + 10
	resp, err = ssc.finalizeAllocation(tx, mustEncode(t, &req), balances)
	require.NoError(t, err)

	// check out all the balances

	_, err = ssc.getChallengePool(allocID, balances)
	require.Error(t, err, "challenge pool should be removed")

	tp += 720

	_, err = ssc.getAllocation(allocID, balances)
	require.Error(t, err)
}

func Test_finalize_allocation_do_not_remove_challenge_ready(t *testing.T) {

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		client   = newClient(1000*x10, balances)
		tp       = int64(0)
		err      error
	)

	setConfig(t, balances)

	tp += 1000
	var allocID, blobs = addAllocation(t, ssc, client, tp, 0, 0, 0, 0, 0, balances, false, false, false)

	// bind another allocation to the blobber

	// blobbers: stake 10k, balance 40k

	sa, err := ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	confMinAllocSize := 1024
	mockBlobberCapacity := 2000 * confMinAllocSize

	alloc.Stats = &StorageAllocationStats{
		UsedSize:          int64(alloc.DataShards+alloc.ParityShards) * int64(mockBlobberCapacity) / 2,
		SuccessChallenges: int64(alloc.DataShards+alloc.ParityShards) * 100,
		FailedChallenges:  int64(alloc.DataShards+alloc.ParityShards) * 2,
		TotalChallenges:   int64(alloc.DataShards+alloc.ParityShards) * 102,
		OpenChallenges:    0,
	}

	for _, ba := range alloc.BlobberAllocs {
		ba.Stats = &StorageAllocationStats{
			UsedSize:          int64(mockBlobberCapacity) / 2,
			SuccessChallenges: 100,
			FailedChallenges:  2,
			TotalChallenges:   102,
			OpenChallenges:    0,
		}

		ba.LatestFinalizedChallCreatedAt = 0
		ba.ChallengePoolIntegralValue = 0

		blobber, err := ssc.getBlobber(ba.BlobberID, balances)
		require.NoError(t, err)
		//nolint:errcheck
		blobber.mustUpdateBase(func(b *storageNodeBase) error {
			b.SavedData = int64(mockBlobberCapacity) / 2
			return nil
		})
		_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
		require.NoError(t, err)
	}

	_, err = balances.InsertTrieNode(sa.GetKey(ADDRESS), sa)
	if err != nil {
		return
	}

	var b1 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[0].BlobberID {
			b1 = b
			break
		}
	}
	require.NotNil(t, b1)

	// bind one more allocation to b1
	err = partitionsBlobberAllocationsAdd(balances, b1.id, encryption.Hash("new_allocation_id"))
	require.NoError(t, err)

	// add 10 validators
	var valids []*Client
	tp += 1000
	for i := 0; i < 10; i++ {
		valids = append(valids, addValidator(t, ssc, tp, balances))
	}

	// generate some challenges to fill challenge pool

	const allocRoot = "alloc-root-1"

	// write 100 MB
	tp += 1000
	var cc = &BlobberCloseConnection{
		AllocationRoot:     allocRoot,
		PrevAllocationRoot: "",
		WriteMarker:        &WriteMarker{},
	}
	wm1 := &writeMarkerV1{
		AllocationRoot:         allocRoot,
		PreviousAllocationRoot: "",
		AllocationID:           allocID,
		Size:                   10 * 1024 * 1024, // 100 MB
		BlobberID:              b1.id,
		Timestamp:              common.Timestamp(tp),
		ClientID:               client.id,
	}
	wm1.Signature, err = client.scheme.Sign(
		encryption.Hash(wm1.GetHashData()))
	require.NoError(t, err)
	cc.WriteMarker.SetEntity(wm1)

	// write
	tp += 1000
	var tx = newTransaction(b1.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	var resp string
	resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc), balances)
	require.NoError(t, err)
	require.NotZero(t, resp)

	// until the end
	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()

	//load validators
	validators, err := getValidatorsList(balances)
	require.NoError(t, err)

	// load blobber
	var blobber *StorageNode
	blobber, err = ssc.getBlobber(b1.id, balances)
	require.NoError(t, err)

	var (
		step    = (int64(alloc.Expiration) - tp) / 10
		challID string
	)

	// expire the allocation challenging it (+ last challenge)
	for i := int64(0); i < 2; i++ {
		tp += step / 2

		challID = fmt.Sprintf("chall-%d", i)
		currentRound := balances.GetBlock().Round
		genChall(t, ssc, tp, currentRound-200*(i-2), challID, i, validators, alloc.ID, blobber, balances)

		var chall = new(ChallengeResponse)
		chall.ID = challID

		for _, val := range valids {
			chall.ValidationTickets = append(chall.ValidationTickets,
				val.validTicket(t, chall.ID, b1.id, true, tp))
		}

		tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		b := &block.Block{}
		b.Round = 100 + i
		balances.setBlock(t, b)
		_, err = ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
		require.NoError(t, err)
	}

	// balances
	_, err = ssc.getChallengePool(allocID, balances)
	require.NoError(t, err)

	// expire the allocation
	tp += int64(alloc.Until(time.Duration(tp)))

	// finalize it

	var req lockRequest
	req.AllocationID = allocID

	tx = newTransaction(client.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	_, err = ssc.finalizeAllocation(tx, mustEncode(t, &req), balances)
	require.NoError(t, err)

	// check out all the balances

	_, err = ssc.getChallengePool(allocID, balances)
	require.Error(t, err, "challenge pool should be removed")

	tp += int64(alloc.Until(time.Duration(tp)))

	_, err = ssc.getAllocation(allocID, balances)
	require.Error(t, util.ErrValueNotPresent, err)
}

func TestBlobberVerifyAuthTicket(t *testing.T) {
	balances := newTestBalances(t, false)
	wallet := newClient(1000*x10, balances)
	b0Wallet := newClient(1000*x10, balances)
	blobber0AuthTicket, err := b0Wallet.scheme.Sign(wallet.id)
	require.NoError(t, err)

	ok, err := verifyBlobberAuthTicket(balances, wallet.id, blobber0AuthTicket, b0Wallet.pk)
	require.NoError(t, err)
	require.True(t, ok)
}
