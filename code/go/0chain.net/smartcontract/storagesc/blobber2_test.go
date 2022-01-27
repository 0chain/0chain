package storagesc

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"github.com/stretchr/testify/require"
)

const (
	CHUNK_SIZE = 64 * KB

	allocationId         = "my allocation id"
	payerId              = "peter"
	delegateWallet       = "my wallet"
	errCommitBlobber     = "commit_blobber_read"
	errReadMarker        = "invalid_read_marker"
	errFieldLength       = "length validations of fields failed"
	errPreviousMarker    = "validations with previous marker failed"
	errEarlyAllocation   = "early reading, allocation not started yet"
	errExpiredAllocation = "late reading, allocation expired"
	errNoTokensReadPool  = "no tokens in read pool for allocation"
	errNotEnoughTokens   = "not enough tokens in read pool "
)

type mockReadMarker struct {
	readCounter int64
	timestamp   common.Timestamp
}
type mockAllocation struct {
	startTime  common.Timestamp
	expiration common.Timestamp
}
type mockAllocationPool struct {
	balance          float64
	expires          common.Timestamp
	blobberBalance   float64
	numberOfBlobbers int64
}

type mockReadPools struct {
	thisAllocation   []mockAllocationPool
	otherAllocations int
}

type cbrResponse struct {
	Pool_id string
	Balance float64
}

var (
	blobberYaml = mockBlobberYaml{
		serviceCharge:           0.3,
		readPrice:               0.01,
		writePrice:              0.01,
		challengeCompletionTime: 2 * time.Minute,
	}
)

func TestCommitBlobberRead(t *testing.T) {
	var lastRead = mockReadMarker{
		readCounter: 0,
		timestamp:   0,
	}
	var now common.Timestamp = 100
	var read = mockReadMarker{
		readCounter: 500,
		timestamp:   now,
	}
	var allocation = mockAllocation{
		startTime:  5,
		expiration: 2 * now,
	}
	var stakes = []mockStakePool{
		{2, now - 1},
		{3, now + 1},
		{5, 0},
		{3, now * 10},
	}
	var rPools = mockReadPools{
		thisAllocation: []mockAllocationPool{
			{2.3, now, 19.2, 1},
			{2.3, now * 3, 19.2, 3},
			{2.3, now - 1, 19.2, 1},
		},
		otherAllocations: 4,
	}

	t.Run("test commit blobber read", func(t *testing.T) {
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, read, allocation, stakes, rPools,
		)
		require.NoError(t, err)
	})

	t.Run("check blobber sort needed", func(t *testing.T) {
		var bRPools = rPools
		bRPools.thisAllocation = []mockAllocationPool{
			{2.3, now * 3, 19.2, 3},
		}
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, read, allocation, stakes, bRPools,
		)
		require.NoError(t, err)
	})

	t.Run(errFieldLength+" -> read counter", func(t *testing.T) {
		var faultyRead = read
		faultyRead.readCounter = 0
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, faultyRead, allocation, stakes, rPools,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errReadMarker))
		require.True(t, strings.Contains(err.Error(), errFieldLength))
	})

	t.Run(errFieldLength+" -> timestamp", func(t *testing.T) {
		var faultyRead = read
		faultyRead.timestamp = 0
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, faultyRead, allocation, stakes, rPools,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errReadMarker))
		require.True(t, strings.Contains(err.Error(), errFieldLength))
	})

	t.Run(errPreviousMarker+" -> timestamp", func(t *testing.T) {
		var faultyLastRead = lastRead
		faultyLastRead.timestamp = read.timestamp + 1
		var err = testCommitBlobberRead(
			t, blobberYaml, faultyLastRead, read, allocation, stakes, rPools,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errReadMarker))
		require.True(t, strings.Contains(err.Error(), errPreviousMarker))
	})

	t.Run(errEarlyAllocation, func(t *testing.T) {
		var faultyLastRead = lastRead
		faultyLastRead.readCounter = read.readCounter + 1
		var err = testCommitBlobberRead(
			t, blobberYaml, faultyLastRead, read, allocation, stakes, rPools,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errReadMarker))
		require.True(t, strings.Contains(err.Error(), errPreviousMarker))
	})

	t.Run(errEarlyAllocation, func(t *testing.T) {
		var faultyRead = read
		faultyRead.timestamp = allocation.startTime - 1
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, faultyRead, allocation, stakes, rPools,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errEarlyAllocation))
	})

	t.Run(errExpiredAllocation, func(t *testing.T) {
		var faultyRead = read
		faultyRead.timestamp = allocation.expiration +
			toSeconds(blobberYaml.challengeCompletionTime) + 1
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, faultyRead, allocation, stakes, rPools,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errExpiredAllocation))
	})

	t.Run(errNoTokensReadPool+" expired blobbers", func(t *testing.T) {
		var expiredReadPools = rPools
		expiredReadPools.thisAllocation = []mockAllocationPool{
			{2.3, 0, 19.2, 1},
			{2.3, now - 2, 19.2, 3},
			{2.3, now - 1, 19.2, 1},
		}
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, read, allocation, stakes, expiredReadPools,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errNoTokensReadPool))
	})

	t.Run(errNotEnoughTokens+" expired blobbers", func(t *testing.T) {
		var expiredReadPools = rPools
		expiredReadPools.thisAllocation = []mockAllocationPool{
			{2.3, now * 3, 0.00001, 1},
			{2.3, now - 1, 19.2, 1},
		}
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, read, allocation, stakes, expiredReadPools,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errNotEnoughTokens))
	})

}

