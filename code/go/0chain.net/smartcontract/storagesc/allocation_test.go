package storagesc

import (
	"0chain.net/chaincore/tokenpool"
	"errors"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

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
	"0chain.net/chaincore/state"
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
		return &StorageNode{
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
		}
	}

	setup := func(t *testing.T, args args) (
		StorageSmartContract, StorageAllocation, StorageNodes, chainState.StateContextI) {
		var balances = &mocks.StateContextI{}
		var ssc = StorageSmartContract{

			SmartContract: sci.NewSC(ADDRESS),
		}
		var sa = StorageAllocation{
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
			require.EqualValues(t, int64(sa.Size+size-1)/size, outSize)

			for _, blobber := range outBlobbers {
				t.Log(blobber)
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

func (sc *StorageSmartContract) selectBlobbers(
	creationDate time.Time,
	allBlobbersList StorageNodes,
	sa *StorageAllocation,
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
		return b.LastHealthCheck <= (now - blobberHealthTime), nil
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
		*StorageAllocation,
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
			bcPart, err = partitionsChallengeReadyBlobbers(balances)
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

			blobber := &StorageNode{
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
			}

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
			require.NoError(t, sp.Save(spenum.Blobber, blobber.ID, balances))

			_, err := balances.InsertTrieNode(blobber.GetKey(), blobber)
			require.NoError(t, err)
			blobbers = append(blobbers, blobber)
		}

		alloc := &StorageAllocation{
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

	validate := func(want want, arg args, sa *StorageAllocation, sc *StorageSmartContract, balances chainState.StateContextI) {
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
				bcPart, err := partitionsChallengeReadyBlobbers(balances)
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
			_, err := sa.changeBlobbers(&Config{TimeUnit: confTimeUnit}, blobbers, addID, removeID, now, balances, sc, clientId)
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
		mockExpiration     = common.Timestamp(2592000)
		mockStake          = 3
		mockMinLockDemand  = 0.1
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
		return &StorageNode{
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
		}
	}

	setup := func(
		t *testing.T, args args,
	) (
		StorageSmartContract,
		*transaction.Transaction,
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
			CreationDate: now * 10,
			Value:        args.value,
		}
		txn.Hash = mockHash
		if txn.Value > 0 {
			balances.On(
				"GetClientBalance", txn.ClientID,
			).Return(txn.Value+1, nil).Once()
			balances.On(
				"AddTransfer", &state.Transfer{
					ClientID:   txn.ClientID,
					ToClientID: txn.ToClientID,
					Amount:     txn.Value,
				},
			).Return(nil).Once()
		}

		var sa = StorageAllocation{
			ID:              mockAllocationId,
			DataShards:      mockDataShards,
			ParityShards:    mockParityShards,
			Owner:           mockOwner,
			OwnerPublicKey:  mockPublicKey,
			Expiration:      now + mockExpiration,
			Size:            mocksSize,
			ReadPriceRange:  PriceRange{mockMinPrice, mockMaxPrice},
			WritePriceRange: PriceRange{mockMinPrice, mockMaxPrice},
			TimeUnit:        mockTimeUnit,
			WritePool:       args.poolFunds * 1e10,
		}

		bCount := sa.DataShards + sa.ParityShards
		var blobbers []*StorageNode
		for i := 0; i < mockNumAllBlobbers; i++ {
			mockBlobber := makeMockBlobber(i)

			if i < sa.DataShards+sa.ParityShards {
				blobbers = append(blobbers, mockBlobber)
				sa.BlobberAllocs = append(sa.BlobberAllocs, &BlobberAllocation{
					BlobberID:     mockBlobber.ID,
					MinLockDemand: zcnToBalance(mockMinLockDemand),
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
							mockPoolId: {},
						},
					},
				}
				sp.Pools[mockPoolId].Balance = zcnToBalance(mockStake)
				balances.On("GetTrieNode", stakePoolKey(spenum.Blobber, mockBlobber.ID),
					mock.MatchedBy(func(s *stakePool) bool {
						*s = sp
						return true
					})).Return(nil).Once()
				balances.On("InsertTrieNode", stakePoolKey(spenum.Blobber, mockBlobber.ID),
					mock.Anything).Return("", nil).Once()
				balances.On("EmitEvent", event.TypeStats,
					event.TagUpdateBlobber, mock.Anything, mock.Anything).Return().Maybe()
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

		return ssc, &txn, sa, blobbers, balances
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
					Size:        zcnToInt64(31),
					FileOptions: 63,
					Extend:      true,
				},
				expiration: mockExpiration,
				value:      0.1e10,
				poolFunds:  10.0,
			},
		},
		{
			name: "ok_unfounded",
			args: args{
				request: updateAllocationRequest{
					ID:          mockAllocationId,
					OwnerID:     mockOwner,
					Size:        zcnToInt64(61),
					FileOptions: 63,
					Extend:      true,
				},
				expiration: mockExpiration,
				value:      0.0,
				poolFunds:  0.0,
			},
			want: want{
				err:    true,
				errMsg: "allocation_extending_failed: adjust_challenge_pool: insufficient funds 0 in write pool to pay 8084397",
			},
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ssc, txn, sa, aBlobbers, balances := setup(t, tt.args)
			conf := &Config{
				TimeUnit: confTimeUnit,
			}

			err := ssc.extendAllocation(
				txn,
				conf,
				&sa,
				aBlobbers,
				&tt.args.request,
				balances,
			)
			if tt.want.err != (err != nil) {
				require.EqualValues(t, tt.want.err, err != nil)
			}
			if err != nil {
				if tt.want.errMsg != err.Error() {
					require.EqualValues(t, tt.want.errMsg, err.Error())
				}
			} else {
				mock.AssertExpectationsForObjects(t, balances)
			}
		})
	}
}

