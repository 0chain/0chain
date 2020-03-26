package storagesc

import (
	"testing"
	"time"

	chainState "0chain.net/chaincore/chain/state"
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
	require.Equal(t, alloc.DataShards, nar.DataShards)
	require.Equal(t, alloc.ParityShards, nar.ParityShards)
	require.Equal(t, alloc.Size, nar.Size)
	require.Equal(t, alloc.Expiration, nar.Expiration)
	require.Equal(t, alloc.Owner, nar.Owner)
	require.Equal(t, alloc.OwnerPublicKey, nar.OwnerPublicKey)
	require.True(t, isEqualStrings(alloc.PreferredBlobbers,
		nar.PreferredBlobbers))
	require.Equal(t, alloc.ReadPriceRange, nar.ReadPriceRange)
	require.Equal(t, alloc.WritePriceRange, nar.WritePriceRange)
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

func newTestAllBlobbers() (all *StorageNodes) {
	all = new(StorageNodes)
	all.Nodes = []*StorageNode{
		&StorageNode{
			ID:      "b1",
			BaseURL: "http://blobber1.test.ru:9100/api",
			Terms: Terms{
				ReadPrice:               20,
				WritePrice:              200,
				MinLockDemand:           0.1,
				MaxOfferDuration:        200 * time.Second,
				ChallengeCompletionTime: 15 * time.Second,
			},
			Capacity:        20 * GB, // 20 GB
			Used:            5 * GB,  //  5 GB
			LastHealthCheck: 0,
		},
		&StorageNode{
			ID:      "b2",
			BaseURL: "http://blobber2.test.ru:9100/api",
			Terms: Terms{
				ReadPrice:               25,
				WritePrice:              250,
				MinLockDemand:           0.05,
				MaxOfferDuration:        250 * time.Second,
				ChallengeCompletionTime: 10 * time.Second,
			},
			Capacity:        20 * GB, // 20 GB
			Used:            10 * GB, // 10 GB
			LastHealthCheck: 0,
		},
	}
	return
}

func TestStorageSmartContract_newAllocationRequest(t *testing.T) {

	const (
		txHash, clientID, pubKey = "a5f4c3d2_tx_hex", "client_hex",
			"pub_key_hex"

		errMsg1 = "allocation_creation_failed: " +
			"No Blobbers registered. Failed to create a storage allocation"
		errMsg2 = "allocation_creation_failed: " +
			"No health Blobbers registered. Failed to create an allocation"
		errMsg3 = "allocation_creation_failed: " +
			"Invalid client in the transaction. No client id in transaction"
		errMsg4 = "allocation_creation_failed: malformed request: " +
			"invalid character '}' looking for beginning of value"
		errMsg5 = "allocation_creation_failed: " +
			"invalid request: invalid read_price range"
		errMsg6 = "allocation_creation_failed: " +
			"Not enough blobbers to honor the allocation"
		errMsg7 = "allocation_request_failed: " +
			"can't get blobber's stake pool: value not present"
		errMsg8 = "allocation_request_failed: " +
			"not enough tokens to create allocation: 0 < 325"
		errMsg9 = "allocation_request_failed: " +
			"can't fill write pool: no tokens to lock"
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
	tx.Value = 400
	tx.ClientID = clientID
	tx.CreationDate = toSeconds(2 * time.Hour)

	balances.txn = &tx

	conf.MaxChallengeCompletionTime = 20 * time.Second
	conf.MinAllocDuration = 20 * time.Second
	conf.MinAllocSize = 20 * GB

	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), &conf)
	require.NoError(t, err)

	// 1.

	_, err = ssc.newAllocationRequest(&tx, nil, balances)
	requireErrMsg(t, err, errMsg1)

	// setup unhealthy blobbers
	var allBlobbers = newTestAllBlobbers()
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbers)
	require.NoError(t, err)

	// 2.

	_, err = ssc.newAllocationRequest(&tx, nil, balances)
	requireErrMsg(t, err, errMsg2)

	// make the blobbers health
	allBlobbers.Nodes[0].LastHealthCheck = tx.CreationDate
	allBlobbers.Nodes[1].LastHealthCheck = tx.CreationDate
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbers)
	require.NoError(t, err)

	// 3.

	tx.ClientID = ""
	_, err = ssc.newAllocationRequest(&tx, nil, balances)
	requireErrMsg(t, err, errMsg3)

	// 4.

	tx.ClientID = clientID
	_, err = ssc.newAllocationRequest(&tx, []byte("} malformed {"), balances)
	requireErrMsg(t, err, errMsg4)

	// 5. invalid request

	var nar newAllocationRequest
	nar.ReadPriceRange = PriceRange{20, 10}

	_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances)
	requireErrMsg(t, err, errMsg5)

	// 6. not enough blobbers (filtered by request, by max_offer_duration)

	nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
	nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
	nar.Size = 20 * GB
	nar.DataShards = 1
	nar.ParityShards = 1
	nar.Expiration = tx.CreationDate + toSeconds(48*time.Hour)
	nar.Owner = "" // not set
	nar.OwnerPublicKey = pubKey
	nar.PreferredBlobbers = nil // not set

	_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances)
	requireErrMsg(t, err, errMsg6)

	// 7. missing stake pools

	nar.Expiration = tx.CreationDate + toSeconds(100*time.Second)

	_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances)
	requireErrMsg(t, err, errMsg7)

	// 8. not enough tokens

	var sp1, sp2 = newStakePool(), newStakePool()
	require.NoError(t, sp1.save(ssc.ID, "b1", balances))
	require.NoError(t, sp2.save(ssc.ID, "b2", balances))

	tx.Value = 0
	_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances)
	requireErrMsg(t, err, errMsg8)

	// 9. no tokens to lock (client balance check)

	allBlobbers.Nodes[0].Used = 5 * GB
	allBlobbers.Nodes[1].Used = 10 * GB
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbers)
	require.NoError(t, err)

	tx.Value = 400
	resp, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances)
	requireErrMsg(t, err, errMsg9)

	// 10. ok

	allBlobbers.Nodes[0].Used = 5 * GB
	allBlobbers.Nodes[1].Used = 10 * GB
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbers)
	require.NoError(t, err)

	balances.balances[clientID] = 1100

	tx.Value = 400
	resp, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances)
	require.NoError(t, err)

	// check response
	var aresp StorageAllocation
	require.NoError(t, aresp.Decode([]byte(resp)))

	assert.Equal(t, txHash, aresp.ID)
	assert.Equal(t, 1, aresp.DataShards)
	assert.Equal(t, 1, aresp.ParityShards)
	assert.Equal(t, int64(20*GB), aresp.Size)
	assert.Equal(t, tx.CreationDate+100, aresp.Expiration)

	// expected blobbers after the allocation
	var sb = newTestAllBlobbers()
	sb.Nodes[0].LastHealthCheck = tx.CreationDate
	sb.Nodes[1].LastHealthCheck = tx.CreationDate
	sb.Nodes[0].Used += 10 * GB
	sb.Nodes[1].Used += 10 * GB

	// blobbers of the allocation
	assert.EqualValues(t, sb.Nodes, aresp.Blobbers)
	// blobbers saved in all blobbers list
	allBlobbers, err = ssc.getBlobbersList(balances)
	require.NoError(t, err)
	assert.EqualValues(t, sb.Nodes, allBlobbers.Nodes)
	// independent saved blobbers
	var b1, b2 *StorageNode
	b1, err = ssc.getBlobber("b1", balances)
	require.NoError(t, err)
	assert.EqualValues(t, sb.Nodes[0], b1)
	b2, err = ssc.getBlobber("b2", balances)
	require.NoError(t, err)
	assert.EqualValues(t, sb.Nodes[1], b2)

	assert.Equal(t, clientID, aresp.Owner)
	assert.Equal(t, pubKey, aresp.OwnerPublicKey)

	if assert.NotNil(t, aresp.Stats) {
		assert.Zero(t, *aresp.Stats)
	}

	assert.Nil(t, aresp.PreferredBlobbers)
	assert.Equal(t, PriceRange{10, 40}, aresp.ReadPriceRange)
	assert.Equal(t, PriceRange{100, 400}, aresp.WritePriceRange)
	assert.Equal(t, 15*time.Second, aresp.ChallengeCompletionTime) // max
	assert.Equal(t, tx.CreationDate, aresp.StartTime)
	assert.False(t, aresp.Finalized)

	// details
	var details = []*BlobberAllocation{
		&BlobberAllocation{
			BlobberID:     "b1",
			AllocationID:  txHash,
			Size:          10 * GB,
			Stats:         &StorageAllocationStats{},
			Terms:         sb.Nodes[0].Terms,
			MinLockDemand: 200, // write_price * (size/GB) * min_lock_demand
			Spent:         0,
		},
		&BlobberAllocation{
			BlobberID:     "b2",
			AllocationID:  txHash,
			Size:          10 * GB,
			Stats:         &StorageAllocationStats{},
			Terms:         sb.Nodes[1].Terms,
			MinLockDemand: 125, // write_price * (size/GB) * min_lock_demand
			Spent:         0,
		},
	}

	assert.EqualValues(t, details, aresp.BlobberDetails)

	// check out pools created and changed:
	//  - write pool, should be created and filled with value of transaction
	//  - stake pool, offer should be added
	//  - challenge pool, should be created

	// 1. write pool
	var wp *writePool
	wp, err = ssc.getWritePool(aresp.ID, balances)
	require.NoError(t, err)
	assert.Equal(t, state.Balance(400), wp.Balance)

	// 2. stake pool offers
	var expire = aresp.Expiration + toSeconds(aresp.ChallengeCompletionTime)

	sp1, err = ssc.getStakePool("b1", balances)
	require.NoError(t, err)
	assert.EqualValues(t, &offerPool{
		Lock:   10 * sb.Nodes[0].Terms.WritePrice,
		Expire: expire,
	}, sp1.Offers[aresp.ID])

	sp2, err = ssc.getStakePool("b2", balances)
	require.NoError(t, err)
	assert.EqualValues(t, &offerPool{
		Lock:   10 * sb.Nodes[1].Terms.WritePrice,
		Expire: expire,
	}, sp2.Offers[aresp.ID])

	// 3. challenge pool existence
	var cp *challengePool
	cp, err = ssc.getChallengePool(aresp.ID, balances)
	require.NoError(t, err)

	assert.Zero(t, cp.Balance)

	// write pool expiration, challenge pool expiration
	assert.Equal(t, &tokenLock{
		StartTime: tx.CreationDate,
		Duration:  time.Duration(expire-tx.CreationDate) * time.Second,
		Owner:     tx.ClientID,
	}, wp.TokenLockInterface)
	assert.Equal(t, &tokenLock{
		StartTime: tx.CreationDate,
		Duration:  time.Duration(expire-tx.CreationDate) * time.Second,
	}, cp.TokenLockInterface)
}

