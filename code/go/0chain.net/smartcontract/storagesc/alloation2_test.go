package storagesc

import (
	"0chain.net/smartcontract/provider"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"github.com/stretchr/testify/require"
)

type blobberStakes []int64

const (
	errValueNotPresent   = "value not present"
	ownerId              = "owin"
	ErrCancelFailed      = "alloc_cancel_failed"
	ErrExpired           = "trying to cancel expired allocation"
	ErrNotOwner          = "only owner can cancel an allocation"
	ErrNotEnoughFailiars = "not enough failed challenges of allocation to cancel"
	ErrNotEnoughLock     = "paying min_lock for"
	ErrFinalizedFailed   = "fini_alloc_failed"
	ErrFinalizedTooSoon  = "allocation is not expired yet, or waiting a challenge completion"
)

func TestNewAllocation(t *testing.T) {
	var stakes = blobberStakes{}
	var now = common.Timestamp(10000)
	scYaml = &Config{
		MinAllocSize:               1027,
		MinAllocDuration:           5 * time.Minute,
		MaxChallengeCompletionTime: 30 * time.Minute,
		MaxStake:                   zcnToBalance(100.0),
	}
	var blobberYaml = mockBlobberYaml{
		readPrice:               0.01,
		writePrice:              0.10,
		challengeCompletionTime: scYaml.MaxChallengeCompletionTime,
	}

	var request = newAllocationRequest{
		Owner:                      clientId,
		OwnerPublicKey:             "my public key",
		Size:                       scYaml.MinAllocSize,
		DataShards:                 3,
		ParityShards:               5,
		Expiration:                 common.Timestamp(scYaml.MinAllocDuration.Seconds()) + now,
		ReadPriceRange:             PriceRange{0, zcnToBalance(blobberYaml.readPrice) + 1},
		WritePriceRange:            PriceRange{0, zcnToBalance(blobberYaml.writePrice) + 1},
		MaxChallengeCompletionTime: blobberYaml.challengeCompletionTime + 1,
		PreferredBlobbers:          []string{"mockBaseUrl1", "mockBaseUrl3"},
	}
	var goodBlobber = StorageNode{
		Capacity: 536870912,
		Used:     73,
		Terms: Terms{
			MaxOfferDuration:        1000 * scYaml.MinAllocDuration,
			ReadPrice:               zcnToBalance(blobberYaml.readPrice),
			ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
		},
		Provider: provider.Provider{
			LastHealthCheck: now - blobberHealthTime,
		},
	}
	var blobbers = new(SortedBlobbers)
	var stake = int64(scYaml.MaxStake)
	var writePrice = blobberYaml.writePrice
	for i := 0; i < request.DataShards+request.ParityShards+4; i++ {
		var nextBlobber = goodBlobber
		nextBlobber.ID = strconv.Itoa(i)
		nextBlobber.Terms.WritePrice = zcnToBalance(writePrice)
		nextBlobber.BaseURL = "mockBaseUrl" + strconv.Itoa(i)
		writePrice *= 0.9
		blobbers.add(&nextBlobber)
		stakes = append(stakes, stake)
		stake = stake / 10
	}

	t.Run("new allocation random blobbers", func(t *testing.T) {
		request := request
		request.DiversifyBlobbers = false
		err := testNewAllocation(t, request, *blobbers, *scYaml, blobberYaml, stakes)
		require.NoError(t, err)
	})

	t.Run("new allocation diverse blobbers", func(t *testing.T) {
		request := request
		request.DiversifyBlobbers = true
		err := testNewAllocation(t, request, *blobbers, *scYaml, blobberYaml, stakes)
		require.NoError(t, err)
	})

	t.Run("new allocation", func(t *testing.T) {
		var request2 = request
		request2.Size = 100 * GB

		err := testNewAllocation(t, request, *blobbers, *scYaml, blobberYaml, stakes)
		require.NoError(t, err)
	})
}

