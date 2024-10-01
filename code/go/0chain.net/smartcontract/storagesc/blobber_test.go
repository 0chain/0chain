package storagesc

import (
	"0chain.net/smartcontract/dbs/event"
	"encoding/hex"
	"fmt"
	"github.com/minio/sha256-simd"
	"math"
	"testing"
	"time"

	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/util"

	"0chain.net/core/common"
	"0chain.net/core/encryption"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateBlobberSettings(t *testing.T) {
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)

		tp int64 = 100

		updateWritePrice    = 1e10
		updateServiceCharge = 0.1
		updateReadPrice     = 1e10
		updateNumDelegates  = 10
		updateCapacity      = 10 * GB
		url                 = "https://new-base-url.com"
	)
	setConfig(t, balances)
	var (
		blob   = addBlobber(t, ssc, 2*GB, tp, avgTerms, 50*x10, balances, false, false)
		b, err = ssc.getBlobber(blob.id, balances)
	)
	require.NoError(t, err)

	// Update write price
	b.mustUpdateBase(func(b *storageNodeBase) error {
		b.Terms.WritePrice += currency.Coin(updateWritePrice)
		updateWritePrice = float64(b.Terms.WritePrice)
		return nil
	})
	tp += 100
	_, err = updateBlobber(t, b, 0, tp, ssc, balances)
	require.NoError(t, err)

	b, err = ssc.getBlobber(blob.id, balances)
	require.NoError(t, err)
	require.Equal(t, updateWritePrice, float64(b.mustBase().Terms.WritePrice))

	// Update service charge
	b.mustUpdateBase(func(b *storageNodeBase) error {
		b.StakePoolSettings.ServiceChargeRatio += updateServiceCharge
		updateServiceCharge = b.StakePoolSettings.ServiceChargeRatio
		return nil
	})
	tp += 100
	_, err = updateBlobber(t, b, 0, tp, ssc, balances)
	require.NoError(t, err)

	b, err = ssc.getBlobber(blob.id, balances)
	require.NoError(t, err)
	require.Equal(t, updateServiceCharge, b.mustBase().StakePoolSettings.ServiceChargeRatio)

	// Update read price
	b.mustUpdateBase(func(b *storageNodeBase) error {
		b.Terms.ReadPrice += currency.Coin(updateReadPrice)
		updateReadPrice = float64(b.Terms.ReadPrice)
		return nil
	})
	tp += 100
	_, err = updateBlobber(t, b, 0, tp, ssc, balances)
	require.NoError(t, err)

	b, err = ssc.getBlobber(blob.id, balances)
	require.NoError(t, err)
	require.Equal(t, updateReadPrice, float64(b.mustBase().Terms.ReadPrice))

	// Update number of delegates
	b.mustUpdateBase(func(b *storageNodeBase) error {
		b.StakePoolSettings.MaxNumDelegates += updateNumDelegates
		updateNumDelegates = b.StakePoolSettings.MaxNumDelegates
		return nil
	})
	tp += 100
	_, err = updateBlobber(t, b, 0, tp, ssc, balances)
	require.NoError(t, err)

	b, err = ssc.getBlobber(blob.id, balances)
	require.NoError(t, err)
	require.Equal(t, updateNumDelegates, b.mustBase().StakePoolSettings.MaxNumDelegates)

	// Update capacity
	b.mustUpdateBase(func(b *storageNodeBase) error {
		b.Capacity = int64(updateCapacity)
		updateCapacity = int(b.Capacity)
		return nil
	})
	tp += 100
	_, err = updateBlobber(t, b, 0, tp, ssc, balances)
	require.NoError(t, err)

	b, err = ssc.getBlobber(blob.id, balances)
	require.NoError(t, err)
	require.Equal(t, int64(updateCapacity), b.mustBase().Capacity)

	// Update not available
	b.mustUpdateBase(func(b *storageNodeBase) error {
		b.NotAvailable = true
		return nil
	})
	tp += 100
	_, err = updateBlobber(t, b, 0, tp, ssc, balances)
	require.NoError(t, err)

	b, err = ssc.getBlobber(blob.id, balances)
	require.NoError(t, err)
	require.Equal(t, true, b.mustBase().NotAvailable)

	// Update URL
	b.mustUpdateBase(func(b *storageNodeBase) error {
		b.BaseURL = url
		return nil
	})
	tp += 100
	_, err = updateBlobber(t, b, 0, tp, ssc, balances)
	require.NoError(t, err)

	b, err = ssc.getBlobber(blob.id, balances)
	require.NoError(t, err)
	require.Equal(t, url, b.mustBase().BaseURL)
}

