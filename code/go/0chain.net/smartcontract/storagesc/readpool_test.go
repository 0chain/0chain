package storagesc

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	require.NoError(t, lrd.decode(mustEncode(t, &lre)))
	assert.EqualValues(t, lre, lrd)
}

func Test_unlockRequest_decode(t *testing.T) {
	var ure, urd unlockRequest
	ure.PoolID = "pool_hex"
	require.NoError(t, urd.decode(mustEncode(t, ure)))
	assert.EqualValues(t, ure, urd)
}

func Test_newReadPool(t *testing.T) {
	var rp = newReadPool()
	assert.NotZero(t, rp.ZcnLockingPool, "missing")
}

func Test_readPool_encode_decode(t *testing.T) {
	var rpe, rpd = newReadPool(), newReadPool()
	require.NoError(t, rpd.decode(rpe.encode()))
	assert.EqualValues(t, rpe, rpd)
}

func Test_newReadPools(t *testing.T) {
	var rps = newReadPools()
	assert.NotZero(t, rps.Pools)
}

func Test_readPools_Encode_Decode(t *testing.T) {
	var (
		rpse, rpsd = newReadPools(), newReadPools()
		rp         = newReadPool()
	)
	require.NoError(t, rpse.addPool(rp))
	require.NoError(t, rpsd.Decode(rpse.Encode()))
	assert.EqualValues(t, rpse, rpsd)
}

func Test_readPoolsKey(t *testing.T) {
	assert.NotZero(t, readPoolsKey("scKey", "clientID"))
}

func Test_readPools_addPool_delPool(t *testing.T) {
	var (
		rps = newReadPools()
		rp  = newReadPool()
	)
	require.NoError(t, rps.addPool(rp))
	require.Error(t, rps.addPool(rp))
	rps.delPool(rp.ID)
	assert.Len(t, rps.Pools, 0, "not deleted")
}

func Test_readPools_moveToBlobber(t *testing.T) {

	const (
		blobberID = "blobber_id"
		errMsg    = "not enough tokens in read pool"
	)

	var (
		rps                 = newReadPools()
		sp                  = newStakePool(blobberID)
		now                 = common.Now()
		value state.Balance = 90
	)

	requireErrMsg(t, rps.moveToBlobber(now, sp, value), errMsg)

	// unlocked
	var unlocked = newReadPool()
	unlocked.ID = "unlocked_id"
	unlocked.Balance = 150
	unlocked.TokenLockInterface = &tokenLock{
		StartTime: now - 100,
		Duration:  10 * time.Second,
	}

	require.NoError(t, rps.addPool(unlocked))
	requireErrMsg(t, rps.moveToBlobber(now, sp, value), errMsg)

	// not enough tokens
	var small = newReadPool()
	small.ID = "small_id"
	small.Balance = 50
	small.TokenLockInterface = &tokenLock{
		StartTime: now,
		Duration:  100 * time.Second,
	}

	require.NoError(t, rps.addPool(small))

	// enough tokens and should left a bit
	var enough = newReadPool()
	enough.ID = "enough_id"
	enough.Balance = 50
	enough.TokenLockInterface = &tokenLock{
		StartTime: now,
		Duration:  100 * time.Second,
	}

	require.NoError(t, rps.addPool(enough))
	require.NoError(t, rps.moveToBlobber(now, sp, value))

	// check pools
	assert.Equal(t, state.Balance(90), sp.Unlocked.Balance)
	assert.Len(t, rps.Pools, 2)

	var gotIt *readPool
	for k, v := range rps.Pools {
		if k == unlocked.ID {
			continue
		}
		gotIt = v
	}

	assert.Equal(t, state.Balance(10), gotIt.Balance)

	requireErrMsg(t, rps.moveToBlobber(now, sp, value), errMsg)
}

func Test_readPoolStats_encode_decode(t *testing.T) {
	var se, sd readPoolStats
	se.Stats = append(se.Stats, &readPoolStat{
		ID:        "pool_id",
		StartTime: common.Now(),
		Duration:  10 * time.Second,
		TimeLeft:  150,
		Locked:    true,
		Balance:   90,
	})
	require.NoError(t, sd.decode(se.encode()))
	assert.EqualValues(t, se, sd)
}

