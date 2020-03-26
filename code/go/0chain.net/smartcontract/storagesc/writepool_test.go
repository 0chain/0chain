package storagesc

import (
	"context"
	"net/url"
	"testing"
	"time"

	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_writePoolRequest_decode(t *testing.T) {
	const allocID = "alloc_hex"
	var wrd, wre writePoolRequest
	wre.AllocationID = allocID
	require.NoError(t, wrd.decode(mustEncode(t, &wre)))
	assert.EqualValues(t, wre, wrd)
}

func Test_newWritePool(t *testing.T) {
	var wp = newWritePool()
	assert.NotNil(t, wp.ZcnPool)
}

func Test_writePoolKey(t *testing.T) {
	const sscKey, allocID = "ssc_hex", "alloc_hex"
	assert.NotZero(t, writePoolKey(sscKey, allocID))
}

func newTestWritePool(poolID string, balance state.Balance) (wp *writePool) {
	wp = newWritePool()
	wp.ZcnPool.ID = poolID
	wp.ZcnPool.Balance = balance
	return
}

func Test_writePool_Encode_Decode(t *testing.T) {
	const (
		poolID  = "pool_id"
		balance = state.Balance(100500)
		dur     = 20 * time.Hour
	)

	var we, wd *writePool

	we = newTestWritePool(poolID, balance)
	wd = newWritePool()

	require.NoError(t, wd.Decode(we.Encode()))
	require.NotNil(t, wd.ZcnPool)
	assert.Equal(t, we.ZcnPool.ID, poolID)
	assert.Equal(t, we.ZcnPool.Balance, balance)
}

func Test_writePool_save(t *testing.T) {
	const sscKey, allocID = "ssc_key", "alloc_hex"

	var (
		wp       = newWritePool()
		balances = newTestBalances()
	)

	require.NoError(t, wp.save(sscKey, allocID, balances))
	var seri, err = balances.GetTrieNode(writePoolKey(sscKey, allocID))
	require.NoError(t, err)
	require.IsType(t, &writePool{}, seri)
	assert.Equal(t, seri, wp)
}

func Test_writePool_fill(t *testing.T) {
	const sscKey, clientID, txHash = "ssc_hex", "client_hex", "tx_hash"

	var (
		wp       = newWritePool()
		balances = newTestBalances()
		tx       = &transaction.Transaction{
			ClientID:   clientID,
			ToClientID: sscKey,
			Value:      90,
		}
	)

	tx.Hash = txHash
	balances.txn = tx

	var tr, resp, err = wp.fill(tx, balances)
	require.NoError(t, err)
	assert.NotNil(t, tr)
	assert.NotZero(t, resp)
	assert.Equal(t, state.Balance(90), wp.Balance)
}

func Test_writePool_moveToChallenge(t *testing.T) {
	// moveToChallenge(cp *challengePool, value state.Balance) (err error)

	var (
		wp = newWritePool()
		cp = newChallengePool()
	)

	cp.TokenLockInterface = &tokenLock{
		StartTime: common.Now(),
		Duration:  100 * time.Second,
	}
	cp.TokenPool.ID = "cp_id"

	wp.TokenPool.ID = "wp_id"

	requireErrMsg(t, wp.moveToChallenge(cp, 90),
		"not enough tokens in write pool")
	wp.Balance = 120
	require.NoError(t, wp.moveToChallenge(cp, 90))
	assert.Equal(t, state.Balance(30), wp.Balance)
	assert.Equal(t, state.Balance(90), cp.Balance)
}

func newTestStorageSC() (ssc *StorageSmartContract) {
	ssc = new(StorageSmartContract)
	ssc.SmartContract = new(smartcontractinterface.SmartContract)
	ssc.ID = ADDRESS
	return
}

func TestStorageSmartContract_getWritePoolBytes(t *testing.T) {
	const allocID = "alloc_hex"

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		b, err   = ssc.getWritePoolBytes(allocID, balances)
	)

	require.Equal(t, err, util.ErrValueNotPresent)
	var wp = newWritePool()
	_, err = balances.InsertTrieNode(writePoolKey(ssc.ID, allocID), wp)
	require.NoError(t, err)
	b, err = ssc.getWritePoolBytes(allocID, balances)
	require.NoError(t, err)
	assert.Equal(t, b, wp.Encode())
}

