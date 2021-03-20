package storagesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	CHUNK_SIZE = 64 * KB // hardcoded in blobber.go

	numBlobbers                  = 30
	blobberBalance state.Balance = 50 * x10
	clientBalance  state.Balance = 100 * x10

	rReadSize       = 1 * GB
	rCounter  int64 = rReadSize / CHUNK_SIZE

	aValue                int64 = 15 * x10
	aMaxReadPrice               = 10 * x10
	aMinReadPrice               = 1 * x10
	aMinWritePrice              = 2 * x10
	aMaxWritePrice              = 20 * x10
	aMaxChallengeCompTime       = 200 * time.Hour
	aRequestSize                = 2 * GB // ---- restMinLockDemand -----
	aDataShards                 = 10     // ---- restMinLockDemand -----
	aParityShards               = 10     // ---- restMinLockDemand -----
)

type mock0ChainBlobberYml struct {
	ReadPrice               float64 //token / GB for reading
	WritePrice              float64 //token / GB / time_unit for writing
	Capacity                int64   // 1 GB bytes total blobber capacity
	MinLockDemand           float64
	MaxOfferDuration        time.Duration
	ChallengeCompletionTime time.Duration
	MinStake                float64
	MaxStake                float64
	NumDelegates            int
	ServiceCharge           float64
}

func attachTestBlobbers(t testing.TB, ssc *StorageSmartContract, now int64,
	terms Terms, capacity int64, ctx *testBalances) (blobbers []*Client) {
	for i := 0; i < numBlobbers; i++ {
		var blobber = addBlobber(t, ssc, capacity, now, terms, blobberBalance, ctx)
		blobbers = append(blobbers, blobber)
	}
	return
}

func newTestAllocation(t testing.TB, client *Client, nar newAllocationRequest, value state.Balance,
	ssc *StorageSmartContract, now int64, ctx *testBalances,
) (allocationId string) {
	nar.Owner = client.id
	nar.OwnerPublicKey = client.pk

	resp, err := nar.callNewAllocReq(t, client.id, value, ssc, now, ctx)
	require.NoError(t, err)
	var decodeResp StorageAllocation
	require.NoError(t, decodeResp.Decode([]byte(resp)))
	return decodeResp.ID
}

func attachBlobbersAndNewAllocation(t *testing.T, terms Terms, aRequest newAllocationRequest, capacity int64,
) (ssc *StorageSmartContract, ctx *testBalances,
	blobbers []*Client, now int64, allocationId string, client *Client, testBlobber *Client,
) {
	ssc = newTestStorageSC()
	ctx = newTestBalances(t, false)
	_ = *setConfig(t, ctx)
	client = newClient(clientBalance, ctx)
	now += 100
	blobbers = attachTestBlobbers(t, ssc, now, terms, capacity, ctx)

	now += 100
	allocationId = newTestAllocation(t, client, aRequest,
		state.Balance(aValue), ssc, now, ctx)

	allocation, err := ssc.getAllocation(allocationId, ctx)
	require.NoError(t, err)
	for _, b := range blobbers {
		if b.id == allocation.BlobberDetails[0].BlobberID {
			testBlobber = b
			break
		}
	}
	return
}

