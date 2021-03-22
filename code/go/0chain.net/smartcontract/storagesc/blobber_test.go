package storagesc

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const CHUNK_SIZE = 64 * KB // hardcoded in blobber.go

type mockBlobberYml struct {
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

var (
	blobberYaml = mockBlobberYml{
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

func TestStorageSmartContract_addBlobber_invalidParams(t *testing.T) {
	var (
		ssc            = newTestStorageSC()        //
		balances       = newTestBalances(t, false) //
		terms          = avgTerms                  // copy
		tp       int64 = 100                       //
	)

	var add = func(t *testing.T, ssc *StorageSmartContract, cap, now int64,
		terms Terms, balacne state.Balance, balances chainState.StateContextI) (
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

	terms.ChallengeCompletionTime = conf.MaxChallengeCompletionTime +
		1*time.Second

	err = add(t, ssc, 2*GB, tp, terms, 0, balances)
	require.Error(t, err)

	terms.ChallengeCompletionTime = conf.MaxChallengeCompletionTime -
		1*time.Second
	terms.MaxOfferDuration = conf.MinOfferDuration - 1*time.Second
	err = add(t, ssc, 2*GB, tp, terms, 0, balances)
	require.Error(t, err)
}

// Test payments for some simple cases
// read - Checks service charge, payment to the blobber and increment to read pool
// write - Checks read lock added to challenge pool for later blobber payment and
//         subtracted from locked allocation amount.
func TestFeesPayments(t *testing.T) {
	const (
		numBlobbers                  = 30
		blobberBalance state.Balance = 50 * x10
		clientBalance  state.Balance = 100 * x10

		readSize        = 1 * GB
		readCount int64 = readSize / CHUNK_SIZE

		// allocation setup constants
		aValue                state.Balance = 15 * x10
		aMaxReadPrice                       = 10 * x10
		aMinReadPrice                       = 1 * x10
		aMinWritePrice                      = 2 * x10
		aMaxWritePrice                      = 20 * x10
		aMaxChallengeCompTime               = 200 * time.Hour
		aRequestSize                        = 2 * GB
		aDataShards                         = 10
		aParityShards                       = 10

		// write
		allocationRoot        = "root-1"
		writeSize             = 100 * 1024 * 1024 // 100 MB
		lockedFundsPerBlobber = 2 * 1e10
	)
	var (
		aExpiration int64 = int64(toSeconds(time.Hour))
		terms             = Terms{
			ReadPrice:               zcnToBalance(blobberYaml.ReadPrice),
			WritePrice:              zcnToBalance(blobberYaml.WritePrice),
			MinLockDemand:           blobberYaml.MinLockDemand,
			MaxOfferDuration:        blobberYaml.MaxOfferDuration,
			ChallengeCompletionTime: blobberYaml.ChallengeCompletionTime,
		}
		allocationRequest = newAllocationRequest{ // input to ./zbox --newallocation
			DataShards:                 aDataShards,                                // --data
			ParityShards:               aParityShards,                              // -parity
			Expiration:                 common.Timestamp(aExpiration),              // --expire
			ReadPriceRange:             PriceRange{aMinReadPrice, aMaxReadPrice},   // --lock
			WritePriceRange:            PriceRange{aMinWritePrice, aMaxWritePrice}, // --lock
			Size:                       aRequestSize,                               // --size
			MaxChallengeCompletionTime: aMaxChallengeCompTime,                      // --mcct
		}
		f formulae = formulae{
			blobber: blobberYaml,
			ar:      allocationRequest,
		}
	)

	t.Run("new allocation", func(t *testing.T) {
		ssc, ctx, _, _, allocationId, _, _ := attachBlobbersAndNewAllocation(t, terms,
			allocationRequest, blobberYaml.Capacity, clientBalance, blobberBalance, aValue, numBlobbers)

		_, err := ssc.getAllocation(allocationId, ctx)
		require.NoError(t, err)
	})

	t.Run("read", func(t *testing.T) {
		ssc, ctx, _, now, allocationId, client, testBlobber :=
			attachBlobbersAndNewAllocation(t, terms, allocationRequest,
				blobberYaml.Capacity, clientBalance, blobberBalance, aValue, numBlobbers)
		require.NotNil(t, testBlobber)
		var readMarker = ReadConnection{
			ReadMarker: &ReadMarker{
				ClientID:        client.id,
				ClientPublicKey: client.pk,
				BlobberID:       testBlobber.id,
				AllocationID:    allocationId, // --alocation
				OwnerID:         client.id,
				Timestamp:       common.Timestamp(now),
				ReadCounter:     readCount,
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
		_, err = ssc.getAllocation(allocationId, ctx)
		require.NoError(t, err)
		var readPoolFund = state.Balance(allocationRequest.DataShards+allocationRequest.ParityShards) *
			lockedFundsPerBlobber

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

		// check out ctx
		sPool, err := ssc.getStakePool(testBlobber.id, ctx)
		require.NoError(t, err)

		f.readMarker = *readMarker.ReadMarker
		f.sc = *setConfig(t, ctx)
		require.EqualValues(t, f.readServiceCharge(), sPool.Rewards.Charge)
		require.EqualValues(t, f.readRewardsBlobber(), sPool.Rewards.Blobber)

		rPool, err = ssc.getReadPool(client.id, ctx)
		require.NoError(t, err)
		require.EqualValues(t, readPoolFund-f.readCost(),
			rPool.Pools.allocTotal(allocationId, now))
		require.EqualValues(t, f.readCost(),
			rPool.Pools.allocBlobberTotal(allocationId, testBlobber.id, now))

		_, err = ssc.getAllocation(allocationId, ctx)
		require.NoError(t, err)
	})

	t.Run("write", func(t *testing.T) {
		ssc, ctx, blobbers, now, allocationId, client, testBlobber1 :=
			attachBlobbersAndNewAllocation(t, terms, allocationRequest,
				blobberYaml.Capacity, clientBalance, blobberBalance, aValue, numBlobbers)

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

		challengePool, err := ssc.getChallengePool(allocationId, ctx)
		require.NoError(t, err)

		writePool, err := ssc.getWritePool(client.id, ctx)
		require.NoError(t, err)

		var writePoolTotal = writePool.Pools.allocTotal(allocationId, until)
		require.EqualValues(t, aValue, writePoolTotal)
		require.EqualValues(t, 0, challengePool.Balance)

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
		f.writeMarker = *cc.WriteMarker
		f.sc = *setConfig(t, ctx)
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

		stakePool, err := ssc.getStakePool(testBlobber1.id, ctx)
		require.NoError(t, err)
		require.EqualValues(t, 0, stakePool.Rewards.Blobber+stakePool.Rewards.Charge)

		challengePool, err = ssc.getChallengePool(allocationId, ctx)
		require.NoError(t, err)
		require.EqualValues(t, f.lockCostForWrite(), challengePool.Balance)

		writePool, err = ssc.getWritePool(client.id, ctx)
		require.NoError(t, err)
		require.EqualValues(t, state.Balance(aValue)-f.lockCostForWrite(),
			writePool.Pools.allocTotal(allocationId, now))
	})
}

func inspectCPIV(t *testing.T, name string, ssc *StorageSmartContract,
	allocID string, balances *testBalances) {

	t.Helper()

	var alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	for _, d := range alloc.BlobberDetails {
		if d.ChallengePoolIntegralValue == 0 {
			continue
		}
		t.Log(name, "CPIV", d.BlobberID, d.ChallengePoolIntegralValue)
	}
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
		if b.id == alloc.BlobberDetails[0].BlobberID {
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

		inspectCPIV(t, "before", ssc, allocID, balances)

		// write
		tp += 100
		var tx = newTransaction(b4.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		var resp string
		resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
			balances)
		require.NoError(t, err)
		require.NotZero(t, resp)

		inspectCPIV(t, "after commit", ssc, allocID, balances)

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
			balances.setTransaction(t, tx)
			var resp string
			resp, err = ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
			require.NoError(t, err)
			require.NotZero(t, resp)

			inspectCPIV(t, fmt.Sprintf("after challenge %d", i), ssc, allocID,
				balances)

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
				require.NotNil(t, vsp)
				require.NotNil(t, vsp.Rewards)
				// assert.True(t, validsl[i] < vsp.Rewards.Validator)
				validsl[i] = vsp.Rewards.Validator
			}

			// next stage
			prevID = challID
		}

	})

}

func isAllocBlobber(id string, alloc *StorageAllocation) bool {
	for _, d := range alloc.BlobberDetails {
		if d.BlobberID == id {
			return true
		}
	}
	return false
}

// no challenge responses, finalize
func Test_flow_no_challenge_responses_finalize(t *testing.T) {

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

	var wp *writePool
	wp, err = ssc.getWritePool(client.id, balances)
	require.NoError(t, err)
	var wps = wp.allocUntil(alloc.ID, alloc.Until())

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

		var wp *writePool
		wp, err = ssc.getWritePool(client.id, balances)
		require.NoError(t, err)

		// offer balance, stake pool total balance
		for _, b := range blobs {
			if !isAllocBlobber(b.id, alloc) {
				continue
			}
			var sp *stakePool
			sp, err = ssc.getStakePool(b.id, balances)
			require.NoError(t, err)

			var offer = sp.findOffer(allocID)
			require.NotNil(t, offer)
			require.EqualValues(t, 10e10, stakePoolTotal(sp))
			require.EqualValues(t, 5000000027, offer.Lock)
		}

		// values before
		var (
			wpb = wp.allocUntil(alloc.ID, alloc.Until())
			cpb = cp.Balance
		)

		require.EqualValues(t, wps, wpb+cpb)

		// until the end
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		// load validators
		var validators *ValidatorNodes
		validators, err = ssc.getValidatorsList(balances)
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
					validators.Nodes, alloc.ID, blobber, allocRoot, balances)
				gfc++
			}
		}

		// let expire all the challenges
		tp += int64(toSeconds(avgTerms.ChallengeCompletionTime))

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
		wp, err = ssc.getWritePool(client.id, balances)
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
			require.Nil(t, sp.findOffer(allocID)) // no offers expected
			require.EqualValues(t, 10e10, stakePoolTotal(sp))
		}

		// values before
		var (
			wpa = wp.allocUntil(alloc.ID, alloc.Until())
			cpa = cp.Balance
		)

		require.Zero(t, cpa)
		require.EqualValues(t, wpa, wps)
		require.EqualValues(t, wps, wp.Pools.gimmeAll())

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
			assert.Zero(t, vsp.Rewards.Validator)
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

	var wp *writePool
	wp, err = ssc.getWritePool(client.id, balances)
	require.NoError(t, err)
	var wps = wp.allocUntil(alloc.ID, alloc.Until())

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

		var wp *writePool
		wp, err = ssc.getWritePool(client.id, balances)
		require.NoError(t, err)

		// offer balance, stake pool total balance
		for _, b := range blobs {
			if !isAllocBlobber(b.id, alloc) {
				continue
			}
			var sp *stakePool
			sp, err = ssc.getStakePool(b.id, balances)
			require.NoError(t, err)

			var offer = sp.findOffer(allocID)
			require.NotNil(t, offer)
			require.EqualValues(t, 10e10, stakePoolTotal(sp))
			require.EqualValues(t, 5000000027, offer.Lock)
		}

		// values before
		var (
			wpb = wp.allocUntil(alloc.ID, alloc.Until())
			cpb = cp.Balance
		)

		require.EqualValues(t, wps, wpb+cpb)

		// until the end
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		// load validators
		var validators *ValidatorNodes
		validators, err = ssc.getValidatorsList(balances)
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
					validators.Nodes, alloc.ID, blobber, allocRoot, balances)
			}
		}

		// let expire all the challenges
		tp += int64(toSeconds(avgTerms.ChallengeCompletionTime))

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
		_, err = ssc.cacnelAllocationRequest(tx, mustEncode(t, &req), balances)
		require.NoError(t, err)

		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)

		// check out pools, blobbers, validators balances
		wp, err = ssc.getWritePool(client.id, balances)
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
			require.Nil(t, sp.findOffer(allocID)) // no offers expected
			require.EqualValues(t, 10e10, stakePoolTotal(sp))
		}

		// values before
		var (
			wpa = wp.allocUntil(alloc.ID, alloc.Until())
			cpa = cp.Balance
		)

		require.Zero(t, cpa)
		require.EqualValues(t, wpb, wpa)
		require.EqualValues(t, wps, wp.Pools.gimmeAll())

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
			assert.Zero(t, vsp.Rewards.Validator)
			assert.Zero(t, balances.balances[val.id])
		}

	})

}