func Test_updateAllocationRequest_decode(t *testing.T) {
	var ud, ue updateAllocationRequest
	ue.Expiration = -1000
	ue.Size = -200
	require.NoError(t, ud.decode(mustEncode(t, &ue)))
	assert.EqualValues(t, ue, ud)
}

func Test_updateAllocationRequest_validate(t *testing.T) {

	var (
		conf  scConfig
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
	alloc.BlobberDetails = []*BlobberAllocation{&BlobberAllocation{}}
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
	assert.Equal(t, int64(256*MB), uar.getBlobbersSizeDiff(&alloc))

	uar.Size = -1 * GB // sub 1 GB
	assert.Equal(t, -int64(256*MB), uar.getBlobbersSizeDiff(&alloc))

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
		conf        scConfig
		err         error
	)

	tx.Hash = txHash
	tx.Value = 400
	tx.ClientID = clientID
	tx.CreationDate = toSeconds(2 * time.Hour)

	balances.(*testBalances).txn = &tx

	conf.MaxChallengeCompletionTime = 20 * time.Second
	conf.MinAllocDuration = 20 * time.Second
	conf.MinAllocSize = 20 * GB

	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), &conf)
	require.NoError(t, err)

	allBlobbers = newTestAllBlobbers()
	allBlobbers.Nodes[0].LastHealthCheck = tx.CreationDate
	allBlobbers.Nodes[1].LastHealthCheck = tx.CreationDate
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbers)
	require.NoError(t, err)

	nar.ReadPriceRange = PriceRange{Min: 10, Max: 40}
	nar.WritePriceRange = PriceRange{Min: 100, Max: 400}
	nar.Size = 20 * GB
	nar.DataShards = 1
	nar.ParityShards = 1
	nar.Expiration = tx.CreationDate + toSeconds(48*time.Hour)
	nar.Owner = "" // not set
	nar.OwnerPublicKey = pubKey
	nar.PreferredBlobbers = nil // not set

	nar.Expiration = tx.CreationDate + toSeconds(100*time.Second)

	var sp1, sp2 = newStakePool(), newStakePool()
	require.NoError(t, sp1.save(ssc.ID, "b1", balances))
	require.NoError(t, sp2.save(ssc.ID, "b2", balances))

	tx.Value = 400

	allBlobbers.Nodes[0].Used = 5 * GB
	allBlobbers.Nodes[1].Used = 10 * GB
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, allBlobbers)
	require.NoError(t, err)

	balances.(*testBalances).balances[clientID] = 1100

	tx.Value = 400
	_, err = ssc.newAllocationRequest(&tx, mustEncode(t, &nar), balances)
	require.NoError(t, err)
	return
}

