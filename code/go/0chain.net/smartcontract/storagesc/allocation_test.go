package storagesc

import (
	"testing"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//
// use:
//
//      go test -cover -coverprofile=cover.out && go tool cover -html=cover.out -o=cover.html
//
// to test and generate coverage html page
//

func TestStorageSmartContract_getAllocation(t *testing.T) {
	const allocID, clientID, clientPk = "alloc_hex", "client_hex", "pk"
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		alloc    *StorageAllocation
		err      error
	)
	if alloc, err = ssc.getAllocation(allocID, balances); err == nil {
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
	if err != nil {
		t.Fatal(err)
	}
	var got *StorageAllocation
	if got, err = ssc.getAllocation(allocID, balances); err != nil {
		t.Fatal(err)
	}
	if string(got.Encode()) != string(alloc.Encode()) {
		t.Fatal("wrong")
	}
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
	const allocID, clientID, clientPk = "alloc_hex", "client_hex", "pk"
	var nar newAllocationRequest
	nar.DataShards = 2
	nar.ParityShards = 3
	nar.Size = 1024
	nar.Expiration = common.Now()
	nar.Owner = clientID
	nar.OwnerPublicKey = clientPk
	nar.PreferredBlobbers = []string{"one", "two"}
	nar.ReadPriceRange = PriceRange{Min: 10, Max: 20}
	nar.WritePriceRange = PriceRange{Min: 100, Max: 200}
	var alloc = nar.storageAllocation()
	if alloc.DataShards != nar.DataShards {
		t.Error("wrong")
	}
	if alloc.ParityShards != nar.ParityShards {
		t.Error("wrong")
	}
	if alloc.Size != nar.Size {
		t.Error("wrong")
	}
	if alloc.Expiration != nar.Expiration {
		t.Error("wrong")
	}
	if alloc.Owner != nar.Owner {
		t.Error("wrong")
	}
	if alloc.OwnerPublicKey != nar.OwnerPublicKey {
		t.Error("wrong")
	}
	if !isEqualStrings(alloc.PreferredBlobbers, nar.PreferredBlobbers) {
		t.Error("wrong")
	}
	if alloc.ReadPriceRange != nar.ReadPriceRange {
		t.Error("wrong")
	}
	if alloc.WritePriceRange != nar.WritePriceRange {
		t.Error("wrong")
	}
}

func Test_newAllocationRequest_decode(t *testing.T) {
	const clientID, clientPk = "client_id_hex", "client_pk_hex"
	var ne, nd newAllocationRequest
	ne.DataShards = 1
	ne.ParityShards = 1
	ne.Size = 2 * GB
	ne.Expiration = 1240
	ne.Owner = clientID
	ne.OwnerPublicKey = clientPk
	ne.PreferredBlobbers = []string{"b1", "b2"}
	ne.ReadPriceRange = PriceRange{1, 2}
	ne.WritePriceRange = PriceRange{2, 3}
	require.NoError(t, nd.decode(mustEncode(t, &ne)))
	assert.EqualValues(t, &ne, &nd)
}

func TestStorageSmartContract_addBlobbersOffers(t *testing.T) {
	const errMsg = "can't get blobber's stake pool: value not present"
	var (
		alloc    StorageAllocation
		b1, b2   StorageNode
		balances = newTestBalances()
		ssc      = newTestStorageSC()

		err error
	)
	// setup
	alloc.ID, b1.ID, b2.ID = "a1", "b1", "b2"
	alloc.ChallengeCompletionTime = 150 * time.Second
	alloc.Expiration = 100
	alloc.BlobberDetails = []*BlobberAllocation{
		&BlobberAllocation{Size: 20 * 1024, Terms: Terms{WritePrice: 12000}},
		&BlobberAllocation{Size: 20 * 1024, Terms: Terms{WritePrice: 4000}},
	}
	// stake pool not found
	var blobbers = []*StorageNode{&b1, &b2}
	requireErrMsg(t, ssc.addBlobbersOffers(&alloc, blobbers, balances), errMsg)
	// create stake pools
	for _, b := range blobbers {
		var sp = newStakePool()
		_, err = balances.InsertTrieNode(stakePoolKey(ssc.ID, b.ID), sp)
		require.NoError(t, err)
	}
	// add the offers
	require.NoError(t, ssc.addBlobbersOffers(&alloc, blobbers, balances))
	// check out all
	var sp1, sp2 *stakePool
	// stake pool 1
	sp1, err = ssc.getStakePool(b1.ID, balances)
	require.NoError(t, err)
	// offer 1
	var off1 = sp1.findOffer(alloc.ID)
	require.NotNil(t, off1)
	assert.Equal(t, toSeconds(alloc.ChallengeCompletionTime)+alloc.Expiration,
		off1.Expire)
	assert.Equal(t, state.Balance(sizeInGB(20*1024)*12000.0), off1.Lock)
	assert.Len(t, sp1.Offers, 1)
	// stake pool 2
	sp2, err = ssc.getStakePool(b2.ID, balances)
	require.NoError(t, err)
	// offer 2
	var off2 = sp2.findOffer(alloc.ID)
	require.NotNil(t, off1)
	assert.Equal(t, toSeconds(alloc.ChallengeCompletionTime)+alloc.Expiration,
		off2.Expire)
	assert.Equal(t, state.Balance(sizeInGB(20*1024)*4000.0), off2.Lock)
	assert.Len(t, sp2.Offers, 1)

}

func Test_updateBlobbersInAll(t *testing.T) {
	var (
		all        StorageNodes
		balances   = newTestBalances()
		b1, b2, b3 StorageNode
		u1, u2     StorageNode
		decode     StorageNodes

		err error
	)

	b1.ID, b2.ID, b3.ID = "b1", "b2", "b3"
	b1.Capacity, b2.Capacity, b3.Capacity = 100, 100, 100

	all.Nodes = []*StorageNode{&b1, &b2, &b3}

	u1.ID, u2.ID = "b1", "b2"
	u1.Capacity, u2.Capacity = 200, 200

	err = updateBlobbersInAll(&all, []*StorageNode{&u1, &u2}, balances)
	require.NoError(t, err)

	var allSeri, ok = balances.tree[ALL_BLOBBERS_KEY]
	require.True(t, ok)
	require.NotNil(t, allSeri)
	require.NoError(t, decode.Decode(allSeri.Encode()))

	require.Len(t, decode.Nodes, 3)
	assert.Equal(t, "b1", decode.Nodes[0].ID)
	assert.Equal(t, int64(200), decode.Nodes[0].Capacity)
	assert.Equal(t, "b2", decode.Nodes[1].ID)
	assert.Equal(t, int64(200), decode.Nodes[1].Capacity)
	assert.Equal(t, "b3", decode.Nodes[2].ID)
	assert.Equal(t, int64(100), decode.Nodes[2].Capacity)
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

func TestStorageSmartContract_newAllocationRequest(t *testing.T) {

	const (
		txHash, clientID, pubKey = "tx_hex", "client_hex", "pub_key_hex"

		errMsg1 = "allocation_creation_failed: " +
			"No Blobbers registered. Failed to create a storage allocation"
		errMsg2 = "allocation_creation_failed: " +
			"No health Blobbers registered. Failed to create an allocation"
		errMsg3 = "allocation_creation_failed: " +
			"Invalid client in the transaction. No client id in transaction"
		errMsg4 = "allocation_creation_failed: malformed request: " +
			"invalid character '}' looking for beginning of value"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()

		tx   transaction.Transaction
		conf scConfig

		resp string
		err  error
	)

	tx.Hash = txHash
	tx.Value = 100
	tx.ClientID = clientID
	tx.CreationDate = toSeconds(2 * time.Hour)

	balances.txn = &tx

	conf.MaxChallengeCompletionTime = 20 * time.Second
	conf.MinAllocDuration = 20 * time.Second
	conf.MinAllocSize = 20 * 1024

	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), &conf)
	require.NoError(t, err)

	// 1.

	_, err = ssc.newAllocationRequest(&tx, nil, balances)
	requireErrMsg(t, err, errMsg1)

	// setup unhealthy blobbers
	var allBlobbers StorageNodes
	allBlobbers.Nodes = []*StorageNode{
		&StorageNode{ID: "b1", LastHealthCheck: 0},
		&StorageNode{ID: "b2", LastHealthCheck: 0},
	}
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, &allBlobbers)
	require.NoError(t, err)

	// 2.

	_, err = ssc.newAllocationRequest(&tx, nil, balances)
	requireErrMsg(t, err, errMsg2)

	// make the blobbers health
	allBlobbers.Nodes[0].LastHealthCheck = tx.CreationDate
	allBlobbers.Nodes[1].LastHealthCheck = tx.CreationDate
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, &allBlobbers)
	require.NoError(t, err)

	// 3.

	tx.ClientID = ""
	_, err = ssc.newAllocationRequest(&tx, nil, balances)
	requireErrMsg(t, err, errMsg3)

	// 4.

	tx.ClientID = clientID
	_, err = ssc.newAllocationRequest(&tx, []byte("} malformed {"), balances)
	requireErrMsg(t, err, errMsg4)

	_ = resp

}

func Test_updateAllocationRequest_decode(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_updateAllocationRequest_validate(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_updateAllocationRequest_getBlobbersSizeDiff(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_updateAllocationRequest_getNewBlobbersSize(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_getAllocationBlobbers(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_closeAllocation(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_saveUpdatedAllocation(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_allocPeriod_weight(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_allocPeriod_join(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_weightedAverage(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_extendAllocation(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_reduceAllocation(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_updateAllocationRequest(t *testing.T) {
	// TODO (sfxdx): implements tests
}