func TestNewAllocation(t *testing.T) {
	var (
		blobberYaml = mock0ChainBlobberYml{
			Capacity:                2 * GB,
			ReadPrice:               1,
			WritePrice:              5,   // ---- restMinLockDemand -----
			MinLockDemand:           0.1, // ---- restMinLockDemand -----
			MaxOfferDuration:        1 * time.Hour,
			ChallengeCompletionTime: 200 * time.Second,
			MinStake:                0,
			MaxStake:                1000 * x10,
			NumDelegates:            100,
			ServiceCharge:           0.3,
		}
		aExpiration int64 = int64(toSeconds(time.Hour))
		terms             = Terms{
			ReadPrice:               convertZcnToValue(blobberYaml.ReadPrice),
			WritePrice:              convertZcnToValue(blobberYaml.WritePrice),
			MinLockDemand:           blobberYaml.MinLockDemand,
			MaxOfferDuration:        blobberYaml.MaxOfferDuration,
			ChallengeCompletionTime: blobberYaml.ChallengeCompletionTime,
		}
		allocationRequest = newAllocationRequest{
			DataShards:                 aDataShards,
			ParityShards:               aParityShards,
			Expiration:                 common.Timestamp(aExpiration),
			ReadPriceRange:             PriceRange{aMinReadPrice, aMaxReadPrice},
			WritePriceRange:            PriceRange{aMinWritePrice, aMaxWritePrice},
			Size:                       aRequestSize,
			MaxChallengeCompletionTime: aMaxChallengeCompTime,
		}
		f formulae = formulae{
			blobber: blobberYaml,
			ar:      allocationRequest,
		}
	)

	t.Run("new allocation", func(t *testing.T) {
		ssc, ctx, _, now, allocationId, _, _ :=
			attachBlobbersAndNewAllocation(t, terms, allocationRequest, blobberYaml.Capacity)

		allocation, err := ssc.getAllocation(allocationId, ctx)
		require.NoError(t, err)

		f.sc = *setConfig(t, ctx)
		require.EqualValues(t, f.AllocRestMinLockDemandTotal(common.Timestamp(now)),
			allocation.restMinLockDemand())
	})

	t.Run("read as owner", func(t *testing.T) {
		ssc, ctx, _, now, allocationId, client, testBlobber :=
			attachBlobbersAndNewAllocation(t, terms, allocationRequest, blobberYaml.Capacity)

		require.NotNil(t, testBlobber)
		var readMarker = ReadConnection{
			ReadMarker: &ReadMarker{
				ClientID:        client.id,
				ClientPublicKey: client.pk,
				BlobberID:       testBlobber.id,
				AllocationID:    allocationId,
				OwnerID:         client.id,
				Timestamp:       common.Timestamp(now),
				ReadCounter:     rCounter,
				PayerID:         client.id,
			},
		}
		var err error
		readMarker.ReadMarker.Signature, err = client.scheme.Sign(
			encryption.Hash(readMarker.ReadMarker.GetHashData()))
		require.NoError(t, err)

		// create read pool
		now += 100
		var tx = newTransaction(client.id, ssc.ID, 0, now)
		ctx.setTransaction(t, tx)
		_, err = ssc.newReadPool(tx, nil, ctx)
		require.NoError(t, err)

		// read pool lock
		now += 100
		const lockedFundsPerBlobber = 2 * 1e10
		allocation, err := ssc.getAllocation(allocationId, ctx)
		require.NoError(t, err)
		var readPoolFund = state.Balance(len(allocation.BlobberDetails)) * lockedFundsPerBlobber
		tx = newTransaction(client.id, ssc.ID, readPoolFund, now)
		ctx.setTransaction(t, tx)
		_, err = ssc.readPoolLock(tx, mustEncode(t, &lockRequest{
			Duration:     20 * time.Minute,
			AllocationID: allocationId,
		}), ctx)
		require.NoError(t, err)

		rPool, err := ssc.getReadPool(client.id, ctx)
		require.NoError(t, err)
		require.EqualValues(t, lockedFundsPerBlobber,
			rPool.Pools.allocBlobberTotal(allocationId, testBlobber.id, now))

		// read
		now += 100
		tx = newTransaction(testBlobber.id, ssc.ID, 0, now)
		ctx.setTransaction(t, tx)
		_, err = ssc.commitBlobberRead(tx, mustEncode(t, &readMarker), ctx)
		require.NoError(t, err)

		// check out balances
		sPool, err := ssc.getStakePool(testBlobber.id, ctx)
		require.NoError(t, err)

		f.readMarker = *readMarker.ReadMarker
		require.EqualValues(t, f.RmRewardsCharge(), sPool.Rewards.Charge)
		require.EqualValues(t, f.RmRewardsBlobber(), sPool.Rewards.Blobber)
		require.EqualValues(t, f.RmRewardsValidator(), sPool.Rewards.Validator)

		rPool, err = ssc.getReadPool(client.id, ctx)
		require.NoError(t, err)
		require.EqualValues(t, readPoolFund-f.RmValue(),
			rPool.Pools.allocTotal(allocationId, now))
		require.EqualValues(t, f.RmValue(),
			rPool.Pools.allocBlobberTotal(allocationId, testBlobber.id, now))

		allocation, err = ssc.getAllocation(allocationId, ctx)
		require.NoError(t, err)
		// todo work out rest min lock demond
		// require.EqualValues(t, f.AllocRestMinLockDemandTotal2(f.RmValue(), common.Timestamp(now)),
		//	allocation.restMinLockDemand())
		require.EqualValues(t, 186921297, allocation.restMinLockDemand())
	})

	t.Run("read as separate user", func(t *testing.T) {
		ssc, ctx, _, now, allocationId, client, testBlobber :=
			attachBlobbersAndNewAllocation(t, terms, allocationRequest, blobberYaml.Capacity)

		reader := newClient(clientBalance, ctx)
		var at = AuthTicket{
			ClientID:     reader.id,
			OwnerID:      client.id,
			AllocationID: allocationId,
			Expiration:   common.Timestamp(now + 1000),
			Timestamp:    common.Timestamp(now - 10),
		}
		var err error
		at.Signature, err = client.scheme.Sign(
			encryption.Hash(at.getHashData()),
		)
		require.NoError(t, err)
		var readMarker ReadConnection
		readMarker.ReadMarker = &ReadMarker{
			ClientID:        reader.id,
			ClientPublicKey: reader.pk,
			BlobberID:       testBlobber.id,
			AllocationID:    allocationId,
			OwnerID:         client.id,
			Timestamp:       common.Timestamp(now),
			ReadCounter:     rCounter,
			PayerID:         reader.id,
			AuthTicket:      &at,
		}
		readMarker.ReadMarker.Signature, err = reader.scheme.Sign(
			encryption.Hash(readMarker.ReadMarker.GetHashData()))
		require.NoError(t, err)

		now += 100
		var tx = newTransaction(testBlobber.id, ssc.ID, 0, now)
		ctx.setTransaction(t, tx)
		_, err = ssc.commitBlobberRead(tx, mustEncode(t, &readMarker), ctx)
		require.Error(t, err)

		// create read pool
		now += 100
		tx = newTransaction(reader.id, ssc.ID, 0, now)
		ctx.setTransaction(t, tx)
		_, err = ssc.newReadPool(tx, nil, ctx)
		require.NoError(t, err)

		// read pool lock
		now += 100
		allocation, err := ssc.getAllocation(allocationId, ctx)
		require.NoError(t, err)
		require.NotNil(t, allocation)
		const lockedFundsPerBlobber = 2 * 1e10
		var readPoolFund = state.Balance(len(allocation.BlobberDetails)) * lockedFundsPerBlobber
		tx = newTransaction(reader.id, ssc.ID, readPoolFund, now)
		ctx.setTransaction(t, tx)
		_, err = ssc.readPoolLock(tx, mustEncode(t, &lockRequest{
			Duration:     20 * time.Minute,
			AllocationID: allocationId,
		}), ctx)
		require.NoError(t, err)

		// read
		now += 100
		tx = newTransaction(testBlobber.id, ssc.ID, 0, now)
		ctx.setTransaction(t, tx)
		_, err = ssc.commitBlobberRead(tx, mustEncode(t, &readMarker), ctx)
		require.NoError(t, err)

		// check out ctx
		var sPool *stakePool
		sPool, err = ssc.getStakePool(testBlobber.id, ctx)
		require.NoError(t, err)

		f.readMarker = *readMarker.ReadMarker
		require.EqualValues(t, f.RmRewardsCharge(), sPool.Rewards.Charge)
		require.EqualValues(t, f.RmRewardsBlobber(), sPool.Rewards.Blobber)
		require.EqualValues(t, f.RmRewardsValidator(), sPool.Rewards.Validator)

		var rPool *readPool
		rPool, err = ssc.getReadPool(reader.id, ctx)
		require.NoError(t, err)

		require.EqualValues(t, readPoolFund-f.RmValue(),
			rPool.Pools.allocTotal(allocationId, now))
		require.EqualValues(t, f.RmValue(),
			rPool.Pools.allocBlobberTotal(allocationId, testBlobber.id, now))

		// min lock demand reducing
		allocation, err = ssc.getAllocation(allocationId, ctx)
		require.NoError(t, err)
		// todo fix rest min lock demand
		require.EqualValues(t, 186921297, allocation.restMinLockDemand())
	})

	t.Run("write", func(t *testing.T) {
		ssc, ctx, blobbers, now, allocationId, client, testBlobber1 :=
			attachBlobbersAndNewAllocation(t, terms, allocationRequest, blobberYaml.Capacity)

		var allocation, err = ssc.getAllocation(allocationId, ctx)
		require.NoError(t, err)
		require.NotNil(t, allocation)
		var until = int64(allocation.Until())
		var testBlobber2 *Client
		for _, b := range blobbers {
			if b.id == allocation.BlobberDetails[1].BlobberID {
				testBlobber2 = b
				break
			}
		}
		require.NotNil(t, testBlobber2)

		//var err error
		//var cp *challengePool
		challengePool, err := ssc.getChallengePool(allocationId, ctx)
		require.NoError(t, err)

		//var wp *writePool
		writePool, err := ssc.getWritePool(client.id, ctx)
		require.NoError(t, err)

		var writePoolTotal = writePool.Pools.allocTotal(allocationId, until)
		require.EqualValues(t, aValue, writePoolTotal)
		require.EqualValues(t, 0, challengePool.Balance)

		const allocationRoot = "root-1"
		const writeSize = 100 * 1024 * 1024 // 100 MB
		var cc = &BlobberCloseConnection{
			AllocationRoot:     allocationRoot,
			PrevAllocationRoot: "",
			WriteMarker: &WriteMarker{
				AllocationRoot:         allocationRoot,
				PreviousAllocationRoot: "",
				AllocationID:           allocationId,
				Size:                   writeSize,
				BlobberID:              testBlobber2.id,
				Timestamp:              common.Timestamp(now),
				ClientID:               client.id,
			},
		}
		cc.WriteMarker.Signature, err = client.scheme.Sign(
			encryption.Hash(cc.WriteMarker.GetHashData()))
		require.NoError(t, err)

		// write
		now += 100
		var tx = newTransaction(testBlobber2.id, ssc.ID, 0, now)
		ctx.setTransaction(t, tx)
		resp, err := ssc.commitBlobberConnection(tx, mustEncode(t, &cc), ctx)
		require.NoError(t, err)
		require.NotZero(t, resp)

		// check out
		stakePool, err := ssc.getStakePool(testBlobber1.id, ctx)
		require.NoError(t, err)
		//stakePool = stakePool
		require.EqualValues(t, 0,
			stakePool.Rewards.Blobber+stakePool.Rewards.Validator+stakePool.Rewards.Charge)

		challengePool, err = ssc.getChallengePool(allocationId, ctx)
		require.NoError(t, err)

		var moved = int64(sizeInGB(cc.WriteMarker.Size) *
			float64(avgTerms.WritePrice) *
			allocation.restDurationInTimeUnits(cc.WriteMarker.Timestamp))

		require.EqualValues(t, moved, challengePool.Balance)

		writePool, err = ssc.getWritePool(client.id, ctx)
		require.NoError(t, err)

		require.EqualValues(t, 15*x10-moved, writePool.Pools.allocTotal(allocationId, now))

		// min lock demand reducing
		allocation, err = ssc.getAllocation(allocationId, ctx)
		require.NoError(t, err)
		require.EqualValues(t, 186921297, allocation.restMinLockDemand()) // -read above
	})

}

// ConvertToValue converts ZCN tokens to value
func convertZcnToValue(token float64) state.Balance {
	return state.Balance(token * float64(x10))
}

//

//

//
//