func Test_updateAllocationRequest_getNewBlobbersSize(t *testing.T) {

	const allocTxHash, clientID, pubKey = "a5f4c3d2_tx_hex", "client_hex",
		"pub_key_hex"

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()

		uar   updateAllocationRequest
		alloc *StorageAllocation
		err   error
	)

	createNewTestAllocation(t, ssc, allocTxHash, clientID, pubKey, balances)

	alloc, err = ssc.getAllocation(allocTxHash, balances)
	require.NoError(t, err)

	alloc.Size = 10 * GB
	alloc.DataShards = 2
	alloc.ParityShards = 2

	uar.Size = 1 * GB // add 1 GB
	assert.Equal(t, int64(10*GB+256*MB), uar.getNewBlobbersSize(alloc))

	uar.Size = -1 * GB // sub 1 GB
	assert.Equal(t, int64(10*GB-256*MB), uar.getNewBlobbersSize(alloc))

	uar.Size = 0 // no changes
	assert.Equal(t, int64(10*GB), uar.getNewBlobbersSize(alloc))
}

func TestStorageSmartContract_getAllocationBlobbers(t *testing.T) {
	const allocTxHash, clientID, pubKey = "a5f4c3d2_tx_hex", "client_hex",
		"pub_key_hex"

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()

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
		errMsg2 = "allocation_closing_failed: " +
			"doesn't need to close allocation is about to expire"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
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
	require.NoError(t, err)

	// 1. expiring allocation
	alloc.Expiration = 1049
	_, err = ssc.closeAllocation(&tx, alloc, balances)
	requireErrMsg(t, err, errMsg1)

	// 2. close (all related pools has created)
	alloc.Expiration = tx.CreationDate +
		toSeconds(alloc.ChallengeCompletionTime) + 20
	resp, err = ssc.closeAllocation(&tx, alloc, balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)

	// checking out

	// TOTH (sfxdx): redo pools, remove expiration, remove locks

	// 1. check out allocation object
	// 2. check out write pool expiration
	// 3. check out stake pool offer expiration
	// 4. check out challenge pool expiration

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