func TestCancelAllocationRequest(t *testing.T) {
	var blobberStakePools = [][]mockStakePool{}
	var challenges = [][]common.Timestamp{}
	var scYaml = Config{
		MaxMint:                         zcnToBalance(4000000.0),
		StakePool:                       &stakePoolConfig{},
		BlobberSlash:                    0.1,
		ValidatorReward:                 0.025,
		MaxChallengeCompletionTime:      30 * time.Minute,
		TimeUnit:                        720 * time.Hour,
		FailedChallengesToRevokeMinLock: 10,
		MaxStake:                        zcnToBalance(100.0),
	}
	var now = common.Timestamp(scYaml.MaxChallengeCompletionTime) * 5
	var blobberYaml = mockBlobberYaml{
		serviceCharge:           0.30,
		writePrice:              0.1,
		challengeCompletionTime: scYaml.MaxChallengeCompletionTime,
		minLockDemand:           0.1,
	}

	var blobberTemplate = StorageNode{
		Capacity: 536870912,
		Terms: Terms{
			MaxOfferDuration:        1000 * scYaml.MinAllocDuration,
			ReadPrice:               zcnToBalance(blobberYaml.readPrice),
			ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
		},
		Provider: provider.Provider{
			LastHealthCheck: now - blobberHealthTime,
		},
	}
	var allocation = StorageAllocation{
		DataShards:    1,
		ParityShards:  1,
		ID:            ownerId,
		BlobberAllocs: []*BlobberAllocation{},
		Owner:         ownerId,
		Expiration:    now,
		Stats: &StorageAllocationStats{
			OpenChallenges: 3,
		},
		Size:     4560,
		UsedSize: 456,
	}
	var blobbers = new(SortedBlobbers)
	var stake = 100.0
	var writePrice = blobberYaml.writePrice
	var extraBlobbers = 0
	var blobberUsedSize = allocation.UsedSize / int64(allocation.DataShards+allocation.ParityShards)
	for i := 0; i < allocation.DataShards+allocation.ParityShards+extraBlobbers; i++ {
		var nextBlobber = blobberTemplate
		nextBlobber.ID = strconv.Itoa(i)
		nextBlobber.Terms.WritePrice = zcnToBalance(writePrice)
		writePrice *= 0.9
		var minLockDemand = float64(allocation.Size) * writePrice * blobberYaml.minLockDemand
		blobbers.add(&nextBlobber)
		blobberStakePools = append(blobberStakePools, []mockStakePool{})
		blobberStakePools[i] = append(blobberStakePools[i], mockStakePool{
			zcnAmount: stake,
		})
		blobberStakePools[i] = append(blobberStakePools[i], mockStakePool{
			zcnAmount: 0.258,
		})
		stake = stake / 10
		if i < allocation.DataShards+allocation.ParityShards {
			allocation.BlobberAllocs = append(allocation.BlobberAllocs, &BlobberAllocation{
				BlobberID: nextBlobber.ID,
				Terms: Terms{
					ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
				},
				Stats: &StorageAllocationStats{
					UsedSize:          blobberUsedSize,
					OpenChallenges:    int64(i + 1),
					SuccessChallenges: int64(i),
				},
				MinLockDemand: 200 + state.Balance(minLockDemand),
				Spent:         100,
			})
			challenges = append(challenges, []common.Timestamp{})
			for j := 0; j < int(allocation.BlobberAllocs[i].Stats.OpenChallenges); j++ {
				var expires = now - common.Timestamp(float64(j)*float64(blobberYaml.challengeCompletionTime)/3.0)
				challenges[i] = append(challenges[i], expires)
			}
		}
	}

	var challengePoolBalance = int64(700000)
	var thisExpires = common.Timestamp(222)

	var blobberOffer = int64(123000)
	var wpBalance state.Balance = 777777
	var otherWritePools = 0

	t.Run("cancel allocation", func(t *testing.T) {
		err := testCancelAllocation(t, allocation, *blobbers, blobberStakePools, scYaml,
			otherWritePools, challengePoolBalance, challenges, blobberOffer, wpBalance, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run(ErrNotOwner, func(t *testing.T) {
		var allocationNotOwner = allocation
		allocationNotOwner.Owner = "someone else"

		err := testCancelAllocation(t, allocationNotOwner, *blobbers, blobberStakePools, scYaml,
			otherWritePools, challengePoolBalance, challenges, blobberOffer, wpBalance, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), ErrCancelFailed))
		require.True(t, strings.Contains(err.Error(), ErrNotOwner))
	})

	t.Run(ErrExpired, func(t *testing.T) {
		var allocationExpired = allocation
		allocationExpired.Expiration = now - 1

		err := testCancelAllocation(t, allocationExpired, *blobbers, blobberStakePools, scYaml,
			otherWritePools, challengePoolBalance, challenges, blobberOffer, wpBalance, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), ErrCancelFailed))
		require.True(t, strings.Contains(err.Error(), ErrExpired))
	})
}

