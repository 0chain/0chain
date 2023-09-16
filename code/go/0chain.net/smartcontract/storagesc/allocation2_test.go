package storagesc

import (
	"0chain.net/chaincore/block"
	"encoding/json"
	"fmt"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"0chain.net/core/config"

	"0chain.net/smartcontract/provider"

	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

type blobberStakes []int64

const (
	errValueNotPresent  = "value not present"
	ownerId             = "owin"
	ErrCancelFailed     = "alloc_cancel_failed"
	ErrExpired          = "trying to cancel expired allocation"
	ErrNotOwner         = "only owner can cancel an allocation"
	ErrFinalizedFailed  = "fini_alloc_failed"
	ErrFinalizedTooSoon = "allocation is not expired yet"
)

func TestNewAllocation(t *testing.T) {
	var stakes = blobberStakes{}
	var now = common.Timestamp(10000)

	var ctx = &mockStateContext{
		clientBalance: 1e12,
		store:         make(map[string]util.MPTSerializable),
	}

	setConfig(t, ctx)

	scYaml, err := getConfig(ctx)
	require.NoError(t, err)

	var blobberYaml = mockBlobberYaml{
		readPrice:  0.01,
		writePrice: 0.10,
	}

	var request = newAllocationRequest{
		Owner:           clientId,
		OwnerPublicKey:  "my public key",
		Size:            scYaml.MinAllocSize,
		DataShards:      3,
		ParityShards:    5,
		ReadPriceRange:  PriceRange{0, zcnToBalance(blobberYaml.readPrice) + 1},
		WritePriceRange: PriceRange{0, zcnToBalance(blobberYaml.writePrice) + 1},
		Blobbers: []string{"0", "1", "2", "3",
			"4", "5", "6", "7"},
	}
	var blobbers = new(SortedBlobbers)
	var stake = int64(scYaml.MaxStake)
	var writePrice = blobberYaml.writePrice
	for i := 0; i < request.DataShards+request.ParityShards+4; i++ {
		var nextBlobber = StorageNode{
			Provider: provider.Provider{
				ProviderType:    spenum.Blobber,
				LastHealthCheck: now - blobberHealthTime,
			},
			Capacity:  536870912,
			Allocated: 73,
			Terms: Terms{
				ReadPrice: zcnToBalance(blobberYaml.readPrice),
			},
		}
		nextBlobber.ID = strconv.Itoa(i)
		nextBlobber.Terms.WritePrice = zcnToBalance(writePrice)
		nextBlobber.BaseURL = "mockBaseUrl" + strconv.Itoa(i)
		writePrice *= 0.9
		blobbers.add(&nextBlobber)
		stakes = append(stakes, stake)
		stake = stake / 10
	}

	t.Run("new allocation", func(t *testing.T) {
		nar := request
		err := testNewAllocation(t, nar, *blobbers, blobberYaml, stakes, ctx)
		require.NoError(t, err)
	})

	t.Run("new allocation", func(t *testing.T) {
		nar := request
		nar.Size = 100 * GB

		err := testNewAllocation(t, nar, *blobbers, blobberYaml, stakes, ctx)
		require.NoError(t, err)
	})
}

func TestCancelAllocationRequest(t *testing.T) {
	var blobberStakePools [][]mockStakePool
	var challenges [][]int64

	var ctx = &mockStateContext{
		clientBalance: zcnToBalance(3.1),
		store:         make(map[string]util.MPTSerializable),
	}

	bk := &block.Block{}
	bk.Round = 1100
	ctx.StateContext = *cstate.NewStateContext(
		bk,
		&util.MerklePatriciaTrie{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	setConfig(t, ctx)

	scYaml, err := getConfig(ctx)
	require.NoError(t, err)

	var now = common.Timestamp(900)
	var blobberYaml = mockBlobberYaml{
		serviceCharge: 0.30,
		writePrice:    0.1,
	}

	var challengePoolBalance = int64(700000)

	var allocation = StorageAllocation{
		DataShards:    1,
		ParityShards:  1,
		ID:            ownerId,
		BlobberAllocs: []*BlobberAllocation{},
		Owner:         ownerId,
		Expiration:    now * 3,
		Stats: &StorageAllocationStats{
			UsedSize: 1073741824,
		},
		Size:          4560,
		WritePool:     400000000,
		MinLockDemand: scYaml.MinLockDemand,
	}
	var blobbers = new(SortedBlobbers)
	var stake = 100.0
	var writePrice = blobberYaml.writePrice
	var extraBlobbers = 0
	var blobberUsedSize = allocation.Stats.UsedSize / int64(allocation.DataShards)
	allocation.BlobberAllocsMap = make(map[string]*BlobberAllocation)
	for i := 0; i < allocation.DataShards+allocation.ParityShards+extraBlobbers; i++ {
		var nextBlobber = StorageNode{
			Provider: provider.Provider{
				ProviderType:    spenum.Blobber,
				LastHealthCheck: now - blobberHealthTime,
			},
			Capacity: 536870912,
			Terms: Terms{
				ReadPrice: zcnToBalance(blobberYaml.readPrice),
			},
		}
		nextBlobber.ID = strconv.Itoa(i)
		nextBlobber.ProviderType = spenum.Blobber
		nextBlobber.Terms.WritePrice = zcnToBalance(writePrice)
		writePrice *= 0.9
		var minLockDemand = float64(allocation.Size) * writePrice * allocation.MinLockDemand
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
			ba := &BlobberAllocation{
				AllocationID: allocation.ID,
				BlobberID:    nextBlobber.ID,
				Terms: Terms{
					WritePrice: zcnToBalance(blobberYaml.writePrice),
				},
				Stats: &StorageAllocationStats{
					UsedSize:        blobberUsedSize,
					OpenChallenges:  int64(i + 1),
					TotalChallenges: int64(i + 1),
				},
				MinLockDemand:                 200 + currency.Coin(minLockDemand),
				Spent:                         100,
				Size:                          1 * GB,
				LatestFinalizedChallCreatedAt: now - 200,
				ChallengePoolIntegralValue:    currency.Coin(challengePoolBalance / int64(allocation.DataShards+allocation.ParityShards)),
			}

			allocation.BlobberAllocs = append(allocation.BlobberAllocs, ba)
			allocation.BlobberAllocsMap[nextBlobber.ID] = ba
			allocation.Stats.OpenChallenges += ba.Stats.OpenChallenges
			allocation.Stats.TotalChallenges += ba.Stats.TotalChallenges

			challenges = append(challenges, []int64{})
			for j := 0; j < int(allocation.BlobberAllocs[i].Stats.OpenChallenges); j++ {
				var expires = int64(float64(ctx.GetBlock().Round) - float64(j+2)*float64(scYaml.MaxChallengeCompletionRounds)/2.0)
				challenges[i] = append(challenges[i], expires)
			}
		}
	}

	t.Run("cancel allocation", func(t *testing.T) {
		err := testCancelAllocation(t, allocation, *blobbers, blobberStakePools,
			challengePoolBalance, challenges, ctx, now)

		require.NoError(t, err)
	})

	t.Run(ErrNotOwner, func(t *testing.T) {
		var allocationNotOwner = allocation
		allocationNotOwner.Owner = "someone else"

		err := testCancelAllocation(t, allocationNotOwner, *blobbers, blobberStakePools,
			challengePoolBalance, challenges, ctx, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), ErrCancelFailed))
		require.True(t, strings.Contains(err.Error(), ErrNotOwner))
	})

	t.Run(ErrExpired, func(t *testing.T) {
		var allocationExpired = allocation
		allocationExpired.Expiration = now - 1

		err := testCancelAllocation(t, allocationExpired, *blobbers, blobberStakePools,
			challengePoolBalance, challenges, ctx, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), ErrCancelFailed))
		require.True(t, strings.Contains(err.Error(), ErrExpired))
	})
}

