package storagesc

import (
	"fmt"
	"testing"
	"time"

	"0chain.net/chaincore/currency"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageSmartContract_addBlobber(t *testing.T) {
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)

		tp int64 = 100
	)

	setConfig(t, balances)

	var (
		blob   = addBlobber(t, ssc, 2*GB, tp, avgTerms, 50*x10, balances)
		blob2  = addBlobber(t, ssc, 2*GB, tp, avgTerms, 50*x10, balances)
		b, err = ssc.getBlobber(blob.id, balances)
		b2, _  = ssc.getBlobber(blob2.id, balances)
	)
	require.NoError(t, err)

	// remove
	b.Capacity = 0
	tp += 100
	_, err = updateBlobber(t, b, 0, tp, ssc, balances)
	require.NoError(t, err)

	// reborn
	b.Capacity = 2 * GB
	tp += 100
	_, err = updateBlobber(t, b, 10*x10, tp, ssc, balances)
	require.NoError(t, err)

	var ab *StorageNode
	ab, err = ssc.getBlobber(b.ID, balances)
	require.NoError(t, err)
	require.NotNil(t, ab)

	// can update URL
	const NEW_BASE_URL = "https://new-base-url.com"
	b.BaseURL = NEW_BASE_URL
	b.Capacity = b.Capacity * 2
	tp += 100
	_, err = updateBlobber(t, b, 0, tp, ssc, balances)
	require.NoError(t, err)

	ab, err = ssc.getBlobber(b.ID, balances)
	require.NoError(t, err)
	require.Equal(t, ab.BaseURL, NEW_BASE_URL)
	require.Equal(t, ab.Capacity, b.Capacity)
	// can update URL

	b2.BaseURL = NEW_BASE_URL
	b.Capacity = b2.Capacity * 2
	tp += 100
	_, err = updateBlobber(t, b2, 0, tp, ssc, balances)
	require.Error(t, err)

}

func TestStorageSmartContract_addBlobber_invalidParams(t *testing.T) {
	var (
		ssc            = newTestStorageSC()        //
		balances       = newTestBalances(t, false) //
		terms          = avgTerms                  // copy
		tp       int64 = 100                       //
	)

	var add = func(t *testing.T, ssc *StorageSmartContract, cap, now int64,
		terms Terms, balacne currency.Coin, balances chainState.StateContextI) (
		err error) {

		var blob = newClient(0, balances)
		blob.terms = terms
		blob.cap = cap

		_, err = blob.callAddBlobber(t, ssc, now, balances)
		return
	}

	setConfig(t, balances)

	var conf, err = ssc.getConfig(balances, false)
	require.NoError(t, err)

	terms.MaxOfferDuration = conf.MinOfferDuration - 1*time.Second
	err = add(t, ssc, 2*GB, tp, terms, 0, balances)
	require.Error(t, err)
}

func TestStorageSmartContract_addBlobber_preventDuplicates(t *testing.T) {
	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(t, false)
		tp       int64 = 100
		err      error
	)

	setConfig(t, balances)

	var blob = newClient(0, balances)
	blob.terms = avgTerms
	blob.cap = 2 * GB

	_, err = blob.callAddBlobber(t, ssc, tp, balances)
	require.NoError(t, err)

	_, err = blob.callAddBlobber(t, ssc, tp, balances)
	require.NoError(t, err)

	_, err = ssc.getBlobber(blob.id, balances)
	require.NoError(t, err)
}

func TestStorageSmartContract_addBlobber_updateSettings(t *testing.T) {
	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(t, false)
		tp       int64 = 100
		err      error
	)

	setConfig(t, balances)

	var blob = newClient(0, balances)
	blob.terms = avgTerms
	blob.cap = 2 * GB

	_, err = blob.callAddBlobber(t, ssc, tp, balances)
	require.NoError(t, err)

	_, err = blob.callAddBlobber(t, ssc, tp, balances)
	require.NoError(t, err)

	_, err = ssc.getBlobber(blob.id, balances)
	require.NoError(t, err)
}

