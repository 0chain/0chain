package storagesc

import (
	"fmt"
	"testing"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageSmartContract_addBlobber(t *testing.T) {
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()

		tp int64 = 100
	)

	setConfig(t, balances)

	var (
		blob   = addBlobber(t, ssc, 2*GB, tp, avgTerms, 50*x10, balances)
		b, err = ssc.getBlobber(blob.id, balances)
	)
	require.NoError(t, err)

	// remove
	b.Capacity = 0
	tp += 100
	_, err = updateBlobber(t, b, 0, tp, ssc, balances)
	require.NoError(t, err)

	var all *StorageNodes
	all, err = ssc.getBlobbersList(balances)
	require.NoError(t, err)
	require.Len(t, all.Nodes, 0)

	// reborn
	b.Capacity = 2 * GB
	tp += 100
	_, err = updateBlobber(t, b, 10*x10, tp, ssc, balances)
	require.NoError(t, err)

	all, err = ssc.getBlobbersList(balances)
	require.NoError(t, err)
	require.Len(t, all.Nodes, 1)
	var ab, ok = all.Nodes.get(b.ID)
	require.True(t, ok)
	require.NotNil(t, ab)
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
		balances       = newTestBalances()
		client         = newClient(100*x10, balances)
		tp, exp  int64 = 0, int64(toSeconds(time.Hour))

		// no owner
		reader = newClient(100*x10, balances)
		err    error
	)

	setConfig(t, balances)

	tp += 100
	var allocID, blobs = addAllocation(t, ssc, client, tp, exp, balances)

	// blobbers: stake 10k, balance 40k

	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	var b1 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberDetails[0].BlobberID {
			b1 = b
			break
		}
	}
	require.NotNil(t, b1)

	require.EqualValues(t, 10000000040, alloc.minLockDemandLeft())

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
			ReadCounter:     (1 * GB) / (64 * KB),
		}
		rm.ReadMarker.Signature, err = client.scheme.Sign(
			encryption.Hash(rm.ReadMarker.GetHashData()))
		require.NoError(t, err)

		tp += 100
		var tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.txn = tx
		_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
		require.Error(t, err)

		// create read pool
		tp += 100
		tx = newTransaction(client.id, ssc.ID, 0, tp)
		balances.txn = tx
		_, err = ssc.newReadPool(tx, nil, balances)
		require.NoError(t, err)

		// read pool lock
		tp += 100
		var readPoolFund = int64(len(alloc.BlobberDetails)) * 2 * 1e10
		tx = newTransaction(client.id, ssc.ID, readPoolFund, tp)
		balances.txn = tx
		_, err = ssc.readPoolLock(tx, mustEncode(t, &lockRequest{
			Duration:     20 * time.Minute,
			AllocationID: allocID,
		}), balances)
		require.NoError(t, err)

		var rp *readPool
		rp, err = ssc.getReadPool(client.id, balances)
		require.NoError(t, err)
		require.EqualValues(t, 2e10, rp.allocBlobberTotal(allocID, b1.id, tp))

		// read
		tp += 100
		tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.txn = tx
		_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
		require.NoError(t, err)

		// check out balances
		var sp *stakePool
		sp, err = ssc.getStakePool(b1.id, balances)
		require.NoError(t, err)
		require.EqualValues(t, 1e10, sp.Rewards.Balance)

		rp, err = ssc.getReadPool(client.id, balances)
		require.NoError(t, err)
		require.EqualValues(t, readPoolFund-1e10, rp.allocTotal(allocID, tp))
		require.EqualValues(t, 1e10, rp.allocBlobberTotal(allocID, b1.id, tp))

		// min lock demand reducing
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		require.EqualValues(t, 9500000038, alloc.minLockDemandLeft())
	})

	t.Run("read as separate user", func(t *testing.T) {
		tp += 100
		var rm ReadConnection
		rm.ReadMarker = &ReadMarker{
			ClientID:        reader.id,
			ClientPublicKey: reader.pk,
			BlobberID:       b1.id,
			AllocationID:    allocID,
			OwnerID:         client.id,
			Timestamp:       common.Timestamp(tp),
			ReadCounter:     (1 * GB) / (64 * KB),
		}
		rm.ReadMarker.Signature, err = reader.scheme.Sign(
			encryption.Hash(rm.ReadMarker.GetHashData()))
		require.NoError(t, err)

		tp += 100
		var tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.txn = tx
		_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
		require.Error(t, err)

		// create read pool
		tp += 100
		tx = newTransaction(reader.id, ssc.ID, 0, tp)
		balances.txn = tx
		_, err = ssc.newReadPool(tx, nil, balances)
		require.NoError(t, err)

		// read pool lock
		tp += 100
		tx = newTransaction(reader.id, ssc.ID,
			int64(len(alloc.BlobberDetails))*2*x10, tp)
		balances.txn = tx
		_, err = ssc.readPoolLock(tx, mustEncode(t, &lockRequest{
			Duration:     20 * time.Minute,
			AllocationID: allocID,
		}), balances)
		require.NoError(t, err)

		// read
		tp += 100
		tx = newTransaction(b1.id, ssc.ID, 0, tp)
		balances.txn = tx
		_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
		require.NoError(t, err)

		// check out balances
		var sp *stakePool
		sp, err = ssc.getStakePool(b1.id, balances)
		require.NoError(t, err)
		require.EqualValues(t, 2e10, sp.Rewards.Balance)

		var rp *readPool
		rp, err = ssc.getReadPool(reader.id, balances)
		require.NoError(t, err)

		require.EqualValues(t, 10000000000,
			rp.allocBlobberTotal(allocID, b1.id, tp))

		// min lock demand reducing
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		require.EqualValues(t, 9500000038, alloc.minLockDemandLeft())
	})

	var b2 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberDetails[1].BlobberID {
			b2 = b
			break
		}
	}
	require.NotNil(t, b2)

	var until = int64(alloc.Until())

	t.Run("write", func(t *testing.T) {

		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var wp *writePool
		wp, err = ssc.getWritePool(client.id, balances)
		require.NoError(t, err)

		var wpb, cpb = wp.allocTotal(allocID, until), cp.Balance
		require.EqualValues(t, 15*x10, wpb)
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
		balances.txn = tx
		var resp string
		resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
			balances)
		require.NoError(t, err)
		require.NotZero(t, resp)

		// check out
		var sp *stakePool
		sp, err = ssc.getStakePool(b1.id, balances)
		require.NoError(t, err)
		require.EqualValues(t, 2e10, sp.Rewards.Balance)

		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var moved = int64(sizeInGB(cc.WriteMarker.Size) *
			float64(avgTerms.WritePrice))
		require.EqualValues(t, moved, cp.Balance)

		wp, err = ssc.getWritePool(client.id, balances)
		require.NoError(t, err)

		require.EqualValues(t, 15*x10-moved, wp.allocTotal(allocID, until))

		// min lock demand reducing
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		require.EqualValues(t, 9000000036, alloc.minLockDemandLeft()) // -read above
	})

	t.Run("delete", func(t *testing.T) {

		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var wp *writePool
		wp, err = ssc.getWritePool(client.id, balances)
		require.NoError(t, err)

		var wpb, cpb = wp.allocTotal(allocID, until), cp.Balance
		require.EqualValues(t, 145117187500, wpb)
		require.EqualValues(t, 4882812500, cpb)

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
		balances.txn = tx
		var resp string
		resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
			balances)
		require.NoError(t, err)
		require.NotZero(t, resp)

		// check out
		var sp *stakePool
		sp, err = ssc.getStakePool(b1.id, balances)
		require.NoError(t, err)
		require.EqualValues(t, 2e10, sp.Rewards.Balance)

		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		// nothing has moved
		require.EqualValues(t, int64(cpb), cp.Balance)

		wp, err = ssc.getWritePool(client.id, balances)
		require.NoError(t, err)

		require.EqualValues(t, int64(wpb), wp.allocTotal(allocID, until))

		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		require.EqualValues(t, 9000000036, alloc.minLockDemandLeft()) // -read above
	})

	var b3 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberDetails[2].BlobberID {
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

		var wp *writePool
		wp, err = ssc.getWritePool(client.id, balances)
		require.NoError(t, err)

		var blobb1 = balances.balances[b3.id]

		var wpb1, cpb1 = wp.allocTotal(allocID, until), cp.Balance
		require.EqualValues(t, 145117187500, wpb1)
		require.EqualValues(t, 4882812500, cpb1)
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
		balances.txn = tx
		var resp string
		resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
			balances)
		require.NoError(t, err)
		require.NotZero(t, resp)

		// balances
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		wp, err = ssc.getWritePool(client.id, balances)
		require.NoError(t, err)

		var blobb2 = balances.balances[b3.id]

		var wpb2, cpb2 = wp.allocTotal(allocID, until), cp.Balance
		require.EqualValues(t, 140234375000, wpb2)
		require.EqualValues(t, 9765625000, cpb2)
		require.EqualValues(t, 40*x10, blobb2)

		// until the end
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		// load validators
		var validators *ValidatorNodes
		validators, err = ssc.getValidatorsList(balances)
		require.NoError(t, err)

		// load blobber
		var blobber *StorageNode
		blobber, err = ssc.getBlobber(b3.id, balances)
		require.NoError(t, err)

		var sp *stakePool
		sp, err = ssc.getStakePool(b3.id, balances)
		require.NoError(t, err)

		//
		var (
			step            = (int64(alloc.Expiration) - tp) / 10
			challID, prevID string
			// last loop balances (previous balance)
			cpl     = cpb2
			b3l     = sp.Rewards.Balance
			validsl []state.Balance
		)
		// validators balances
		for _, val := range valids {
			validsl = append(validsl, balances.balances[val.id])
		}
		// expire the allocation challenging it (+ last challenge)
		for i := int64(0); i < 10+1; i++ {
			if i < 10 {
				tp += step / 2
			} else {
				tp += 10 // last challenge, before challenge_completion expired
			}

			challID = fmt.Sprintf("chall-%d", i)
			genChall(t, ssc, b3.id, tp, prevID, challID, i, validators.Nodes,
				alloc.ID, blobber, allocRoot, balances)

			var chall = new(ChallengeResponse)
			chall.ID = challID

			for _, val := range valids {
				chall.ValidationTickets = append(chall.ValidationTickets,
					val.validTicket(t, chall.ID, b3.id, true, tp))
			}

			tp += step / 2
			tx = newTransaction(b3.id, ssc.ID, 0, tp)
			balances.txn = tx
			var resp string
			resp, err = ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
			require.NoError(t, err)
			require.NotZero(t, resp)

			// check out pools, blobbers, validators balances
			wp, err = ssc.getWritePool(client.id, balances)
			require.NoError(t, err)

			// write pool balance should be the same
			require.EqualValues(t, wpb2, wp.allocTotal(allocID, until))

			cp, err = ssc.getChallengePool(allocID, balances)
			require.NoError(t, err)

			// challenge pool tokens should be moved to blobber and validators
			assert.True(t, cp.Balance < cpl)
			cpl = cp.Balance

			// blobber reward
			sp, err = ssc.getStakePool(b3.id, balances)
			require.NoError(t, err)
			assert.True(t, b3l < sp.Rewards.Balance)
			b3l = sp.Rewards.Balance

			// validators reward
			for i, val := range valids {
				var vsp *stakePool
				vsp, err = ssc.getStakePool(val.id, balances)
				require.NoError(t, err)
				assert.True(t, validsl[i] < vsp.Rewards.Validator)
				validsl[i] = vsp.Rewards.Validator
			}

			// next stage
			prevID = challID
		}

	})

}