func TestStorageSmartContract_getWritePool(t *testing.T) {
	const allocID, clientID, txHash = "alloc_hex", "client_hex", "tx_hash"

	var (
		ssc      = newTestStorageSC()
		wp       = newWritePool()
		balances = newTestBalances()
		tx       = &transaction.Transaction{
			ClientID:   clientID,
			ToClientID: ssc.ID,
			Value:      90,
		}
		err error
	)

	tx.Hash = txHash
	balances.txn = tx

	_, _, err = wp.fill(tx, balances)
	require.NoError(t, err)

	_, err = ssc.getWritePool(allocID, balances)
	require.Equal(t, err, util.ErrValueNotPresent)
	require.NoError(t, wp.save(ssc.ID, allocID, balances))
	var wd *writePool
	wd, err = ssc.getWritePool(allocID, balances)
	require.NoError(t, err)
	assert.EqualValues(t, wd, wp)
}

func TestStorageSmartContract_newWritePool(t *testing.T) {
	const allocID, clientID = "alloc_hex", "client_hex"

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		wp, err  = ssc.newWritePool(allocID, balances)
	)

	require.NoError(t, err)
	assert.Equal(t, writePoolKey(ssc.ID, allocID), wp.TokenPool.ID)
}

func newTestWritePoolBlobberAllocation(minLockDemand state.Balance) (
	ba *BlobberAllocation) {

	ba = new(BlobberAllocation)
	ba.MinLockDemand = minLockDemand
	return
}

func TestStorageSmartContract_createWritePool(t *testing.T) {
	const (
		allocID, clientID, txHash = "alloc_hex", "client_hex", "tx_hash"

		errMsg1 = "not enough tokens to create allocation: 90 < 150"
		errMsg2 = "can't fill write pool: no tokens to lock"
		errMsg3 = "can't fill write pool: lock amount is greater than balance"
	)
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		tx       = &transaction.Transaction{
			ClientID:   clientID,
			ToClientID: ssc.ID,
			Value:      90,
		}
		sa = &StorageAllocation{
			ID: allocID,
			BlobberDetails: []*BlobberAllocation{
				newTestWritePoolBlobberAllocation(50), // }
				newTestWritePoolBlobberAllocation(50), // } 150
				newTestWritePoolBlobberAllocation(50), // }
			},
		}
		wp  *writePool
		err error
	)
	tx.Hash = txHash
	balances.txn = tx
	requireErrMsg(t, ssc.createWritePool(tx, sa, balances), errMsg1)
	sa.BlobberDetails = []*BlobberAllocation{
		newTestWritePoolBlobberAllocation(30), // }
		newTestWritePoolBlobberAllocation(30), // } 90
		newTestWritePoolBlobberAllocation(30), // }
	}
	requireErrMsg(t, ssc.createWritePool(tx, sa, balances), errMsg2)
	balances.balances[clientID] = 85
	requireErrMsg(t, ssc.createWritePool(tx, sa, balances), errMsg3)
	balances.balances[clientID] = 120
	require.NoError(t, ssc.createWritePool(tx, sa, balances))
	wp, err = ssc.getWritePool(allocID, balances)
	require.NoError(t, err)
	require.NotNil(t, wp)
	assert.Equal(t, wp.Balance, state.Balance(90))
	require.Len(t, balances.transfers, 1)
	var transfer = balances.transfers[0]
	assert.Equal(t, clientID, transfer.ClientID)
	assert.Equal(t, ADDRESS, transfer.ToClientID)
	assert.Equal(t, state.Balance(90), transfer.Amount)
}

