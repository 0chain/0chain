package storagesc

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustEncode(t *testing.T, val interface{}) []byte {
	var err error
	b, err := json.Marshal(val)
	require.NoError(t, err)
	return b
}

func Test_lockRequest_decode(t *testing.T) {
	var lre, lrd lockRequest
	lre.Duration = time.Second * 60
	require.NoError(t, lrd.decode(mustEncode(t, lre)))
	assert.EqualValues(t, lrd, lre)
}

func requireErrMsg(t *testing.T, err error, msg string) {
	t.Helper()
	require.Error(t, err, "missing error")
	require.Equal(t, msg, err.Error(), "unexpected error")
}

func Test_lockRequest_validate(t *testing.T) {
	const (
		allocID = "alloc_hex"
		errMsg1 = "insufficient lock period"
		errMsg2 = "lock duration is too big"
		errMsg3 = "missing allocation id"
		errMsg4 = "missing allocation blobbers"
		errMsg5 = "doesn't belong to allocation"
	)
	var (
		lr    lockRequest
		conf  readPoolConfig
		alloc StorageAllocation
		err   error
	)
	alloc.BlobberMap = map[string]*BlobberAllocation{
		"blob1": &BlobberAllocation{BlobberID: "blob1"},
		"blob2": &BlobberAllocation{BlobberID: "blob2"},
		"blob3": &BlobberAllocation{BlobberID: "blob3"},
	}
	conf.MinLockPeriod = 10 * time.Second
	conf.MaxLockPeriod = 20 * time.Second
	lr.Duration = 9 * time.Second
	err = lr.validate(&conf, &alloc)
	requireErrMsg(t, err, errMsg1)
	lr.Duration = 21 * time.Second
	err = lr.validate(&conf, &alloc)
	requireErrMsg(t, err, errMsg2)
	lr.Duration = 15 * time.Second
	err = lr.validate(&conf, &alloc)
	requireErrMsg(t, err, errMsg3)
	lr.AllocationID = allocID
	err = lr.validate(&conf, &alloc)
	requireErrMsg(t, err, errMsg4)
	lr.Blobbers = []string{"unknown"}
	err = lr.validate(&conf, &alloc)
	require.Error(t, err, "missing error")
	require.True(t, strings.Contains(err.Error(), errMsg5), "unexpected error")
	lr.Blobbers = []string{"blob1", "blob2", "blob3"}
	err = lr.validate(&conf, &alloc)
	require.NoError(t, err)
}

func Test_readPoolLock_copy(t *testing.T) {
	var rpl readPoolLock
	rpl.StartTime = common.Now()
	rpl.Duration = 20 * time.Second
	rpl.Value = 50
	var cp = rpl.copy()
	require.False(t, (*cp) != rpl)
}

func Test_readPoolBlobbers_copy(t *testing.T) {
	var rpbs = readPoolBlobbers{
		"blob1": []*readPoolLock{
			&readPoolLock{StartTime: 10, Duration: 10 * time.Second, Value: 10},
			&readPoolLock{StartTime: 20, Duration: 20 * time.Second, Value: 20},
			&readPoolLock{StartTime: 30, Duration: 30 * time.Second, Value: 30},
		},
		"blob2": []*readPoolLock{
			&readPoolLock{StartTime: 40, Duration: 40 * time.Second, Value: 40},
			&readPoolLock{StartTime: 50, Duration: 50 * time.Second, Value: 50},
			&readPoolLock{StartTime: 60, Duration: 60 * time.Second, Value: 60},
		},
	}
	require.EqualValues(t, rpbs.copy(), rpbs)
}

func Test_readPoolAllocs_update(t *testing.T) {
	var (
		rpa readPoolAllocs
		tp  = common.Now()
	)
	require.Zero(t, rpa.update(common.Now()))
	rpa = make(readPoolAllocs)
	// all fresh
	rpa["alloc1"] = readPoolBlobbers{
		"blob11": []*readPoolLock{
			&readPoolLock{
				StartTime: tp + 20,
				Duration:  75 * time.Second,
				Value:     100,
			},
		},
	}
	// all outdated
	rpa["alloc2"] = readPoolBlobbers{
		"blob21": []*readPoolLock{
			&readPoolLock{
				StartTime: tp - 20,
				Duration:  10 * time.Second,
				Value:     50,
			},
			&readPoolLock{
				StartTime: tp - 20,
				Duration:  10 * time.Second,
				Value:     73,
			},
		},
	}
	require.True(t, rpa.update(tp) == 50+73)
	require.EqualValues(t, readPoolAllocs{
		"alloc1": readPoolBlobbers{
			"blob11": []*readPoolLock{
				&readPoolLock{
					StartTime: tp + 20,
					Duration:  75 * time.Second,
					Value:     100,
				},
			},
		},
	}, rpa)
}

