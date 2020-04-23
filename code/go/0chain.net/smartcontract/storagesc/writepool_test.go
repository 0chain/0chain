package storagesc

import (
	// "context"
	"encoding/json"
	// "net/url"
	"testing"
	"time"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//
// test extension
//
func (wp *writePool) total(now int64) state.Balance {
	return wp.Pools.total(now)
}

func (wp *writePool) allocTotal(allocID string,
	now int64) state.Balance {

	return wp.Pools.allocTotal(allocID, now)
}

func (wp *writePool) allocBlobberTotal(allocID, blobberID string,
	now int64) state.Balance {

	return wp.Pools.allocBlobberTotal(allocID, blobberID, now)
}

func mustEncode(t *testing.T, val interface{}) []byte {
	var err error
	b, err := json.Marshal(val)
	require.NoError(t, err)
	return b
}

func requireErrMsg(t *testing.T, err error, msg string) {
	t.Helper()
	require.Error(t, err, "missing error")
	require.Equal(t, msg, err.Error(), "unexpected error")
}

func Test_lockRequest_decode(t *testing.T) {
	var lre, lrd lockRequest
	lre.Duration = time.Second * 60
	lre.AllocationID = "alloc_hex"
	lre.BlobberID = "blobber_hex"
	require.NoError(t, lrd.decode(mustEncode(t, &lre)))
	assert.EqualValues(t, lre, lrd)
}

func Test_unlockRequest_decode(t *testing.T) {
	var ure, urd unlockRequest
	ure.PoolID = "pool_hex"
	require.NoError(t, urd.decode(mustEncode(t, ure)))
	assert.EqualValues(t, ure, urd)
}

func Test_writePool_Encode_Decode(t *testing.T) {
	var rpe, rpd writePool
	rpe.Pools.add(&allocationPool{
		ZcnPool: tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID: "IDENTIFIER", Balance: 100500,
			},
		},
		AllocationID: "ALLOCATION ID",
		Blobbers: blobberPools{
			&blobberPool{
				BlobberID: "BLOBBER ID",
				Balance:   10300,
			},
		},
		ExpireAt: 90210,
	})
	require.NoError(t, json.Unmarshal(mustEncode(t, rpe), &rpd))
	assert.EqualValues(t, rpe, rpd)
}

func Test_writePoolKey(t *testing.T) {
	assert.NotZero(t, writePoolKey("scKey", "clientID"))
}

func TestStorageSmartContract_getWritePoolBytes(t *testing.T) {
	const (
		clientID = "client_id"
		errMsg1  = "value not present"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()

		wp *writePool

		b, err = ssc.getWritePoolBytes(clientID, balances)
	)

	requireErrMsg(t, err, errMsg1)
	wp = new(writePool)
	require.NoError(t, wp.save(ssc.ID, clientID, balances))
	b, err = ssc.getWritePoolBytes(clientID, balances)
	require.NoError(t, err)
	assert.EqualValues(t, wp.Encode(), b)
}

func TestStorageSmartContract_getWritePool(t *testing.T) {
	const (
		clientID = "client_id"
		errMsg1  = "value not present"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		wps, err = ssc.getWritePool(clientID, balances)
		nrps     = new(writePool)
	)

	requireErrMsg(t, err, errMsg1)
	nrps = new(writePool)
	require.NoError(t, nrps.save(ssc.ID, clientID, balances))
	wps, err = ssc.getWritePool(clientID, balances)
	require.NoError(t, err)
	require.EqualValues(t, nrps, wps)
}

func TestStorageSmartContract_newWritePool(t *testing.T) {
	const (
		clientID, txHash = "client_id", "tx_hash"
		errMsg           = "new_write_pool_failed: already exist"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		tx       = transaction.Transaction{
			ClientID:   clientID,
			ToClientID: ssc.ID,
			Value:      0,
		}
		resp string
		err  error
	)

	balances.txn = &tx
	tx.Hash = txHash

	resp, err = ssc.newWritePool(&tx, nil, balances)
	require.NoError(t, err)
	var nrp = new(writePool)
	assert.Equal(t, string(nrp.Encode()), resp)

	_, err = ssc.newWritePool(&tx, nil, balances)
	requireErrMsg(t, err, errMsg)
}

func testSetWritePoolConfig(t *testing.T, wpc *writePoolConfig,
	balances chainState.StateContextI, sscID string) {

	var (
		conf scConfig
		err  error
	)
	conf.WritePool = wpc
	_, err = balances.InsertTrieNode(scConfigKey(sscID), &conf)
	require.NoError(t, err)
}

func TestStorageSmartContract_writePoolLock(t *testing.T) {
	const (
		allocID, txHash = "alloc_hex", "tx_hash"

		errMsg1 = "write_pool_lock_failed: value not present"
		errMsg2 = "write_pool_lock_failed: " +
			"invalid character '}' looking for beginning of value"
		errMsg3 = "write_pool_lock_failed: no tokens to lock"
		errMsg4 = "write_pool_lock_failed: insufficient amount to lock"
		errMsg5 = "write_pool_lock_failed: " +
			"duration (5s) is shorter than min lock period (10s)"
		errMsg6 = "write_pool_lock_failed: " +
			"duration (2m30s) is longer than max lock period (1m40s)"
		errMsg7 = "write_pool_lock_failed: user already has this write pool"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		client   = newClient(0, balances)
		tx       = transaction.Transaction{
			ClientID:   client.id,
			ToClientID: ssc.ID,
			Value:      0,
		}
		lr   lockRequest
		resp string
		err  error
	)

	// setup transaction

	balances.txn = &tx
	tx.Hash = txHash

	// setup config

	testSetWritePoolConfig(t, &writePoolConfig{
		MinLock:       10,
		MinLockPeriod: 10 * time.Second,
		MaxLockPeriod: 100 * time.Second,
	}, balances, ssc.ID)

	// 1. no pool
	_, err = ssc.writePoolLock(&tx, nil, balances)
	requireErrMsg(t, err, errMsg1)

	tx.Hash = "new_write_pool_tx_hash"
	_, err = ssc.newWritePool(&tx, nil, balances)
	require.NoError(t, err)
	tx.Hash = txHash
	// 2. malformed request
	_, err = ssc.writePoolLock(&tx, []byte("} malformed {"), balances)
	requireErrMsg(t, err, errMsg2)
	// 3. min lock
	tx.Value = 5
	lr.Duration = 5 * time.Second
	lr.AllocationID = allocID
	_, err = ssc.writePoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg4)
	// 5. min lock period
	tx.Value = 15
	balances.balances[client.id] = 15
	_, err = ssc.writePoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg5)
	// 6. max lock period
	lr.Duration = 150 * time.Second
	_, err = ssc.writePoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg6)
	// 7. no such allocation
	lr.Duration = 15 * time.Second
	resp, err = ssc.writePoolLock(&tx, mustEncode(t, &lr), balances)
	require.Error(t, err)

	balances.balances[client.id] = 200e10
	var aid, _ = addAllocation(t, ssc, client, 10, int64(toSeconds(time.Hour)),
		balances)
	// lock
	lr.AllocationID = aid
	resp, err = ssc.writePoolLock(&tx, mustEncode(t, &lr), balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)
}
