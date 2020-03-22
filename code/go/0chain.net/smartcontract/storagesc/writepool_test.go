package storagesc

import (
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
	const clientID = "client_hex"
	var wp = newWritePool(clientID)
	assert.NotNil(t, wp.ZcnLockingPool)
	assert.Equal(t, wp.ClientID, clientID)
}

func Test_writePoolKey(t *testing.T) {
	const sscKey, allocID = "ssc_hex", "alloc_hex"
	assert.NotZero(t, writePoolKey(sscKey, allocID))
}

func newTestWritePool(clientID, poolID string, balance state.Balance,
	startTime common.Timestamp, duration time.Duration) (wp *writePool) {

	wp = newWritePool(clientID)
	wp.ZcnLockingPool.ID = poolID
	wp.ZcnLockingPool.Balance = balance
	wp.ZcnLockingPool.TokenLockInterface = &tokenLock{
		StartTime: startTime,
		Duration:  duration,
		Owner:     clientID,
	}
	return
}

func Test_writePool_Encode_Decode(t *testing.T) {
	const (
		clientID, poolID = "client_hex", "pool_id"
		balance          = state.Balance(100500)
		dur              = 20 * time.Hour
	)

	var (
		sp     = common.Now() // start point
		we, wd *writePool
	)

	we = newTestWritePool(clientID, poolID, balance, sp, dur)
	wd = newWritePool("")

	require.NoError(t, wd.Decode(we.Encode()))
	assert.Equal(t, wd.ClientID, clientID)
	require.NotNil(t, wd.ZcnLockingPool)
	assert.Equal(t, we.ZcnLockingPool.ID, poolID)
	assert.Equal(t, we.ZcnLockingPool.Balance, balance)
	assert.NotNil(t, we.ZcnLockingPool.TokenLockInterface)
	assert.IsType(t, &tokenLock{}, we.ZcnLockingPool.TokenLockInterface)
	assert.EqualValues(t, &tokenLock{
		StartTime: sp,
		Duration:  dur,
		Owner:     clientID,
	}, we.ZcnLockingPool.TokenLockInterface)
}

func Test_writePool_save(t *testing.T) {
	const sscKey, allocID = "ssc_key", "alloc_hex"

	var (
		wp       = newWritePool("")
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
		wp       = newWritePool("")
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

func Test_writePool_setExpiration(t *testing.T) {
	const (
		clientID = "client_hex"
		dur      = 60 * time.Second
	)

	var (
		tp = common.Now()
		wp = newWritePool(clientID)
	)

	wp.ZcnLockingPool.TokenLockInterface = &tokenLock{
		StartTime: tp,
	}

	require.NoError(t, wp.setExpiration(tp+toSeconds(dur)))
	require.IsType(t, &tokenLock{}, wp.ZcnLockingPool.TokenLockInterface)
	assert.Equal(t, &tokenLock{
		StartTime: tp,
		Duration:  dur,
	}, wp.ZcnLockingPool.TokenLockInterface)
}

func Test_writePoolStat_decode(t *testing.T) {
	var state, statd writePoolStat

	state.ID = "pool_hex"
	state.StartTime = common.Now()
	state.Duartion = 60 * time.Second
	state.TimeLeft = 90 * time.Second
	state.Locked = true
	state.Balance = 150

	require.NoError(t, statd.decode(mustEncode(t, &state)))
	assert.EqualValues(t, state, statd)
}

func newTestStorageSC() (ssc *StorageSmartContract) {
	ssc = new(StorageSmartContract)
	ssc.SmartContract = new(smartcontractinterface.SmartContract)
	ssc.ID = ADDRESS
	return
}

func TestStorageSmartContract_getWritePoolBytes(t *testing.T) {
	const allocID, clientID = "alloc_hex", "client_hex"

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		b, err   = ssc.getWritePoolBytes(allocID, balances)
	)

	require.Equal(t, err, util.ErrValueNotPresent)
	var wp = newWritePool(clientID)
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
		wp       = newWritePool(clientID)
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
		tp       = common.Now()
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		wp, err  = ssc.newWritePool(allocID, clientID, tp, tp, balances)
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
	// TODO (sfxdx): implement tests
}

func TestStorageSmartContract_writePoolUnlock(t *testing.T) {
	// TODO (sfxdx): implement tests
}

func TestStorageSmartContract_getWritePoolStatHandler(t *testing.T) {
	// TODO (sfxdx): implement tests
}