func TestFinalizeAllocation(t *testing.T) {
	var now = common.Timestamp(300)
	var blobberStakePools = [][]mockStakePool{}
	var challenges [][]int64

	var ctx = &mockStateContext{
		clientBalance: zcnToBalance(3.1),
		store:         make(map[string]util.MPTSerializable),
	}

	bk := &block.Block{}
	bk.Round = 1100
	ctx.StateContext = *cstate.NewStateContext(
		bk,
		&util.MerklePatriciaTrie{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	setConfig(t, ctx)

	scYaml, err := getConfig(ctx)
	require.NoError(t, err)

	var blobberYaml = mockBlobberYaml{
		serviceCharge: 0.30,
		writePrice:    0.1,
	}

	var allocation = StorageAllocation{
		DataShards:    5,
		ParityShards:  5,
		ID:            ownerId,
		BlobberAllocs: []*BlobberAllocation{},
		Owner:         ownerId,
		Expiration:    now - 180,
		Stats: &StorageAllocationStats{
			UsedSize:       205,
			OpenChallenges: 3,
		},
		Size: 4560,
	}
	var blobbers = new(SortedBlobbers)
	var stake = 100.0
	var writePrice = blobberYaml.writePrice
	var extraBlobbers = 0
	var blobberUsedSize = int64(float64(allocation.Stats.UsedSize) / float64(allocation.DataShards))

	allocation.BlobberAllocsMap = make(map[string]*BlobberAllocation)
	for i := 0; i < allocation.DataShards+allocation.ParityShards+extraBlobbers; i++ {
		var nextBlobber = StorageNode{
			Capacity: 536870912,
			Provider: provider.Provider{
				ProviderType:    spenum.Blobber,
				LastHealthCheck: now - blobberHealthTime,
			},
			Terms: Terms{
				ReadPrice: zcnToBalance(blobberYaml.readPrice),
			},
		}
		nextBlobber.ID = strconv.Itoa(i)
		nextBlobber.ProviderType = spenum.Blobber
		nextBlobber.Terms.WritePrice = zcnToBalance(writePrice)
		writePrice *= 0.9
		var minLockDemand = float64(allocation.Size) * writePrice * allocation.MinLockDemand
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
			ba := &BlobberAllocation{
				AllocationID: allocation.ID,
				BlobberID:    nextBlobber.ID,
				Terms: Terms{
					WritePrice: 1e9,
					ReadPrice:  0,
				},
				Stats: &StorageAllocationStats{
					UsedSize:        blobberUsedSize,
					OpenChallenges:  int64(i + 1),
					TotalChallenges: int64(i + 1), // add open challenges and success  challenges
				},
				MinLockDemand:                  200 + currency.Coin(minLockDemand),
				Spent:                          100,
				Size:                           1 * GB,
				LatestFinalizedChallCreatedAt:  allocation.Expiration / 6,
				LatestSuccessfulChallCreatedAt: allocation.Expiration / 8,
				ChallengePoolIntegralValue:     10000,
			}

			allocation.BlobberAllocs = append(allocation.BlobberAllocs, ba)
			allocation.BlobberAllocsMap[nextBlobber.ID] = ba
			allocation.Stats.OpenChallenges += ba.Stats.OpenChallenges
			allocation.Stats.TotalChallenges += ba.Stats.TotalChallenges

			challenges = append(challenges, []int64{})
			for j := 0; j < int(allocation.BlobberAllocs[i].Stats.OpenChallenges); j++ {
				var expires = int64(float64(ctx.GetBlock().Round) - float64(j)*float64(scYaml.MaxChallengeCompletionRounds)/3.0)
				challenges[i] = append(challenges[i], expires)
			}
		}
	}
	var challengePoolBalance = int64(7000000)

	allocation.WritePool = currency.Coin(10000000000000000000)

	t.Run("finalize allocation", func(t *testing.T) {
		err := testFinalizeAllocation(t, allocation, *blobbers, blobberStakePools, challengePoolBalance, allocation.Expiration, challenges, ctx)
		require.NoError(t, err)
	})

	t.Run(ErrFinalizedTooSoon, func(t *testing.T) {
		var allocationExpired = allocation
		allocationExpired.Expiration = now - toSeconds(0) + 1

		err := testFinalizeAllocation(t, allocationExpired, *blobbers, blobberStakePools, challengePoolBalance, now, challenges, ctx)
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
	challengePoolBalance int64,
	challenges [][]int64,
	ctx *mockStateContext,
	now common.Timestamp,
) error {

	scYaml, err := getConfig(ctx)
	require.NoError(t, err)

	var f = formulaeFinalizeAllocation{
		t:                    t,
		scYaml:               *scYaml,
		allocation:           sAllocation,
		blobbers:             blobbers,
		bStakes:              bStakes,
		challengePoolBalance: challengePoolBalance,
		now:                  now,
		challengeCreation:    challenges,
	}

	var ssc, txn, input = setupMocksFinishAllocation(
		t, sAllocation, blobbers, bStakes,
		currency.Coin(challengePoolBalance), now, ctx,
	)

	require.True(t, len(challenges) <= len(blobbers))

	ac := AllocationChallenges{
		AllocationID: sAllocation.ID,
	}

	for i, blobberChallenges := range challenges {
		blobberID := strconv.Itoa(i)

		err := partitionsChallengeReadyBlobberAddOrUpdate(ctx, blobberID, 1000)
		require.NoError(t, err)

		for _, created := range blobberChallenges {
			ac.OpenChallenges = append(ac.OpenChallenges, &AllocOpenChallenge{
				ID:             fmt.Sprintf("%s:%s:%v", sAllocation.ID, blobberID, created),
				BlobberID:      blobberID,
				RoundCreatedAt: created,
			})
		}
		_, err = ctx.InsertTrieNode(ac.GetKey(ssc.ID), &ac)
		require.NoError(t, err)
	}

	f.setFinilizationPassRates(ssc, ctx, *scYaml, now)

	resp, err := ssc.cancelAllocationRequest(txn, input, ctx)
	if err != nil {
		return err
	}
	require.EqualValues(t, "canceled", resp)

	require.NoError(t, err)
	newCp, err := ssc.getChallengePool(sAllocation.ID, ctx)
	require.NoError(t, err)
	var sps []*stakePool
	for _, blobber := range blobbers {
		sp, err := ssc.getStakePool(spenum.Blobber, blobber.ID, ctx)
		require.NoError(t, err)
		sps = append(sps, sp)
	}

	var cancellationCharges []int64
	totalCancellationCharge, _ := sAllocation.cancellationCharge(0.2)

	totalWritePrice := currency.Coin(0)

	for _, ba := range f.allocation.BlobberAllocs {
		totalWritePrice, err = currency.AddCoin(totalWritePrice, ba.Terms.WritePrice)
	}

	for i, ba := range f.allocation.BlobberAllocs {

		blobberWritePriceWeight := float64(ba.Terms.WritePrice) / float64(totalWritePrice)
		reward, err := currency.Float64ToCoin(float64(totalCancellationCharge) * blobberWritePriceWeight * f._passRates[i])

		if err != nil {
			return fmt.Errorf("failed to convert float to coin: %v", err)
		}

		cancellationCharges = append(cancellationCharges, int64(reward))
	}

	confirmFinalizeAllocation(t, f, *newCp, sps, cancellationCharges, *scYaml)

	var req lockRequest
	req.decode(input)
	allocation, _ := ssc.getAllocation(req.AllocationID, ctx)
	remainingWritePool, _ := allocation.WritePool.Int64()
	require.Equal(t, int64(100660834), remainingWritePool)

	return nil
}

func testFinalizeAllocation(t *testing.T, sAllocation StorageAllocation, blobbers SortedBlobbers, bStakes [][]mockStakePool, challengePoolBalance int64, now common.Timestamp, challenges [][]int64, ctx *mockStateContext) error {

	var ssc, txn, input = setupMocksFinishAllocation(
		t, sAllocation, blobbers, bStakes,
		currency.Coin(challengePoolBalance), now, ctx,
	)

	scYaml, err := getConfig(ctx)
	require.NoError(t, err)

	var f = formulaeFinalizeAllocation{
		t:                    t,
		scYaml:               *scYaml,
		allocation:           sAllocation,
		blobbers:             blobbers,
		bStakes:              bStakes,
		challengePoolBalance: challengePoolBalance,
		now:                  now,
		challengeCreation:    challenges,
	}

	ac := AllocationChallenges{
		AllocationID: sAllocation.ID,
	}

	for i, blobberChallenges := range challenges {
		blobberID := strconv.Itoa(i)

		err := partitionsChallengeReadyBlobberAddOrUpdate(ctx, blobberID, 1000)
		require.NoError(t, err)

		for _, created := range blobberChallenges {
			ac.OpenChallenges = append(ac.OpenChallenges, &AllocOpenChallenge{
				ID:             fmt.Sprintf("%s:%s:%v", sAllocation.ID, blobberID, created),
				BlobberID:      blobberID,
				RoundCreatedAt: created,
			})
		}
		_, err = ctx.InsertTrieNode(ac.GetKey(ssc.ID), &ac)
		require.NoError(t, err)
	}

	f.setFinilizationPassRates(ssc, ctx, *scYaml, now)

	resp, err := ssc.finalizeAllocation(txn, input, ctx)
	if err != nil {
		return err
	}

	require.EqualValues(t, "finalized", resp)
	require.NoError(t, err)
	newCp, err := ssc.getChallengePool(sAllocation.ID, ctx)
	require.NoError(t, err)
	require.NoError(t, err)
	var sps []*stakePool
	for _, blobber := range blobbers {
		sp, err := ssc.getStakePool(spenum.Blobber, blobber.ID, ctx)
		require.NoError(t, err)
		sps = append(sps, sp)
	}

	var cancellationCharges []int64
	totalCancellationCharge, _ := sAllocation.cancellationCharge(0.2)

	totalWritePrice := currency.Coin(0)

	for _, ba := range f.allocation.BlobberAllocs {
		totalWritePrice, err = currency.AddCoin(totalWritePrice, ba.Terms.WritePrice)
	}

	for i, ba := range f.allocation.BlobberAllocs {

		blobberWritePriceWeight := float64(ba.Terms.WritePrice) / float64(totalWritePrice)
		reward, err := currency.Float64ToCoin(float64(totalCancellationCharge) * blobberWritePriceWeight * f._passRates[i])

		if err != nil {
			return fmt.Errorf("failed to convert float to coin: %v", err)
		}

		cancellationCharges = append(cancellationCharges, int64(reward))
	}

	confirmFinalizeAllocation(t, f, *newCp, sps, cancellationCharges, *scYaml)

	return nil
}

func confirmFinalizeAllocation(
	t *testing.T,
	f formulaeFinalizeAllocation,
	challengePool challengePool,
	sps []*stakePool,
	cancellationCharge []int64,
	scYaml Config,
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

	for i, sp := range sps {
		minLockServiceCharge := f.minLockServiceCharge(i)
		serviceCharge := f.blobberServiceCharge(i, cancellationCharge[i], scYaml) + minLockServiceCharge
		require.InDelta(t, serviceCharge, int64(sp.Reward), errDelta)

		orderedPoolIds := sp.OrderedPoolIds()
		for _, poolId := range orderedPoolIds {
			dp := sp.Pools[poolId]
			wSplit := strings.Split(poolId, " ")
			dId, err := strconv.Atoi(wSplit[2])
			require.NoError(t, err)
			reward := f.blobberDelegateReward(i, dId, cancellationCharge[i], scYaml) + f.minLockDelegatePayment(i, dId)
			require.InDelta(t, reward, int64(dp.Reward), errDelta)
		}
	}

}

func setupMocksFinishAllocation(
	t *testing.T,
	sAllocation StorageAllocation,
	blobbers SortedBlobbers,
	bStakes [][]mockStakePool,
	challengePoolBalance currency.Coin,
	now common.Timestamp,
	ctx *mockStateContext,
) (*StorageSmartContract, *transaction.Transaction, []byte) {
	var err error
	var txn = &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: datastore.Key(transactionHash),
		},
		ClientID:     sAllocation.ID,
		ToClientID:   storageScId,
		CreationDate: now,
	}

	block := &block.Block{}
	block.Round = 1100

	ctx.StateContext = *cstate.NewStateContext(
		block,
		&util.MerklePatriciaTrie{},
		txn,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	var ssc = &StorageSmartContract{
		&sci.SmartContract{
			ID: storageScId,
		},
	}

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
	require.NoError(t, cPool.save(ssc.ID, &sAllocation, ctx))

	require.EqualValues(t, len(blobbers), len(bStakes))
	for i, blobber := range blobbers {
		var id = strconv.Itoa(i)
		var sp = newStakePool()
		sp.Settings.ServiceChargeRatio = blobberYaml.serviceCharge
		sp.TotalOffers = currency.Coin(200000000000)
		for j, stake := range bStakes[i] {
			var jd = strconv.Itoa(j)
			var delegatePool = &stakepool.DelegatePool{}
			delegatePool.Balance = zcnToBalance(stake.zcnAmount)
			delegatePool.DelegateID = encryption.Hash("delegate " + id + " " + jd)
			//delegatePool.MintAt = stake.MintAt
			sp.Pools["paula "+id+" "+jd] = delegatePool
			sp.Pools["paula "+id+" "+jd] = delegatePool
		}
		sp.Settings.DelegateWallet = blobberId + " " + id + " wallet"
		require.NoError(t, sp.Save(spenum.Blobber, blobber.ID, ctx))

		_, err = ctx.InsertTrieNode(blobber.GetKey(), blobber)
		require.NoError(t, err)
	}

	setConfig(t, ctx)

	var request = lockRequest{
		AllocationID: sAllocation.ID,
	}
	input, err := json.Marshal(&request)
	require.NoError(t, err)

	for _, ba := range sAllocation.BlobberAllocs {
		err = partitionsBlobberAllocationsAdd(ctx, ba.BlobberID, ba.AllocationID)
		require.NoError(t, err)
	}

	return ssc, txn, input
}

type formulaeFinalizeAllocation struct {
	t                    *testing.T
	scYaml               Config
	now                  common.Timestamp
	allocation           StorageAllocation
	blobbers             SortedBlobbers
	bStakes              [][]mockStakePool
	challengePoolBalance int64
	challengeCreation    [][]int64
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

func (f *formulaeFinalizeAllocation) blobberServiceCharge(blobberIndex int, cancellationCharge int64, scYaml Config) int64 {
	var serviceCharge = blobberYaml.serviceCharge

	var blobberRewards = f._blobberReward(blobberIndex, cancellationCharge, scYaml)

	return int64(blobberRewards * serviceCharge)
}

func (f *formulaeFinalizeAllocation) blobberDelegateReward(bIndex, dIndex int, cancellationCharge int64, scYaml Config) int64 {
	require.True(f.t, bIndex < len(f.bStakes))
	require.True(f.t, dIndex < len(f.bStakes[bIndex]))
	var totalStake = 0.0
	for _, stake := range f.bStakes[bIndex] {
		totalStake += stake.zcnAmount
	}
	var delegateStake = f.bStakes[bIndex][dIndex].zcnAmount
	var totalDelegateReward = f._blobberReward(bIndex, cancellationCharge, scYaml) - float64(f.blobberServiceCharge(bIndex, cancellationCharge, scYaml))

	require.True(f.t, totalStake > 0)
	return int64(float64(totalDelegateReward) * delegateStake / totalStake)
}

func (f *formulaeFinalizeAllocation) _blobberReward(blobberIndex int, cancellationCharge int64, scYaml Config) float64 {

	ba := f.allocation.BlobberAllocs[blobberIndex]

	challengePoolIntegralValue := float64(ba.ChallengePoolIntegralValue)

	var passRate = f._passRates[blobberIndex]

	dtu := float64(ba.LatestFinalizedChallCreatedAt - ba.LatestSuccessfulChallCreatedAt)
	rdtu := float64(f.allocation.Expiration - ba.LatestSuccessfulChallCreatedAt)
	move := currency.Coin((dtu / rdtu) * challengePoolIntegralValue)
	cv, _ := currency.MinusCoin(currency.Coin(challengePoolIntegralValue), move)
	challengePoolIntegralValue = float64(cv)

	dtu = float64(f.now - ba.LatestFinalizedChallCreatedAt)
	rdtu = float64(f.allocation.Expiration - ba.LatestFinalizedChallCreatedAt)

	if rdtu <= 0 {
		return float64(cancellationCharge)
	}

	move = currency.Coin((dtu / rdtu) * challengePoolIntegralValue)
	cv, _ = currency.MinusCoin(currency.Coin(challengePoolIntegralValue), move)

	moveFloat64, _ := move.Float64()
	moveFloat64 *= passRate

	return moveFloat64 + float64(cancellationCharge)
}

func DeepCopyBlobberAllocsMap(original map[string]*BlobberAllocation) map[string]*BlobberAllocation {
	var copyAllocation map[string]*BlobberAllocation
	jsonData, _ := json.Marshal(original)
	err := json.Unmarshal(jsonData, &copyAllocation)
	if err != nil {
		return map[string]*BlobberAllocation{}
	}
	return copyAllocation
}

func DeepCopyAlloc(original StorageAllocation) StorageAllocation {
	var copyAllocation StorageAllocation
	jsonData, _ := json.Marshal(original)
	err := json.Unmarshal(jsonData, &copyAllocation)
	if err != nil {
		return StorageAllocation{}
	}
	return copyAllocation
}

func (f *formulaeFinalizeAllocation) setFinilizationPassRates(ssc *StorageSmartContract, balances cstate.StateContextI, scYaml Config, now common.Timestamp) {
	f._passRates = []float64{}

	alloc := DeepCopyAlloc(f.allocation)

	blobberAllocMaps := DeepCopyBlobberAllocsMap(f.allocation.BlobberAllocsMap)

	if alloc.Stats == nil {
		alloc.Stats = &StorageAllocationStats{}
	}
	passRates := make([]float64, 0, len(alloc.BlobberAllocs))

	allocChallenges, err := ssc.getAllocationChallenges(alloc.ID, balances)
	switch err {
	case util.ErrValueNotPresent:
	case nil:
		for _, oc := range allocChallenges.OpenChallenges {
			ba, ok := blobberAllocMaps[oc.BlobberID]
			if !ok {
				continue
			}

			if ba.Stats == nil {
				ba.Stats = new(StorageAllocationStats) // make sure
			}

			var expire = oc.RoundCreatedAt + scYaml.MaxChallengeCompletionRounds

			ba.Stats.OpenChallenges--
			alloc.Stats.OpenChallenges--

			currentRound := balances.GetBlock().Round

			if expire < currentRound {
				ba.Stats.FailedChallenges++
				alloc.Stats.FailedChallenges++
			} else {
				ba.Stats.SuccessChallenges++
				alloc.Stats.SuccessChallenges++
			}
		}

	default:
		return
	}

	for _, d := range alloc.BlobberAllocs {
		ba := blobberAllocMaps[d.BlobberID]
		if ba.Stats.OpenChallenges > 0 {
			logging.Logger.Warn("not all challenges canceled", zap.Int64("remaining", ba.Stats.OpenChallenges))

			ba.Stats.SuccessChallenges += ba.Stats.OpenChallenges
			alloc.Stats.SuccessChallenges += ba.Stats.OpenChallenges

			ba.Stats.OpenChallenges = 0
		}

		if ba.Stats.TotalChallenges == 0 {
			passRates = append(passRates, 1.0)
			continue
		}
		// success rate for the blobber allocation
		passRates = append(passRates, float64(ba.Stats.SuccessChallenges)/float64(ba.Stats.TotalChallenges))
	}

	alloc.Stats.OpenChallenges = 0

	f._passRates = passRates
}

func testNewAllocation(t *testing.T, request newAllocationRequest, blobbers SortedBlobbers,
	blobberYaml mockBlobberYaml, stakes blobberStakes, ctx *mockStateContext,
) (err error) {
	require.EqualValues(t, len(blobbers), len(stakes))

	scYaml, err := getConfig(ctx)
	require.NoError(t, err)

	var f = formulaeCommitNewAllocation{
		scYaml:      *scYaml,
		blobberYaml: blobberYaml,
		request:     request,
		blobbers:    blobbers,
		stakes:      stakes,
	}

	expectedAllocationCost := int64((float64(request.Size) * blobberYaml.writePrice * float64(request.DataShards+request.ParityShards)) / float64(request.DataShards))
	val, err := currency.Int64ToCoin(expectedAllocationCost)
	require.NoError(t, err)

	var txn = &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: transactionHash,
		},
		Value:        val,
		ClientID:     clientId,
		ToClientID:   storageScId,
		CreationDate: creationDate,
	}
	access := config.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            os.Getenv("POSTGRES_USER"),
		Password:        os.Getenv("POSTGRES_PASSWORD"),
		Host:            os.Getenv("POSTGRES_HOST"),
		Port:            os.Getenv("POSTGRES_PORT"),
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
	}
	t.Skip("only for local debugging, requires local postgresql")
	eventDb, err := event.NewEventDbWithoutWorker(access, config.DbSettings{})
	if err != nil {
		return
	}
	defer eventDb.Close()

	ctx.StateContext = *cstate.NewStateContext(
		nil,
		&util.MerklePatriciaTrie{},
		txn,
		nil,
		nil,
		nil,
		nil,
		nil,
		eventDb,
	)

	var ssc = &StorageSmartContract{
		&sci.SmartContract{
			ID: storageScId,
		},
	}

	input, err := json.Marshal(request)
	require.NoError(t, err)

	for i, blobber := range blobbers {
		var stakePool = newStakePool()
		stakePool.Pools["paula"] = &stakepool.DelegatePool{}
		stakePool.Pools["paula"].Balance = currency.Coin(stakes[i])
		require.NoError(t, stakePool.Save(spenum.Blobber, blobber.ID, ctx))
	}

	for _, blobber := range blobbers {
		// Save the blobber
		_, err = ctx.InsertTrieNode(blobber.GetKey(), blobber)
		if err != nil {
			require.NoError(t, err)
		}
	}

	_, err = ssc.newAllocationRequest(txn, input, ctx, nil)
	if err != nil {
		return err
	}

	require.NoError(t, err)
	var individualBlobbers = SortedBlobbers{}
	for _, id := range request.Blobbers {
		var b *StorageNode
		b, err = ssc.getBlobber(id, ctx)
		if err != nil && err.Error() == errValueNotPresent {
			continue
		}
		require.NoError(t, err)
		individualBlobbers.add(b)
	}

	var newStakePools = []*stakePool{}
	for _, blobber := range individualBlobbers {
		var sp, err = ssc.getStakePool(spenum.Blobber, blobber.ID, ctx)
		require.NoError(t, err)
		newStakePools = append(newStakePools, sp)
	}

	confirmTestNewAllocation(t, f, individualBlobbers, txn, ctx)

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
	var usedAlready = thisBlobber.Allocated
	var newAllocament = f.sizePerUsedBlobber()

	return usedAlready + newAllocament
}

func confirmTestNewAllocation(t *testing.T, f formulaeCommitNewAllocation,
	blobbers SortedBlobbers, txn *transaction.Transaction, ctx cstate.StateContextI,
) {
	var transfers = ctx.GetTransfers()
	require.Len(t, transfers, 1)
	require.EqualValues(t, clientId, transfers[0].ClientID)
	require.EqualValues(t, storageScId, transfers[0].ToClientID)
	require.EqualValues(t, txn.Value, transfers[0].Amount)

	var countUsedBlobbers = 0
	for _, blobber := range blobbers {
		b, ok := f.blobbers.get(blobber.ID)
		require.True(t, ok)
		if blobber.Allocated > b.Allocated {
			require.EqualValues(t, f.capacityUsedBlobber(t, blobber.ID), blobber.Allocated)
			countUsedBlobbers++
		}
	}
	require.EqualValues(t, f.blobbersUsed(), countUsedBlobbers)

	require.EqualValues(t, f.blobbersUsed(), len(blobbers))
	for _, blobber := range blobbers {
		require.EqualValues(t, f.capacityUsedBlobber(t, blobber.ID), blobber.Allocated)
	}
}