func testCommitBlobberRead(
	t *testing.T,
	blobberYaml mockBlobberYaml,
	lastRead mockReadMarker,
	read mockReadMarker,
	allocation mockAllocation,
	stakes []mockStakePool,
	readPools mockReadPools,
) error {
	var err error
	var f = formulaeCommitBlobberRead{
		blobberYaml: blobberYaml,
		lastRead:    lastRead,
		read:        read,
		allocation:  allocation,
		stakes:      stakes,
		readPools:   readPools,
	}
	var txn = &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: datastore.Key(transactionHash),
		},
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
			nil,
		),
		store: make(map[datastore.Key]util.Serializable),
	}

	var client = &Client{
		balance: 10000,
		scheme:  encryption.NewBLS0ChainScheme(),
	}
	require.NoError(t, client.scheme.GenerateKeys())
	client.pk = client.scheme.GetPublicKey()
	client.id = encryption.Hash(client.pk)

	var ssc = &StorageSmartContract{
		&sci.SmartContract{
			ID: storageScId,
		},
	}

	var lastReadConnection = &ReadConnection{
		ReadMarker: &ReadMarker{
			ReadCounter: lastRead.readCounter,
			BlobberID:   blobberId,
			ClientID:    client.id,
			Timestamp:   lastRead.timestamp,
		},
	}
	lastReadConnection.ReadMarker.ClientID = clientId
	var readConection = &ReadConnection{
		ReadMarker: &ReadMarker{
			ClientPublicKey: client.pk,
			ReadCounter:     read.readCounter,
			BlobberID:       lastReadConnection.ReadMarker.BlobberID,
			ClientID:        lastReadConnection.ReadMarker.ClientID,
			Timestamp:       read.timestamp,
			PayerID:         payerId,
			AuthTicket:      nil,
			AllocationID:    allocationId,
		},
	}
	readConection.ReadMarker.Signature, err = client.scheme.Sign(
		encryption.Hash(readConection.ReadMarker.GetHashData()))
	require.NoError(t, err)
	var input = readConection.Encode()

	_, err = ctx.InsertTrieNode(readConection.GetKey(ssc.ID), lastReadConnection)
	require.NoError(t, err)
	var storageAllocation = &StorageAllocation{
		ID:                      allocationId,
		StartTime:               allocation.startTime,
		ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
		Expiration:              allocation.expiration,
		BlobberDetails: []*BlobberAllocation{
			{
				BlobberID: blobberId,
				Terms: Terms{
					ReadPrice: zcnToBalance(blobberYaml.readPrice),
				},
			},
		},
		Owner: payerId,
	}
	_, err = ctx.InsertTrieNode(storageAllocation.GetKey(ssc.ID), storageAllocation)

	var rPool = readPool{
		Pools: []*allocationPool{},
	}
	for i := 0; i < len(readPools.thisAllocation)+readPools.otherAllocations; i++ {
		var id = strconv.Itoa(i)
		rPool.Pools.add(&allocationPool{
			AllocationID: id,
		})
	}
	var startBlock = 0
	for i, aPool := range readPools.thisAllocation {
		rPool.Pools[startBlock+i].AllocationID = allocationId
		rPool.Pools[startBlock+i].ID = blobberId
		rPool.Pools[startBlock+i].Balance = zcnToBalance(aPool.balance)
		rPool.Pools[startBlock+i].ExpireAt = aPool.expires
		rPool.Pools[startBlock+i].Blobbers = []*blobberPool{}
		var myBlobberIndex = 0
		for j := 0; j < int(aPool.numberOfBlobbers); j++ {
			var id = strconv.Itoa(i)
			var pool = &blobberPool{BlobberID: id}
			if j == myBlobberIndex {
				pool.BlobberID = blobberId
				pool.Balance = zcnToBalance(aPool.blobberBalance)
			}
			rPool.Pools[startBlock+i].Blobbers.add(pool)
		}
	}
	require.NoError(t, rPool.save(ssc.ID, payerId, ctx))

	var sPool = stakePool{
		Pools: make(map[string]*delegatePool),
		Settings: stakePoolSettings{
			ServiceCharge:  blobberYaml.serviceCharge,
			DelegateWallet: delegateWallet,
		},
	}
	for i, stake := range stakes {
		var id = strconv.Itoa(i)
		sPool.Pools["pool"+id] = &delegatePool{
			DelegateID: strconv.Itoa(i),
			ZcnPool: tokenpool.ZcnPool{
				TokenPool: tokenpool.TokenPool{
					ID:      id,
					Balance: zcnToBalance(stake.zcnAmount),
				},
			},
			MintAt: stake.MintAt,
		}
	}
	sPool.Pools["pool0"].ZcnPool.TokenPool.ID = blobberId
	require.NoError(t, sPool.save(ssc.ID, blobberId, ctx))

	ss := &StorageStats{}
	ss.Stats = &StorageAllocationStats{}
	_, err = ctx.InsertTrieNode(ss.GetKey(ssc.ID), ss)
	require.NoError(t, err)

	resp, err := ssc.commitBlobberRead(txn, input, ctx)
	if err != nil {
		return err
	}

	newRp, err := ssc.getReadPool(payerId, ctx)
	require.NoError(t, err)

	newSp, err := ssc.getStakePool(blobberId, ctx)
	require.NoError(t, err)

	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	statsBytes, err := ctx.GetTrieNode(stats.GetKey(ssc.ID))
	require.NoError(t, err)
	require.NotNil(t, statsBytes)
	require.NoError(t, stats.Decode(statsBytes.Encode()))

	confirmCommitBlobberRead(t, f, resp, stats, newRp, newSp, ctx)
	return nil
}