func Test_readPoolAllocs_copy(t *testing.T) {
	var (
		rpa readPoolAllocs
		tp  = common.Now()
	)
	rpa = make(readPoolAllocs)
	rpa["alloc1"] = readPoolBlobbers{
		"blob11": []*readPoolLock{
			&readPoolLock{
				StartTime: tp + 20,
				Duration:  75 * time.Second,
				Value:     100,
			},
		},
	}
	rpa["alloc2"] = readPoolBlobbers{
		"blob21": []*readPoolLock{
			&readPoolLock{
				StartTime: tp - 20,
				Duration:  10 * time.Second,
				Value:     50,
			},
			&readPoolLock{
				StartTime: tp - 20,
				Duration:  10 * time.Second,
				Value:     73,
			},
		},
	}
	require.EqualValues(t, rpa, rpa.copy())
}

func Test_newReadPool(t *testing.T) {
	var rp = newReadPool()
	require.NotNil(t, rp.Locked)
	require.NotNil(t, rp.Unlocked)
	require.NotNil(t, rp.Locks)
}

func Test_readPool_Encode_Decode(t *testing.T) {
	const allocID, dur, value = "alloc_hex", 10 * time.Second, 150
	var (
		now      = common.Now()
		rpe, rpd = newReadPool(), newReadPool()
		blobbers = []string{"b1_hex", "b2_hex", "b3_hex"}
		alloc    = StorageAllocation{
			BlobberDetails: []*BlobberAllocation{
				&BlobberAllocation{
					BlobberID: "b1_hex", Terms: Terms{ReadPrice: 10},
				},
				&BlobberAllocation{
					BlobberID: "b2_hex", Terms: Terms{ReadPrice: 10},
				},
				&BlobberAllocation{
					BlobberID: "b3_hex", Terms: Terms{ReadPrice: 10},
				},
			},
		}
		lr lockRequest
	)

	lr.AllocationID = "all"
	lr.Blobbers = blobbers
	lr.Duration = dur

	require.Zero(t, rpe.addLocks(now, value, &lr, &alloc))
	require.NoError(t, rpd.Decode(rpe.Encode()))
	require.EqualValues(t, rpe, rpd)
}

func Test_readPoolsKey(t *testing.T) {
	require.NotZero(t, readPoolKey("scKey", "clientID"))
}

func Test_readPool_fill(t *testing.T) {
	const sscKey, clientID, txHash = "ssc_hex", "client_hex", "tx_hash"
	var (
		rp       = newReadPool()
		balances = newTestBalances()
		tx       = &transaction.Transaction{
			ClientID:   clientID,
			ToClientID: sscKey,
			Value:      90,
		}
	)
	tx.Hash = txHash
	balances.txn = tx
	var transfer, resp, err = rp.fill(tx, balances)
	require.NoError(t, err)
	require.NotZero(t, resp)
	require.NotNil(t, transfer)
	require.True(t, rp.Locked.Balance == 90)
}

func Test_readPool_addLocks(t *testing.T) {
	//
}

func Test_readPool_update(t *testing.T) {
	var (
		rp = newReadPool()
		tp = common.Now()
	)
	// all fresh
	rp.Locks["alloc1"] = readPoolBlobbers{
		"blob11": []*readPoolLock{
			&readPoolLock{
				StartTime: tp + 20,
				Duration:  75 * time.Second,
				Value:     100,
			},
		},
	}
	// all outdated
	rp.Locks["alloc2"] = readPoolBlobbers{
		"blob21": []*readPoolLock{
			&readPoolLock{
				StartTime: tp - 20,
				Duration:  10 * time.Second,
				Value:     50,
			},
			&readPoolLock{
				StartTime: tp - 20,
				Duration:  10 * time.Second,
				Value:     73,
			},
		},
	}
	rp.Locked.Balance = 100 + 50 + 73
	var ul, err = rp.update(tp)
	require.NoError(t, err)
	assert.True(t, ul == 50+73)
	assert.True(t, rp.Locked.Balance == 100)
	assert.True(t, rp.Unlocked.Balance == 50+73)

	ul, err = rp.update(tp)
	require.NoError(t, err)
	assert.Zero(t, ul)
}

