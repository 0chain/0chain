package vestingsc

import (
	"context"
	"net/url"
	"testing"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_toSeconds(t *testing.T) {
	assert.Equal(t, common.Timestamp(1),
		toSeconds(1*time.Second+500*time.Millisecond))
}

func Test_lockRequest_decode(t *testing.T) {
	var lr lockRequest
	require.NoError(t, lr.decode([]byte(`{"pool_id":"pool_hex"}`)))
	assert.Equal(t, "pool_hex", lr.PoolID)
}

func Test_addRequest_decode(t *testing.T) {
	var are, ard addRequest
	are.Description = "for something"
	are.StartTime = 10
	are.Duration = 2 * time.Second
	are.Friquency = 3 * time.Second
	are.Destinations = []string{"one", "two"}
	are.Amount = 400
	require.NoError(t, ard.decode(mustEncode(t, &are)))
	assert.EqualValues(t, &are, &ard)
}

func Test_addRequest_validate(t *testing.T) {
	var (
		conf = avgConfig()
		ar   addRequest
	)
	ar.Description = "very very very long description"
	assertErrMsg(t, ar.validate(10, conf), "entry description is too long")
	ar.Description = "short desc."

	ar.StartTime = 1
	assertErrMsg(t, ar.validate(10, conf), "vesting starts before now")
	ar.StartTime = 20

	assertErrMsg(t, ar.validate(10, conf), "vesting duration is too short")
	ar.Duration = 2 * time.Hour
	assertErrMsg(t, ar.validate(10, conf), "vesting duration is too long")
	ar.Duration = 1 * time.Minute

	assertErrMsg(t, ar.validate(10, conf), "vesting friquency is too low")
	ar.Friquency = 2 * time.Hour
	assertErrMsg(t, ar.validate(10, conf), "vesting friquency is too high")
	ar.Friquency = 1 * time.Minute

	assertErrMsg(t, ar.validate(10, conf), "no destinations")
	ar.Destinations = []string{"one", "two", "three"}
	assertErrMsg(t, ar.validate(10, conf), "too many destinations")
	ar.Destinations = []string{"one", "two"}

	assert.NoError(t, ar.validate(10, conf))
	ar.StartTime = 0
	assert.NoError(t, ar.validate(10, conf))
}

func Test_vestingPool(t *testing.T) {
	const poolID, clientID = "pool_hex", "client_hex"
	require.NotZero(t, poolKey(ADDRESS, poolID))
	var vp = newVestingPool()
	assert.NotNil(t, vp)
	assert.NotNil(t, vp.TokenPool)
	var ar addRequest
	ar.Description = "for something"
	ar.StartTime = 10
	ar.Duration = 2 * time.Second
	ar.Friquency = 3 * time.Second
	ar.Destinations = []string{"one", "two"}
	ar.Amount = 400
	vp = newVestingPoolFromReqeust(clientID, &ar)
	assert.NotNil(t, vp)
	assert.NotNil(t, vp.TokenPool)

	assert.Zero(t, vp.Last)

	assert.Equal(t, vp.Description, ar.Description)
	assert.Equal(t, vp.StartTime, ar.StartTime)
	assert.Equal(t, vp.ExpireAt, ar.StartTime+toSeconds(ar.Duration))
	assert.Equal(t, vp.Friquency, ar.Friquency)
	assert.Equal(t, vp.Destinations, ar.Destinations)
	assert.Equal(t, vp.Amount, ar.Amount)

	var vpd = new(vestingPool)
	require.NoError(t, vpd.Decode(vp.Encode()))
	assert.Equal(t, vp, vpd)

	var inf = vpd.info()
	assert.Equal(t, vp.Description, inf.Description)
	assert.Equal(t, vp.StartTime, inf.StartTime)
	assert.Equal(t, vp.ExpireAt, inf.ExpireAt)
	assert.Equal(t, vp.Friquency, inf.Friquency)
	assert.Equal(t, vp.Destinations, inf.Destinations)
	assert.Equal(t, vp.Amount, inf.Amount)
	assert.Equal(t, state.Balance(0), inf.Balance)
	assert.Equal(t, common.Timestamp(0), inf.Last)
}

func TestVestingSmartContract_getPoolBytes_getPool(t *testing.T) {
	const txHash, clientID = "tx_hash", "client_hex"
	var (
		vsc      = newTestVestingSC()
		balances = newTestBalances()
		err      error
	)
	_, err = vsc.getPoolBytes(poolKey(vsc.ID, txHash), balances)
	require.Equal(t, util.ErrValueNotPresent, err)
	_, err = vsc.getPool(poolKey(vsc.ID, txHash), balances)
	require.Equal(t, util.ErrValueNotPresent, err)
	var vp = newVestingPoolFromReqeust(clientID, &addRequest{
		Description:  "for something",
		StartTime:    10,
		Duration:     2 * time.Second,
		Friquency:    3 * time.Second,
		Destinations: []string{"one", "two"},
		Amount:       400,
	})
	vp.ID = poolKey(vsc.ID, txHash)
	require.NoError(t, vp.save(balances))
	var (
		poolb []byte
		got   *vestingPool
	)
	poolb, err = vsc.getPoolBytes(poolKey(vsc.ID, txHash), balances)
	require.NoError(t, err)
	assert.Equal(t, string(vp.Encode()), string(poolb))
	got, err = vsc.getPool(poolKey(vsc.ID, txHash), balances)
	require.NoError(t, err)
	assert.EqualValues(t, vp, got)
}

func TestVestingSmartContract_add(t *testing.T) {
	var (
		vsc      = newTestVestingSC()
		balances = newTestBalances()
		tp       = common.Timestamp(10)
		client   = newClient(0, balances)
		tx       = newTransaction(client.id, vsc.ID, 0, tp)
		ar       addRequest
		err      error
	)

	balances.txn = tx

	// 1. malformed request
	_, err = vsc.add(tx, []byte(`} malformed {`), balances)
	assertErrMsg(t, err, `create_vesting_pool_failed: malformed request:`+
		` invalid character '}' looking for beginning of value`)

	// 2. invalid
	setConfig(t, balances)
	ar.Description = "for something"
	ar.StartTime = 10
	ar.Duration = 0
	ar.Friquency = 3 * time.Second
	ar.Destinations = []string{"one", "two"}
	ar.Amount = 400
	_, err = vsc.add(tx, mustEncode(t, &ar), balances)
	assertErrMsg(t, err, `create_vesting_pool_failed: invalid request:`+
		` vesting duration is too short`)

	// 3. empty client id
	ar.Duration = 2 * time.Second
	tx.ClientID = ""
	_, err = vsc.add(tx, mustEncode(t, &ar), balances)
	assertErrMsg(t, err, `create_vesting_pool_failed: `+
		`empty client_id of transaction`)

	// 4. min lock
	tx = newTransaction(client.id, vsc.ID, 1, tp)
	balances.txn = tx
	_, err = vsc.add(tx, mustEncode(t, &ar), balances)
	assertErrMsg(t, err, `create_vesting_pool_failed: `+
		`insufficient amount to lock`)

	// 5. no tokens
	tx = newTransaction(client.id, vsc.ID, 800, tp)
	balances.txn = tx
	_, err = vsc.add(tx, mustEncode(t, &ar), balances)
	assertErrMsg(t, err, `create_vesting_pool_failed: `+
		`can't fill pool: lock amount is greater than balance`)

	// 6. ok
	balances.balances[client.id] = 1200
	tx = newTransaction(client.id, vsc.ID, 800, tp)
	balances.txn = tx
	var resp string
	resp, err = vsc.add(tx, mustEncode(t, &ar), balances)
	require.NoError(t, err)
	var deco vestingPool
	require.NoError(t, deco.Decode([]byte(resp)))
	assert.NotZero(t, deco.ID)
	assert.Equal(t, client.id, deco.ClientID)
	assert.Equal(t, state.Balance(800), deco.Balance)

	// 7. client pools
	var cp *clientPools
	cp, err = vsc.getClientPools(client.id, balances)
	require.NoError(t, err)
	assert.Equal(t, []string{deco.ID}, cp.Pools)
}

func TestVestingSmartContract_delete(t *testing.T) {
	var (
		vsc      = newTestVestingSC()
		balances = newTestBalances()
		client   = newClient(1200, balances)
		tp       = common.Timestamp(0)
		tx       = newTransaction(client.id, vsc.ID, 0, tp)
		dr       lockRequest
		err      error
	)

	balances.txn = tx
	setConfig(t, balances)

	// 1. malformed (lock, unlock)
	_, err = vsc.delete(tx, []byte("} malformed {"), balances)
	assertErrMsg(t, err, "delete_vesting_pool_failed: invalid request:"+
		" invalid character '}' looking for beginning of value")

	// 2. pool_id = ""
	_, err = vsc.delete(tx, mustEncode(t, &dr), balances)
	assertErrMsg(t, err, "delete_vesting_pool_failed: invalid request:"+
		" missing pool id")

	// 3. invalid transaction
	dr.PoolID = "pool_id"
	tx.ClientID = ""
	_, err = vsc.delete(tx, mustEncode(t, &dr), balances)
	assertErrMsg(t, err, "delete_vesting_pool_failed: "+
		"empty client id of transaction")

	// 4. not found
	tx.ClientID = client.id
	_, err = vsc.delete(tx, mustEncode(t, &dr), balances)
	assertErrMsg(t, err, "delete_vesting_pool_failed: "+
		"can't get pool: value not present")

	// 5. another client
	var resp string
	resp, err = client.add(t, vsc, &addRequest{
		Description:  "for something",
		StartTime:    10,
		Duration:     2 * time.Second,
		Friquency:    3 * time.Second,
		Destinations: []string{"one", "two"},
		Amount:       400,
	}, 800, tp, balances)
	require.NoError(t, err)
	var set vestingPool
	require.NoError(t, set.Decode([]byte(resp)))
	dr.PoolID = set.ID

	tx.ClientID = "another_one"
	balances.txn = tx
	_, err = vsc.delete(tx, mustEncode(t, &dr), balances)
	assertErrMsg(t, err, "delete_vesting_pool_failed: "+
		"only pool owner can do that")

	// 6. delete
	tx.ClientID = client.id
	resp, err = vsc.delete(tx, mustEncode(t, &dr), balances)
	require.NoError(t, err)
	assert.EqualValues(t, "deleted", resp)

	assert.Zero(t, balances.tree[set.ID])
	assert.Zero(t, balances.tree[clientPoolsKey(vsc.ID, client.id)])
}

func TestVestingSmartContract_lock(t *testing.T) {
	var (
		vsc      = newTestVestingSC()
		balances = newTestBalances()
		client   = newClient(1200, balances)
		tp       = common.Timestamp(0)
		tx       = newTransaction(client.id, vsc.ID, 0, tp)
		lr       lockRequest
		err      error
	)

	balances.txn = tx
	setConfig(t, balances)

	// 1. malformed (lock, unlock)
	_, err = vsc.lock(tx, []byte("} malformed {"), balances)
	assertErrMsg(t, err, "lock_vesting_pool_failed: invalid request:"+
		" invalid character '}' looking for beginning of value")

	// 2. pool_id = ""
	_, err = vsc.lock(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "lock_vesting_pool_failed: invalid request:"+
		" missing pool id")

	// 3. not found
	lr.PoolID = "pool_hex"
	_, err = vsc.lock(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "lock_vesting_pool_failed: "+
		"can't get pool: value not present")

	// 4. another client
	var resp string
	resp, err = client.add(t, vsc, &addRequest{
		Description:  "for something",
		StartTime:    10,
		Duration:     2 * time.Second,
		Friquency:    3 * time.Second,
		Destinations: []string{"one", "two"},
		Amount:       400,
	}, 800, tp, balances)
	require.NoError(t, err)
	var set vestingPool
	require.NoError(t, set.Decode([]byte(resp)))
	lr.PoolID = set.ID

	tx.ClientID = "another_one"
	balances.txn = tx
	_, err = vsc.lock(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "lock_vesting_pool_failed: "+
		"only owner can lock more tokens to the pool")

	// 6. min lock
	tx.Value = 1
	tx.ClientID = client.id
	_, err = vsc.lock(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "lock_vesting_pool_failed: "+
		"insufficient amount to lock")

	// 7. no tokens
	tx.Value = 2000
	_, err = vsc.lock(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "lock_vesting_pool_failed: "+
		"lock amount is greater than balance")

	// 8. lock
	balances.balances[client.id] = 4000
	resp, err = vsc.lock(tx, mustEncode(t, &lr), balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)

	var got *vestingPool
	got, err = vsc.getPool(set.ID, balances)
	require.NoError(t, err)
	assert.Equal(t, state.Balance(2800), got.Balance)

}

func TestVestingSmartContract_unlock(t *testing.T) {
	var (
		vsc      = newTestVestingSC()
		balances = newTestBalances()
		client   = newClient(1200, balances)
		tp       = common.Timestamp(0)
		tx       = newTransaction(client.id, vsc.ID, 0, tp)
		lr       lockRequest
		err      error
	)

	balances.txn = tx
	setConfig(t, balances)

	// 1. malformed
	_, err = vsc.unlock(tx, []byte("} malformed {"), balances)
	assertErrMsg(t, err, "unlock_vesting_pool_failed: invalid request:"+
		" invalid character '}' looking for beginning of value")

	// 2. pool_id = ""
	_, err = vsc.unlock(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "unlock_vesting_pool_failed: invalid request:"+
		" missing pool id")

	// 3. not found
	lr.PoolID = "pool_hex"
	_, err = vsc.unlock(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "unlock_vesting_pool_failed: "+
		"can't get pool: value not present")

	// 4. another client
	var resp string
	resp, err = client.add(t, vsc, &addRequest{
		Description:  "for something",
		StartTime:    10,
		Duration:     2 * time.Second,
		Friquency:    3 * time.Second,
		Destinations: []string{"one", "two"},
		Amount:       400,
	}, 800, tp, balances)
	require.NoError(t, err)
	var set vestingPool
	require.NoError(t, set.Decode([]byte(resp)))
	lr.PoolID = set.ID

	tx.ClientID = "another_one"
	balances.txn = tx
	_, err = vsc.unlock(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "unlock_vesting_pool_failed: "+
		"only owner can unlock tokens from the pool")

	// 6. min lock
	tx.ClientID = client.id
	resp, err = vsc.unlock(tx, mustEncode(t, &lr), balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)

	var got *vestingPool
	got, err = vsc.getPool(set.ID, balances)
	require.NoError(t, err)
	assert.Equal(t, state.Balance(0), got.Balance)
}

func TestVestingSmartContract_trigger(t *testing.T) {
	var (
		vsc      = newTestVestingSC()
		balances = newTestBalances()
		client   = newClient(1200, balances)
		tp       = common.Timestamp(0)
		tx       = newTransaction(client.id, vsc.ID, 0, tp)
		conf     *config
		lr       lockRequest
		err      error
	)

	balances.txn = tx
	setConfig(t, balances)

	// 0. not allowed
	_, err = vsc.trigger(tx, nil, balances)
	assertErrMsg(t, err,
		"trigger_vesting_pool_failed: not allowed for this client")

	conf, err = vsc.getConfig(balances, false)
	require.NoError(t, err)
	conf.Triggers = append(conf.Triggers, tx.ClientID, "another_one")
	_, err = balances.InsertTrieNode(configKey(vsc.ID), conf)
	require.NoError(t, err)

	// 1. malformed
	_, err = vsc.trigger(tx, []byte("} malformed {"), balances)
	assertErrMsg(t, err, "trigger_vesting_pool_failed: invalid request:"+
		" invalid character '}' looking for beginning of value")

	// 2. pool_id = ""
	_, err = vsc.trigger(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "trigger_vesting_pool_failed: invalid request:"+
		" missing pool id")

	// 3. not found
	lr.PoolID = "pool_hex"
	_, err = vsc.trigger(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "trigger_vesting_pool_failed: "+
		"can't get pool: value not present")

	// 4. vesting is not started yet
	var resp string
	resp, err = client.add(t, vsc, &addRequest{
		Description:  "for something",
		StartTime:    10,
		Duration:     10 * time.Second,
		Friquency:    3 * time.Second,
		Destinations: []string{"one", "two"},
		Amount:       4000,
	}, 800, tp, balances)
	require.NoError(t, err)
	var set vestingPool
	require.NoError(t, set.Decode([]byte(resp)))
	lr.PoolID = set.ID

	tx.ClientID = "another_one"
	balances.txn = tx
	_, err = vsc.trigger(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "trigger_vesting_pool_failed: "+
		"early vesting")

	// 5. not enough tokens
	tx.CreationDate = 15
	_, err = vsc.trigger(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "trigger_vesting_pool_failed: "+
		"vesting: not enough tokens")

	// 6. vest
	set.Balance = 32000
	require.NoError(t, set.save(balances))
	resp, err = vsc.trigger(tx, mustEncode(t, &lr), balances)
	require.NoError(t, err)
	assert.NotZero(t, resp)

	var got *vestingPool
	got, err = vsc.getPool(set.ID, balances)
	require.NoError(t, err)
	assert.Equal(t, state.Balance(24000), got.Balance)

	// 7. expired pool
	set.Last = 19
	require.NoError(t, set.save(balances))
	tx.CreationDate = 21 // 10 (start_time) + 10 (duration) + 1 over
	_, err = vsc.trigger(tx, mustEncode(t, &lr), balances)
	assertErrMsg(t, err, "trigger_vesting_pool_failed: "+
		"expired pool")
}

func TestVestingSmartContract_getPoolInfoHandler(t *testing.T) {
	var (
		vsc      = newTestVestingSC()
		balances = newTestBalances()
		ctx      = context.Background()
		params   = make(url.Values)
		client   = newClient(0, balances)
		resp     interface{}
		err      error
	)
	setConfig(t, balances)
	params.Set("pool_id", "pool_unknown")

	_, err = vsc.getPoolInfoHandler(ctx, params, balances)
	require.Equal(t, util.ErrValueNotPresent, err)

	var set string
	set, err = client.add(t, vsc, &addRequest{
		Description:  "for something",
		StartTime:    10,
		Duration:     10 * time.Second,
		Friquency:    3 * time.Second,
		Destinations: []string{"one", "two"},
		Amount:       4000,
	}, 0, 0, balances)
	require.NoError(t, err)
	var deco vestingPool
	require.NoError(t, deco.Decode([]byte(set)))

	params.Set("pool_id", deco.ID)
	resp, err = vsc.getPoolInfoHandler(ctx, params, balances)
	require.NoError(t, err)
	assert.EqualValues(t, deco.info(), resp)
}