func TestStorageSmartContract_getAllocation(t *testing.T) {
	const allocID, clientID, clientPk = "alloc_hex", "client_hex", "pk"
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		alloc    *StorageAllocation
		err      error
	)
	if _, err = ssc.getAllocation(allocID, balances); err == nil {
		t.Fatal("missing error")
	}
	if err != util.ErrValueNotPresent {
		t.Fatal("unexpected error:", err)
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
	var alloc = nar.storageAllocation(conf, now)
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

func newTestAllBlobbers() (all *StorageNodes) {
	all = new(StorageNodes)
	all.Nodes = []*StorageNode{
		{
			Provider: provider.Provider{
				ID:              "b1",
				ProviderType:    spenum.Blobber,
				LastHealthCheck: 0,
			},
			BaseURL: "http://blobber1.test.ru:9100/api",
			Terms: Terms{
				ReadPrice:  20,
				WritePrice: 200,
			},
			Capacity:     25 * GB, // 20 GB
			Allocated:    5 * GB,  //  5 GB
			NotAvailable: false,
		},
		{
			Provider: provider.Provider{
				ID:              "b2",
				ProviderType:    spenum.Blobber,
				LastHealthCheck: 0,
			},
			BaseURL: "http://blobber2.test.ru:9100/api",
			Terms: Terms{
				ReadPrice:  25,
				WritePrice: 250,
			},
			Capacity:     20 * GB, // 20 GB
			Allocated:    10 * GB, // 10 GB
			NotAvailable: false,
		},
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
		errMsg8 = "allocation_creation_failed: " +
			"no tokens to lock"
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
	tx.Value = 400
	tx.ClientID = clientID
	tx.CreationDate = toSeconds(2 * time.Hour)

	balances.setTransaction(t, &tx)

	conf = setConfig(t, balances)
	conf.MaxChallengeCompletionTime = 20 * time.Second
	conf.MinAllocSize = 10 * GB
	conf.TimeUnit = 20 * time.Second

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
		b0.LastHealthCheck = tx.CreationDate
		b1 := allBlobbers.Nodes[1]
		b1.LastHealthCheck = tx.CreationDate
		nar.Blobbers = append(nar.Blobbers, b0.ID)
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		nar.Blobbers = append(nar.Blobbers, b1.ID)
		_, err = balances.InsertTrieNode(b1.GetKey(), b1)
		require.NoError(t, err)

		_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances, nil)
		requireErrMsg(t, err, errMsg7)
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
		b0.LastHealthCheck = tx.CreationDate
		b1 := allBlobbers.Nodes[1]
		b1.LastHealthCheck = tx.CreationDate
		nar.Blobbers = append(nar.Blobbers, b0.ID)
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		nar.Blobbers = append(nar.Blobbers, b1.ID)
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
		b0.LastHealthCheck = tx.CreationDate
		b1 := allBlobbers.Nodes[1]
		b1.LastHealthCheck = tx.CreationDate
		b0.Allocated = 5 * GB
		b1.Allocated = 10 * GB

		nar.Blobbers = append(nar.Blobbers, b0.ID)
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		nar.Blobbers = append(nar.Blobbers, b1.ID)
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
		b0.LastHealthCheck = tx.CreationDate
		b1 := allBlobbers.Nodes[1]
		b1.LastHealthCheck = tx.CreationDate
		b0.Allocated = 5 * GB
		b1.Allocated = 10 * GB

		nar.Blobbers = append(nar.Blobbers, b0.ID)
		_, err = balances.InsertTrieNode(b0.GetKey(), b0)
		nar.Blobbers = append(nar.Blobbers, b1.ID)
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

		balances.balances[clientID] = 1100 + 4500

		tx.Hash = encryption.Hash("ok")
		tx.Value = 5000
		resp, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances, nil)
		require.NoError(t, err)

		// check response
		var aresp NewAllocationTxnOutput
		require.NoError(t, aresp.Decode([]byte(resp)))

		assert.Equal(t, tx.Hash, aresp.ID)
		assert.Equal(t, len(aresp.Blobber_ids), 4)

		// expected blobbers after the allocation
		var sb = newTestAllBlobbers()
		sb.Nodes[0].LastHealthCheck = tx.CreationDate
		sb.Nodes[1].LastHealthCheck = tx.CreationDate
		sb.Nodes[0].Allocated += 10 * GB
		sb.Nodes[1].Allocated += 10 * GB

		// blobbers saved in all blobbers list
		var ab []*StorageNode
		loaded0, err := ssc.getBlobber(b0.ID, balances)
		loaded1, err := ssc.getBlobber(b1.ID, balances)
		ab = append(ab, loaded0)
		ab = append(ab, loaded1)
		require.NoError(t, err)
		for i, sbn := range sb.Nodes {
			assert.EqualValues(t, *sbn, *ab[i])
		}
		assert.EqualValues(t, sb.Nodes, ab)
		// independent saved blobbers
		var blob1, blob2 *StorageNode
		blob1, err = ssc.getBlobber("b1", balances)
		require.NoError(t, err)
		assert.EqualValues(t, sb.Nodes[0], blob1)
		blob2, err = ssc.getBlobber("b2", balances)
		require.NoError(t, err)
		assert.EqualValues(t, sb.Nodes[1], blob2)

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
}

