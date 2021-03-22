package storagesc

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/encryption"

	"github.com/stretchr/testify/require"
)

func Test_challenge_pool_moveToWritePool(t *testing.T) {

	const allocID, until, earlier = "alloc_hex", 20, 10

	var (
		wp = new(writePool)
		ap = wp.allocPool(allocID, until)
	)

	require.Nil(t, ap)

	ap = new(allocationPool)
	ap.AllocationID = allocID
	ap.ExpireAt = 0
	wp.Pools.add(ap)

	require.NotNil(t, wp.allocPool(allocID, until))
	require.NotNil(t, wp.allocPool(allocID, earlier))
}

func Test_challenge_timeout(t *testing.T) {

	// create a challenge
	var (
		storageSC                     = newTestStorageSC()
		balances                      = newTestBalances(t, true)
		client                        = newClient(100000*x10, balances)
		tp, challengeExpiration int64 = 0, int64(toSeconds(time.Second * 3))
	)

	defer balances.mpts.Close()

	balances.skipMerge = true
	storageSCConfig := setConfig(t, balances)

	t.Log("add 1 blobbers")
	tp += 1
	balances.skipMerge = true // don't merge transactions for now
	_, blobbers := addAllocation(t, storageSC, client, tp, challengeExpiration, 1, balances)

	if len(blobbers) != 1 {
		t.Errorf("unexpected number of blobbers")
	}

	t.Log("add 1 corresponding validators")
	tp += 1
	tx := newTransaction(blobbers[0].id, storageSC.ID, 0, tp)
	_, err := storageSC.addValidator(tx, blobbers[0].addValidatorRequest(t), balances)
	require.NoError(t, err)

	storageSCConfig.MinAllocSize = 1 * KB
	mustSave(t, scConfigKey(ADDRESS), storageSCConfig, balances)

	t.Log("add 1 allocations")
	var allocationID string
	var nar = new(newAllocationRequest)
	nar.DataShards = 10
	nar.ParityShards = 10
	nar.Expiration = common.Timestamp(challengeExpiration)
	nar.Owner = client.id
	nar.OwnerPublicKey = client.pk
	nar.ReadPriceRange = PriceRange{1 * x10, 10 * x10}
	nar.WritePriceRange = PriceRange{2 * x10, 20 * x10}
	nar.Size = 1 * KB
	nar.MaxChallengeCompletionTime = 200 * time.Hour

	resp, err := nar.callNewAllocReq(t, client.id, 15*x10, storageSC, tp,
		balances)
	require.NoError(t, err)

	var deco StorageAllocation
	require.NoError(t, deco.Decode([]byte(resp)))

	allocationID = deco.ID

	t.Log("write 10 files")
	var stats StorageStats
	stats.Stats = new(StorageAllocationStats)
	var alloc *StorageAllocation
	alloc, err = storageSC.getAllocation(allocationID, balances)
	require.NoError(t, err)
	alloc.Stats = new(StorageAllocationStats)
	alloc.Stats.NumWrites += 10 // 10 files
	for _, d := range alloc.BlobberDetails {
		d.AllocationRoot = "allocation-root"
	}
	_, err = balances.InsertTrieNode(alloc.GetKey(storageSC.ID), alloc)
	require.NoError(t, err)
	stats.Stats.NumWrites += 10    // total stats
	stats.Stats.UsedSize += 1 * GB // fake size just for the challenges
	_, err = balances.InsertTrieNode(stats.GetKey(storageSC.ID), &stats)
	require.NoError(t, err)

	t.Log("merge all into p node db")
	balances.skipMerge = false
	balances.mpts.merge(t)

	var valids *ValidatorNodes
	valids, err = storageSC.getValidatorsList(balances)
	require.NoError(t, err)

	tp += 1
	alloc, err = storageSC.getAllocation(allocationID, balances)
	require.NoError(t, err)

	var challID = encryption.Hash(fmt.Sprintf("chall-%d", tp))
	_, err = storageSC.addChallenge(alloc, valids, challID,
		common.Timestamp(tp), rand.New(rand.NewSource(tp)), tp, balances)
	require.NoError(t, err)

	// check if challenge has been added
	blobberChallengeObj, err := storageSC.getBlobberChallenge(blobbers[0].id, balances)
	require.NoError(t, err)

	if len(blobberChallengeObj.Challenges) != 1 {
		t.Errorf("unexpected number of challenges")
	}

	if blobberChallengeObj.Challenges[0].ID != challID {
		t.Errorf("challenge created with wrong challenge id")
	}

	// wait for challenge to expire
	tp += 4

	// check if challenge expired without clean up
	blobberChallengeObj, err = storageSC.getBlobberChallenge(blobbers[0].id, balances)
	require.NoError(t, err)

	if len(blobberChallengeObj.Challenges) != 1 {
		t.Errorf("unexpected number of challenges")
	}

	if blobberChallengeObj.Challenges[0].ID != challID {
		t.Errorf("challenge created with wrong challenge id")
	}

	// trigger challenge clean up
	err = storageSC.clearExpiredChallenges(common.Timestamp(tp), balances)
	require.NoError(t, err)

	// check if challenge expired
	blobberChallengeObj, err = storageSC.getBlobberChallenge(blobbers[0].id, balances)
	require.NoError(t, err)

	if len(blobberChallengeObj.Challenges) != 0 {
		t.Errorf("unexpected number of challenges")
	}
}
