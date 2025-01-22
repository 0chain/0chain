package storagesc

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/statecache"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

const (
	allocationId         = "my allocation id"
	delegateWallet       = "my wallet"
	errCommitBlobber     = "commit_blobber_read"
	errReadMarker        = "invalid_read_marker"
	errFieldLength       = "length validations of fields failed"
	errPreviousMarker    = "validations with previous marker failed"
	errEarlyAllocation   = "early reading, allocation not started yet"
	errExpiredAllocation = "late reading, allocation expired"
	errNotEnoughTokens   = "not enough tokens"
)

type mockReadMarker struct {
	readCounter int64
	timestamp   common.Timestamp
}
type mockAllocation struct {
	startTime  common.Timestamp
	expiration common.Timestamp
}

type mockReadPool struct {
	Balance currency.Coin `json:"balance"`
}

type cbrResponse struct {
	Pool_id string
	Balance float64
}

var (
	blobberYaml = mockBlobberYaml{
		serviceCharge: 0.3,
		readPrice:     0.01,
		writePrice:    0.1,
	}
	freeReadBlobberYaml = mockBlobberYaml{
		serviceCharge: 0.3,
	}
	validatorYamls = []mockBlobberYaml{
		{serviceCharge: 0.2}, {serviceCharge: 0.25}, {serviceCharge: 0.3},
	}
)

func TestCommitBlobberRead(t *testing.T) {

	var lastRead = mockReadMarker{
		readCounter: 0,
		timestamp:   0,
	}

	var now common.Timestamp = 100
	var nowRound int64 = 10
	var read = mockReadMarker{
		readCounter: 500,
		timestamp:   now,
	}
	var allocation = mockAllocation{
		startTime:  5,
		expiration: 2 * now,
	}
	var stakes = []mockStakePool{
		{2, nowRound - 1},
		{3, nowRound + 1},
		{5, 0},
		{3, nowRound * 10},
	}
	var rPool = mockReadPool{
		11 * 1e10,
	}

	t.Run("test commit blobber read", func(t *testing.T) {
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, read, allocation, stakes, rPool,
		)
		require.NoError(t, err)
	})
	t.Run("test commit blobber read empty pool", func(t *testing.T) {
		var err = testCommitBlobberRead(
			t, freeReadBlobberYaml, lastRead, read, allocation, stakes, mockReadPool{},
		)
		require.NoError(t, err)
	})

	t.Run("check blobber sort needed", func(t *testing.T) {
		var bRPool = mockReadPool{11 * 1e10}
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, read, allocation, stakes, bRPool,
		)
		require.NoError(t, err)
	})

	t.Run(errFieldLength+" -> read counter", func(t *testing.T) {
		var faultyRead = read
		faultyRead.readCounter = 0
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, faultyRead, allocation, stakes, rPool,
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
			t, blobberYaml, lastRead, faultyRead, allocation, stakes, rPool,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errReadMarker))
		require.True(t, strings.Contains(err.Error(), errFieldLength))
	})

	t.Run(errEarlyAllocation, func(t *testing.T) {
		var faultyLastRead = lastRead
		faultyLastRead.readCounter = read.readCounter + 1
		var err = testCommitBlobberRead(
			t, blobberYaml, faultyLastRead, read, allocation, stakes, rPool,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errReadMarker))
	})

	t.Run(errEarlyAllocation, func(t *testing.T) {
		var faultyRead = read
		faultyRead.timestamp = allocation.startTime - 1
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, faultyRead, allocation, stakes, rPool,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errEarlyAllocation))
	})

	t.Run(errExpiredAllocation, func(t *testing.T) {
		var faultyRead = read
		faultyRead.timestamp = allocation.expiration + 1
		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, faultyRead, allocation, stakes, rPool,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errExpiredAllocation))
	})

	t.Run(errNotEnoughTokens+" expired blobbers", func(t *testing.T) {
		var stingyReadPool = mockReadPool{1}

		var err = testCommitBlobberRead(
			t, blobberYaml, lastRead, read, allocation, stakes, stingyReadPool,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errCommitBlobber))
		require.True(t, strings.Contains(err.Error(), errNotEnoughTokens))
	})

}

func newTxnStateCache() *statecache.TransactionCache {
	bc := statecache.NewBlockCache(statecache.NewStateCache(), statecache.Block{})
	return statecache.NewTransactionCache(bc)
}