func TestAddBlobber(t *testing.T) {
	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(t, false)
		tp       int64 = 100
		err      error
	)

	setConfig(t, balances)

	t.Run("Register normal blobber", func(t *testing.T) {
		var blob = newClient(0, balances)
		blob.terms = avgTerms
		blob.cap = 2 * GB

		_, err = blob.callAddBlobber(t, ssc, tp, balances)
		require.NoError(t, err)

		blobber, err := getBlobber(blob.id, balances)
		require.NoError(t, err)
		require.NotNil(t, blobber)

		require.Equal(t, avgTerms.WritePrice, blobber.mustBase().Terms.WritePrice)
		require.Equal(t, avgTerms.ReadPrice, blobber.mustBase().Terms.ReadPrice)
		require.Equal(t, blob.cap, blobber.mustBase().Capacity)
		if v2, ok := blobber.Entity().(*storageNodeV2); ok && v2.IsRestricted != nil {
			require.Equal(t, false, *v2.IsRestricted)
		}
	})

	t.Run("Register restricted blobber", func(t *testing.T) {
		var blob = newClient(0, balances)
		blob.terms = avgTerms
		blob.cap = 2 * GB
		blob.isRestricted = true

		_, err = blob.callAddBlobber(t, ssc, tp, balances)
		require.NoError(t, err)

		blobber, err := getBlobber(blob.id, balances)
		require.NoError(t, err)
		require.NotNil(t, blobber)

		require.Equal(t, avgTerms.WritePrice, blobber.mustBase().Terms.WritePrice)
		require.Equal(t, avgTerms.ReadPrice, blobber.mustBase().Terms.ReadPrice)
		require.Equal(t, blob.cap, blobber.mustBase().Capacity)

		if v2, ok := blobber.Entity().(*storageNodeV2); ok && v2.IsRestricted != nil {
			require.Equal(t, true, *v2.IsRestricted)
		}
	})

	t.Run("Register Enterprise blobber", func(t *testing.T) {
		var blob = newClient(0, balances)
		blob.terms = avgTerms
		blob.cap = 2 * GB
		blob.isEnterprise = true

		_, err = blob.callAddBlobber(t, ssc, tp, balances)
		require.NoError(t, err)

		blobber, err := getBlobber(blob.id, balances)
		require.NoError(t, err)
		require.NotNil(t, blobber)

		require.Equal(t, avgTerms.WritePrice, blobber.mustBase().Terms.WritePrice)
		require.Equal(t, avgTerms.ReadPrice, blobber.mustBase().Terms.ReadPrice)
		require.Equal(t, blob.cap, blobber.mustBase().Capacity)
		if v2, ok := blobber.Entity().(*storageNodeV2); ok && v2.IsRestricted != nil {
			require.Equal(t, false, *v2.IsRestricted)
		}

		blobberV4 := blobber.Entity().(*storageNodeV4)
		require.Equal(t, true, *blobberV4.IsEnterprise)
		require.Equal(t, false, *blobberV4.IsRestricted)
	})
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
	require.Error(t, err)
	require.EqualError(t, err, fmt.Sprintf("add_or_update_blobber_failed: blobber already exists,with id: %s ", blob.id))

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
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		client   = newClient(2000*x10, balances)
		tp       = int64(0)

		// no owner
		reader = newClient(100*x10, balances)
		err    error
	)

	conf := setConfig(t, balances)

	tp += 100
	var allocID, blobs = addAllocation(t, ssc, client, tp, 0, 0, 0, 0, 0, balances, false, false, false)

	// blobbers: stake 10k, balance 40k

	sa, err := ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	var b1 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[0].BlobberID {
			b1 = b
			break
		}
	}
	require.NotNil(t, b1)

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
			TargetId: client.id,
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
			TargetId: reader.id,
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

	initialWriteMarkerSavedData := int64(0)
	endWriteMarkerSavedData := int64(0)

	t.Run("write", func(t *testing.T) {

		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var apb, cpb = alloc.WritePool, cp.Balance
		require.EqualValues(t, currency.Coin(1000*x10), apb)
		require.EqualValues(t, currency.Coin(0), cpb)

		tp += 100
		var cc = &BlobberCloseConnection{
			AllocationRoot:     "root-1",
			PrevAllocationRoot: "",
			WriteMarker:        &WriteMarker{},
		}
		wm1 := &writeMarkerV1{
			AllocationRoot:         "root-1",
			PreviousAllocationRoot: "",
			AllocationID:           allocID,
			Size:                   100 * 1024 * 1024, // 100 MB
			BlobberID:              b2.id,
			Timestamp:              common.Timestamp(tp),
			ClientID:               client.id,
		}
		wm1.Signature, err = client.scheme.Sign(
			encryption.Hash(wm1.GetHashData()))
		require.NoError(t, err)
		cc.WriteMarker.SetEntity(wm1)

		blobBeforeWrite, err := ssc.getBlobber(b2.id, balances)
		blobBeforeWriteBase := blobBeforeWrite.mustBase()
		savedDataBeforeUpdate := blobBeforeWriteBase.SavedData
		require.EqualValues(t, initialWriteMarkerSavedData, savedDataBeforeUpdate)
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

		blobAfterWrite, err := ssc.getBlobber(b2.id, balances)
		blobAfterWriteBase := blobAfterWrite.mustBase()
		endWriteMarkerSavedData = wm1.Size - initialWriteMarkerSavedData
		require.EqualValues(t, endWriteMarkerSavedData, blobAfterWriteBase.SavedData)

		size := (int64(math.Ceil(float64(wm1.Size) / CHUNK_SIZE))) * CHUNK_SIZE
		rdtu, err := alloc.restDurationInTimeUnits(wm1.Timestamp, conf.TimeUnit)
		require.NoError(t, err)

		var moved = int64(sizeInGB(size) * float64(avgTerms.WritePrice) * rdtu)

		require.EqualValues(t, moved, cp.Balance)
	})

	t.Run("delete", func(t *testing.T) {
		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var wpb, cpb = alloc.WritePool, cp.Balance
		require.EqualValues(t, currency.Coin(10000000000000), wpb)
		require.EqualValues(t, currency.Coin(4881117078), cpb)

		tp += 100
		var cc = &BlobberCloseConnection{
			AllocationRoot:     "root-2",
			PrevAllocationRoot: "root-1",
			WriteMarker:        &WriteMarker{},
		}
		wm1 := &writeMarkerV1{
			AllocationRoot:         "root-2",
			PreviousAllocationRoot: "root-1",
			AllocationID:           allocID,
			Size:                   -50 * 1024 * 1024, // 50 MB
			BlobberID:              b2.id,
			Timestamp:              common.Timestamp(tp),
			ClientID:               client.id,
		}
		wm1.Signature, err = client.scheme.Sign(
			encryption.Hash(wm1.GetHashData()))
		require.NoError(t, err)
		cc.WriteMarker.SetEntity(wm1)

		blobBeforeWrite, err := ssc.getBlobber(b2.id, balances)
		blobBeforeWriteBase := blobBeforeWrite.mustBase()
		require.EqualValues(t, endWriteMarkerSavedData, blobBeforeWriteBase.SavedData)
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

		blobAfterWrite, err := ssc.getBlobber(b2.id, balances)
		blobAfterWriteBase := blobAfterWrite.mustBase()
		// asserting by dividing `endWriteMarkerSavedData` since write marker value would half after delete
		require.EqualValues(t, endWriteMarkerSavedData/2, blobAfterWriteBase.SavedData)

		require.EqualValues(t, currency.Coin(2440746919), cp.Balance)

	})

	var b3 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[2].BlobberID {
			b3 = b
			break
		}
	}
	require.NotNil(t, b3)

	t.Run("write less than 64 KB", func(t *testing.T) {
		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var blobb1 = balances.balances[b3.id]
		var wpb1, cpb1 = alloc.WritePool, cp.Balance

		wpb1i, err2 := wpb1.Int64()
		if err2 != nil {
			t.Error(err2)
		}
		cpb1i, err2 := cpb1.Int64()
		if err2 != nil {
			t.Error(err2)
		}
		require.EqualValues(t, currency.Coin(10000000000000), wpb1i)
		require.EqualValues(t, currency.Coin(2440746919), cpb1i)
		require.EqualValues(t, currency.Coin(40*x10), blobb1)

		// write 10 KB
		tp = 200
		var cc = &BlobberCloseConnection{
			AllocationRoot:     "alloc-root-1",
			PrevAllocationRoot: "",
			WriteMarker:        &WriteMarker{},
		}
		wm1 := &writeMarkerV1{
			AllocationRoot:         "alloc-root-1",
			PreviousAllocationRoot: "",
			AllocationID:           allocID,
			Size:                   10 * KB,
			BlobberID:              b3.id,
			Timestamp:              common.Timestamp(tp),
			ClientID:               client.id,
		}
		wm1.Signature, err = client.scheme.Sign(
			encryption.Hash(wm1.GetHashData()))
		require.NoError(t, err)
		cc.WriteMarker.SetEntity(wm1)

		blobBeforeWrite, err := ssc.getBlobber(b3.id, balances)
		blobBeforeWriteBase := blobBeforeWrite.mustBase()
		require.EqualValues(t, initialWriteMarkerSavedData, blobBeforeWriteBase.SavedData)
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

		apb2i, err2 := apb2.Int64()
		if err2 != nil {
			t.Error(err2)
		}
		cpb2i, err2 := cpb2.Int64()
		if err2 != nil {
			t.Error(err2)
		}
		blobAfterWrite, err := ssc.getBlobber(b3.id, balances)
		blobAfterWriteBase := blobAfterWrite.mustBase()
		ccWMBase := cc.WriteMarker.mustBase()
		endWriteMarkerSavedData = ccWMBase.Size - initialWriteMarkerSavedData
		require.EqualValues(t, endWriteMarkerSavedData, blobAfterWriteBase.SavedData)

		require.EqualValues(t, currency.Coin(10000000000000), apb2i)
		require.EqualValues(t, currency.Coin(2443798559), cpb2i)

		require.EqualValues(t, currency.Coin(40*x10), blobb2)

		sa, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		alloc = sa.mustBase()
	})

	t.Run("delete less than 64 KB", func(t *testing.T) {
		var cp *challengePool
		cp, err = ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var blobb1 = balances.balances[b3.id]
		var wpb1, cpb1 = alloc.WritePool, cp.Balance

		wpb1i, err2 := wpb1.Int64()
		if err2 != nil {
			t.Error(err2)
		}
		cpb1i, err2 := cpb1.Int64()
		if err2 != nil {
			t.Error(err2)
		}
		require.EqualValues(t, 9997556201441, wpb1i)
		require.EqualValues(t, 2443798559, cpb1i)
		require.EqualValues(t, 40*x10, blobb1)

		// delete 10 KB
		tp += 100
		var cc = &BlobberCloseConnection{
			AllocationRoot:     "alloc-root-2",
			PrevAllocationRoot: "alloc-root-1",
			WriteMarker:        &WriteMarker{},
		}
		wm1 := &writeMarkerV1{
			AllocationRoot:         "alloc-root-2",
			PreviousAllocationRoot: "alloc-root-1",
			AllocationID:           allocID,
			Size:                   -10 * KB,
			BlobberID:              b3.id,
			Timestamp:              common.Timestamp(tp),
			ClientID:               client.id,
		}
		wm1.Signature, err = client.scheme.Sign(
			encryption.Hash(wm1.GetHashData()))
		require.NoError(t, err)
		cc.WriteMarker.SetEntity(wm1)

		blobBeforeWrite, err := ssc.getBlobber(b3.id, balances)
		blobBeforeWriteBase := blobBeforeWrite.mustBase()
		require.EqualValues(t, endWriteMarkerSavedData, blobBeforeWriteBase.SavedData)
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

		apb2i, err2 := apb2.Int64()
		if err2 != nil {
			t.Error(err2)
		}
		cpb2i, err2 := cpb2.Int64()
		if err2 != nil {
			t.Error(err2)
		}
		blobAfterWrite, err := ssc.getBlobber(b3.id, balances)
		blobAfterWriteBase := blobAfterWrite.mustBase()
		require.EqualValues(t, initialWriteMarkerSavedData, blobAfterWriteBase.SavedData)
		require.EqualValues(t, 9997556201441, apb2i)
		require.EqualValues(t, 2440747155, cpb2i)
		require.EqualValues(t, 40*x10, blobb2)

		sa, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		alloc = sa.mustBase()
	})
}