// - create allocation
// - write
// - read as owner
// - read as not an owner
// - delete
// - challenge passed
func Test_flow_reward(t *testing.T) {

	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(t, false)
		client         = newClient(100*x10, balances)
		tp, exp  int64 = 0, int64(toSeconds(time.Hour))

		// no owner
		reader = newClient(100*x10, balances)
		err    error
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

	require.EqualValues(t, 202546280, alloc.restMinLockDemand())

	t.Run("read as owner", func(t *testing.T) {
		tp += 100
		var rm ReadConnection
		rm.ReadMarker = &ReadMarker{
			ClientID:        client.id,
			ClientPublicKey: client.pk,
			BlobberID:       b1.id,
			AllocationID:    allocID,
			OwnerID:         client.id,
			Timestamp:       common.Timestamp(tp),
			ReadCounter:     1 * GB / (64 * KB),
		}
		rm.ReadMarker.Signature, err = client.scheme.Sign(
			encryption.Hash(rm.ReadMarker.GetHashData()))
		require.NoError(t, err)

		tp += 100
		var tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
		require.Error(t, err)

		// read pool lock
		tp += 100
		var readPoolFund currency.Coin
		readPoolFund, err = currency.ParseZCN(float64(len(alloc.BlobberAllocs)) * 2)
		require.NoError(t, err)
		tx = newTransaction(client.id, ssc.ID, readPoolFund, tp)
		balances.setTransaction(t, tx)
		_, err = ssc.readPoolLock(tx, mustEncode(t, &readPoolLockRequest{
			TargetId:   client.id,
			MintTokens: false,
		}), balances)
		require.NoError(t, err)

		var rp *readPool
		rp, err = ssc.getReadPool(client.id, balances)
		require.NoError(t, err)
		require.EqualValues(t, readPoolFund, int64(rp.Balance))

		// read
		tp += 100
		tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
		require.NoError(t, err)

		// check out balances
		require.NoError(t, err)
		rp, err = ssc.getReadPool(client.id, balances)
		require.NoError(t, err)
		require.EqualValues(t, readPoolFund-1e10, int64(rp.Balance))

		// min lock demand reducing
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		require.EqualValues(t, 192418966, alloc.restMinLockDemand())
	})

	t.Run("read as unauthorized separate user", func(t *testing.T) {
		tp += 100
		require.NoError(t, err)
		var rm ReadConnection
		rm.ReadMarker = &ReadMarker{
			ClientID:        reader.id,
			ClientPublicKey: reader.pk,
			BlobberID:       b1.id,
			AllocationID:    allocID,
			OwnerID:         client.id,
			Timestamp:       common.Timestamp(tp),
			ReadCounter:     1 * GB / (64 * KB),
		}
		rm.ReadMarker.Signature, err = reader.scheme.Sign(
			encryption.Hash(rm.ReadMarker.GetHashData()))
		require.NoError(t, err)

		tp += 100
		var tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
		require.Error(t, err)

		// read pool lock
		tp += 100

		readPoolFund, err := currency.ParseZCN(float64(len(alloc.BlobberAllocs)) * 2)
		require.NoError(t, err)
		tx = newTransaction(reader.id, ssc.ID, readPoolFund, tp)
		balances.setTransaction(t, tx)
		_, err = ssc.readPoolLock(tx, mustEncode(t, &readPoolLockRequest{
			TargetId:   reader.id,
			MintTokens: false,
		}), balances)
		require.NoError(t, err)

		var rp *readPool
		rp, err = ssc.getReadPool(reader.id, balances)
		require.NoError(t, err)
		require.EqualValues(t, readPoolFund, int64(rp.Balance))

		// read
		tp += 100
		tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
		require.NoError(t, err)
	})

	var b2 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[1].BlobberID {
			b2 = b
			break
		}
	}
	require.NotNil(t, b2)

	t.Run("write", func(t *testing.T) {

		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var apb, cpb = alloc.WritePool, cp.Balance
		require.EqualValues(t, 15*x10, apb)
		require.EqualValues(t, 0, cpb)

		tp += 100
		var cc = &BlobberCloseConnection{
			AllocationRoot:     "root-1",
			PrevAllocationRoot: "",
			WriteMarker: &WriteMarker{
				AllocationRoot:         "root-1",
				PreviousAllocationRoot: "",
				AllocationID:           allocID,
				Size:                   100 * 1024 * 1024, // 100 MB
				BlobberID:              b2.id,
				Timestamp:              common.Timestamp(tp),
				ClientID:               client.id,
			},
		}
		cc.WriteMarker.Signature, err = client.scheme.Sign(
			encryption.Hash(cc.WriteMarker.GetHashData()))
		require.NoError(t, err)

		// write
		tp += 100
		var tx = newTransaction(b2.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		var resp string
		resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
			balances)
		require.NoError(t, err)
		require.NotZero(t, resp)

		// check out
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var moved = int64(sizeInGB(cc.WriteMarker.Size) *
			float64(avgTerms.WritePrice) *
			alloc.restDurationInTimeUnits(cc.WriteMarker.Timestamp))

		require.EqualValues(t, moved, cp.Balance)

		// min lock demand reducing
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		require.EqualValues(t, 182291652, alloc.restMinLockDemand()) // -read above
	})

	t.Run("delete", func(t *testing.T) {

		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var wpb, cpb = alloc.WritePool, cp.Balance
		//require.EqualValues(t, 149932183160, wpb)
		//require.EqualValues(t, 67816840, cpb)
		require.EqualValues(t, 149926531757, wpb)
		require.EqualValues(t, 73468243, cpb)

		tp += 100
		var cc = &BlobberCloseConnection{
			AllocationRoot:     "root-2",
			PrevAllocationRoot: "root-1",
			WriteMarker: &WriteMarker{
				AllocationRoot:         "root-2",
				PreviousAllocationRoot: "root-1",
				AllocationID:           allocID,
				Size:                   -50 * 1024 * 1024, // 50 MB
				BlobberID:              b2.id,
				Timestamp:              common.Timestamp(tp),
				ClientID:               client.id,
			},
		}
		cc.WriteMarker.Signature, err = client.scheme.Sign(
			encryption.Hash(cc.WriteMarker.GetHashData()))
		require.NoError(t, err)

		// write
		tp += 100
		var tx = newTransaction(b2.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		var resp string
		resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
			balances)
		require.NoError(t, err)
		require.NotZero(t, resp)

		// check out
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		require.EqualValues(t, 39559823, cp.Balance)

		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		require.EqualValues(t, 182291652, alloc.restMinLockDemand()) // -read above
	})

	var b3 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[2].BlobberID {
			b3 = b
			break
		}
	}
	require.NotNil(t, b3)

	// add 10 validators
	var valids []*Client
	tp += 100
	for i := 0; i < 10; i++ {
		valids = append(valids, addValidator(t, ssc, tp, balances))
	}

	t.Run("challenge pass", func(t *testing.T) {
		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var blobb1 = balances.balances[b3.id]

		var wpb1, cpb1 = alloc.WritePool, cp.Balance

		require.EqualValues(t, 149960440177, wpb1)
		require.EqualValues(t, 39559823, cpb1)
		require.EqualValues(t, 40*x10, blobb1)

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
				Size:                   100 * 1024 * 1024, // 100 MB
				BlobberID:              b3.id,
				Timestamp:              common.Timestamp(tp),
				ClientID:               client.id,
			},
		}
		cc.WriteMarker.Signature, err = client.scheme.Sign(
			encryption.Hash(cc.WriteMarker.GetHashData()))
		require.NoError(t, err)

		// write
		tp += 100
		var tx = newTransaction(b3.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		var resp string
		resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
			balances)
		require.NoError(t, err)
		require.NotZero(t, resp)

		// balances
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var blobb2 = balances.balances[b3.id]

		var apb2, cpb2 = alloc.WritePool, cp.Balance

		require.EqualValues(t, 149960440177, apb2)
		require.EqualValues(t, 98899558, cpb2)
		require.EqualValues(t, 40*x10, blobb2)

		// until the end
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		// load validators
		validators, err := getValidatorsList(balances)
		require.NoError(t, err)

		// load blobber
		var blobber *StorageNode
		blobber, err = ssc.getBlobber(b3.id, balances)
		require.NoError(t, err)
		//
		var (
			step            = (int64(alloc.Expiration) - tp) / 10
			challID, prevID string
		)
		// expire the allocation challenging it (+ last challenge)
		for i := int64(0); i < 10+1; i++ {
			if i < 10 {
				tp += step / 2
			} else {
				tp += 10 // last challenge, before challenge_completion expired
			}

			challID = fmt.Sprintf("chall-%d", i)
			genChall(t, ssc, b3.id, tp, prevID, challID, i, validators,
				alloc.ID, blobber, allocRoot, balances)

			var chall = new(ChallengeResponse)
			chall.ID = challID

			for _, val := range valids {
				chall.ValidationTickets = append(chall.ValidationTickets,
					val.validTicket(t, chall.ID, b3.id, true, tp))
			}

			tp += step / 2
			tx = newTransaction(b3.id, ssc.ID, 0, tp)
			balances.setTransaction(t, tx)
			var resp string
			resp, err = ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
			if i == 0 {
				require.NoError(t, err)
				require.Equal(t, resp, "challenge passed by blobber")
			} else {
				require.Error(t, err)
				require.Zero(t, resp)
			}
		}

	})

}