func TestFinalizeAllocation(t *testing.T) {
	var now = common.Timestamp(300)
	var blobberStakePools = [][]mockStakePool{}
	var scYaml = Config{
		MaxMint:                         zcnToBalance(4000000.0),
		BlobberSlash:                    0.1,
		ValidatorReward:                 0.025,
		MaxChallengeCompletionTime:      30 * time.Minute,
		TimeUnit:                        720 * time.Hour,
		FailedChallengesToRevokeMinLock: 10,
		MaxStake:                        zcnToBalance(100.0),
	}
	var blobberYaml = mockBlobberYaml{
		serviceCharge:           0.30,
		writePrice:              0.1,
		challengeCompletionTime: scYaml.MaxChallengeCompletionTime,
		minLockDemand:           0.1,
	}
	var blobberTemplate = StorageNode{
		Capacity: 536870912,
		Terms: Terms{
			MaxOfferDuration:        1000 * scYaml.MinAllocDuration,
			ReadPrice:               zcnToBalance(blobberYaml.readPrice),
			ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
		},
		Provider: provider.Provider{
			LastHealthCheck: now - blobberHealthTime,
		},
	}
	var allocation = StorageAllocation{
		DataShards:    5,
		ParityShards:  5,
		ID:            ownerId,
		BlobberAllocs: []*BlobberAllocation{},
		Owner:         ownerId,
		Expiration:    now,
		Stats: &StorageAllocationStats{
			OpenChallenges: 3,
		},
		Size: 4560,
	}
	allocation.UsedSize = 41 * int64(allocation.DataShards+allocation.ParityShards)
	var blobbers = new(SortedBlobbers)
	var stake = 100.0
	var writePrice = blobberYaml.writePrice
	var extraBlobbers = 0
	var blobberUsedSize = int64(float64(allocation.UsedSize) / float64(allocation.DataShards+allocation.ParityShards))
	for i := 0; i < allocation.DataShards+allocation.ParityShards+extraBlobbers; i++ {
		var nextBlobber = blobberTemplate
		nextBlobber.ID = strconv.Itoa(i)
		nextBlobber.Terms.WritePrice = zcnToBalance(writePrice)
		writePrice *= 0.9
		var minLockDemand = float64(allocation.Size) * writePrice * blobberYaml.minLockDemand
		blobbers.add(&nextBlobber)
		blobberStakePools = append(blobberStakePools, []mockStakePool{})
		blobberStakePools[i] = append(blobberStakePools[i], mockStakePool{
			zcnAmount: stake,
		})
		blobberStakePools[i] = append(blobberStakePools[i], mockStakePool{
			zcnAmount: 0.258,
		})
		stake = stake / 10
		if i < allocation.DataShards+allocation.ParityShards {
			allocation.BlobberAllocs = append(allocation.BlobberAllocs, &BlobberAllocation{
				BlobberID: nextBlobber.ID,
				Terms: Terms{
					ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
				},
				Stats: &StorageAllocationStats{
					UsedSize:          blobberUsedSize,
					OpenChallenges:    int64(i + 1),
					SuccessChallenges: int64(i),
				},
				MinLockDemand: 200 + state.Balance(minLockDemand),
				Spent:         100,
			})
		}
	}
	var challengePoolBalance = int64(700000)
	var thisExpires = common.Timestamp(222)

	var blobberOffer = int64(123000)
	var wpBalance state.Balance = 777777
	var otherWritePools = 4

	t.Run("finalize allocation", func(t *testing.T) {
		err := testFinalizeAllocation(t, allocation, *blobbers, blobberStakePools, scYaml,
			otherWritePools, challengePoolBalance, blobberOffer, wpBalance, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run(ErrFinalizedTooSoon, func(t *testing.T) {
		var allocationExpired = allocation
		allocationExpired.Expiration = now - toSeconds(allocation.ChallengeCompletionTime) + 1

		err := testFinalizeAllocation(t, allocationExpired, *blobbers, blobberStakePools, scYaml,
			otherWritePools, challengePoolBalance, blobberOffer, wpBalance, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), ErrFinalizedFailed))
		require.True(t, strings.Contains(err.Error(), ErrFinalizedTooSoon))
	})
}

func testCancelAllocation(
	t *testing.T,
	sAllocation StorageAllocation,
	blobbers SortedBlobbers,
	bStakes [][]mockStakePool,
	scYaml Config,
	otherWritePools int,
	challengePoolBalance int64,
	challenges [][]common.Timestamp,
	blobberOffer int64,
	wpBalance state.Balance,
	thisExpires, now common.Timestamp,
) error {
	var f = formulaeFinalizeAllocation{
		t:                    t,
		scYaml:               scYaml,
		allocation:           sAllocation,
		blobbers:             blobbers,
		bStakes:              bStakes,
		challengePoolBalance: challengePoolBalance,
		now:                  now,
		challengeCreation:    challenges,
	}
	f.setCancelPassRates()

	var ssc, txn, input, ctx = setupMocksFinishAllocation(
		t, sAllocation, blobbers, bStakes, scYaml, otherWritePools,
		state.Balance(challengePoolBalance), blobberOffer, wpBalance, thisExpires, now,
	)

	require.True(t, len(challenges) <= len(blobbers))
	for i, blobberChallenges := range challenges {
		var bc = BlobberChallenges{
			BlobberID: strconv.Itoa(i),
		}

		var ac = AllocationChallenges{
			AllocationID: sAllocation.ID,
			//OpenChallenges:   []*StorageChallenge{},
		}
		for _, created := range blobberChallenges {
			ac.OpenChallenges = append(ac.OpenChallenges, &AllocOpenChallenge{
				//AllocationID: sAllocation.ID,
				BlobberID: bc.BlobberID,
				CreatedAt: created,
			})
		}
		_, err := ctx.InsertTrieNode(bc.GetKey(ssc.ID), &bc)
		require.NoError(t, err)
		_, err = ctx.InsertTrieNode(ac.GetKey(ssc.ID), &ac)
		require.NoError(t, err)
	}

	resp, err := ssc.cancelAllocationRequest(txn, input, ctx)
	if err != nil {
		return err
	}
	require.EqualValues(t, "canceled", resp)

	var newScYaml = &Config{}
	newScYaml, err = ssc.getConfig(ctx, false)

	require.NoError(t, err)
	newAllb, err := ssc.getBlobbersList(ctx)
	require.NoError(t, err)
	newCp, err := ssc.getChallengePool(sAllocation.ID, ctx)
	require.NoError(t, err)
	newWp, err := ssc.getWritePool(sAllocation.Owner, ctx)
	require.NoError(t, err)
	var newAlloc *StorageAllocation
	newAlloc, err = ssc.getAllocation(sAllocation.ID, ctx)
	require.NoError(t, err)
	var sps []*stakePool
	for _, blobber := range blobbers {
		sp, err := ssc.getStakePool(blobber.ID, ctx)
		require.NoError(t, err)
		sps = append(sps, sp)
	}

	confirmFinalizeAllocation(t, f, *newScYaml, *newAllb, *newCp, *newWp, *newAlloc, wpBalance, sps, ctx)
	return nil
}

func testFinalizeAllocation(
	t *testing.T,
	sAllocation StorageAllocation,
	blobbers SortedBlobbers,
	bStakes [][]mockStakePool,
	scYaml Config,
	otherWritePools int,
	challengePoolBalance int64,
	blobberOffer int64,
	wpBalance state.Balance,
	thisExpires, now common.Timestamp,
) error {

	var f = formulaeFinalizeAllocation{
		t:                    t,
		scYaml:               scYaml,
		allocation:           sAllocation,
		blobbers:             blobbers,
		bStakes:              bStakes,
		challengePoolBalance: challengePoolBalance,
		now:                  now,
	}
	f.setFinilizationPassRates()

	var ssc, txn, input, ctx = setupMocksFinishAllocation(
		t, sAllocation, blobbers, bStakes, scYaml, otherWritePools,
		state.Balance(challengePoolBalance), blobberOffer, wpBalance, thisExpires, now,
	)

	resp, err := ssc.finalizeAllocation(txn, input, ctx)
	if err != nil {
		return err
	}
	require.EqualValues(t, "finalized", resp)

	var newScYaml = &Config{}
	newScYaml, err = ssc.getConfig(ctx, false)

	require.NoError(t, err)
	newAllb, err := ssc.getBlobbersList(ctx)
	require.NoError(t, err)
	newCp, err := ssc.getChallengePool(sAllocation.ID, ctx)
	require.NoError(t, err)
	newWp, err := ssc.getWritePool(sAllocation.Owner, ctx)
	require.NoError(t, err)
	var newAlloc *StorageAllocation
	newAlloc, err = ssc.getAllocation(sAllocation.ID, ctx)
	require.NoError(t, err)
	var sps []*stakePool
	for _, blobber := range blobbers {
		sp, err := ssc.getStakePool(blobber.ID, ctx)
		require.NoError(t, err)
		sps = append(sps, sp)
	}

	confirmFinalizeAllocation(t, f, *newScYaml, *newAllb, *newCp, *newWp, *newAlloc, wpBalance, sps, ctx)
	return nil
}

func confirmFinalizeAllocation(
	t *testing.T,
	f formulaeFinalizeAllocation,
	scYaml Config,
	_ StorageNodes,
	challengePool challengePool,
	allocationWritePool writePool,
	allocation StorageAllocation,
	wpStartBalance state.Balance,
	sps []*stakePool,
	ctx cstate.StateContextI,
) {
	require.EqualValues(t, 0, challengePool.Balance)

	var rewardDelegateTransfers = [][]bool{}
	var minLockdelegateTransfers = [][]bool{}
	for i := range f.bStakes {
		if len(f.bStakes[i]) > 0 {
			rewardDelegateTransfers = append(rewardDelegateTransfers, []bool{})
			minLockdelegateTransfers = append(minLockdelegateTransfers, []bool{})
			for range f.bStakes[i] {
				rewardDelegateTransfers[i] = append(rewardDelegateTransfers[i], false)
				minLockdelegateTransfers[i] = append(minLockdelegateTransfers[i], false)
			}
		}
	}

	f.blobberServiceCharge(0)
	f.minLockServiceCharge(0)
	f.blobberDelegateReward(0, 0)
	f.minLockDelegatePayment(0, 0)

	for i, sp := range sps {
		serviceCharge := f.blobberServiceCharge(i) + f.minLockServiceCharge(i)
		require.InDelta(t, serviceCharge, int64(sp.Reward), errDelta)
		for poolId, dp := range sp.Pools {
			wSplit := strings.Split(poolId, " ")
			dId, err := strconv.Atoi(wSplit[2])
			require.NoError(t, err)
			reward := f.blobberDelegateReward(i, dId) + f.minLockDelegatePayment(i, dId)
			require.InDelta(t, reward, int64(dp.Reward), errDelta)
		}
	}

}

func setupMocksFinishAllocation(
	t *testing.T,
	sAllocation StorageAllocation,
	blobbers SortedBlobbers,
	bStakes [][]mockStakePool,
	scYaml Config,
	otherWritePools int,
	challengePoolBalance state.Balance,
	blobberOffer int64,
	wpBalance state.Balance,
	thisExpires, now common.Timestamp,
) (*StorageSmartContract, *transaction.Transaction, []byte, cstate.StateContextI) {
	var err error
	var txn = &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: datastore.Key(transactionHash),
		},
		ClientID:     sAllocation.ID,
		ToClientID:   storageScId,
		CreationDate: now,
	}
	var ctx = &mockStateContext{
		ctx: *cstate.NewStateContext(
			nil,
			&util.MerklePatriciaTrie{},
			txn,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		),
		clientBalance: zcnToBalance(3.1),
		store:         make(map[string]util.MPTSerializable),
	}
	var ssc = &StorageSmartContract{
		&sci.SmartContract{
			ID: storageScId,
		},
	}

	sAllocation.WritePoolOwners = []string{sAllocation.Owner}
	_, err = ctx.InsertTrieNode(sAllocation.GetKey(ssc.ID), &sAllocation)
	require.NoError(t, err)

	var cPool = challengePool{
		ZcnPool: &tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      sAllocation.ID,
				Balance: challengePoolBalance,
			},
		},
	}
	require.NoError(t, cPool.save(ssc.ID, sAllocation.ID, ctx))

	var wPool = writePool{
		Pools: allocationPools{},
	}
	var newPool = &allocationPool{}
	newPool.ID = "first_mock_write_pool"
	newPool.Balance = state.Balance(wpBalance)
	newPool.AllocationID = sAllocation.ID
	newPool.Blobbers = blobberPools{}
	newPool.ExpireAt = now
	for i := 0; i < len(sAllocation.BlobberAllocs); i++ {
		newPool.Blobbers.add(&blobberPool{
			BlobberID: blobbers[i].ID,
			Balance:   state.Balance(1),
		})
	}

	awp := &allocationWritePools{
		ownerId:    0,
		ids:        []string{sAllocation.Owner},
		writePools: []*writePool{&wPool},
	}
	awp.allocationPools.add(newPool)

	wPool.Pools.add(newPool)
	for i := 0; i < otherWritePools; i++ {
		var id = strconv.Itoa(i)
		var newPool = &allocationPool{}
		newPool.ID = "mock_write_pool_" + id
		newPool.AllocationID = allocationId + " " + id
		wPool.Pools.add(newPool)
		awp.allocationPools.add(newPool)
	}
	require.NoError(t, wPool.save(ssc.ID, sAllocation.Owner, ctx))
	require.NoError(t, awp.saveWritePools(ssc.ID, ctx))

	var blobberList = new(StorageNodes)
	blobberList.Nodes = blobbers
	_, err = ctx.InsertTrieNode(ALL_BLOBBERS_KEY, blobberList)
	require.NoError(t, err)

	require.EqualValues(t, len(blobbers), len(bStakes))
	for i, blobber := range blobbers {
		var id = strconv.Itoa(i)
		var sp = newStakePool()
		sp.Settings.ServiceCharge = blobberYaml.serviceCharge
		for j, stake := range bStakes[i] {
			var jd = strconv.Itoa(j)
			var delegatePool = &stakepool.DelegatePool{}
			delegatePool.Balance = zcnToBalance(stake.zcnAmount)
			//delegatePool.DelegateID = "delegate " + id + " " + jd
			//delegatePool.MintAt = stake.MintAt
			sp.Pools["paula "+id+" "+jd] = delegatePool
			sp.Pools["paula "+id+" "+jd] = delegatePool
		}
		sp.Settings.DelegateWallet = blobberId + " " + id + " wallet"
		require.NoError(t, sp.save(ssc.ID, blobber.ID, ctx))

		_, err = ctx.InsertTrieNode(blobber.GetKey(ssc.ID), blobber)
		require.NoError(t, err)
	}

	_, err = ctx.InsertTrieNode(scConfigKey(ssc.ID), &scYaml)
	require.NoError(t, err)

	var request = lockRequest{
		AllocationID: sAllocation.ID,
	}
	input, err := json.Marshal(&request)
	require.NoError(t, err)

	return ssc, txn, input, ctx
}

