package storagesc

import (
	"encoding/json"
	"testing"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//
// test extension
//

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
	lre.AllocationID = "alloc_hex"
	require.NoError(t, lrd.decode(mustEncode(t, &lre)))
	assert.EqualValues(t, lre, lrd)
}

func Test_unlockRequest_decode(t *testing.T) {
	var ure, urd unlockRequest
	require.NoError(t, urd.decode(mustEncode(t, ure)))
	assert.EqualValues(t, ure, urd)
}

func Test_readPool_Encode_Decode(t *testing.T) {
	var rpe, rpd readPool
	rpe.add(10)
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

		rp     *readPool
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
		txHash  = "tx_hash"
		errMsg1 = "read_pool_lock_failed: insufficient amount to lock"
		errMsg2 = "read_pool_lock_failed: " +
			"invalid character '}' looking for beginning of value"
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
		lr  readPoolLockRequest
		err error
	)

	// setup transaction
	balances.setTransaction(t, &tx)
	tx.Hash = txHash

	// setup config
	testSetReadPoolConfig(t, &readPoolConfig{
		MinLock: 10,
	}, balances, ssc.ID)

	var fp fundedPools = []string{client.id}
	_, err = balances.InsertTrieNode(fundedPoolsKey(ssc.ID, client.id), &fp)
	require.NoError(t, err)

	// 1. 0 tx value
	_, err = ssc.readPoolLock(&tx, nil, balances)
	requireErrMsg(t, err, errMsg1)

	tx.Hash = "new_read_pool_tx_hash"
	tx.Hash = txHash
	tx.Value = 50
	// 2. malformed request
	_, err = ssc.readPoolLock(&tx, []byte("} malformed {"), balances)
	requireErrMsg(t, err, errMsg2)
	// 3. min lock
	tx.Value = 5
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg1)
	// 5. lock for owned allocations
	var rp *readPool
	tx.Value = 15
	balances.balances[client.id] = 15
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	require.NoError(t, err)
	rp, err = ssc.getReadPool(client.id, balances)
	require.NoError(t, err)
	assert.EqualValues(t, 15, rp.Balance)
}