func Test_updateAllocationRequest_decode(t *testing.T) {
	var ud, ue updateAllocationRequest
	ue.Size = -200
	require.NoError(t, ud.decode(mustEncode(t, &ue)))
	assert.EqualValues(t, ue, ud)
}

func Test_updateAllocationRequest_validate(t *testing.T) {

	var (
		conf  Config
		uar   updateAllocationRequest
		alloc StorageAllocation
	)

	alloc.Size = 10 * GB

	// 1. zero
	assert.Error(t, uar.validate(&conf, &alloc))

	// 2. becomes to small
	var sub = 9.01 * GB
	uar.Size -= int64(sub)
	conf.MinAllocSize = 1 * GB
	assert.Error(t, uar.validate(&conf, &alloc))

	// 3. no blobbers (invalid allocation, panic check)
	uar.Size = 1 * GB
	assert.Error(t, uar.validate(&conf, &alloc))

	// 4. ok
	alloc.BlobberAllocs = []*BlobberAllocation{{}}
	assert.NoError(t, uar.validate(&conf, &alloc))
}

func Test_updateAllocationRequest_getBlobbersSizeDiff(t *testing.T) {
	var (
		uar   updateAllocationRequest
		alloc StorageAllocation
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

	conf.MaxChallengeCompletionTime = 20 * time.Second
	conf.MinAllocSize = 10 * GB
	conf.MaxBlobbersPerAllocation = 4
	conf.TimeUnit = 20 * time.Second

	_, err = balances.InsertTrieNode(scConfigKey(ADDRESS), &conf)
	require.NoError(t, err)

	allBlobbers = newTestAllBlobbers()
	// make the blobbers health
	b0 := allBlobbers.Nodes[0]
	b0.LastHealthCheck = tx.CreationDate
	b1 := allBlobbers.Nodes[1]
	b1.LastHealthCheck = tx.CreationDate
	b0.Allocated = 5 * GB
	b1.Allocated = 10 * GB

	nar.Blobbers = append(nar.Blobbers, b0.ID)
	_, err = balances.InsertTrieNode(b0.GetKey(), b0)
	nar.Blobbers = append(nar.Blobbers, b1.ID)
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

		uar   updateAllocationRequest
		alloc *StorageAllocation
		err   error
	)

	createNewTestAllocation(t, ssc, allocTxHash, clientID, pubKey, balances)

	alloc, err = ssc.getAllocation(allocTxHash, balances)
	require.NoError(t, err)

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
	blobbers, err = ssc.getAllocationBlobbers(alloc, balances)
	require.NoError(t, err)

	assert.Len(t, blobbers, 2)
}