func Test_readPoolStats_addStat(t *testing.T) {
	var (
		stats readPoolStats
		stat  = &readPoolStat{
			ID:        "pool_id",
			StartTime: common.Now(),
			Duration:  10 * time.Second,
			TimeLeft:  150,
			Locked:    true,
			Balance:   90,
		}
	)
	stats.addStat(stat)
	assert.Len(t, stats.Stats, 1)
	stats.addStat(stat)
	assert.Len(t, stats.Stats, 2)
}

func Test_readPoolStat_encode_decode(t *testing.T) {
	var se, sd readPoolStat
	se.ID = "pool_id"
	se.StartTime = common.Now()
	se.Duration = 10 * time.Second
	se.TimeLeft = 150
	se.Locked = true
	se.Balance = 90
	require.NoError(t, sd.decode(se.encode()))
	assert.EqualValues(t, se, sd)

}

func Test_tokenLock_IsLocked(t *testing.T) {
	var tl tokenLock
	tl.StartTime = common.Now()
	tl.Duration = 10 * time.Second
	assert.True(t, tl.IsLocked(false))
	assert.True(t, tl.IsLocked(time.Now()))
	tl.StartTime = common.Now() - 11
	assert.False(t, tl.IsLocked(time.Now()))
}

func Test_tokenLock_LockStats(t *testing.T) {
	var (
		tl   tokenLock
		stat readPoolStat
		now  common.Timestamp = 210
	)
	tl.StartTime = now
	tl.Duration = 10 * time.Second
	tl.Owner = "client_id"
	assert.Error(t, stat.decode(tl.LockStats(false)))
	assert.NoError(t, stat.decode(tl.LockStats(time.Unix(215, 0))))
	assert.Equal(t, stat, readPoolStat{
		StartTime: now,
		Duration:  10 * time.Second,
		TimeLeft:  5 * time.Second,
		Locked:    true,
	})
	assert.NoError(t, stat.decode(tl.LockStats(time.Unix(225, 0))))
	assert.Equal(t, stat, readPoolStat{
		StartTime: now,
		Duration:  10 * time.Second,
		TimeLeft:  -5 * time.Second,
		Locked:    false,
	})
}

func TestStorageSmartContract_getReadPoolsBytes(t *testing.T) {
	const (
		clientID = "client_id"
		errMsg1  = "value not present"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()

		rps *readPools

		b, err = ssc.getReadPoolsBytes(clientID, balances)
	)

	requireErrMsg(t, err, errMsg1)
	rps = newReadPools()
	require.NoError(t, rps.save(ssc.ID, clientID, balances))
	b, err = ssc.getReadPoolsBytes(clientID, balances)
	require.NoError(t, err)
	assert.EqualValues(t, rps.Encode(), b)
}

func TestStorageSmartContract_getReadPools(t *testing.T) {
	const (
		clientID = "client_id"
		errMsg1  = "value not present"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		rps, err = ssc.getReadPools(clientID, balances)
		nrps     = newReadPools()
	)

	requireErrMsg(t, err, errMsg1)
	nrps = newReadPools()
	require.NoError(t, nrps.save(ssc.ID, clientID, balances))
	rps, err = ssc.getReadPools(clientID, balances)
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

	resp, err = ssc.newReadPool(&tx, nil, balances)
	require.NoError(t, err)
	assert.Equal(t, string(newReadPools().Encode()), resp)

	_, err = ssc.newReadPool(&tx, nil, balances)
	requireErrMsg(t, err, errMsg)
}

func testSetReadPoolConfig(t *testing.T, rpc *readPoolConfig,
	balances chainState.StateContextI, sscID string) {

	var (
		conf scConfig
		err  error
	)
	conf.ReadPool = rpc
	_, err = balances.InsertTrieNode(scConfigKey(sscID), &conf)
	require.NoError(t, err)
}