func inspectCPIV(t *testing.T, ssc *StorageSmartContract, allocID string, balances *testBalances) {

	t.Helper()

	var _, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
}

// challenge failed
func Test_flow_penalty(t *testing.T) {
	t.Skip("rewrite this tests")
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		client   = newClient(2000*x10, balances)
		tp       = int64(0)

		err error
	)

	setConfig(t, balances)

	tp += 100
	var allocID, blobs = addAllocation(t, ssc, client, tp, 0, 0, 0, 0, 0, balances, false, false, false)

	// blobbers: stake 10k, balance 40k

	sa, err := ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	var b1 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[0].BlobberID {
			b1 = b
			break
		}
	}
	require.NotNil(t, b1)

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
			WriteMarker:        &WriteMarker{},
		}
		wm1 := &writeMarkerV1{
			AllocationRoot:         allocRoot,
			PreviousAllocationRoot: "",
			AllocationID:           allocID,
			Size:                   100 * 1024 * 1024, // 100 MB
			BlobberID:              b4.id,
			Timestamp:              common.Timestamp(tp),
			ClientID:               client.id,
		}
		wm1.Signature, err = client.scheme.Sign(
			encryption.Hash(wm1.GetHashData()))
		require.NoError(t, err)
		cc.WriteMarker.SetEntity(wm1)

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
		_, err = ssc.getStakePool(spenum.Blobber, b4.id, balances)
		require.NoError(t, err)

		// until the end
		sa, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		alloc = sa.mustBase()

		// load validators
		validators, err := getValidatorsList(balances)
		require.NoError(t, err)

		// load blobber
		var blobber *StorageNode
		blobber, err = ssc.getBlobber(b4.id, balances)
		require.NoError(t, err)

		//
		var (
			step    = (int64(alloc.Expiration) - tp) / 10
			challID string

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

			currentRound := balances.GetBlock().Round
			genChall(t, ssc, tp, currentRound-200*(i-2), challID, i, validators, alloc.ID, blobber, balances)

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
			//sp, err = ssc.getStakePool(spenum.Blobber, b4.id, balances)
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
			//	_, err = ssc.getStakePool(spenum.Blobber, val.id, balances)
			//	require.NoError(t, err)
			//}
			//
			//// next stage
			//prevID = challID
		}

	})

}