func TestStorageSmartContract_closeAllocation(t *testing.T) {

	const (
		allocTxHash, clientID, pubKey, closeTxHash = "a5f4c3d2_tx_hex",
			"client_hex", "pub_key_hex", "close_tx_hash"

		errMsg1 = "allocation_closing_failed: " +
			"doesn't need to close allocation is about to expire"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		tx       transaction.Transaction

		alloc *StorageAllocation
		resp  string
		err   error
	)

	createNewTestAllocation(t, ssc, allocTxHash, clientID, pubKey, balances)

	tx.Hash = closeTxHash
	tx.ClientID = clientID
	tx.CreationDate = 1050

	alloc, err = ssc.getAllocation(allocTxHash, balances)
	storageAllocationToAllocationTable(alloc)

	require.NoError(t, err)

	// 1. expiring allocation
	alloc.Expiration = 1049
	var conf = Config{
		MaxChallengeCompletionTime: 30 * time.Minute,
	}

	_, err = ssc.closeAllocation(&tx, alloc, conf.MaxChallengeCompletionTime, balances)
	requireErrMsg(t, err, errMsg1)

	// 2. close (all related pools has created)
	alloc.Expiration = tx.CreationDate +
		toSeconds(conf.MaxChallengeCompletionTime) + 20
	resp, err = ssc.closeAllocation(&tx, alloc, conf.MaxChallengeCompletionTime, balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)
	// checking out

	alloc, err = ssc.getAllocation(alloc.ID, balances)
	require.NoError(t, err)

	require.Equal(t, tx.CreationDate, alloc.Expiration)
}

func (alloc *StorageAllocation) deepCopy(t *testing.T) (cp *StorageAllocation) {
	cp = new(StorageAllocation)
	require.NoError(t, cp.Decode(mustEncode(t, alloc)))
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

		bcpartition, err := partitionsChallengeReadyBlobbers(balances)
		require.NoError(t, err)

		for i := 0; i < arg.numBlobbers; i++ {
			blobberID := "blobber_" + strconv.Itoa(i)
			err := bcpartition.Add(balances, &ChallengeReadyBlobber{BlobberID: blobberID})
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

		err = bcpartition.Save(balances)
		require.NoError(t, err)

		return ssc, balances, removeID, allocationID
	}

	validate := func(want want, balances chainState.StateContextI) {
		bcPart, err := partitionsChallengeReadyBlobbers(balances)
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
				allocationID, removeBlobberID)
			require.NoError(t, err)
			validate(tt.want, balances)
		})
	}
}