type formulaeFinalizeAllocation struct {
	t                    *testing.T
	scYaml               Config
	now                  common.Timestamp
	allocation           StorageAllocation
	blobbers             SortedBlobbers
	bStakes              [][]mockStakePool
	challengePoolBalance int64
	challengeCreation    [][]common.Timestamp
	_passRates           []float64
}

func (f *formulaeFinalizeAllocation) _challengePool() int64 {
	return f.challengePoolBalance
}

func (f *formulaeFinalizeAllocation) _minLockPayment(blobber int) int64 {
	require.True(f.t, blobber < len(f.allocation.BlobberAllocs))
	var details = f.allocation.BlobberAllocs[blobber]
	var minLock = int64(details.MinLockDemand)

	var spent = int64(details.Spent)

	if minLock > spent {
		return minLock - spent
	} else {
		return 0
	}
}

func (f *formulaeFinalizeAllocation) minLockServiceCharge(blobber int) int64 {
	var serviceCharge = blobberYaml.serviceCharge
	var blobberMinLock = float64(f._minLockPayment(blobber))

	return int64(blobberMinLock * serviceCharge)
}

func (f *formulaeFinalizeAllocation) minLockDelegatePayment(blobber, delegate int) int64 {
	require.True(f.t, blobber < len(f.bStakes))
	require.True(f.t, delegate < len(f.bStakes[blobber]))
	var totalStake = 0.0
	for _, stake := range f.bStakes[blobber] {
		totalStake += stake.zcnAmount
	}
	var delegateStake = f.bStakes[blobber][delegate].zcnAmount
	var delegateMinLock = float64(f._minLockPayment(blobber) - f.minLockServiceCharge(blobber))

	require.True(f.t, totalStake > 0)
	return int64(delegateMinLock * delegateStake / totalStake)
}