func isAllocBlobber(id string, alloc *storageAllocationBase) bool {
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

	_, err = balances.InsertTrieNode(scConfigKey(ADDRESS), conf)
	require.NoError(t, err)

	tp += 100
	var allocID, blobs = addAllocation(t, ssc, client, tp, 0, 0, 0, 0, 0, balances, false, false, false)

	sa, err := ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

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
				WriteMarker:        &WriteMarker{},
			}
			wm1 := &writeMarkerV1{
				AllocationRoot:         allocRoot,
				PreviousAllocationRoot: "",
				AllocationID:           allocID,
				Size:                   100 * 1024 * 1024, // 100 MB
				BlobberID:              b.id,
				Timestamp:              common.Timestamp(tp),
				ClientID:               client.id,
			}
			wm1.Signature, err = client.scheme.Sign(
				encryption.Hash(wm1.GetHashData()))
			require.NoError(t, err)
			cc.WriteMarker.SetEntity(wm1)
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

		require.NoError(t, err)
		require.EqualValues(t, wps, wpb+cpb)

		// until the end
		sa, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		alloc = sa.mustBase()

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

				var challID string
				challID = fmt.Sprintf("chall-%s-%d", b.id, i)
				currentRound := balances.GetBlock().Round
				genChall(t, ssc, tp, currentRound-100, challID, 0, validators, alloc.ID, blobber, balances)
				gfc++
			}
		}

		// let expire all the challenges
		balances.block.Round += int64(MaxChallengeCompletionRounds)
		tp += 180

		// add open challenges to allocation stats
		sa, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		alloc = sa.mustBase()

		if alloc.Stats == nil {
			alloc.Stats = new(StorageAllocationStats)
		}
		alloc.Stats.OpenChallenges = 50 // just a non-zero number
		_, err = balances.InsertTrieNode(sa.GetKey(ssc.ID), sa)
		require.NoError(t, err)

		tp += exp // expire the allocation

		var req lockRequest
		req.AllocationID = allocID

		var tx = newTransaction(client.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		_, err = ssc.finalizeAllocation(tx, mustEncode(t, &req), balances)
		require.NoError(t, err)

		sa, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		alloc = sa.mustBase()

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
			sp, err = ssc.getStakePool(spenum.Blobber, b.id, balances)
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

		require.NoError(t, err)
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
			vsp, err = ssc.getStakePool(spenum.Blobber, val.id, balances)
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
		client   = newClient(1000*x10, balances)
		tp       = int64(0)
		conf     = setConfig(t, balances)

		err error
	)

	balances.block.Round = 100000

	_, err = balances.InsertTrieNode(scConfigKey(ADDRESS), conf)
	require.NoError(t, err)

	tp += 100
	var allocID, blobs = addAllocation(t, ssc, client, tp, 0, 0, 0, 0, 0, balances, false, false, false)

	sa, err := ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	for _, ba := range alloc.BlobberAllocs {
		ba.LatestFinalizedChallCreatedAt = 0
		ba.ChallengePoolIntegralValue = 0
	}

	sa.mustUpdateBase(func(base *storageAllocationBase) error {
		alloc.deepCopy(base)
		return nil
	})

	_, err = balances.InsertTrieNode(sa.GetKey(ADDRESS), sa)
	if err != nil {
		return
	}

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
				WriteMarker:        &WriteMarker{},
			}
			wm1 := &writeMarkerV1{
				AllocationRoot:         allocRoot,
				PreviousAllocationRoot: "",
				AllocationID:           allocID,
				Size:                   100 * 1024 * 1024, // 100 MB
				BlobberID:              b.id,
				Timestamp:              alloc.StartTime,
				ClientID:               client.id,
			}
			wm1.Signature, err = client.scheme.Sign(
				encryption.Hash(wm1.GetHashData()))
			require.NoError(t, err)
			cc.WriteMarker.SetEntity(wm1)
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
			sp, err = ssc.getStakePool(spenum.Blobber, b.id, balances)
			require.NoError(t, err)
			spTotal, err := stakePoolTotal(sp)
			require.NoError(t, err)
			require.EqualValues(t, 10e10, spTotal)
		}

		afterSA, err := ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		afterAlloc := afterSA.mustBase()

		require.EqualValues(t, wps, afterAlloc.WritePool+cp.Balance)

		// until the end
		sa, err = ssc.getAllocation(allocID, balances)
		require.NoError(t, err)
		alloc = sa.mustBase()

		// load validators
		validators, err := getValidatorsList(balances)
		require.NoError(t, err)

		// ---------------

		tp += 10

		//generate challenges leaving them without a response
		for i := int64(0); i < 10; i++ {
			for _, b := range blobs {
				if !isAllocBlobber(b.id, alloc) {
					continue
				}
				// load blobber
				var blobber *StorageNode
				blobber, err = ssc.getBlobber(b.id, balances)
				require.NoError(t, err)

				var challID string
				challID = fmt.Sprintf("chall-%s-%d", b.id, i)
				currentRound := balances.GetBlock().Round
				genChall(t, ssc, tp, currentRound-10000+i, challID, i, validators, alloc.ID, blobber, balances)
			}
		}

		// let expire all the challenges
		balances.block.Round += int64(MaxChallengeCompletionRounds)
		tp += 180

		tp += 10 // a not expired allocation to cancel

		var req lockRequest
		req.AllocationID = allocID

		var tx = newTransaction(client.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		_, err = ssc.cancelAllocationRequest(tx, mustEncode(t, &req), balances)
		require.NoError(t, err)

		_, err = ssc.getAllocation(allocID, balances)
		require.Error(t, util.ErrValueNotPresent, err)

		// challenge pool should be empty
		_, err = ssc.getChallengePool(allocID, balances)
		require.Error(t, err, "challenge pool should be deleted")

		// offer balance, stake pool total balance
		for _, b := range blobs {
			if !isAllocBlobber(b.id, alloc) {
				continue
			}
			var sp *stakePool
			sp, err = ssc.getStakePool(spenum.Blobber, b.id, balances)
			require.NoError(t, err)
			spTotal, err := stakePoolTotal(sp)
			require.NoError(t, err)
			require.EqualValues(t, 10e10, float64(spTotal))
		}

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
			vsp, err = ssc.getStakePool(spenum.Validator, val.id, balances)
			require.NoError(t, err)
			assert.Zero(t, vsp.Reward)
			assert.Zero(t, balances.balances[val.id])
		}

	})

}

func TestBlobberHealthCheck(t *testing.T) {

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)

		tp int64 = 100
	)

	setConfig(t, balances)

	var (
		blob   = addBlobber(t, ssc, 2*GB, tp, avgTerms, 50*x10, balances, false, false)
		b, err = ssc.getBlobber(blob.id, balances)
	)
	require.NoError(t, err)

	// check health
	_, err = healthCheckBlobber(t, b, 0, tp, ssc, balances)
	require.NoError(t, err)

}

func TestOnlyAdd(t *testing.T) {

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)

		tp int64 = 100
	)

	setConfig(t, balances)

	var (
		blob   = addBlobber(t, ssc, 2*GB, tp, avgTerms, 50*x10, balances, false, false)
		b, err = ssc.getBlobber(blob.id, balances)
	)
	require.NoError(t, err)

	b.mustUpdateBase(func(b *storageNodeBase) error {
		b.BaseURL = "https://newabcurl.com"
		return nil
	})

	//should fail as only add is allowed
	_, err = updateBlobberUsingAddBlobber(t, b, 0, tp, ssc, balances)
	require.Error(t, err)

}

type LoopRequest struct {
	TestingIterations            int
	IterationsStartIndex         int
	Tp                           *int64
	Ssc                          *StorageSmartContract
	Balances                     *testBalances
	ChainSize                    *int64
	ChainData                    *[]byte
	Client                       *Client
	Conf                         *Config
	Blobber                      *Client
	InitialWriteMarkerSavedData  *int64
	EndWriteMarkerSavedData      *int64
	EndWriteMarkerAllocSavedData *int64
	MovedBalance                 *currency.Coin
	PrevAllocRoot                *string
	AllocId                      *string
	PrevHash                     *string
	WmSize                       []int64
	AllocatedRootsArray          *[]string
	PrevAllocatedRootsArray      *[]string
	WmSizeAllocatedArray         *[]int64
	IsRollbackRequest            bool
	NumberOfWrites               *int64
}