func TestStorageSmartContract_readPoolLock(t *testing.T) {
	const (
		clientID, txHash = "client_id", "tx_hash"

		errMsg1 = "read_pool_lock_failed: value not present"
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
		balances = newTestBalances()
		tx       = transaction.Transaction{
			ClientID:   clientID,
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

	testSetReadPoolConfig(t, &readPoolConfig{
		MinLock:       10,
		MinLockPeriod: 10 * time.Second,
		MaxLockPeriod: 100 * time.Second,
	}, balances, ssc.ID)

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
	// 3. no tokens
	tx.Value = 5
	lr.Duration = 5 * time.Second
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg3)
	// 4. min lock
	balances.balances[clientID] = 5
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg4)
	tx.Value = 15
	balances.balances[clientID] = 15
	// 5. min lock period
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg5)
	// 6. max lock period
	lr.Duration = 150 * time.Second
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg6)
	// lock
	lr.Duration = 15 * time.Second
	resp, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)
	// 7. already exists
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg7)
}

func TestStorageSmartContract_readPoolUnlock(t *testing.T) {
	const (
		clientID, txHash, readPoolID = "client_id", "tx_hash", "pool_id"

		errMsg1 = "read_pool_unlock_failed: value not present"
		errMsg2 = "read_pool_unlock_failed: " +
			"invalid character '}' looking for beginning of value"
		errMsg3 = "read_pool_unlock_failed: pool not found"
		errMsg4 = "read_pool_unlock_failed: " +
			"emptying pool failed: pool is still locked"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		tx       = transaction.Transaction{
			ClientID:   clientID,
			ToClientID: ssc.ID,
			Value:      0,
		}
		lr   lockRequest
		ur   unlockRequest
		resp string
		err  error
	)

	balances.txn = &tx
	tx.Hash = txHash

	// 1. no read pools
	_, err = ssc.readPoolUnlock(&tx, nil, balances)
	requireErrMsg(t, err, errMsg1)

	// create read pool
	tx.Hash = "create_read_pool_tx"
	_, err = ssc.newReadPool(&tx, nil, balances)
	require.NoError(t, err)
	tx.Hash = txHash

	// 2. malformed request
	_, err = ssc.readPoolUnlock(&tx, []byte("} malformed {"), balances)
	requireErrMsg(t, err, errMsg2)

	// 3. no read pool
	_, err = ssc.readPoolUnlock(&tx, mustEncode(t, &ur), balances)
	requireErrMsg(t, err, errMsg3)

	// lock tokens
	testSetReadPoolConfig(t, &readPoolConfig{
		MinLock:       10,
		MinLockPeriod: 10 * time.Second,
		MaxLockPeriod: 100 * time.Second,
	}, balances, ssc.ID)
	tx.Hash = readPoolID
	lr.Duration = 15 * time.Second
	balances.balances[clientID] = 150
	tx.Value = 150
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	require.NoError(t, err)

	delete(balances.balances, clientID)
	tx.Value = 0

	// 4. not expired
	ur.PoolID = readPoolID
	_, err = ssc.readPoolUnlock(&tx, mustEncode(t, &ur), balances)
	requireErrMsg(t, err, errMsg4)

	tx.CreationDate = common.Timestamp(20 * time.Second)

	// 5. unlock (ok)
	resp, err = ssc.readPoolUnlock(&tx, mustEncode(t, &ur), balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)

}

func TestStorageSmartContract_getReadPoolsStatsHandler(t *testing.T) {

	const (
		clientID = "client_id"

		errMsg1 = "value not present"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		ctx      = context.Background()
		params   = url.Values{
			"client_id": []string{clientID},
		}
		resp, err = ssc.getReadPoolsStatsHandler(ctx, params, balances)
	)

	requireErrMsg(t, err, errMsg1)

	var (
		rps = newReadPools()
		rp  = newReadPool()
	)

	rp.ID = "pool_id"
	rp.TokenLockInterface = &tokenLock{
		StartTime: 150,
		Duration:  10 * time.Second,
		Owner:     "owner_id",
	}
	rp.Balance = 150

	require.NoError(t, rps.addPool(rp))
	require.NoError(t, rps.save(ssc.ID, clientID, balances))

	resp, err = ssc.getReadPoolsStatsHandler(ctx, params, balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)
}