func (f *formulaeFinalizeAllocation) blobberServiceCharge(blobberIndex int) int64 {
	var serviceCharge = blobberYaml.serviceCharge
	var blobberRewards = float64(f._blobberReward(blobberIndex))

	return int64(blobberRewards * serviceCharge)
}

func (f *formulaeFinalizeAllocation) blobberDelegateReward(bIndex, dIndex int) int64 {
	require.True(f.t, bIndex < len(f.bStakes))
	require.True(f.t, dIndex < len(f.bStakes[bIndex]))
	var totalStake = 0.0
	for _, stake := range f.bStakes[bIndex] {
		totalStake += stake.zcnAmount
	}
	var delegateStake = f.bStakes[bIndex][dIndex].zcnAmount
	var totalDelegateReward = f._blobberReward(bIndex) - float64(f.blobberServiceCharge(bIndex))

	require.True(f.t, totalStake > 0)
	return int64(float64(totalDelegateReward) * delegateStake / totalStake)
}

func (f *formulaeFinalizeAllocation) _blobberReward(blobberIndex int) float64 {
	var challengePool = float64(f._challengePool())

	var used = float64(f.allocation.BlobberAllocs[blobberIndex].Stats.UsedSize)
	var totalUsed = float64(f.allocation.UsedSize)
	var abdUsed int64 = 0
	for _, d := range f.allocation.BlobberAllocs {
		abdUsed += d.Stats.UsedSize
	}
	require.InDelta(f.t, totalUsed, abdUsed, errDelta)

	var ratio = used / totalUsed
	var passRate = f._passRates[blobberIndex]

	return challengePool * ratio * passRate
}