func inspectCPIV(t *testing.T, ssc *StorageSmartContract, allocID string, balances *testBalances) {

	t.Helper()

	var _, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
}

// challenge failed
func Test_flow_penalty(t *testing.T) {

	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(t, false)
		client         = newClient(100*x10, balances)
		tp, exp  int64 = 0, int64(toSeconds(time.Hour))

		err error
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

	require.EqualValues(t, 202546280, alloc.restMinLockDemand())

	// add 10 validators
	var valids []*Client
	tp += 100
	for i := 0; i < 10; i++ {
		valids = append(valids, addValidator(t, ssc, tp, balances))
	}

	var b4 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[3].BlobberID {
			b4 = b
			break
		}
	}
	require.NotNil(t, b4)

	t.Run("challenge penalty", func(t *testing.T) {

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
				Size:                   100 * 1024 * 1024, // 100 MB
				BlobberID:              b4.id,
				Timestamp:              common.Timestamp(tp),
				ClientID:               client.id,
			},
		}
		cc.WriteMarker.Signature, err = client.scheme.Sign(
			encryption.Hash(cc.WriteMarker.GetHashData()))
		require.NoError(t, err)

		inspectCPIV(t, ssc, allocID, balances)

		// write
		tp += 100
		var tx = newTransaction(b4.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		var resp string
		resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
			balances)
		require.NoError(t, err)
		require.NotZero(t, resp)

		inspectCPIV(t, ssc, allocID, balances)

		// balances
		//var cp *challengePool
		_, err := ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		//var sp *stakePool
		_, err = ssc.getStakePool(b4.id, balances)
		require.NoError(t, err)

		// until the end
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		// load validators
		validators, err := getValidatorsList(balances)
		require.NoError(t, err)

		// load blobber
		var blobber *StorageNode
		blobber, err = ssc.getBlobber(b4.id, balances)
		require.NoError(t, err)

		//
		var (
			step            = (int64(alloc.Expiration) - tp) / 10
			challID, prevID string

			//until = alloc.Until()
			// last loop balances (previous balance)
			//spl = sp.stake()
			//wpl = wp.allocUntil(allocID, until)
			//cpl = cp.Balance
			//b4l = balances.balances[b4.id]
		)
		// expire the allocation challenging it (+ last challenge)
		for i := int64(0); i < 10+1; i++ {
			if i < 10 {
				tp += step / 2
			} else {
				tp += 10 // last challenge, before challenge_completion expired
			}

			challID = fmt.Sprintf("chall-%d", i)
			genChall(t, ssc, b4.id, tp, prevID, challID, i, validators,
				alloc.ID, blobber, allocRoot, balances)

			var chall = new(ChallengeResponse)
			chall.ID = challID

			// failure tickets
			for _, val := range valids {
				chall.ValidationTickets = append(chall.ValidationTickets,
					val.validTicket(t, chall.ID, b4.id, false, tp))
			}

			tp += step / 2
			tx = newTransaction(b4.id, ssc.ID, 0, tp)
			balances.setTransaction(t, tx)
			var resp string
			resp, err = ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
			require.NoError(t, err)
			require.EqualValues(t, "Challenge Failed by Blobber", resp)
			continue

			//TODO: unreachable code below
			//
			//inspectCPIV(t, ssc, allocID, balances)
			//
			//// check out pools, blobbers, validators balances
			//wp, err = ssc.getWritePool(client.id, balances)
			//require.NoError(t, err)
			//
			//// write pool balance should grow (stake -> write_pool)
			//require.True(t, wpl < wp.allocUntil(allocID, until))
			//wpl = wp.allocUntil(allocID, until)
			//
			//// challenge pool should be reduced (validators reward)
			//cp, err = ssc.getChallengePool(allocID, balances)
			//require.NoError(t, err)
			//
			//// challenge pool tokens should be moved to blobber and validators
			//assert.True(t, cp.Balance < cpl)
			//cpl = cp.Balance
			//
			//// offer pool should be reduced (blobber slash)
			//sp, err = ssc.getStakePool(b4.id, balances)
			//require.NoError(t, err)
			//assert.True(t, sp.stake() < spl)
			//spl = sp.stake()
			//
			//// no rewards for the blobber
			//assert.True(t, b4l == balances.balances[b4.id])
			//b4l = balances.balances[b4.id]
			//
			//// validators reward
			//for _, val := range valids {
			//	_, err = ssc.getStakePool(val.id, balances)
			//	require.NoError(t, err)
			//}
			//
			//// next stage
			//prevID = challID
		}

	})

}