func confirmCommitBlobberRead(
	t *testing.T,
	f formulaeCommitBlobberRead,
	resp string,
	stats *StorageStats,
	newReadPool *readPool,
	newStakePool *stakePool,
	ctx *mockStateContext,
) {
	var respArray = []cbrResponse{}
	require.NoError(t, json.Unmarshal([]byte(resp), &respArray))
	require.Len(t, respArray, 1)
	require.EqualValues(t, blobberId, respArray[0].Pool_id)
	require.InDelta(t, f.blobberReward(), respArray[0].Balance, errDelta)

	require.EqualValues(t, f.read.readCounter, stats.Stats.NumReads)
	require.Len(t, newReadPool.Pools, len(f.readPools.thisAllocation)+f.readPools.otherAllocations)

	require.InDelta(t, f.blobberCharge(), int64(newStakePool.Rewards.Charge), errDelta)
	require.InDelta(t, f.blobberReward()-f.blobberCharge(), int64(newStakePool.Rewards.Blobber), errDelta)

	require.True(t, true)
	for _, transfer := range ctx.GetTransfers() {
		require.EqualValues(t, storageScId, transfer.ClientID)
		if transfer.ToClientID == delegateWallet {
			require.InDelta(t, f.blobberCharge(), int64(transfer.Amount), errDelta)
		} else {
			index, err := strconv.Atoi(transfer.ToClientID)
			require.NoError(t, err)
			require.InDelta(t, f.delegateRward(int64(index)), int64(transfer.Amount), errDelta)
		}
	}
}

type formulaeCommitBlobberRead struct {
	blobberYaml mockBlobberYaml
	lastRead    mockReadMarker
	read        mockReadMarker
	allocation  mockAllocation
	stakes      []mockStakePool
	readPools   mockReadPools
}

func (f formulaeCommitBlobberRead) blobberReward() int64 {
	var readSize = float64(f.read.readCounter*CHUNK_SIZE) / GB
	var readPrice = float64(zcnToInt64(f.blobberYaml.readPrice))

	return int64(readSize * readPrice)
}

func (f formulaeCommitBlobberRead) blobberCharge() int64 {
	var blobberRward = float64(f.blobberReward())
	var serviceCharge = blobberYaml.serviceCharge

	return int64(blobberRward * serviceCharge)
}

func (f formulaeCommitBlobberRead) delegateRward(id int64) int64 {
	var totalStaked = int64(0)
	for _, stake := range f.stakes {
		totalStaked += zcnToInt64(stake.zcnAmount)
	}
	var delegateStake = float64(zcnToInt64(f.stakes[id].zcnAmount))
	var shareRatio = float64(delegateStake) / float64(totalStaked)
	var blobberEarnings = float64(f.blobberReward())
	var serviceCharge = f.blobberYaml.serviceCharge

	return int64(blobberEarnings * shareRatio * (1 - serviceCharge))
}