func Test_readPool_save(t *testing.T) {
	const sscID, clientID = "ssc_hex", "client_hex"
	var (
		rp       = newReadPool()
		balances = newTestBalances()
		err      = rp.save(sscID, clientID, balances)
	)
	require.NoError(t, err)
	assert.NotZero(t, balances.tree[readPoolKey(sscID, clientID)])
}

func Test_readPool_stat(t *testing.T) {
	var (
		rp = newReadPool()
		tp = common.Now()
	)
	// all fresh
	rp.Locks["alloc1"] = readPoolBlobbers{
		"blob11": []*readPoolLock{
			&readPoolLock{
				StartTime: tp + 20,
				Duration:  75 * time.Second,
				Value:     100,
			},
		},
	}
	// all outdated
	rp.Locks["alloc2"] = readPoolBlobbers{
		"blob21": []*readPoolLock{
			&readPoolLock{
				StartTime: tp - 20,
				Duration:  10 * time.Second,
				Value:     50,
			},
			&readPoolLock{
				StartTime: tp - 20,
				Duration:  10 * time.Second,
				Value:     73,
			},
		},
	}
	rp.Locked.Balance = 100 + 50 + 73
	rp.Unlocked.Balance = 200
	var stat = rp.stat()
	assert.EqualValues(t, &readPoolStat{
		Locked:   100,
		Unlocked: 200 + 50 + 73,
		Locks: readPoolAllocs{
			"alloc1": readPoolBlobbers{
				"blob11": []*readPoolLock{
					&readPoolLock{
						StartTime: tp + 20,
						Duration:  75 * time.Second,
						Value:     100,
					},
				},
			},
		},
	}, stat)
}

func TestStorageSmartContract_getReadPoolBytes(t *testing.T) {
	const clientID = "client_hex"
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		b, err   = ssc.getReadPoolBytes(clientID, balances)
	)
	require.Equal(t, util.ErrValueNotPresent, err)
	var rp = newReadPool()
	require.NoError(t, rp.save(ssc.ID, clientID, balances))
	b, err = ssc.getReadPoolBytes(clientID, balances)
	require.NoError(t, err)
	assert.Equal(t, b, rp.Encode())
}

func TestStorageSmartContract_getReadPool(t *testing.T) {
	const clientID = "client_hex"
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		rp, err  = ssc.getReadPool(clientID, balances)
	)
	require.Equal(t, util.ErrValueNotPresent, err)
	var nrp = newReadPool()
	require.NoError(t, nrp.save(ssc.ID, clientID, balances))
	rp, err = ssc.getReadPool(clientID, balances)
	require.NoError(t, err)
	assert.EqualValues(t, nrp, rp)
}

func TestStorageSmartContract_newReadPool(t *testing.T) {
	const clientID = "client_hex"
	var (
		ssc       = newTestStorageSC()
		balances  = newTestBalances()
		tx        = transaction.Transaction{ClientID: clientID}
		resp, err = ssc.newReadPool(&tx, nil, balances)
	)
	require.NoError(t, err)
	assert.NotZero(t, resp)
	var rp = newReadPool()
	require.NoError(t, rp.Decode([]byte(resp)))
	assert.Equal(t, readPoolKey(ssc.ID, clientID)+":locked", rp.Locked.ID)
	assert.Equal(t, readPoolKey(ssc.ID, clientID)+":unlocked", rp.Unlocked.ID)
	require.NotZero(t, balances.tree[readPoolKey(ssc.ID, clientID)])

	_, err = ssc.newReadPool(&tx, nil, balances)
	assert.Error(t, err)
}

func TestStorageSmartContract_checkFill(t *testing.T) {
	const (
		clientID = "client_hex"
		errMsg1  = "no tokens to lock"
		errMsg2  = "lock amount is greater than balance"
		errMsg3  = "negative transaction value given"
	)
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		tx       = transaction.Transaction{ClientID: clientID, Value: 100}
	)
	requireErrMsg(t, ssc.checkFill(&tx, balances), errMsg1)
	balances.balances[clientID] = 90
	requireErrMsg(t, ssc.checkFill(&tx, balances), errMsg2)
	tx.Value = -1
	requireErrMsg(t, ssc.checkFill(&tx, balances), errMsg3)
}