// challenge failed
func Test_flow_penalty(t *testing.T) {

	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances()
		client         = newClient(100*x10, balances)
		tp, exp  int64 = 0, int64(toSeconds(time.Hour))

		err error
	)

	setConfig(t, balances)

	tp += 100
	var allocID, blobs = addAllocation(t, ssc, client, tp, exp, balances)

	// blobbers: stake 10k, balance 40k

	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	var b1 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberDetails[0].BlobberID {
			b1 = b
			break
		}
	}
	require.NotNil(t, b1)

	require.EqualValues(t, 10000000040, alloc.minLockDemandLeft())

	// add 10 validators
	var valids []*Client
	tp += 100
	for i := 0; i < 10; i++ {
		valids = append(valids, addValidator(t, ssc, tp, balances))
	}

	var b4 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberDetails[3].BlobberID {
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

		// write
		tp += 100
		var tx = newTransaction(b4.id, ssc.ID, 0, tp)
		balances.txn = tx
		var resp string
		resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
			balances)
		require.NoError(t, err)
		require.NotZero(t, resp)

		// balances
		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var wp *writePool
		wp, err = ssc.getWritePool(client.id, balances)
		require.NoError(t, err)

		var sp *stakePool
		sp, err = ssc.getStakePool(b4.id, balances)
		require.NoError(t, err)

		var offer = sp.findOffer(allocID)
		require.NotNil(t, offer)

		// until the end
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		// load validators
		var validators *ValidatorNodes
		validators, err = ssc.getValidatorsList(balances)
		require.NoError(t, err)

		// load blobber
		var blobber *StorageNode
		blobber, err = ssc.getBlobber(b4.id, balances)
		require.NoError(t, err)

		//
		var (
			step            = (int64(alloc.Expiration) - tp) / 10
			challID, prevID string

			until = alloc.Until()
			// last loop balances (previous balance)
			spl     = sp.stake()
			wpl     = wp.allocUntil(allocID, until)
			opl     = offer.Lock
			cpl     = cp.Balance
			b4l     = balances.balances[b4.id]
			validsl []state.Balance
		)
		// validators balances
		for _, val := range valids {
			validsl = append(validsl, balances.balances[val.id])
		}
		// expire the allocation challenging it (+ last challenge)
		for i := int64(0); i < 10+1; i++ {
			if i < 10 {
				tp += step / 2
			} else {
				tp += 10 // last challenge, before challenge_completion expired
			}

			challID = fmt.Sprintf("chall-%d", i)
			genChall(t, ssc, b4.id, tp, prevID, challID, i, validators.Nodes,
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
			balances.txn = tx
			var resp string
			resp, err = ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
			require.NoError(t, err)
			require.NotZero(t, resp)

			// check out pools, blobbers, validators balances
			wp, err = ssc.getWritePool(client.id, balances)
			require.NoError(t, err)

			// write pool balance should grow (stake -> write_pool)
			require.True(t, wpl < wp.allocUntil(allocID, until))
			wpl = wp.allocUntil(allocID, until)

			// challenge pool should be reduced (validators reward)
			cp, err = ssc.getChallengePool(allocID, balances)
			require.NoError(t, err)

			// challenge pool tokens should be moved to blobber and validators
			assert.True(t, cp.Balance < cpl)
			cpl = cp.Balance

			// offer pool should be reduced (blobber slash)
			sp, err = ssc.getStakePool(b4.id, balances)
			require.NoError(t, err)
			assert.True(t, sp.stake() < spl)
			spl = sp.stake()

			offer = sp.findOffer(allocID)
			require.NotNil(t, offer)
			assert.True(t, opl > offer.Lock)
			opl = offer.Lock

			// no rewards for the blobber
			assert.True(t, b4l == balances.balances[b4.id])
			b4l = balances.balances[b4.id]

			// validators reward
			for i, val := range valids {
				var vsp *stakePool
				vsp, err = ssc.getStakePool(val.id, balances)
				require.NoError(t, err)
				assert.True(t, validsl[i] < vsp.Rewards.Validator)
				validsl[i] = vsp.Rewards.Validator
			}

			// next stage
			prevID = challID
		}

	})

}