func (lr *LoopRequest) createChainData(t *testing.T, iterIndx int, allocationRoot string) (chainHash string, err error) {
	// chainData
	byteSlice := make([]byte, 32)
	for i := 0; i < len(byteSlice); i++ {
		byteSlice[i] = byte(i + 1 + iterIndx) // Example: fill with increasing byte values starting from 1+wmIndx
	}
	*lr.ChainData = append(*lr.ChainData, byteSlice...)

	// chainhash
	hasher := sha256.New()
	if iterIndx != 1 {
		prevChainHash, _ := hex.DecodeString(*lr.PrevHash)
		hasher.Write(prevChainHash)
	}
	for i := 0; i < len(*lr.ChainData); i += 32 {
		hasher.Write((*lr.ChainData)[i : i+32]) //nolint:errcheck
		sum := hasher.Sum(nil)
		hasher.Reset()
		hasher.Write(sum) //nolint:errcheck
	}
	allocRootBytes, err := hex.DecodeString(allocationRoot)
	require.NoError(t, err)

	hasher.Write(allocRootBytes)
	chainHash = hex.EncodeToString(hasher.Sum(nil))
	return
}

func (lr *LoopRequest) createBlobberCloseConnection(t *testing.T, allocationRoot, chainHash string, wmSize int64) (*BlobberCloseConnection, *writeMarkerV2) {

	var cc = &BlobberCloseConnection{
		AllocationRoot:     allocationRoot,
		PrevAllocationRoot: *lr.PrevAllocRoot,
		WriteMarker:        &WriteMarker{},
		ChainData:          *lr.ChainData,
	}
	wm := &writeMarkerV2{
		Version:                writeMarkerV2Version,
		AllocationRoot:         allocationRoot,
		PreviousAllocationRoot: *lr.PrevAllocRoot,
		FileMetaRoot:           "",
		AllocationID:           *lr.AllocId,
		Size:                   wmSize,
		ChainSize:              *lr.ChainSize,
		ChainHash:              chainHash,
		BlobberID:              lr.Blobber.id,
		Timestamp:              common.Timestamp(*lr.Tp),
		ClientID:               lr.Client.id,
	}

	var err error
	wm.Signature, err = lr.Client.scheme.Sign(encryption.Hash(wm.GetHashData()))
	require.NoError(t, err)
	cc.WriteMarker.SetEntity(wm)
	return cc, wm
}

