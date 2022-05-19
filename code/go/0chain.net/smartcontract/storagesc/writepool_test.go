package storagesc

import (
	"0chain.net/pkg/currency"
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

func (wp *writePool) allocTotal(allocID string,
	now int64) currency.Coin {

	return wp.Pools.allocTotal(allocID, now)
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
		balances = newTestBalances(t, false)

		_, err = ssc.getWritePool(clientID, balances)
	)

	requireErrMsg(t, err, errMsg1)
	wp := new(writePool)
	require.NoError(t, wp.save(ssc.ID, clientID, balances))
	wwp, err := ssc.getWritePool(clientID, balances)
	require.NoError(t, err)
	assert.EqualValues(t, wp, wwp)
}

func TestStorageSmartContract_getWritePool(t *testing.T) {
	const (
		clientID = "client_id"
		errMsg1  = "value not present"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)
		nrps     = new(writePool)
		_, err   = ssc.getWritePool(clientID, balances)
	)

	requireErrMsg(t, err, errMsg1)
	require.NoError(t, nrps.save(ssc.ID, clientID, balances))
	wps, err := ssc.getWritePool(clientID, balances)
	require.NoError(t, err)
	require.EqualValues(t, nrps, wps)
}

func testSetWritePoolConfig(t *testing.T, wpc *writePoolConfig,
	balances chainState.StateContextI, sscID string) {

	var (
		conf Config
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
			"duration (5s) is shorter than min lock period (20s)"
		errMsg6 = "write_pool_lock_failed: " +
			"duration (3h0m0s) is longer than max lock period (2h0m0s)"
		errMsg7 = "write_pool_lock_failed: user already has this write pool"
		errMsg8 = "write_pool_lock_failed: unexpected end of JSON input"
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

	testSetWritePoolConfig(t, &writePoolConfig{
		MinLock:       10,
		MinLockPeriod: 20 * time.Second,
		MaxLockPeriod: 2 * time.Hour,
	}, balances, ssc.ID)

	var fp fundedPools = []string{client.id}
	_, err = balances.InsertTrieNode(fundedPoolsKey(ssc.ID, client.id), &fp)
	require.NoError(t, err)

	var alloc = StorageAllocation{
		ID: allocID,
		BlobberAllocs: []*BlobberAllocation{
			&BlobberAllocation{MinLockDemand: 10, Spent: 0},
			&BlobberAllocation{MinLockDemand: 10, Spent: 0},
			&BlobberAllocation{MinLockDemand: 10, Spent: 0},
			&BlobberAllocation{MinLockDemand: 10, Spent: 0},
		},
		Expiration:              10,
		ChallengeCompletionTime: 200 * time.Second,
		Owner:                   client.id,
	}

	// 1. no pool
	_, err = ssc.writePoolLock(&tx, nil, balances)
	requireErrMsg(t, err, errMsg8)

	tx.Hash = "new_write_pool_tx_hash"
	tx.Value, balances.balances[client.id] = 40, 40 // set {
	err = ssc.createWritePool(&tx, &alloc, false, balances)
	require.NoError(t, err)
	tx.Hash = txHash
	tx.Value, balances.balances[client.id] = 0, 0 // } reset

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
	lr.Duration = 3 * time.Hour
	_, err = ssc.writePoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg6)
	// 7. no such allocation
	lr.Duration = 15 * time.Second
	_, err = ssc.writePoolLock(&tx, mustEncode(t, &lr), balances)
	require.Error(t, err)

	balances.balances[client.id] = 200e10
	var aid, _ = addAllocation(t, ssc, client, 10, int64(toSeconds(time.Hour)),
		0, balances)
	// lock
	lr.AllocationID = aid
	resp, err = ssc.writePoolLock(&tx, mustEncode(t, &lr), balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)
	// 8. 0 lock
	tx.Value = 0
	lr.Duration = 5 * time.Second
	lr.AllocationID = allocID
	_, err = ssc.writePoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg4)
}