func isAllocBlobber(id string, alloc *StorageAllocation) bool {
	for _, d := range alloc.BlobberAllocs {
		if d.BlobberID == id {
			return true
		}
	}
	return false
}

// no challenge responses, finalize
func Test_flow_no_challenge_responses_finalize(t *testing.T) {
	t.Skip("Assumes blobbers do not get a reward form finilizeAllocation")
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		client   = newClient(100*x10, balances)
		tp, exp  = int64(0), int64(toSeconds(time.Hour))
		conf     = setConfig(t, balances)

		err error
	)

	conf.FailedChallengesToCancel = 100
	conf.FailedChallengesToRevokeMinLock = 50
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	require.NoError(t, err)

	tp += 100
	var allocID, blobs = addAllocation(t, ssc, client, tp, exp, 0, balances)

	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	require.EqualValues(t, 202546280, alloc.restMinLockDemand())

	// add 10 validators
	var valids []*Client
	tp += 100
	for i := 0; i < 10; i++ {
		var valid = addValidator(t, ssc, tp, balances)
		valids = append(valids, valid)
		balances.balances[valid.id] = 0 // reset the balance
	}

	// reset all blobbers balances (blobber stakes itself)
	for _, b := range blobs {
		balances.balances[b.id] = 0 // reset the balance
	}

	require.NoError(t, err)
	var wps = alloc.WritePool

	t.Run("challenges without a response", func(t *testing.T) {

		const allocRoot = "alloc-root-1"

		tp += 10

		for _, b := range blobs {
			if !isAllocBlobber(b.id, alloc) {
				continue
			}
			// write 100 MB
			var cc = &BlobberCloseConnection{
				AllocationRoot:     allocRoot,
				PrevAllocationRoot: "",
				WriteMarker: &WriteMarker{
					AllocationRoot:         allocRoot,
					PreviousAllocationRoot: "",
					AllocationID:           allocID,
					Size:                   100 * 1024 * 1024, // 100 MB
					BlobberID:              b.id,
					Timestamp:              common.Timestamp(tp),
					ClientID:               client.id,
				},
			}
			cc.WriteMarker.Signature, err = client.scheme.Sign(
				encryption.Hash(cc.WriteMarker.GetHashData()))
			require.NoError(t, err)
			// write
			var tx = newTransaction(b.id, ssc.ID, 0, tp)
			balances.setTransaction(t, tx)
			var resp string
			resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
				balances)
			require.NoError(t, err)
			require.NotZero(t, resp)
		}

		// balances
		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		require.NoError(t, err)

		// offer balance, stake pool total balance
		for _, b := range blobs {
			if !isAllocBlobber(b.id, alloc) {
				continue
			}
		}

		// values before
		var (
			wpb = alloc.WritePool
			cpb = cp.Balance
		)

		require.EqualValues(t, wps, wpb+cpb)

		// until the end
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		// load validators
		validators, err := getValidatorsList(balances)
		require.NoError(t, err)

		// ---------------

		tp += 10

		var gfc int

		// generate challenges leaving them without a response
		// (don't got the 'failed challenges to revoke min lock')
		for i := int64(0); i < 2; i++ {
			for _, b := range blobs {
				if !isAllocBlobber(b.id, alloc) {
					continue
				}
				// load blobber
				var blobber *StorageNode
				blobber, err = ssc.getBlobber(b.id, balances)
				require.NoError(t, err)

				var challID, prevID string
				challID = fmt.Sprintf("chall-%s-%d", b.id, i)
				if i > 0 {
					prevID = fmt.Sprintf("chall-%s-%d", b.id, i-1)
				}
				genChall(t, ssc, b.id, tp, prevID, challID, i,
					validators, alloc.ID, blobber, allocRoot, balances)
				gfc++
			}
		}

		// let expire all the challenges
		tp += int64(toSeconds(getMaxChallengeCompletionTime()))

		// add open challenges to allocation stats
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		if alloc.Stats == nil {
			alloc.Stats = new(StorageAllocationStats)
		}
		alloc.Stats.OpenChallenges = 50 // just a non-zero number
		_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), alloc)
		require.NoError(t, err)

		tp += exp // expire the allocation

		var req lockRequest
		req.AllocationID = allocID

		var tx = newTransaction(client.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		_, err = ssc.finalizeAllocation(tx, mustEncode(t, &req), balances)
		require.NoError(t, err)

		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		// check out pools, blobbers, validators balances
		// challenge pool should be empty
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)
		assert.Zero(t, cp.Balance)

		// offer balance, stake pool total balance
		for _, b := range blobs {
			if !isAllocBlobber(b.id, alloc) {
				continue
			}
			var sp *stakePool
			sp, err = ssc.getStakePool(b.id, balances)
			require.NoError(t, err)
			spTotal, err := stakePoolTotal(sp)
			require.NoError(t, err)
			require.EqualValues(t, 10e10, spTotal)
		}

		// values before
		var (
			apa = alloc.WritePool
			cpa = cp.Balance
		)

		require.Zero(t, cpa)
		require.EqualValues(t, apa, wps)

		require.Equal(t, alloc.MovedBack, cpb)

		// no rewards for the blobber
		for _, b := range blobs {
			if !isAllocBlobber(b.id, alloc) {
				continue
			}
			assert.Zero(t, balances.balances[b.id])
		}

		// no rewards for validators
		for _, val := range valids {
			var vsp *stakePool
			vsp, err = ssc.getStakePool(val.id, balances)
			require.NoError(t, err)
			assert.Zero(t, vsp.Reward)
			assert.Zero(t, balances.balances[val.id])
		}

	})

}