func (lr *LoopRequest) checkDataPostCommit(t *testing.T, wm *writeMarkerV2) {

	// check out
	cp, err := lr.Ssc.getChallengePool(*lr.AllocId, lr.Balances)
	require.NoError(t, err)

	sa, err := lr.Ssc.getAllocation(*lr.AllocId, lr.Balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	blobAfterWrite, err := lr.Ssc.getBlobber(lr.Blobber.id, lr.Balances)
	blobAfterWriteBase := blobAfterWrite.mustBase()

	//adding current WM size data
	*lr.EndWriteMarkerSavedData = wm.Size + *lr.InitialWriteMarkerSavedData
	*lr.InitialWriteMarkerSavedData = *lr.EndWriteMarkerSavedData // for next iteration of new wm, initial saved data is not 0
	require.EqualValues(t, *lr.EndWriteMarkerSavedData, blobAfterWriteBase.SavedData)

	size := (int64(math.Ceil(float64(wm.Size) / CHUNK_SIZE))) * CHUNK_SIZE
	rdtu, err := alloc.restDurationInTimeUnits(wm.Timestamp, lr.Conf.TimeUnit)
	require.NoError(t, err)

	var moved = int64(sizeInGB(size) * float64(avgTerms.WritePrice) * rdtu)
	*lr.MovedBalance += currency.Coin(moved)
	require.EqualValues(t, *lr.MovedBalance, cp.Balance)

	require.EqualValues(t, alloc.Stats.NumWrites, *lr.NumberOfWrites)
	require.EqualValues(t, alloc.Stats.NumReads, 0)

	*lr.EndWriteMarkerAllocSavedData += int64(float64(wm.Size) * float64(alloc.DataShards) / float64(alloc.DataShards+alloc.ParityShards))
	require.EqualValues(t, alloc.Stats.UsedSize, *lr.EndWriteMarkerAllocSavedData)
}

func (lr *LoopRequest) checkEmitEvents(t *testing.T, wm *writeMarkerV2) {
	events := lr.Balances.GetEvents()
	require.EqualValues(t, 5, len(events))
	sa, err := lr.Ssc.getAllocation(*lr.AllocId, lr.Balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	requiredEventTags := []event.EventTag{event.TagToChallengePool, event.TagAddOrUpdateChallengePool, event.TagAddWriteMarker, event.TagUpdateAllocationStat, event.TagUpdateBlobberStat, event.TagFromChallengePool}
	for i, evnt := range events {
		require.Contains(t, requiredEventTags, evnt.Tag)
		if evnt.Tag == event.TagAddWriteMarker {
			myInst := *events[i].Data.(*event.WriteMarker)
			require.EqualValues(t, wm.Size, myInst.Size)
			require.EqualValues(t, wm.AllocationRoot, myInst.AllocationRoot)
			require.EqualValues(t, wm.Signature, myInst.Signature)
		} else if evnt.Tag == event.TagUpdateAllocationStat {
			myInst := *events[i].Data.(*event.Allocation)
			require.EqualValues(t, *lr.EndWriteMarkerAllocSavedData, myInst.UsedSize)
			require.EqualValues(t, *lr.NumberOfWrites, myInst.NumWrites)
			require.EqualValues(t, alloc.MovedToChallenge, myInst.MovedToChallenge)
			require.EqualValues(t, alloc.MovedBack, myInst.MovedBack)
			require.EqualValues(t, alloc.WritePool, myInst.WritePool)
		} else if evnt.Tag == event.TagUpdateBlobberStat {
			myInst := (events[i].Data).(event.Blobber)
			//changeSize := int64(float64(wm.Size) * float64(alloc.DataShards) / float64(alloc.DataShards+alloc.ParityShards))
			require.EqualValues(t, wm.Size, myInst.SavedData)
		}
	}
}

func (lr *LoopRequest) checkDataPostLoop(t *testing.T) {

	// check out
	cp, err := lr.Ssc.getChallengePool(*lr.AllocId, lr.Balances)
	require.NoError(t, err)

	sa, err := lr.Ssc.getAllocation(*lr.AllocId, lr.Balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	blobAfterWrite, err := lr.Ssc.getBlobber(lr.Blobber.id, lr.Balances)
	blobAfterWriteBase := blobAfterWrite.mustBase()

	require.EqualValues(t, *lr.EndWriteMarkerSavedData, blobAfterWriteBase.SavedData)
	require.EqualValues(t, *lr.MovedBalance, cp.Balance)
	require.EqualValues(t, alloc.Stats.NumWrites, *lr.NumberOfWrites)
	require.EqualValues(t, alloc.Stats.NumReads, 0)
	require.EqualValues(t, alloc.Stats.UsedSize, *lr.EndWriteMarkerAllocSavedData)
}

func (req *LoopRequest) runWmRequestInLoopAndTest(t *testing.T) {

	var (
		rollbackIndx = 1
		wmIndx       int
	)
	for wmIndx = req.IterationsStartIndex; wmIndx < req.IterationsStartIndex+req.TestingIterations; wmIndx++ {

		var (
			allocationRoot string
			wmSize         int64
		)

		if req.IsRollbackRequest {
			//startedIndex will be 11 if earlier 10WMs have been processed
			allocationRoot = (*req.AllocatedRootsArray)[req.IterationsStartIndex-rollbackIndx-1]
			wmSize = -1 * (*req.WmSizeAllocatedArray)[req.IterationsStartIndex-rollbackIndx-1]
			rollbackIndx++
		} else {
			allocationRootString := fmt.Sprintf("root%d", wmIndx)
			allocationRootHex := []byte(allocationRootString)
			allocationRoot = hex.EncodeToString(allocationRootHex)

			*req.AllocatedRootsArray = append(*req.AllocatedRootsArray, allocationRoot)
			*req.PrevAllocatedRootsArray = append(*req.PrevAllocatedRootsArray, *req.PrevAllocRoot)

			wmSize = req.WmSize[wmIndx-req.IterationsStartIndex]
		}
		*req.WmSizeAllocatedArray = append(*req.WmSizeAllocatedArray, wmSize)

		// chainSize
		*req.ChainSize += wmSize
		chainHash, err := req.createChainData(t, wmIndx, allocationRoot)
		//todo error handling
		//fmt.Sprintln(allocationRoot)

		//create BCC
		cc, wm := req.createBlobberCloseConnection(t, allocationRoot, chainHash, wmSize)

		//pre-commit checks
		blobBeforeWrite, err := req.Ssc.getBlobber(req.Blobber.id, req.Balances)
		blobBeforeWriteBase := blobBeforeWrite.mustBase()
		savedDataBeforeUpdate := blobBeforeWriteBase.SavedData
		require.EqualValues(t, *req.InitialWriteMarkerSavedData, savedDataBeforeUpdate)

		// write
		*req.Tp += 100
		var tx = newTransaction(req.Blobber.id, req.Ssc.ID, 0, *req.Tp) // why this value is 0 in previous tests?
		req.Balances.setTransaction(t, tx)
		var resp string
		resp, err = req.Ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
			req.Balances)
		require.NoError(t, err)
		require.NotZero(t, resp)

		//number of total writes to be increased
		*req.NumberOfWrites++

		req.checkDataPostCommit(t, wm)
		req.checkEmitEvents(t, wm)
		req.flushEventsListFromBalances(t)
		*req.PrevHash = chainHash
		*req.PrevAllocRoot = allocationRoot
	}
	req.checkDataPostLoop(t)
}

func (req *LoopRequest) flushEventsListFromBalances(t *testing.T) {
	req.Balances.events = []event.Event{}
	return
}

func TestCommitBlobberConnection(t *testing.T) {

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		client   = newClient(2000*x10, balances)
		tp       = int64(0)
		err      error
	)

	conf := setConfig(t, balances)

	tp += 100

	var allocID, blobs = addAllocation(t, ssc, client, tp, 0, 0, 0, 0, 0, balances, false, false, false)

	// blobbers: stake 10k, balance 40k

	sa, err := ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	var b1 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[0].BlobberID {
			b1 = b
			break
		}
	}
	require.NotNil(t, b1)

	movedBalance := currency.Coin(0)
	totalNumWrites := int64(0)
	endWriteMarkerAllocSavedData := int64(0)

	t.Run("write 10 write-markers in a row", func(t *testing.T) {

		initialWriteMarkerSavedData := int64(0)
		endWriteMarkerSavedData := int64(0)
		prevAllocRoot := ""

		var chainSize int64 = 0
		var chainData []byte
		var wmSize []int64
		var allocationRootArr, prevAllocationRootArr []string
		var wmSizeArr []int64

		cp, err := ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var apb, cpb = alloc.WritePool, cp.Balance
		require.EqualValues(t, currency.Coin(1000*x10), apb)
		require.EqualValues(t, movedBalance, cpb)

		testingIterations := 10
		tp += 100
		var prevHash string

		for i := 0; i < testingIterations; i++ {
			wmSize = append(wmSize, 1024*1024)
		}
		loopRequest := LoopRequest{
			TestingIterations:            testingIterations,
			IterationsStartIndex:         1,
			Tp:                           &tp,
			Ssc:                          ssc,
			Balances:                     balances,
			ChainSize:                    &chainSize,
			ChainData:                    &chainData,
			Client:                       client,
			Conf:                         conf,
			Blobber:                      b1,
			InitialWriteMarkerSavedData:  &initialWriteMarkerSavedData,
			EndWriteMarkerSavedData:      &endWriteMarkerSavedData,
			EndWriteMarkerAllocSavedData: &endWriteMarkerAllocSavedData,
			MovedBalance:                 &movedBalance,
			PrevAllocRoot:                &prevAllocRoot,
			AllocId:                      &allocID,
			PrevHash:                     &prevHash,
			AllocatedRootsArray:          &allocationRootArr,
			PrevAllocatedRootsArray:      &prevAllocationRootArr,
			WmSizeAllocatedArray:         &wmSizeArr,
			WmSize:                       wmSize,
			NumberOfWrites:               &totalNumWrites,
		}
		loopRequest.flushEventsListFromBalances(t)
		loopRequest.runWmRequestInLoopAndTest(t)
	})

	var b2 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[1].BlobberID {
			b2 = b
			break
		}
	}
	require.NotNil(t, b2)

	t.Run("write 10 write-markers then 10 roll back then add Wm", func(t *testing.T) {

		initialWriteMarkerSavedData := int64(0)
		endWriteMarkerSavedData := int64(0)
		prevAllocRoot := ""
		var chainSize int64 = 0
		var chainData []byte
		var wmSize []int64

		cp, err := ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var apb, cpb = alloc.WritePool, cp.Balance
		require.EqualValues(t, currency.Coin(1000*x10), apb)
		require.EqualValues(t, movedBalance, cpb)

		testingIterations := 10
		rollbackTestingIterations := 10
		tp += 100
		var prevHash string
		var allocationRootArr, prevAllocationRootArr []string
		var wmSizeArr []int64

		for i := 0; i < testingIterations; i++ {
			wmSize = append(wmSize, 1024*1024)
		}
		loopRequest := LoopRequest{
			TestingIterations:            testingIterations,
			IterationsStartIndex:         1,
			Tp:                           &tp,
			Ssc:                          ssc,
			Balances:                     balances,
			ChainSize:                    &chainSize,
			ChainData:                    &chainData,
			Client:                       client,
			Conf:                         conf,
			Blobber:                      b2,
			InitialWriteMarkerSavedData:  &initialWriteMarkerSavedData,
			EndWriteMarkerSavedData:      &endWriteMarkerSavedData,
			EndWriteMarkerAllocSavedData: &endWriteMarkerAllocSavedData,
			MovedBalance:                 &movedBalance,
			PrevAllocRoot:                &prevAllocRoot,
			AllocId:                      &allocID,
			PrevHash:                     &prevHash,
			WmSize:                       wmSize,
			AllocatedRootsArray:          &allocationRootArr,
			PrevAllocatedRootsArray:      &prevAllocationRootArr,
			WmSizeAllocatedArray:         &wmSizeArr,
			IsRollbackRequest:            false,
			NumberOfWrites:               &totalNumWrites,
		}
		//10 +ve WMs
		loopRequest.flushEventsListFromBalances(t)
		loopRequest.runWmRequestInLoopAndTest(t)

		//rollback of last 10+ve WMs
		loopRequest.IsRollbackRequest = true
		loopRequest.TestingIterations = rollbackTestingIterations
		loopRequest.IterationsStartIndex = 11
		loopRequest.runWmRequestInLoopAndTest(t)

		//2+ve WMs
		var wmSizeNew []int64
		for i := 0; i < 2; i++ {
			wmSizeNew = append(wmSizeNew, 1024*1024)
		}
		loopRequest.IsRollbackRequest = false
		loopRequest.TestingIterations = 2
		loopRequest.IterationsStartIndex = 21
		loopRequest.WmSize = wmSizeNew
		loopRequest.runWmRequestInLoopAndTest(t)

	})

	var b3 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[2].BlobberID {
			b3 = b
			break
		}
	}
	t.Run("write 10 write-markers then 10 delete of same size", func(t *testing.T) {

		initialWriteMarkerSavedData := int64(0)
		endWriteMarkerSavedData := int64(0)
		prevAllocRoot := ""
		var chainSize int64 = 0
		var chainData []byte
		var wmSize []int64
		var allocationRootArr, prevAllocationRootArr []string
		var wmSizeArr []int64

		cp, err := ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var apb, cpb = alloc.WritePool, cp.Balance
		require.EqualValues(t, currency.Coin(1000*x10), apb)
		require.EqualValues(t, movedBalance, cpb)

		testingIterations := 10
		delTestingIterations := 10
		tp += 100
		var prevHash string

		for i := 0; i < testingIterations; i++ {
			wmSize = append(wmSize, 1024*1024)
		}
		loopRequest := LoopRequest{
			TestingIterations:            testingIterations,
			IterationsStartIndex:         1,
			Tp:                           &tp,
			Ssc:                          ssc,
			Balances:                     balances,
			ChainSize:                    &chainSize,
			ChainData:                    &chainData,
			Client:                       client,
			Conf:                         conf,
			Blobber:                      b3,
			InitialWriteMarkerSavedData:  &initialWriteMarkerSavedData,
			EndWriteMarkerSavedData:      &endWriteMarkerSavedData,
			EndWriteMarkerAllocSavedData: &endWriteMarkerAllocSavedData,
			MovedBalance:                 &movedBalance,
			PrevAllocRoot:                &prevAllocRoot,
			AllocId:                      &allocID,
			PrevHash:                     &prevHash,
			WmSize:                       wmSize,
			AllocatedRootsArray:          &allocationRootArr,
			PrevAllocatedRootsArray:      &prevAllocationRootArr,
			WmSizeAllocatedArray:         &wmSizeArr,
			NumberOfWrites:               &totalNumWrites,
		}

		loopRequest.flushEventsListFromBalances(t)
		loopRequest.runWmRequestInLoopAndTest(t)

		var WmSizeDel []int64
		for i := 0; i < delTestingIterations; i++ {
			WmSizeDel = append(WmSizeDel, -1*1024*1024)
		}

		loopRequest.TestingIterations = delTestingIterations
		loopRequest.IterationsStartIndex = 11
		loopRequest.WmSize = WmSizeDel

		loopRequest.runWmRequestInLoopAndTest(t)
	})

	var b4 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[3].BlobberID {
			b4 = b
			break
		}
	}
	t.Run("write 10 write-markers then last 3 roll back then add some Wms", func(t *testing.T) {

		initialWriteMarkerSavedData := int64(0)
		endWriteMarkerSavedData := int64(0)
		prevAllocRoot := ""
		var chainSize int64 = 0
		var chainData []byte
		var wmSize []int64

		cp, err := ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var apb, cpb = alloc.WritePool, cp.Balance
		require.EqualValues(t, currency.Coin(1000*x10), apb)
		require.EqualValues(t, movedBalance, cpb)

		testingIterations := 10
		rollbackTestingIterations := 3
		tp += 100
		var prevHash string
		var allocationRootArr, prevAllocationRootArr []string
		var wmSizeArr []int64

		for i := 0; i < testingIterations; i++ {
			wmSize = append(wmSize, 1024*1024)
		}
		loopRequest := LoopRequest{
			TestingIterations:            testingIterations,
			IterationsStartIndex:         1,
			Tp:                           &tp,
			Ssc:                          ssc,
			Balances:                     balances,
			ChainSize:                    &chainSize,
			ChainData:                    &chainData,
			Client:                       client,
			Conf:                         conf,
			Blobber:                      b4,
			InitialWriteMarkerSavedData:  &initialWriteMarkerSavedData,
			EndWriteMarkerSavedData:      &endWriteMarkerSavedData,
			EndWriteMarkerAllocSavedData: &endWriteMarkerAllocSavedData,
			MovedBalance:                 &movedBalance,
			PrevAllocRoot:                &prevAllocRoot,
			AllocId:                      &allocID,
			PrevHash:                     &prevHash,
			WmSize:                       wmSize,
			AllocatedRootsArray:          &allocationRootArr,
			PrevAllocatedRootsArray:      &prevAllocationRootArr,
			WmSizeAllocatedArray:         &wmSizeArr,
			IsRollbackRequest:            false,
			NumberOfWrites:               &totalNumWrites,
		}
		loopRequest.flushEventsListFromBalances(t)
		loopRequest.runWmRequestInLoopAndTest(t)

		loopRequest.IsRollbackRequest = true
		loopRequest.TestingIterations = rollbackTestingIterations
		loopRequest.IterationsStartIndex = 11
		loopRequest.runWmRequestInLoopAndTest(t)

		var wmSizeNew []int64
		for i := 0; i < 2; i++ {
			wmSizeNew = append(wmSizeNew, 1024*1024)
		}
		loopRequest.IsRollbackRequest = false
		loopRequest.TestingIterations = 2
		loopRequest.IterationsStartIndex = 14
		loopRequest.WmSize = wmSizeNew
		loopRequest.runWmRequestInLoopAndTest(t)
	})

	var b5 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[4].BlobberID {
			b5 = b
			break
		}
	}
	require.NotNil(t, b1)

	t.Run("write 10 write-markers then 10 delete then 10 rollback of delete", func(t *testing.T) {

		initialWriteMarkerSavedData := int64(0)
		endWriteMarkerSavedData := int64(0)
		prevAllocRoot := ""
		var chainSize int64 = 0
		var chainData []byte
		var wmSize []int64
		var allocationRootArr, prevAllocationRootArr []string
		var wmSizeArr []int64

		cp, err := ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var apb, cpb = alloc.WritePool, cp.Balance
		require.EqualValues(t, currency.Coin(1000*x10), apb)
		require.EqualValues(t, movedBalance, cpb)

		testingIterations := 10
		delTestingIterations := 10
		rollbackTestingIterations := 10
		tp += 100
		var prevHash string

		for i := 0; i < testingIterations; i++ {
			wmSize = append(wmSize, 1024*1024)
		}
		loopRequest := LoopRequest{
			TestingIterations:            testingIterations,
			IterationsStartIndex:         1,
			Tp:                           &tp,
			Ssc:                          ssc,
			Balances:                     balances,
			ChainSize:                    &chainSize,
			ChainData:                    &chainData,
			Client:                       client,
			Conf:                         conf,
			Blobber:                      b5,
			InitialWriteMarkerSavedData:  &initialWriteMarkerSavedData,
			EndWriteMarkerSavedData:      &endWriteMarkerSavedData,
			EndWriteMarkerAllocSavedData: &endWriteMarkerAllocSavedData,
			MovedBalance:                 &movedBalance,
			PrevAllocRoot:                &prevAllocRoot,
			AllocId:                      &allocID,
			PrevHash:                     &prevHash,
			WmSize:                       wmSize,
			AllocatedRootsArray:          &allocationRootArr,
			PrevAllocatedRootsArray:      &prevAllocationRootArr,
			WmSizeAllocatedArray:         &wmSizeArr,
			NumberOfWrites:               &totalNumWrites,
		}
		loopRequest.flushEventsListFromBalances(t)
		//+ve 10 WM
		loopRequest.runWmRequestInLoopAndTest(t)

		//-ve 10 wm, delete
		var WmSizeDel []int64
		for i := 0; i < delTestingIterations; i++ {
			WmSizeDel = append(WmSizeDel, -1*1024*1024)
		}
		loopRequest.TestingIterations = delTestingIterations
		loopRequest.IterationsStartIndex = 11
		loopRequest.WmSize = WmSizeDel
		loopRequest.runWmRequestInLoopAndTest(t)

		loopRequest.IsRollbackRequest = true
		loopRequest.TestingIterations = rollbackTestingIterations
		loopRequest.IterationsStartIndex = 21
		loopRequest.runWmRequestInLoopAndTest(t)

		var WmSizeNew []int64
		for i := 0; i < 2; i++ {
			WmSizeNew = append(WmSizeNew, 1024*1024)
		}
		loopRequest.IsRollbackRequest = false
		loopRequest.TestingIterations = 2
		loopRequest.IterationsStartIndex = 31
		loopRequest.WmSize = WmSizeNew
		loopRequest.runWmRequestInLoopAndTest(t)
	})

	var b6 *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[5].BlobberID {
			b6 = b
			break
		}
	}
	t.Run("write 10 write-markers then 10 delete then 3 rollback of delete", func(t *testing.T) {

		initialWriteMarkerSavedData := int64(0)
		endWriteMarkerSavedData := int64(0)
		prevAllocRoot := ""
		var chainSize int64 = 0
		var chainData []byte
		var wmSize []int64
		var allocationRootArr, prevAllocationRootArr []string
		var wmSizeArr []int64

		cp, err := ssc.getChallengePool(allocID, balances)
		require.NoError(t, err)

		var apb, cpb = alloc.WritePool, cp.Balance
		require.EqualValues(t, currency.Coin(1000*x10), apb)
		require.EqualValues(t, movedBalance, cpb)

		testingIterations := 10
		delTestingIterations := 10
		rollbackTestingIterations := 3
		tp += 100
		var prevHash string

		for i := 0; i < testingIterations; i++ {
			wmSize = append(wmSize, 1024*1024)
		}
		loopRequest := LoopRequest{
			TestingIterations:            testingIterations,
			IterationsStartIndex:         1,
			Tp:                           &tp,
			Ssc:                          ssc,
			Balances:                     balances,
			ChainSize:                    &chainSize,
			ChainData:                    &chainData,
			Client:                       client,
			Conf:                         conf,
			Blobber:                      b6,
			InitialWriteMarkerSavedData:  &initialWriteMarkerSavedData,
			EndWriteMarkerSavedData:      &endWriteMarkerSavedData,
			EndWriteMarkerAllocSavedData: &endWriteMarkerAllocSavedData,
			MovedBalance:                 &movedBalance,
			PrevAllocRoot:                &prevAllocRoot,
			AllocId:                      &allocID,
			PrevHash:                     &prevHash,
			WmSize:                       wmSize,
			AllocatedRootsArray:          &allocationRootArr,
			PrevAllocatedRootsArray:      &prevAllocationRootArr,
			WmSizeAllocatedArray:         &wmSizeArr,
			NumberOfWrites:               &totalNumWrites,
		}
		loopRequest.flushEventsListFromBalances(t)
		//+ve 10 WM
		loopRequest.runWmRequestInLoopAndTest(t)

		//-ve 10 wm, delete
		var WmSizeDel []int64
		for i := 0; i < delTestingIterations; i++ {
			WmSizeDel = append(WmSizeDel, -1*1024*1024)
		}
		loopRequest.TestingIterations = delTestingIterations
		loopRequest.IterationsStartIndex = 11
		loopRequest.WmSize = WmSizeDel
		loopRequest.runWmRequestInLoopAndTest(t)

		loopRequest.IsRollbackRequest = true
		loopRequest.TestingIterations = rollbackTestingIterations
		loopRequest.IterationsStartIndex = 21
		loopRequest.runWmRequestInLoopAndTest(t)

		var WmSizeNew []int64
		for i := 0; i < 2; i++ {
			WmSizeNew = append(WmSizeNew, 1024*1024)
		}
		loopRequest.IsRollbackRequest = false
		loopRequest.TestingIterations = 2
		loopRequest.IterationsStartIndex = 24
		loopRequest.WmSize = WmSizeNew
		loopRequest.runWmRequestInLoopAndTest(t)

	})

}