func (f *formulaeFinalizeAllocation) setCancelPassRates() {
	f._passRates = []float64{}
	var deadline = f.now - toSeconds(blobberYaml.challengeCompletionTime)

	for i, details := range f.allocation.BlobberAllocs {
		var successful = float64(details.Stats.SuccessChallenges)
		var failed = float64(details.Stats.FailedChallenges)

		require.Len(f.t, f.challengeCreation[i], int(details.Stats.OpenChallenges))
		for _, created := range f.challengeCreation[i] {
			if created < deadline {
				failed++
			} else {
				successful++
			}
		}
		var total = successful + failed
		//fmt.Println("pass rate i", i, "successful", successful, "failed", failed)
		if total == 0 {
			f._passRates = append(f._passRates, 1.0)
		} else {
			f._passRates = append(f._passRates, successful/total)
		}
	}
}

func (f *formulaeFinalizeAllocation) setFinilizationPassRates() {
	f._passRates = []float64{}

	for _, details := range f.allocation.BlobberAllocs {
		var successful = float64(details.Stats.SuccessChallenges)
		var failed = float64(details.Stats.FailedChallenges + details.Stats.OpenChallenges)
		var total = successful + failed
		if total == 0 {
			f._passRates = append(f._passRates, 1.0)
		} else {
			f._passRates = append(f._passRates, successful/total)
		}
	}
}