func TestStorageSmartContract_readPoolLock(t *testing.T) {
	const (
		allocID, clientID = "alloc_hex", "client_hex"

		errMsg1 = "read_pool_lock_failed: insufficient amount to lock"
		errMsg2 = "read_pool_lock_failed: invalid character '}' looking for" +
			" beginning of value"
		errMsg3 = "read_pool_lock_failed: can't get allocation:" +
			" value not present"
		errMsg4 = "read_pool_lock_failed: blobber \"blob4\" doesn't belong" +
			" to allocation \"alloc_hex\""
		errMsg5 = "read_pool_lock_failed: can't get read pool:" +
			" value not present"
		errMsg6 = "read_pool_lock_failed: no tokens to lock"
	)

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances()
		tx       = transaction.Transaction{
			ClientID: clientID, ToClientID: ssc.ID, Value: 5,
			CreationDate: common.Now(),
		}
		lr    lockRequest
		alloc StorageAllocation
		input []byte
		conf  scConfig
		resp  string
		err   error
	)

	// setup SC configurations
	conf.ReadPool = new(readPoolConfig)
	conf.ReadPool.MinLock = 10
	conf.ReadPool.MinLockPeriod = 10 * time.Second
	conf.ReadPool.MaxLockPeriod = 100 * time.Second

	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), &conf)
	require.NoError(t, err)

	// 1. value is less then configured min lock
	_, err = ssc.readPoolLock(&tx, nil, balances)
	requireErrMsg(t, err, errMsg1)

	// 2. invalid request JSON
	tx.Value = 100
	input = []byte("}{ invalid JSON }{")
	_, err = ssc.readPoolLock(&tx, input, balances)
	requireErrMsg(t, err, errMsg2)

	// setup request
	lr.AllocationID = allocID
	lr.Blobbers = []string{"blob1", "blob2", "blob3", "blob4"}
	lr.Duration = 20 * time.Second

	// 3. missing allocation
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg3)

	// setup allocation
	alloc.ID = allocID
	alloc.BlobberDetails = []*BlobberAllocation{
		&BlobberAllocation{BlobberID: "blob1", Terms: Terms{ReadPrice: 10}},
		&BlobberAllocation{BlobberID: "blob2", Terms: Terms{ReadPrice: 20}},
		&BlobberAllocation{BlobberID: "blob3", Terms: Terms{ReadPrice: 40}},
	}
	alloc.Blobbers = []*StorageNode{
		&StorageNode{ID: "blob1"},
		&StorageNode{ID: "blob2"},
		&StorageNode{ID: "blob3"},
	}
	_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), &alloc)
	require.NoError(t, err)

	// 4. blobber not of the allocation
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg4)

	// 5. missing read pool
	lr.Blobbers = []string{"blob1", "blob2", "blob3"}
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg5)

	var rp = newReadPool()
	require.NoError(t, rp.save(ssc.ID, clientID, balances))

	// 6. client has no money
	_, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	requireErrMsg(t, err, errMsg6)

	balances.balances[clientID] = 500
	balances.txn = &tx

	// 7. ok
	resp, err = ssc.readPoolLock(&tx, mustEncode(t, &lr), balances)
	require.NoError(t, err)
	assert.Equal(t, `{"value":100,"from_client":"`+clientID+`","to_client":`+
		`"`+ssc.ID+`"}`,
		resp)
	if assert.Len(t, balances.transfers, 1) {
		assert.EqualValues(t, &state.Transfer{
			ClientID:   clientID,
			ToClientID: ssc.ID,
			Amount:     100,
		}, balances.transfers[0])
	}
	rp, err = ssc.getReadPool(clientID, balances)
	require.NoError(t, err)
	assert.Equal(t, rp.Locked.Balance, state.Balance(99))
	assert.Equal(t, rp.Unlocked.Balance, state.Balance(1))
	assert.EqualValues(t, readPoolAllocs{
		allocID: readPoolBlobbers{
			"blob1": []*readPoolLock{
				&readPoolLock{
					StartTime: tx.CreationDate,
					Duration:  lr.Duration,
					Value:     14,
				},
			},
			"blob2": []*readPoolLock{
				&readPoolLock{
					StartTime: tx.CreationDate,
					Duration:  lr.Duration,
					Value:     28,
				},
			},
			"blob3": []*readPoolLock{
				&readPoolLock{
					StartTime: tx.CreationDate,
					Duration:  lr.Duration,
					Value:     57,
				},
			},
		},
	}, rp.Locks)
}

func TestStorageSmartContract_readPoolUnlock(t *testing.T) {
	// TODO (sfxdx): implement
}

func TestStorageSmartContract_getReadPoolStatHandler(t *testing.T) {
	// TODO (sfxdx): implement
}

func TestStorageSmartContract_getReadPoolBlobberHandler(t *testing.T) {
	// TODO (sfxdx): implement
}