func TestStorageSmartContract_updateAllocationRequest(t *testing.T) {
	var (
		ssc                  = newTestStorageSC()
		balances             = newTestBalances(t, false)
		client               = newClient(2000*x10, balances)
		otherClient          = newClient(50*x10, balances)
		tp, exp        int64 = 100, 1000
		allocID, blobs       = addAllocation(t, ssc, client, tp, exp, 0, balances)
		alloc          *StorageAllocation
		resp           string
		err            error
	)

	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	cp := &StorageAllocation{}
	err = cp.Decode(alloc.Encode())
	require.NoError(t, err)

	// change terms
	tp += 100
	for _, b := range blobs {
		var blob *StorageNode
		blob, err = ssc.getBlobber(b.id, balances)
		require.NoError(t, err)
		blob.Terms.WritePrice = currency.Coin(5 * x10)
		blob.Terms.ReadPrice = currency.Coin(0.8 * x10)
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
	tp += 100
	resp, err = uar.callUpdateAllocReq(t, client.id, 300*x10, tp, ssc, balances)
	require.NoError(t, err)

	var deco StorageAllocation
	require.NoError(t, deco.Decode([]byte(resp)))

	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	require.EqualValues(t, alloc, &deco)

	assert.Equal(t, cp.Size*2, alloc.Size)
	assert.Equal(t, common.Timestamp(tp+3600), alloc.Expiration)

	var tbs, mld int64
	for i, d := range alloc.BlobberAllocs {
		if i == alloc.DataShards {
			break
		}
		tbs += d.Size
		mld += int64(d.MinLockDemand)
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
	tp += 100
	resp, err = req.callUpdateAllocReq(t, client.id, 20*x10, tp, ssc, balances)
	require.NoError(t, err)

	// Others cannot extend the allocation if `third_party_extendable` = false
	req = updateAllocationRequest{
		ID:     alloc.ID,
		Size:   100,
		Extend: true,
	}
	tp += 100
	resp, err = req.callUpdateAllocReq(t, otherClient.id, 20*x10, tp, ssc, balances)
	require.Error(t, err)
	assert.Equal(t, "allocation_updating_failed: only owner can update the allocation", err.Error())

	// Owner can change `third_party_extendable`
	req = updateAllocationRequest{
		ID:                      alloc.ID,
		SetThirdPartyExtendable: true,
	}
	tp += 100
	resp, err = req.callUpdateAllocReq(t, client.id, 20*x10, tp, ssc, balances)
	require.NoError(t, err)

	// Others can extend the allocation if `third_party_extendable` = true
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	req = updateAllocationRequest{
		ID:     alloc.ID,
		Size:   100,
		Extend: true,
	}
	tp += 100
	expectedSize := alloc.Size + 100
	resp, err = req.callUpdateAllocReq(t, otherClient.id, 20*x10, tp, ssc, balances)
	require.NoError(t, err)
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	assert.Equal(t, expectedSize, alloc.Size)
	assert.Equal(t, common.Timestamp(tp+3600), alloc.Expiration)

	// Other cannot perform any other action than extending.
	req = updateAllocationRequest{
		ID:                 alloc.ID,
		FileOptions:        61,
		FileOptionsChanged: true,
	}
	tp += 100
	expectedFileOptions := alloc.FileOptions
	resp, err = req.callUpdateAllocReq(t, otherClient.id, 20*x10, tp, ssc, balances)
	require.Error(t, err)

	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	assert.Equal(t, expectedFileOptions, alloc.FileOptions)

	//
	// add blobber
	//
	tp += 100
	nb := addBlobber(t, ssc, 2*GB, tp, avgTerms, 50*x10, balances)
	tp += 100
	req = updateAllocationRequest{
		ID:           alloc.ID,
		AddBlobberId: nb.id,
	}
	resp, err = req.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
	require.NoError(t, err)

	// assert that the new blobber offer is updated
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	nblobAlloc, ok := alloc.BlobberAllocsMap[nb.id]
	require.True(t, ok)

	nsp, err := ssc.getStakePool(spenum.Blobber, nb.id, balances)
	require.NoError(t, err)
	require.Equal(t, nsp.TotalOffers, nblobAlloc.Offer())

	// assert the blobber allocation is added
	baParts, err := partitionsBlobberAllocations(nb.id, balances)
	require.NoError(t, err)
	var it BlobberAllocationNode

	err = baParts.Get(balances, alloc.ID, &it)
	require.NoError(t, err)
	require.Equal(t, alloc.ID, it.ID)

	//
	// remove blobber
	//

	tp += 100
	nb2 := addBlobber(t, ssc, 2*GB, tp, avgTerms, 50*x10, balances)
	tp += 100

	req = updateAllocationRequest{
		ID:              alloc.ID,
		AddBlobberId:    nb2.id,
		RemoveBlobberId: nb.id,
	}
	resp, err = req.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
	require.NoError(t, err)

	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	// assert blobber is removed from allocation
	_, ok = alloc.BlobberAllocsMap[nb.id]
	require.False(t, ok)

	// assert allocation is removed from blobber
	baParts, err = partitionsBlobberAllocations(nb.id, balances)
	require.NoError(t, err)
	var noneIt BlobberAllocationNode
	err = baParts.Get(balances, alloc.ID, &noneIt)
	require.True(t, partitions.ErrItemNotFound(err))

	// commit connection to get update challenge ready partition
	// assert there's no challenge ready partition before commit connection
	challengeReadyParts, err := partitionsChallengeReadyBlobbers(balances)
	require.NoError(t, err)
	var cit ChallengeReadyBlobber
	err = challengeReadyParts.Get(balances, nb2.id, &cit)
	require.True(t, partitions.ErrItemNotFound(err))

	tp += 100
	// write
	const allocRoot = "alloc-root-1"
	var cc = &BlobberCloseConnection{
		AllocationRoot:     allocRoot,
		PrevAllocationRoot: "",
		WriteMarker: &WriteMarker{
			AllocationRoot:         allocRoot,
			PreviousAllocationRoot: "",
			AllocationID:           allocID,
			Size:                   10 * 1024 * 1024, // 100 MB
			BlobberID:              nb2.id,
			Timestamp:              common.Timestamp(tp),
			ClientID:               client.id,
		},
	}
	cc.WriteMarker.Signature, err = client.scheme.Sign(
		encryption.Hash(cc.WriteMarker.GetHashData()))
	require.NoError(t, err)
	var tx = newTransaction(nb2.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc), balances)
	require.NoError(t, err)
	require.NotZero(t, resp)

	// assert nb2 is challenge ready
	challengeReadyParts, err = partitionsChallengeReadyBlobbers(balances)
	require.NoError(t, err)
	err = challengeReadyParts.Get(balances, nb2.id, &cit)
	require.NoError(t, err)
	require.Equal(t, cit.BlobberID, nb2.id)

	//
	// remove blobber nb2, assert it self is removed from challenge ready partition
	//

	tp += 100
	nb3 := addBlobber(t, ssc, 3*GB, tp, avgTerms, 50*x10, balances)

	tp += 100
	req = updateAllocationRequest{
		ID:              alloc.ID,
		AddBlobberId:    nb3.id,
		RemoveBlobberId: nb2.id,
	}

	resp, err = req.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
	require.NoError(t, err)

	// assert blobber nb2 is removed from challenge ready partition
	challengeReadyParts, err = partitionsChallengeReadyBlobbers(balances)
	require.NoError(t, err)
	err = challengeReadyParts.Get(balances, nb2.id, &cit)
	require.True(t, partitions.ErrItemNotFound(err))

	//
	// increase duration
	//

	cp = &StorageAllocation{}
	err = cp.Decode(alloc.Encode())
	require.NoError(t, err)

	uar.ID = alloc.ID
	uar.Size = -(alloc.Size / 2)
	uar.Extend = true

	tp += 100
	resp, err = uar.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
	require.NoError(t, err)
	require.NoError(t, deco.Decode([]byte(resp)))

	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	require.EqualValues(t, alloc, &deco)

	assert.Equal(t, cp.Size/2, alloc.Size)
	assert.Equal(t, common.Timestamp(tp+3600), alloc.Expiration)

	tbs, mld = 0, 0
	for i, detail := range alloc.BlobberAllocs {
		if i == alloc.DataShards {
			break
		}
		tbs += detail.Size
		mld += int64(detail.MinLockDemand)
	}
	numb = int64(alloc.DataShards + alloc.ParityShards)
	bsize = (alloc.Size + (numb - 1)) / numb
	assert.True(t, math.Abs(float64(bsize*numb-tbs)) < 100)

	//
	// change owner and owner public key
	//

	cp = &StorageAllocation{}
	err = cp.Decode(alloc.Encode())
	require.NoError(t, err)

	var uarOwnerUpdate = updateAllocationRequest{
		ID:             alloc.ID,
		OwnerID:        otherClient.id,
		OwnerPublicKey: otherClient.pk,
	}

	tp += 100
	resp, err = uarOwnerUpdate.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
	require.NoError(t, err)
	require.NoError(t, deco.Decode([]byte(resp)))

	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	require.EqualValues(t, alloc.Owner, otherClient.id)
	require.EqualValues(t, alloc.OwnerPublicKey, otherClient.pk)
	require.EqualValues(t, alloc, &deco)

	//
	// reduce
	//

	cp = alloc.deepCopy(t)

	uar.ID = alloc.ID
	uar.Size = -(alloc.Size / 2)

	tp += 100
	resp, err = uar.callUpdateAllocReq(t, client.id, 0, tp, ssc, balances)
	require.Error(t, err)

}

// - finalize allocation
func Test_finalize_allocation(t *testing.T) {
	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(t, false)
		client         = newClient(1000*x10, balances)
		tp, exp  int64 = 0, int64(toSeconds(time.Hour))
		err      error
	)

	setConfig(t, balances)

	tp += 100
	var allocID, blobs = addAllocation(t, ssc, client, tp, exp, 0, balances)

	// blobbers: stake 10k, balance 40k

	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

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
	tp += 100
	for i := 0; i < 10; i++ {
		valids = append(valids, addValidator(t, ssc, tp, balances))
	}

	// generate some challenges to fill challenge pool

	const allocRoot = "alloc-root-1"

	// write 100 MB
	tp += 100
	var cc = &BlobberCloseConnection{
		AllocationRoot:     allocRoot,
		PrevAllocationRoot: "",
		WriteMarker: &WriteMarker{
			AllocationRoot:         allocRoot,
			PreviousAllocationRoot: "",
			AllocationID:           allocID,
			Size:                   10 * 1024 * 1024, // 100 MB
			BlobberID:              b1.id,
			Timestamp:              common.Timestamp(tp),
			ClientID:               client.id,
		},
	}
	cc.WriteMarker.Signature, err = client.scheme.Sign(
		encryption.Hash(cc.WriteMarker.GetHashData()))
	require.NoError(t, err)

	// write
	tp += 100
	var tx = newTransaction(b1.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	var resp string
	resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc), balances)
	require.NoError(t, err)
	require.NotZero(t, resp)

	// until the end
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

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
		genChall(t, ssc, tp, challID, i, validators, alloc.ID, blobber, balances)

		var chall = new(ChallengeResponse)
		chall.ID = challID

		for _, val := range valids {
			chall.ValidationTickets = append(chall.ValidationTickets,
				val.validTicket(t, chall.ID, b1.id, true, tp))
		}

		tp += step / 2
		tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		b := &block.Block{}
		b.Round = 100 + i
		balances.setBlock(t, b)

		_, err = ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
		require.NoError(t, err)
	}

	// balances
	var cp *challengePool
	_, err = ssc.getChallengePool(allocID, balances)
	require.NoError(t, err)

	// expire the allocation
	var conf *Config
	conf, err = getConfig(balances)
	require.NoError(t, err)

	require.NoError(t, err)
	tp += int64(alloc.Until(conf.MaxChallengeCompletionTime))

	// finalize it

	var req lockRequest
	req.AllocationID = allocID

	tx = newTransaction(client.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	_, err = ssc.finalizeAllocation(tx, mustEncode(t, &req), balances)
	require.NoError(t, err)

	// check out all the balances

	cp, err = ssc.getChallengePool(allocID, balances)
	require.NoError(t, err)

	tp += int64(toSeconds(conf.MaxChallengeCompletionTime))
	assert.Zero(t, cp.Balance, "should be drained")

	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	assert.True(t, alloc.Finalized)
	assert.True(t,
		alloc.BlobberAllocs[0].MinLockDemand <= alloc.BlobberAllocs[0].Spent,
		"should receive min_lock_demand")
}