func testNewAllocation(t *testing.T, request newAllocationRequest, blobbers SortedBlobbers,
	scYaml Config, blobberYaml mockBlobberYaml, stakes blobberStakes,
) (err error) {
	require.EqualValues(t, len(blobbers), len(stakes))
	var f = formulaeCommitNewAllocation{
		scYaml:      scYaml,
		blobberYaml: blobberYaml,
		request:     request,
		blobbers:    blobbers,
		stakes:      stakes,
	}

	var txn = &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: datastore.Key(transactionHash),
		},
		Value:        request.Size,
		ClientID:     clientId,
		ToClientID:   storageScId,
		CreationDate: creationDate,
	}
	var ctx = &mockStateContext{
		ctx: *cstate.NewStateContext(
			nil,
			&util.MerklePatriciaTrie{},
			txn,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		),
		clientBalance: zcnToBalance(3),
		store:         make(map[string]util.MPTSerializable),
	}
	var ssc = &StorageSmartContract{
		&sci.SmartContract{
			ID: storageScId,
		},
	}

	input, err := json.Marshal(request)
	require.NoError(t, err)

	var blobberList = new(StorageNodes)
	blobberList.Nodes = blobbers
	_, err = ctx.InsertTrieNode(ALL_BLOBBERS_KEY, blobberList)
	require.NoError(t, err)

	for i, blobber := range blobbers {
		var stakePool = newStakePool()
		stakePool.Pools["paula"] = &stakepool.DelegatePool{}
		stakePool.Pools["paula"].Balance = state.Balance(stakes[i])
		require.NoError(t, stakePool.save(ssc.ID, blobber.ID, ctx))
	}

	var wPool = writePool{}
	require.NoError(t, wPool.save(ssc.ID, clientId, ctx))

	_, err = ctx.InsertTrieNode(scConfigKey(ssc.ID), &scYaml)
	require.NoError(t, err)

	_, err = ssc.newAllocationRequest(txn, input, ctx)
	if err != nil {
		return err
	}

	allBlobbersList, err := ssc.getBlobbersList(ctx)
	require.NoError(t, err)
	var individualBlobbers = SortedBlobbers{}
	for _, blobber := range allBlobbersList.Nodes {
		var b *StorageNode
		b, err = ssc.getBlobber(blobber.ID, ctx)
		if err != nil && err.Error() == errValueNotPresent {
			continue
		}
		require.NoError(t, err)
		individualBlobbers.add(b)
	}

	var newStakePools = []*stakePool{}
	for _, blobber := range allBlobbersList.Nodes {
		var sp, err = ssc.getStakePool(blobber.ID, ctx)
		require.NoError(t, err)
		newStakePools = append(newStakePools, sp)
	}
	var wp *writePool
	wp, err = ssc.getWritePool(clientId, ctx)
	require.NoError(t, err)

	confirmTestNewAllocation(t, f, allBlobbersList.Nodes, individualBlobbers, newStakePools, *wp, ctx)

	return nil
}

