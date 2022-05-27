package storagesc

import (
	"0chain.net/chaincore/currency"
	// "context"
	"encoding/json"
	// "net/url"
	"testing"
	"time"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//
// test extension
//

func (rp *readPool) allocTotal(allocID string,
	now int64) currency.Coin {

	return rp.Pools.allocTotal(allocID, now)
}

func mustEncode(t testing.TB, val interface{}) []byte {
	var err error
	b, err := json.Marshal(val)
	require.NoError(t, err)
	return b
}

func mustDecode(t testing.TB, b []byte, val interface{}) {
	require.NoError(t, json.Unmarshal(b, val))
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

func Test_readPool_Encode_Decode(t *testing.T) {
	var rpe, rpd readPool
	rpe.Pools.add(&allocationPool{
		ZcnPool: tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID: "IDENTIFIER", Balance: 100500,
			},
		},
		AllocationID: "ALLOCATION ID",
		ExpireAt:     90210,
	})
	require.NoError(t, json.Unmarshal(mustEncode(t, rpe), &rpd))
	assert.EqualValues(t, rpe, rpd)
}

func Test_readPoolKey(t *testing.T) {
	assert.NotZero(t, readPoolKey("scKey", "clientID"))
}

func TestStorageSmartContract_getReadPoolBytes(t *testing.T) {
	const (
		clientID = "client_id"
		errMsg1  = "value not present"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)

		rp *readPool

		_, err = ssc.getReadPool(clientID, balances)
	)

	requireErrMsg(t, err, errMsg1)
	rp = new(readPool)
	require.NoError(t, rp.save(ssc.ID, clientID, balances))
	b, err := ssc.getReadPool(clientID, balances)
	require.NoError(t, err)
	assert.EqualValues(t, rp, b)
}

func TestStorageSmartContract_getReadPool(t *testing.T) {
	const (
		clientID = "client_id"
		errMsg1  = "value not present"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		_, err   = ssc.getReadPool(clientID, balances)
		nrps     = new(readPool)
	)

	requireErrMsg(t, err, errMsg1)
	require.NoError(t, nrps.save(ssc.ID, clientID, balances))
	rps, err := ssc.getReadPool(clientID, balances)
	require.NoError(t, err)
	require.EqualValues(t, nrps, rps)
}

func TestStorageSmartContract_newReadPool(t *testing.T) {
	const (
		clientID, txHash = "client_id", "tx_hash"
		errMsg           = "new_read_pool_failed: already exist"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		tx       = transaction.Transaction{
			ClientID:   clientID,
			ToClientID: ssc.ID,
			Value:      0,
		}
		resp string
		err  error
	)

	balances.setTransaction(t, &tx)
	tx.Hash = txHash

	resp, err = ssc.newReadPool(&tx, nil, balances)
	require.NoError(t, err)
	var nrp = new(readPool)
	assert.Equal(t, string(nrp.Encode()), resp)

	_, err = ssc.newReadPool(&tx, nil, balances)
	requireErrMsg(t, err, errMsg)
}

func testSetReadPoolConfig(t *testing.T, rpc *readPoolConfig,
	balances chainState.StateContextI, sscID string) {

	var (
		conf Config
		err  error
	)
	conf.ReadPool = rpc
	_, err = balances.InsertTrieNode(scConfigKey(sscID), &conf)
	require.NoError(t, err)
}

func TestStorageSmartContract_readPoolLock(t *testing.T) {
	const (
		allocID, txHash = "alloc_hex", "tx_hash"

		errMsg1 = "read_pool_lock_failed: unexpected end of JSON input"
		errMsg2 = "read_pool_lock_failed: " +
			"invalid character '}' looking for beginning of value"
		errMsg3 = "read_pool_lock_failed: no tokens to lock"
		errMsg4 = "read_pool_lock_failed: insufficient amount to lock"
		errMsg5 = "read_pool_lock_failed: " +
			"duration (5s) is shorter than min lock period (10s)"
		errMsg6 = "read_pool_lock_failed: " +
			"duration (2m30s) is longer than max lock period (1m40s)"
		errMsg7 = "read_pool_lock_failed: user already has this read pool"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
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

	balances.setTransaction(t, &tx)
	tx.Hash = txHash

	// setup config

	testSetReadPoolConfig(t, &readPoolConfig{
		MinLock:       10,
		MinLockPeriod: 10 * time.Second,
		MaxLockPeriod: 100 * time.Second,
	}, balances, ssc.ID)

	var fp fundedPools = []string{client.id}
	_, err = balances.InsertTrieNode(fundedPoolsKey(ssc.ID, client.id), &fp)
	require.NoError(t, err)

	// 1. no pool
	_, err = ssc.readPoolLock(&tx, nil, balances)
	requireErrMsg(t, err, errMsg1)

	tx.Hash = "new_read_pool_tx_hash"
	_, err = ssc.newReadPool(&tx, nil, balances)
	require.NoError(t, err)
	tx.Hash = txHash
	// 2. malformed request
	_, err = ssc.readPoolLock(&tx, []byte("} malformed {"), balances)
	requireErrMsg(t, err, errMsg2)
	// 3. min lock
	tx.Value = 5
	lr.Duration = 5 * time.Second
	lr.AllocationID = allocID
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg4)
	// 5. min lock period
	tx.Value = 15
	balances.balances[client.id] = 15
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg5)
	// 6. max lock period
	lr.Duration = 150 * time.Second
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg6)
	// 7. no such allocation
	lr.Duration = 15 * time.Second
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	require.Error(t, err)

	balances.balances[client.id] = 200e10
	var aid, _ = addAllocation(t, ssc, client, 10, int64(toSeconds(time.Hour)),
		0, balances)
	// lock
	lr.AllocationID = aid
	resp, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)
}