func Test_finalize_allocation_do_not_remove_challenge_ready(t *testing.T) {
	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(t, false)
		client         = newClient(1000*x10, balances)
		tp, exp  int64 = 0, int64(toSeconds(time.Hour))
		err      error
	)

	setConfig(t, balances)

	tp += 100
	var allocID, blobs = addAllocation(t, ssc, client, tp, exp, 0, balances)

	// bind another allocation to the blobber

	// blobbers: stake 10k, balance 40k

	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

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
	tp += 100
	for i := 0; i < 10; i++ {
		valids = append(valids, addValidator(t, ssc, tp, balances))
	}

	// generate some challenges to fill challenge pool

	const allocRoot = "alloc-root-1"

	// write 100 MB
	tp += 100
	var cc = &BlobberCloseConnection{
		AllocationRoot:     allocRoot,
		PrevAllocationRoot: "",
		WriteMarker: &WriteMarker{
			AllocationRoot:         allocRoot,
			PreviousAllocationRoot: "",
			AllocationID:           allocID,
			Size:                   10 * 1024 * 1024, // 100 MB
			BlobberID:              b1.id,
			Timestamp:              common.Timestamp(tp),
			ClientID:               client.id,
		},
	}
	cc.WriteMarker.Signature, err = client.scheme.Sign(
		encryption.Hash(cc.WriteMarker.GetHashData()))
	require.NoError(t, err)

	// write
	tp += 100
	var tx = newTransaction(b1.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	var resp string
	resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc), balances)
	require.NoError(t, err)
	require.NotZero(t, resp)

	// until the end
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

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
		genChall(t, ssc, tp, challID, i, validators, alloc.ID, blobber, balances)

		var chall = new(ChallengeResponse)
		chall.ID = challID

		for _, val := range valids {
			chall.ValidationTickets = append(chall.ValidationTickets,
				val.validTicket(t, chall.ID, b1.id, true, tp))
		}

		tp += step / 2
		tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		b := &block.Block{}
		b.Round = 100 + i
		balances.setBlock(t, b)
		_, err = ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
		require.NoError(t, err)
	}

	// balances
	var cp *challengePool
	_, err = ssc.getChallengePool(allocID, balances)
	require.NoError(t, err)

	// expire the allocation
	var conf *Config
	conf, err = getConfig(balances)
	require.NoError(t, err)

	require.NoError(t, err)
	tp += int64(alloc.Until(conf.MaxChallengeCompletionTime))

	// finalize it

	var req lockRequest
	req.AllocationID = allocID

	tx = newTransaction(client.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	_, err = ssc.finalizeAllocation(tx, mustEncode(t, &req), balances)
	require.NoError(t, err)

	// check out all the balances

	cp, err = ssc.getChallengePool(allocID, balances)
	require.NoError(t, err)

	tp += int64(toSeconds(conf.MaxChallengeCompletionTime))
	assert.Zero(t, cp.Balance, "should be drained")

	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	assert.True(t, alloc.Finalized)
	assert.True(t,
		alloc.BlobberAllocs[0].MinLockDemand <= alloc.BlobberAllocs[0].Spent,
		"should receive min_lock_demand")
}
