package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

type blobberStakes []int64

const (
	errValueNotPresent = "value not present"
	ownerId            = "owin"
)

func TestNewAllocation(t *testing.T) {
	t.Skip()
	var stakes = blobberStakes{}
	var now = common.Timestamp(10000)
	scYaml = &scConfig{
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
	}
	var goodBlobber = StorageNode{
		Capacity: 536870912,
		Used:     73,
		Terms: Terms{
			MaxOfferDuration:        1000 * scYaml.MinAllocDuration,
			ReadPrice:               zcnToBalance(blobberYaml.readPrice),
			ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
		},
		LastHealthCheck: now - blobberHealthTime,
	}
	var blobbers = new(sortedBlobbers)
	var stake = int64(scYaml.MaxStake)
	var writePrice = blobberYaml.writePrice
	for i := 0; i < request.DataShards+request.ParityShards+4; i++ {
		var nextBlobber = goodBlobber
		nextBlobber.ID = strconv.Itoa(i)
		nextBlobber.Terms.WritePrice = zcnToBalance(writePrice)
		writePrice *= 0.9
		blobbers.add(&nextBlobber)
		stakes = append(stakes, stake)
		stake = stake / 10
	}

	t.Run("new allocation", func(t *testing.T) {
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

/*
func TestCancelAllocationRequest(t *testing.T) {
	var now = common.Timestamp(300)
	var blobberStakePools = [][]mockStakePool{}
	var challenges = [][]common.Timestamp{}
	var scYaml = scConfig{
		MaxMint: zcnToBalance(4000000.0),
		StakePool: &stakePoolConfig{
			InterestRate:     0.0000334,
			InterestInterval: time.Minute,
		},
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
		Used:     73,
		Terms: Terms{
			MaxOfferDuration:        1000 * scYaml.MinAllocDuration,
			ReadPrice:               zcnToBalance(blobberYaml.readPrice),
			ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
		},
		LastHealthCheck: now - blobberHealthTime,
	}
	var allocation = StorageAllocation{
		DataShards:     2, //5,
		ParityShards:   2, //6,
		ID:             ownerId,
		BlobberDetails: []*BlobberAllocation{},
		Owner:          ownerId,
		Expiration:     now,
		Stats: &StorageAllocationStats{
			OpenChallenges: 3,
		},
		Size:     4560,
		UsedSize: 456,
	}
	var blobbers = new(sortedBlobbers)
	var stake = 100.0
	var writePrice = blobberYaml.writePrice
	var writePoolBalances = []int64{}
	var extraBlobbers = 0
	var period = common.Timestamp(scYaml.StakePool.InterestInterval.Seconds())
	var blobberUsedSize = allocation.UsedSize / int64(allocation.DataShards+allocation.ParityShards+extraBlobbers)
	for i := 0; i < allocation.DataShards+allocation.ParityShards+extraBlobbers; i++ {
		var nextBlobber = blobberTemplate
		nextBlobber.ID = strconv.Itoa(i)
		nextBlobber.Terms.WritePrice = zcnToBalance(writePrice)
		writePrice *= 0.9
		var minLockDemand = float64(allocation.Size) * writePrice * blobberYaml.minLockDemand
		blobbers.add(&nextBlobber)
		blobberStakePools = append(blobberStakePools, []mockStakePool{})
		blobberStakePools[i] = append(blobberStakePools[i], mockStakePool{
			zcnAmount: stake, MintAt: now - 2*period,
		})
		blobberStakePools[i] = append(blobberStakePools[i], mockStakePool{
			zcnAmount: 0.258, MintAt: now - 3*period,
		})
		stake = stake / 10
		if i < allocation.DataShards+allocation.ParityShards {
			challenges = append(challenges, []common.Timestamp{
				now - toSeconds(blobberYaml.challengeCompletionTime) - 1,
				now - toSeconds(blobberYaml.challengeCompletionTime) - 10,
			})
			allocation.BlobberDetails = append(allocation.BlobberDetails, &BlobberAllocation{
				BlobberID: nextBlobber.ID,
				Terms: Terms{
					ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
				},
				Stats: &StorageAllocationStats{
					UsedSize:          blobberUsedSize,
					TotalChallenges:   int64(len(challenges[i]) + i),
					OpenChallenges:    int64(len(challenges[i])),
					SuccessChallenges: int64(i),
				},
				MinLockDemand: 200 + state.Balance(minLockDemand),
				Spent:         100,
			})
		}
		writePoolBalances = append(writePoolBalances, 3000000)
	}

	var challengePoolIntegralValue = state.Balance(73000000)
	var challengePoolBalance = state.Balance(700000)
	var partial = 0.9
	var preiviousChallenge = common.Timestamp(3)
	var thisChallenge = common.Timestamp(5)
	var thisExpires = common.Timestamp(222)

	var blobberOffer = int64(123000)
	var otherWritePools = 4

	t.Run("cancelAllocationRequest", func(t *testing.T) {
		err := testFinalizeAllocation(t, allocation, *blobbers, blobberStakePools, scYaml, blobberYaml,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, blobberOffer, challenges, preiviousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

}
*/
func TestFinalizeAllocation(t *testing.T) {
	//t.Skip()
	var now = common.Timestamp(300)
	var blobberStakePools = [][]mockStakePool{}
	//var challenges = [][]common.Timestamp{}
	var scYaml = scConfig{
		MaxMint: zcnToBalance(4000000.0),
		StakePool: &stakePoolConfig{
			InterestRate:     0.0000334,
			InterestInterval: time.Minute,
		},
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
		//Used:     73,
		Terms: Terms{
			MaxOfferDuration:        1000 * scYaml.MinAllocDuration,
			ReadPrice:               zcnToBalance(blobberYaml.readPrice),
			ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
		},
		LastHealthCheck: now - blobberHealthTime,
	}
	var allocation = StorageAllocation{
		DataShards:     2, //5,
		ParityShards:   2, //6,
		ID:             ownerId,
		BlobberDetails: []*BlobberAllocation{},
		Owner:          ownerId,
		Expiration:     now,
		Stats: &StorageAllocationStats{
			OpenChallenges: 3,
		},
		Size:     4560,
		UsedSize: 456,
	}
	var blobbers = new(sortedBlobbers)
	var stake = 100.0
	var writePrice = blobberYaml.writePrice
	var writePoolBalances = []int64{}
	var extraBlobbers = 0
	var period = common.Timestamp(scYaml.StakePool.InterestInterval.Seconds())
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
			zcnAmount: stake, MintAt: now - 2*period,
		})
		blobberStakePools[i] = append(blobberStakePools[i], mockStakePool{
			zcnAmount: 0.258, MintAt: now - 3*period,
		})
		stake = stake / 10
		if i < allocation.DataShards+allocation.ParityShards {
			allocation.BlobberDetails = append(allocation.BlobberDetails, &BlobberAllocation{
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
		writePoolBalances = append(writePoolBalances, 3000000)
	}

	var challengePoolIntegralValue = state.Balance(73000000)
	var challengePoolBalance = state.Balance(700000)
	var partial = 0.9
	var preiviousChallenge = common.Timestamp(3)
	var thisChallenge = common.Timestamp(5)
	var thisExpires = common.Timestamp(222)

	var blobberOffer = int64(123000)
	var otherWritePools = 4

	t.Run("finialzeAllocation", func(t *testing.T) {
		err := testFinalizeAllocation(t, allocation, *blobbers, blobberStakePools, scYaml, blobberYaml,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, blobberOffer, preiviousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})
}

func testFinalizeAllocation(
	t *testing.T,
	sAllocation StorageAllocation,
	blobbers sortedBlobbers,
	bStakes [][]mockStakePool,
	scYaml scConfig,
	blobberYaml mockBlobberYaml,
	wpBalances []int64,
	otherWritePools int,
	challengePoolIntegralValue, challengePoolBalance state.Balance,
	partial float64,
	blobberOffer int64,
	//_ [][]common.Timestamp,
	previous, thisChallange, thisExpires, now common.Timestamp,
) error {

	var f = formulaeFinalChallenge{
		t:                    t,
		scYaml:               scYaml,
		allocation:           sAllocation,
		blobbers:             blobbers,
		bStakes:              bStakes,
		challengePoolBalance: int64(challengePoolBalance),
		now:                  now,
	}

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
			&state.Deserializer{},
			txn,
			nil,
			nil,
			nil,
			nil,
		),
		clientBalance: zcnToBalance(3.1),
		store:         make(map[datastore.Key]util.Serializable),
	}
	var ssc = &StorageSmartContract{
		&sci.SmartContract{
			ID: storageScId,
		},
	}

	_, err = ctx.InsertTrieNode(sAllocation.GetKey(ssc.ID), &sAllocation)
	require.NoError(t, err)

	var allications = Allocations{}
	allications.List.add(sAllocation.ID)
	_, err = ctx.InsertTrieNode(ALL_ALLOCATIONS_KEY, &allications)

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
	require.True(t, len(blobbers) >= len(wpBalances))
	for i, balance := range wpBalances {
		var newPool = &allocationPool{}
		newPool.Balance = state.Balance(balance)
		newPool.AllocationID = sAllocation.ID
		newPool.Blobbers = blobberPools{}
		newPool.Blobbers.add(&blobberPool{
			BlobberID: blobbers[i].ID,
			Balance:   state.Balance(balance),
		})
		newPool.ExpireAt = now
		wPool.Pools.add(newPool)
	}
	for i := 0; i < otherWritePools; i++ {
		var id = strconv.Itoa(i)
		var newPool = &allocationPool{}
		newPool.AllocationID = allocationId + " " + id
		wPool.Pools.add(newPool)
	}
	require.NoError(t, wPool.save(ssc.ID, sAllocation.Owner, ctx))

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
			var delegatePool = &delegatePool{}
			delegatePool.Balance = zcnToBalance(stake.zcnAmount)
			delegatePool.DelegateID = "delegate " + id + " " + jd
			delegatePool.MintAt = stake.MintAt
			sp.Pools["paula "+id+" "+jd] = delegatePool
			sp.Pools["paula "+id+" "+jd] = delegatePool
		}
		sp.Offers[sAllocation.ID] = &offerPool{
			Expire: thisExpires,
			Lock:   state.Balance(blobberOffer),
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
	/*
		require.True(t, len(challenges) <= len(blobbers))
		for i, blobbersChallenges := range challenges {
			var bc = BlobberChallenge{
				BlobberID:  strconv.Itoa(i),
				//Challenges: []*StorageChallenge{},
			}
			for _, created := range blobbersChallenges {
				bc.Challenges = append(bc.Challenges, &StorageChallenge{
					Created: created,
				})
			}
			_, err = ctx.InsertTrieNode(bc.GetKey(ssc.ID), &bc)
			require.NoError(t, err)
		}
	*/
	allAllocationsBefore, err := ssc.getAllAllocationsList(ctx)

	resp, err := ssc.finalizeAllocation(txn, input, ctx)
	if err != nil {
		return err
	}
	require.EqualValues(t, "finalized", resp)

	allAllocationsAfter, err := ssc.getAllAllocationsList(ctx)
	require.EqualValues(t, len(allAllocationsBefore.List)-1, len(allAllocationsAfter.List))

	var newScYaml = &scConfig{}
	newScYaml, err = ssc.getConfig(ctx, false)
	allb, err := ssc.getBlobbersList(ctx)
	cp, err := ssc.getChallengePool(request.AllocationID, ctx)
	wp, err := ssc.getWritePool(sAllocation.Owner, ctx)
	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(request.AllocationID, ctx)

	confirmFinalizeAllocation(t, f, *newScYaml, *allb, *cp, *wp, *alloc, ctx)
	return nil
}

func confirmFinalizeAllocation(t *testing.T, f formulaeFinalChallenge, scYaml scConfig,
	allBlobbers StorageNodes, challengePool challengePool, allocationWritePool writePool,
	allocation StorageAllocation, ctx cstate.StateContextI) {
	var minted = f.scYaml.Minted
	require.EqualValues(t, 0, challengePool.Balance)

	var delegateMints = [][]bool{}
	for i := range f.bStakes {
		if len(f.bStakes[i]) > 0 {
			delegateMints = append(delegateMints, []bool{})
			for _ = range f.bStakes[i] {
				delegateMints[i] = append(delegateMints[i], false)
			}
		}
	}
	for _, mint := range ctx.GetMints() {
		require.EqualValues(t, storageScId, mint.Minter)
		var wSplit = strings.Split(mint.ToClientID, " ")
		require.Len(t, wSplit, 3)
		require.EqualValues(t, wSplit[0], "delegate")
		dIndex, err := strconv.Atoi(wSplit[2])
		require.NoError(t, err)
		bIndex, err := strconv.Atoi(wSplit[1])
		require.False(t, delegateMints[bIndex][dIndex])
		require.InDelta(t, f.delegateInterest(wSplit[1], dIndex), int64(mint.Amount), errDelta)
		minted += mint.Amount
		delegateMints[bIndex][dIndex] = true
	}
	require.EqualValues(t, minted, scYaml.Minted)
	for i := range delegateMints {
		for j, minted := range delegateMints[i] {
			if !minted {
				require.InDelta(t, f.delegateInterest(strconv.Itoa(i), j), 0, errDelta)
			}
		}
	}

	var rewardTransfers = []bool{}
	var minLockTransfers = []bool{}
	var rewardDelegateTransfers = [][]bool{}
	var minLockdelegateTransfers = [][]bool{}
	for i := range f.bStakes {
		rewardTransfers = append(rewardTransfers, false)
		minLockTransfers = append(minLockTransfers, false)
		if len(f.bStakes[i]) > 0 {
			rewardDelegateTransfers = append(rewardDelegateTransfers, []bool{})
			minLockdelegateTransfers = append(minLockdelegateTransfers, []bool{})
			for _ = range f.bStakes[i] {
				rewardDelegateTransfers[i] = append(rewardDelegateTransfers[i], false)
				minLockdelegateTransfers[i] = append(minLockdelegateTransfers[i], false)
			}
		}
	}

	var amountTransferred = int64(0)
	for _, transfer := range ctx.GetTransfers() {
		amountTransferred += int64(transfer.Amount)
		require.EqualValues(t, storageScId, transfer.ClientID)
		var wSplit = strings.Split(transfer.ToClientID, " ")
		require.Len(t, wSplit, 3)
		bId, err := strconv.Atoi(wSplit[1])
		require.NoError(t, err)
		if wSplit[0] == blobberId {
			if !rewardTransfers[bId] {
				var fbsc = f.blobberServiceCharge(bId)
				fbsc = fbsc
				var abs = math.Abs(float64(f.blobberServiceCharge(bId) - int64(transfer.Amount)))
				abs = abs
				if math.Abs(float64(f.blobberServiceCharge(bId)-int64(transfer.Amount))) <= errDelta {
					rewardTransfers[bId] = true
					continue
				}
			}
			require.False(t, minLockTransfers[bId])
			require.InDelta(t, f.blobberMinLockPayment(bId), int64(transfer.Amount), errDelta)
			minLockTransfers[bId] = true
			continue
		}
		dId, err := strconv.Atoi(wSplit[2])
		require.NoError(t, err)
		if !rewardDelegateTransfers[bId][dId] {
			if math.Abs(float64(f.blobberDelegateReward(bId, dId)-int64(transfer.Amount))) <= errDelta {
				rewardDelegateTransfers[bId][dId] = true
				continue
			}
		}
		require.False(t, minLockdelegateTransfers[bId][dId])
		require.InDelta(t, f.delegateMinLockPayment(bId, dId), int64(transfer.Amount), errDelta)
		minLockdelegateTransfers[bId][dId] = true
	}

	for i, transfered := range minLockTransfers {
		if !transfered {
			require.InDelta(t, f.blobberMinLockPayment(i), 0, errDelta)
		}
	}
	for i, transfered := range rewardTransfers {
		if !transfered {
			require.InDelta(t, f.blobberServiceCharge(i), 0, errDelta)
		}
	}

	for i := range rewardDelegateTransfers {
		for j, transfered := range rewardDelegateTransfers[i] {
			if !transfered {
				require.InDelta(t, f.blobberDelegateReward(i, j), 0, errDelta)
			}
		}
	}
	for i := range minLockdelegateTransfers {
		for j, transfered := range minLockdelegateTransfers[i] {
			if !transfered {
				require.InDelta(t, f.blobberDelegateReward(i, j), 0, errDelta)
			}
		}
	}

}

type formulaeFinalChallenge struct {
	t                    *testing.T
	scYaml               scConfig
	now                  common.Timestamp
	allocation           StorageAllocation
	blobbers             sortedBlobbers
	bStakes              [][]mockStakePool
	challengePoolBalance int64
}

func (f formulaeFinalChallenge) minLockPayment(blobber int) int64 {
	require.True(f.t, blobber < len(f.allocation.BlobberDetails))
	var details = f.allocation.BlobberDetails[blobber]
	var minLock = int64(details.MinLockDemand)

	var spentBefore = int64(details.Spent)
	var spentFinalising = f.blobberReward(blobber)
	var totalSpent = spentBefore + spentFinalising

	if minLock > totalSpent {
		return minLock - totalSpent
	} else {
		return 0
	}
}

func (f formulaeFinalChallenge) blobberMinLockPayment(blobber int) int64 {
	var serviceCharge = blobberYaml.serviceCharge
	var blobberMinLock = float64(f.minLockPayment(blobber))

	return int64(blobberMinLock * serviceCharge)
}

func (f formulaeFinalChallenge) delegateMinLockPayment(blobber, delegate int) int64 {
	require.True(f.t, blobber < len(f.bStakes))
	require.True(f.t, delegate < len(f.bStakes[blobber]))
	var totalStake = 0.0
	for _, stake := range f.bStakes[blobber] {
		totalStake += stake.zcnAmount
	}
	var delegateStake = f.bStakes[blobber][delegate].zcnAmount
	var delegateMinLock = float64(f.minLockPayment(blobber) - f.blobberMinLockPayment(blobber))

	require.True(f.t, totalStake > 0)
	return int64(delegateMinLock * delegateStake / totalStake)
}

func (f formulaeFinalChallenge) delegateInterest(blobber string, delegate int) int64 {
	var interestRate = f.scYaml.StakePool.InterestRate
	blobberIndex, err := strconv.Atoi(blobber)
	require.NoError(f.t, err)
	var numberOfPayments = float64(f.numberOfInterestPayments(blobberIndex, delegate))
	var stake = float64(zcnToInt64(f.bStakes[blobberIndex][delegate].zcnAmount))

	return int64(stake * numberOfPayments * interestRate)
}

func (f formulaeFinalChallenge) numberOfInterestPayments(blobberIndex, delegate int) int64 {
	var activeTime = int64(f.now - f.bStakes[blobberIndex][delegate].MintAt)
	var period = int64(f.scYaml.StakePool.InterestInterval.Seconds())
	var periods = activeTime / period

	// round down to previous integer
	if activeTime%period == 0 {
		if periods-1 >= 0 {
			return periods - 1
		} else {
			return 0
		}
	} else {
		return periods
	}
}

func (f formulaeFinalChallenge) blobberServiceCharge(blobberIndex int) int64 {
	var serviceCharge = blobberYaml.serviceCharge
	var blobberRewards = float64(f.blobberReward(blobberIndex))

	return int64(blobberRewards * serviceCharge)
}

func (f formulaeFinalChallenge) blobberDelegateReward(bIndex, dIndex int) int64 {
	require.True(f.t, bIndex < len(f.bStakes))
	require.True(f.t, dIndex < len(f.bStakes[bIndex]))
	var totalStake = 0.0
	for _, stake := range f.bStakes[bIndex] {
		totalStake += stake.zcnAmount
	}
	var delegateStake = f.bStakes[bIndex][dIndex].zcnAmount
	var totalDelegateReward = float64(f.blobberReward(bIndex) - f.blobberServiceCharge(bIndex))

	require.True(f.t, totalStake > 0)
	return int64(totalDelegateReward * delegateStake / totalStake)
}

func (f formulaeFinalChallenge) blobberReward(blobberIndex int) int64 {
	var challengePool = float64(f.challengePoolBalance)

	var used = float64(f.allocation.BlobberDetails[blobberIndex].Stats.UsedSize)
	var totalUsed = float64(f.allocation.UsedSize)
	var ratio = used / totalUsed

	var passRate = f.passRate(blobberIndex)

	return int64(challengePool * ratio * passRate)
}

func (f formulaeFinalChallenge) passRate(blobberIndex int) float64 {
	var stats = f.allocation.BlobberDetails[blobberIndex].Stats
	var successful = float64(stats.SuccessChallenges)
	var failed = float64(stats.FailedChallenges + stats.OpenChallenges)
	var total = successful + failed

	if total == 0 {
		return 1.0
	} else {
		return successful / total
	}
}

func testNewAllocation(t *testing.T, request newAllocationRequest, blobbers sortedBlobbers,
	scYaml scConfig, blobberYaml mockBlobberYaml, stakes blobberStakes,
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
			&state.Deserializer{},
			txn,
			nil,
			nil,
			nil,
			nil,
		),
		clientBalance: zcnToBalance(3),
		store:         make(map[datastore.Key]util.Serializable),
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
		stakePool.Pools["paula"] = &delegatePool{}
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
	var individualBlobbers = sortedBlobbers{}
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
	wp = wp

	confirmTestNewAllocation(t, f, allBlobbersList.Nodes, individualBlobbers, newStakePools, *wp, ctx)

	return nil
}

type formulaeCommitNewAllocation struct {
	scYaml      scConfig
	blobberYaml mockBlobberYaml
	request     newAllocationRequest
	blobbers    sortedBlobbers
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

func (f formulaeCommitNewAllocation) offerBlobber(index int) int64 {
	var amount = sizeInGB(f.sizePerUsedBlobber())
	var writePrice = float64(f.blobbers[index].Terms.WritePrice)

	return int64(amount * writePrice)
}

func (f formulaeCommitNewAllocation) capacityUsedBlobber(t *testing.T, id string) int64 {
	var thisBlobber, ok = f.blobbers.get(id)
	require.True(t, ok)
	var usedAlready = thisBlobber.Used
	var newAllocament = f.sizePerUsedBlobber()

	return usedAlready + newAllocament
}

func (f formulaeCommitNewAllocation) offerExpiration() common.Timestamp {
	var expiration = f.request.Expiration
	var challangeTime = f.request.MaxChallengeCompletionTime

	return expiration + toSeconds(challangeTime)
}

func confirmTestNewAllocation(t *testing.T, f formulaeCommitNewAllocation,
	blobbers1, blobbers2 sortedBlobbers, stakes []*stakePool, wp writePool, ctx cstate.StateContextI,
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

	var countOffers = 0
	for i, stake := range stakes {
		offer, ok := stake.Offers[transactionHash]
		if ok {
			require.EqualValues(t, f.offerBlobber(i), int64(offer.Lock))
			require.EqualValues(t, f.offerExpiration(), offer.Expire)
			countOffers++
		}
	}
	require.EqualValues(t, f.blobbersUsed(), countOffers)
}