type formulaeCommitNewAllocation struct {
	scYaml      Config
	blobberYaml mockBlobberYaml
	request     newAllocationRequest
	blobbers    SortedBlobbers
	stakes      blobberStakes
}

func (f formulaeCommitNewAllocation) blobbersUsed() int {
	return f.request.ParityShards + f.request.DataShards
}

func (f formulaeCommitNewAllocation) blobberEarnt(t *testing.T, id string, used []string) int64 {
	var totalWritePrice = 0.0
	var found = false
	for _, bId := range used {
		if bId == id {
			found = true
		}
		b, ok := f.blobbers.get(bId)
		require.True(t, ok)
		totalWritePrice += float64(b.Terms.WritePrice)
	}
	require.True(t, found)

	thisBlobber, ok := f.blobbers.get(id)
	require.True(t, ok)
	var ratio = float64(thisBlobber.Terms.WritePrice) / totalWritePrice
	var sizeOfWrite = float64(f.request.Size)

	return int64(sizeOfWrite * ratio)
}

func (f formulaeCommitNewAllocation) sizePerUsedBlobber() int64 {
	var numBlobbersUsed = int64(f.blobbersUsed())
	var writeSize = f.request.Size

	return (writeSize + numBlobbersUsed - 1) / numBlobbersUsed
}

func (f formulaeCommitNewAllocation) capacityUsedBlobber(t *testing.T, id string) int64 {
	var thisBlobber, ok = f.blobbers.get(id)
	require.True(t, ok)
	var usedAlready = thisBlobber.Used
	var newAllocament = f.sizePerUsedBlobber()

	return usedAlready + newAllocament
}

func confirmTestNewAllocation(t *testing.T, f formulaeCommitNewAllocation,
	blobbers1, blobbers2 SortedBlobbers, stakes []*stakePool, wp writePool, ctx cstate.StateContextI,
) {
	var transfers = ctx.GetTransfers()
	require.Len(t, transfers, 1)
	require.EqualValues(t, clientId, transfers[0].ClientID)
	require.EqualValues(t, storageScId, transfers[0].ToClientID)
	require.EqualValues(t, f.request.Size, transfers[0].Amount)

	require.Len(t, wp.Pools, 1)
	require.EqualValues(t, transactionHash, wp.Pools[0].ID)
	require.EqualValues(t, transactionHash, wp.Pools[0].AllocationID)
	require.EqualValues(t, f.request.Size, wp.Pools[0].Balance)
	require.Len(t, wp.Pools[0].Blobbers, f.blobbersUsed())
	var blobbersUsed []string
	for _, blobber := range wp.Pools[0].Blobbers {
		blobbersUsed = append(blobbersUsed, blobber.BlobberID)
	}
	for _, blobber := range wp.Pools[0].Blobbers {
		require.EqualValues(t, f.blobberEarnt(t, blobber.BlobberID, blobbersUsed), blobber.Balance)
	}

	var countUsedBlobbers = 0
	for _, blobber := range blobbers1 {
		b, ok := f.blobbers.get(blobber.ID)
		require.True(t, ok)
		if blobber.Used > b.Used {
			require.EqualValues(t, f.capacityUsedBlobber(t, blobber.ID), blobber.Used)
			countUsedBlobbers++
		}
	}
	require.EqualValues(t, f.blobbersUsed(), countUsedBlobbers)

	require.EqualValues(t, f.blobbersUsed(), len(blobbers2))
	for _, blobber := range blobbers2 {
		require.EqualValues(t, f.capacityUsedBlobber(t, blobber.ID), blobber.Used)
	}
}