func testCommitBlobberRead(
	t *testing.T,
	blobberYaml mockBlobberYaml,
	lastRead mockReadMarker,
	read mockReadMarker,
	allocation mockAllocation,
	stakes []mockStakePool,
	readPoolIn mockReadPool,
) error {
	var err error
	var f = formulaeCommitBlobberRead{
		blobberYaml: blobberYaml,
		read:        read,
		allocation:  allocation,
		stakes:      stakes,
		readPool:    readPoolIn,
	}
	var client = &Client{
		balance: 10000,
		scheme:  encryption.NewBLS0ChainScheme(),
	}
	require.NoError(t, client.scheme.GenerateKeys())
	client.pk = client.scheme.GetPublicKey()
	pub := bls.PublicKey{}
	err = pub.DeserializeHexStr(client.pk)
	require.Nil(t, err)
	client.id = encryption.Hash(pub.Serialize())

	var txn = &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: datastore.Key(transactionHash),
		},
		ClientID:     client.id,
		ToClientID:   storageScId,
		CreationDate: creationDate,
	}
	var ctx = &mockStateContext{
		StateContext: *cstate.NewStateContext(
			&block.Block{},
			util.NewMerklePatriciaTrie(nil, 0, nil, statecache.NewEmpty()),
			txn,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil, nil, nil,
		),
		store: make(map[datastore.Key]util.MPTSerializable),
	}

	setConfig(t, ctx)

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
	lastReadConnection.ReadMarker.ClientID = client.id

	var readConnection = &ReadConnection{
		ReadMarker: &ReadMarker{
			ClientPublicKey: client.pk,
			ReadCounter:     read.readCounter,
			BlobberID:       lastReadConnection.ReadMarker.BlobberID,
			ClientID:        lastReadConnection.ReadMarker.ClientID,
			Timestamp:       read.timestamp,
			AllocationID:    allocationId,
		},
	}
	readConnection.ReadMarker.Signature, err = client.scheme.Sign(
		encryption.Hash(readConnection.ReadMarker.GetHashData()))
	require.NoError(t, err)
	var input = readConnection.Encode()

	_, err = ctx.InsertTrieNode(readConnection.GetKey(ssc.ID), lastReadConnection)
	require.NoError(t, err)

	var sa = &StorageAllocation{}
	sa.SetEntity(&storageAllocationV2{
		ID:         allocationId,
		StartTime:  allocation.startTime,
		Expiration: allocation.expiration,
		BlobberAllocs: []*BlobberAllocation{
			{
				BlobberID: blobberId,
				Terms: Terms{
					ReadPrice: zcnToBalance(blobberYaml.readPrice),
				},
				Stats: &StorageAllocationStats{},
			},
		},
		Owner: client.id,
		Stats: &StorageAllocationStats{},
	})

	_, err = ctx.InsertTrieNode(sa.GetKey(ssc.ID), sa)
	require.NoError(t, err)

	blobber := &StorageNode{}
	blobber.SetEntity(&storageNodeV3{
		Provider: provider.Provider{
			ID:           blobberId,
			ProviderType: spenum.Blobber,
		},
		Terms: Terms{
			ReadPrice:  zcnToBalance(blobberYaml.readPrice),
			WritePrice: zcnToBalance(blobberYaml.writePrice),
		},
	})

	_, err = ctx.InsertTrieNode(blobber.GetKey(), blobber)

	var rPool = readPool{readPoolIn.Balance}

	if readPoolIn.Balance != 0 {
		require.NoError(t, rPool.save(ssc.ID, client.id, ctx))
	}

	var sPool = stakePool{
		StakePool: &stakepool.StakePool{
			Pools: make(map[string]*stakepool.DelegatePool),
			Settings: stakepool.Settings{
				ServiceChargeRatio: blobberYaml.serviceCharge,
				DelegateWallet:     delegateWallet,
			},
		},
	}
	for i, stake := range stakes {
		var id = strconv.Itoa(i)
		sPool.Pools["pool"+id] = &stakepool.DelegatePool{
			DelegateID:   encryption.Hash("delegate_pool_" + id),
			Balance:      zcnToBalance(stake.zcnAmount),
			RoundCreated: stake.MintAt,
		}
	}
	require.NoError(t, sPool.Save(spenum.Blobber, blobberId, ctx))

	resp, err := ssc.commitBlobberRead(txn, input, ctx)
	if err != nil {
		return err
	}

	newRp, err := ssc.getReadPool(client.id, ctx)
	require.NoError(t, err)

	if blobberYaml.readPrice != 0 {
		require.NotEqualValues(t, rPool.Balance, newRp.Balance)
	} else {
		require.EqualValues(t, 0, newRp.Balance)
	}

	newSp, err := ssc.getStakePool(spenum.Blobber, blobberId, ctx)
	require.NoError(t, err)

	confirmCommitBlobberRead(t, f, resp, newSp)
	return nil
}

func confirmCommitBlobberRead(
	t *testing.T,
	f formulaeCommitBlobberRead,
	resp string,
	newStakePool *stakePool,
) {
	var respArray = []cbrResponse{}
	require.NoError(t, json.Unmarshal([]byte(resp), &respArray))
	require.Len(t, respArray, 1)
	require.EqualValues(t, blobberId, respArray[0].Pool_id)
	require.InDelta(t, f.blobberReward(), respArray[0].Balance, errDelta)
	require.InDelta(t, f.blobberCharge(), int64(newStakePool.Reward), errDelta)

	for i, id := range newStakePool.OrderedPoolIds() {
		require.InDelta(
			t,
			f.delegateRward(int64(i)),
			int64(newStakePool.Pools[id].Reward),
			errDelta,
		)
	}
}

type formulaeCommitBlobberRead struct {
	blobberYaml mockBlobberYaml
	lastRead    mockReadMarker
	read        mockReadMarker
	allocation  mockAllocation
	stakes      []mockStakePool
	readPool    mockReadPool
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