func TestStorageSmartContract_writePoolLock(t *testing.T) {

	const (
		allocTxHash, clientID, pubKey = "a5f4c3d2_tx_hex",
			"client_hex", "pub_key_hex"

		errMsg1 = "write_pool_lock_failed: insufficient amount to lock"
		errMsg2 = "write_pool_lock_failed: " +
			"invalid character '}' looking for beginning of value"
		errMsg3 = "write_pool_lock_failed: missing allocation_id"
		errMsg4 = "write_pool_lock_failed: " +
			"can't get related allocation: value not present"
		errMsg5 = "write_pool_lock_failed: allocation is finalized"
		errMsg6 = "write_pool_lock_failed: only owner can fill the write pool"
		errMsg7 = "write_pool_lock_failed: lock amount is greater than balance"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		wpr      writePoolRequest

		tx    transaction.Transaction
		conf  scConfig
		alloc *StorageAllocation
		resp  string
		err   error
	)

	// setup write pool configurations
	conf.WritePool = new(writePoolConfig)
	conf.WritePool.MinLock = 10
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), &conf)
	require.NoError(t, err)

	// 1. less then min lock
	_, err = ssc.writePoolLock(&tx, nil, balances)
	requireErrMsg(t, err, errMsg1)
	// 2. malformed request
	tx.Value = 20
	_, err = ssc.writePoolLock(&tx, []byte("} malformed {"), balances)
	requireErrMsg(t, err, errMsg2)
	// 3. missing allocation id
	_, err = ssc.writePoolLock(&tx, mustEncode(t, &wpr), balances)
	requireErrMsg(t, err, errMsg3)
	// 4. no such allocation
	wpr.AllocationID = allocTxHash
	_, err = ssc.writePoolLock(&tx, mustEncode(t, &wpr), balances)
	requireErrMsg(t, err, errMsg4)
	// 5. allocation is finalized
	createNewTestAllocation(t, ssc, allocTxHash, clientID, pubKey, balances)
	// repair write pool configurations ---------------------------------------
	conf.WritePool = new(writePoolConfig)
	conf.WritePool.MinLock = 10
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), &conf)
	require.NoError(t, err)
	// ------------------------------------------------------------------------
	alloc, err = ssc.getAllocation(allocTxHash, balances)
	require.NoError(t, err)
	alloc.Finalized = true
	_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), alloc)
	require.NoError(t, err)
	_, err = ssc.writePoolLock(&tx, mustEncode(t, &wpr), balances)
	requireErrMsg(t, err, errMsg5)
	// 6. not owner
	alloc.Finalized = false
	_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), alloc)
	require.NoError(t, err)
	_, err = ssc.writePoolLock(&tx, mustEncode(t, &wpr), balances)
	requireErrMsg(t, err, errMsg6)
	// 7. no tokens (client balance)
	tx.ClientID = clientID
	balances.balances[clientID] = 19
	_, err = ssc.writePoolLock(&tx, mustEncode(t, &wpr), balances)
	requireErrMsg(t, err, errMsg7)
	// 8. ok
	balances.balances[clientID] = 21
	resp, err = ssc.writePoolLock(&tx, mustEncode(t, &wpr), balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)
	// write pool should be saved
	var wp *writePool
	wp, err = ssc.getWritePool(allocTxHash, balances)
	require.NoError(t, err)
	assert.Equal(t, state.Balance(420), wp.Balance) // 400 by the create alloc
}

func TestStorageSmartContract_getWritePoolStatHandler(t *testing.T) {

	const allocTxHash, clientID, pubKey = "a5f4c3d2_tx_hex",
		"client_hex", "pub_key_hex"

	var (
		ctx      = context.Background()
		ssc      = newTestStorageSC()
		balances = newTestBalances()

		params url.Values
		resp   interface{}
		err    error
	)

	resp, err = ssc.getWritePoolStatHandler(ctx, params, balances)
	requireErrMsg(t, err, "missing allocation_id URL query parameter")

	params = url.Values{"allocation_id": []string{"not_found_hex"}}
	resp, err = ssc.getWritePoolStatHandler(ctx, params, balances)
	requireErrMsg(t, err, "value not present")

	createNewTestAllocation(t, ssc, allocTxHash, clientID, pubKey, balances)
	params = url.Values{"allocation_id": []string{allocTxHash}}
	resp, err = ssc.getWritePoolStatHandler(ctx, params, balances)
	require.NoError(t, err)

	assert.EqualValues(t, &writePoolStat{
		ID:         writePoolKey(ssc.ID, allocTxHash),
		Balance:    400,
		StartTime:  7200,
		Expiration: 7315,
		Finalized:  false,
	}, resp)
}