// no challenge responses, cancel
func Test_flow_no_challenge_responses_cancel(t *testing.T) {

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		client   = newClient(100*x10, balances)
		tp, exp  = int64(0), int64(toSeconds(time.Hour))
		conf     = setConfig(t, balances)

		err error
	)

	conf.FailedChallengesToCancel = 10
	conf.FailedChallengesToRevokeMinLock = 5
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	require.NoError(t, err)

	tp += 100
	var allocID, blobs = addAllocation(t, ssc, client, tp, exp, 0, balances)

	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	require.EqualValues(t, 202546280, alloc.restMinLockDemand())

	// add 10 validators
	var valids []*Client
	tp += 100
	for i := 0; i < 10; i++ {
		var valid = addValidator(t, ssc, tp, balances)
		valids = append(valids, valid)
		balances.balances[valid.id] = 0 // reset the balance
	}

	// reset all blobbers balances (blobber stakes itself)
	for _, b := range blobs {
		balances.balances[b.id] = 0 // reset the balance
	}

	require.NoError(t, err)
	var wps = alloc.WritePool

	t.Run("challenges without a response", func(t *testing.T) {

		const allocRoot = "alloc-root-1"

		tp += 10

		for _, b := range blobs {
			if !isAllocBlobber(b.id, alloc) {
				continue
			}
			// write 100 MB
			var cc = &BlobberCloseConnection{
				AllocationRoot:     allocRoot,
				PrevAllocationRoot: "",
				WriteMarker: &WriteMarker{
					AllocationRoot:         allocRoot,
					PreviousAllocationRoot: "",
					AllocationID:           allocID,
					Size:                   100 * 1024 * 1024, // 100 MB
					BlobberID:              b.id,
					Timestamp:              common.Timestamp(tp),
					ClientID:               client.id,
				},
			}
			cc.WriteMarker.Signature, err = client.scheme.Sign(
				encryption.Hash(cc.WriteMarker.GetHashData()))
			require.NoError(t, err)
			// write
			var tx = newTransaction(b.id, ssc.ID, 0, tp)
			balances.setTransaction(t, tx)
			var resp string
			resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
				balances)
			require.NoError(t, err)
			require.NotZero(t, resp)
		}

		// balances
		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		// offer balance, stake pool total balance
		for _, b := range blobs {
			if !isAllocBlobber(b.id, alloc) {
				continue
			}
			var sp *stakePool
			sp, err = ssc.getStakePool(b.id, balances)
			require.NoError(t, err)
			spTotal, err := stakePoolTotal(sp)
			require.NoError(t, err)
			require.EqualValues(t, 10e10, spTotal)
		}

		// values before
		var (
			wpb = alloc.WritePool
			cpb = cp.Balance
		)
		afterAlloc, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		require.EqualValues(t, wps, afterAlloc.WritePool+cp.Balance)

		// until the end
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		// load validators
		validators, err := getValidatorsList(balances)
		require.NoError(t, err)

		// ---------------

		var fc = int64(maxInt(conf.FailedChallengesToCancel,
			conf.FailedChallengesToRevokeMinLock))

		tp += 10

		// generate challenges leaving them without a response
		for i := int64(0); i < fc; i++ {
			for _, b := range blobs {
				if !isAllocBlobber(b.id, alloc) {
					continue
				}
				// load blobber
				var blobber *StorageNode
				blobber, err = ssc.getBlobber(b.id, balances)
				require.NoError(t, err)

				var challID, prevID string
				challID = fmt.Sprintf("chall-%s-%d", b.id, i)
				if i > 0 {
					prevID = fmt.Sprintf("chall-%s-%d", b.id, i-1)
				}
				genChall(t, ssc, b.id, tp, prevID, challID, i,
					validators, alloc.ID, blobber, allocRoot, balances)
			}
		}

		// let expire all the challenges
		tp += int64(toSeconds(getMaxChallengeCompletionTime()))

		// add open challenges to allocation stats
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		if alloc.Stats == nil {
			alloc.Stats = new(StorageAllocationStats)
		}
		alloc.Stats.OpenChallenges = 50 // just a non-zero number
		_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), alloc)
		require.NoError(t, err)

		tp += 10 // a not expired allocation to cancel

		var req lockRequest
		req.AllocationID = allocID

		var tx = newTransaction(client.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		_, err = ssc.cancelAllocationRequest(tx, mustEncode(t, &req), balances)
		require.NoError(t, err)

		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		// challenge pool should be empty
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)
		assert.Zero(t, cp.Balance)

		// offer balance, stake pool total balance
		for _, b := range blobs {
			if !isAllocBlobber(b.id, alloc) {
				continue
			}
			var sp *stakePool
			sp, err = ssc.getStakePool(b.id, balances)
			require.NoError(t, err)
			spTotal, err := stakePoolTotal(sp)
			require.NoError(t, err)
			require.EqualValues(t, 10e10, spTotal)
		}

		// values before
		var (
			wpa = alloc.WritePool
			cpa = cp.Balance
		)

		require.Zero(t, cpa)
		require.EqualValues(t, wpb, wpa)
		require.Equal(t, alloc.MovedBack, cpb)

		// no rewards for the blobber
		for _, b := range blobs {
			if !isAllocBlobber(b.id, alloc) {
				continue
			}
			assert.Zero(t, balances.balances[b.id])
		}

		// no rewards for validators
		for _, val := range valids {
			var vsp *stakePool
			vsp, err = ssc.getStakePool(val.id, balances)
			require.NoError(t, err)
			assert.Zero(t, vsp.Reward)
			assert.Zero(t, balances.balances[val.id])
		}

	})

}