// Client cancels a transaction before the blobber has written a
// transaction to the blockchain confirming storage.
//
// The storage SC doesn't care about this confirmation. If a
// blobber chosen, then it should be rewarded by the SC regardless
// any its side confirmation. A blobber can loose it rewards only
// by the challenges mechanism.

// Blobber makes an agreement with itself for a huge amount of
// very cheap storage, in the hopes of starving other blobbers.
func Test_blobber_choose_randomization(t *testing.T) {

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		client   = newClient(10000*x10, balances)
		tp, exp  = int64(0), int64(toSeconds(time.Hour))
		conf     = setConfig(t, balances)

		blobs = make([]*Client, 0, 30)
		err   error
	)

	conf.StakePool.MinLock = 1
	conf.MinAllocSize = 10 * MB
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	require.NoError(t, err)

	// terms, capacity ranges
	//
	// read price      [1; 31]
	// write price     [1; 31]
	// min_lock_demand [0.0; 0.03]
	// capacity        [20 GB; 620 GB]

	var terms = avgTerms      // copy
	terms.ReadPrice = 1       // cheapest greater than zero
	terms.WritePrice = 1      // cheapest greater than zero
	terms.MinLockDemand = 0.0 // no min lock demand
	var bcap int64 = 20 * GB  // capacity, starting from 2 GB

	for i := 0; i < 30; i++ {
		tp += 1
		var b = addBlobber(t, ssc, bcap, tp, terms,
			state.Balance(float64(terms.WritePrice)*sizeInGB(bcap)), balances)
		blobs = append(blobs, b)

		terms.ReadPrice++
		terms.WritePrice++
		terms.MinLockDemand += 0.001
		bcap += 20 * GB
	}

	// add few allocations

	// add allocation without adding new 30 blobbers and without setting
	// configurations
	var addAlloc = func(t *testing.T, ssc *StorageSmartContract, client *Client,
		now, exp int64, balances chainState.StateContextI) (allocID string) {

		var nar = new(newAllocationRequest)
		nar.DataShards = 10
		nar.ParityShards = 10
		nar.Expiration = common.Timestamp(exp)
		nar.Owner = client.id
		nar.OwnerPublicKey = client.pk
		nar.ReadPriceRange = PriceRange{0, 10 * x10}
		nar.WritePriceRange = PriceRange{0, 20 * x10}
		nar.Size = 100 * MB // 100 MB
		nar.MaxChallengeCompletionTime = 200 * time.Hour

		var resp, err = nar.callNewAllocReq(t, client.id, 15*x10, ssc, now,
			balances)
		require.NoError(t, err)

		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		return deco.ID
	}

	// sort blobs, since all blobbers list is sorted
	sort.Slice(blobs, func(i, j int) bool {
		return blobs[i].id < blobs[j].id
	})

	const n = 10 + 10 // n is blobbers required for an allocation (data+parity)

	for i := 0; i < 100; i++ {
		tp += 1

		var (
			allocID = addAlloc(t, ssc, client, tp, tp+exp, balances)

			seed     int64
			rnd      *rand.Rand
			expected []string
		)

		// just make sure that blobbers selected pseudo-random transaction
		// hash-based, regardless a price or size
		seed, err = strconv.ParseInt(allocID[0:8], 16, 64)
		require.NoError(t, err)
		rnd = rand.New(rand.NewSource(seed))
	Outer:
		for i := 0; len(expected) < n; i++ {
			var x = rnd.Intn(len(blobs))
			for _, id := range expected {
				if blobs[x].id == id {
					continue Outer // already have the blobber in the list
				}
			}
			expected = append(expected, blobs[x].id)
		}

		var (
			alloc *StorageAllocation
			got   []string
		)
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		for _, d := range alloc.BlobberDetails {
			got = append(got, d.BlobberID)
		}

		require.Equal(t, expected, got)
	}

}

func zcnToBalance(token float64) state.Balance {
	return state.Balance(token * float64(x10))
}

func attachBlobbersAndNewAllocation(t *testing.T, terms Terms, aRequest newAllocationRequest, capacity int64,
	clientBalance, blobberBalance, value state.Balance, numBlobbers int,
) (ssc *StorageSmartContract, ctx *testBalances,
	blobbers []*Client, now int64, allocationId string, client *Client, testBlobber *Client,
) {
	ssc = newTestStorageSC()
	ctx = newTestBalances(t, false)
	_ = setConfig(t, ctx)
	client = newClient(clientBalance, ctx)
	now += 100
	for i := 0; i < numBlobbers; i++ {
		var blobber = addBlobber(t, ssc, capacity, now, terms, blobberBalance, ctx)
		blobbers = append(blobbers, blobber)
	}

	now += 100

	aRequest.Owner = client.id
	aRequest.OwnerPublicKey = client.pk

	resp, err := aRequest.callNewAllocReq(t, client.id, value, ssc, now, ctx)
	require.NoError(t, err)
	var decodeResp StorageAllocation
	require.NoError(t, decodeResp.Decode([]byte(resp)))
	allocationId = decodeResp.ID

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
