package storagesc

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/datastore"
	"0chain.net/core/util"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_newStakePool(t *testing.T) {
	var sp = newStakePool()
	assert.NotNil(t, sp.Pools)
}

func Test_stakePoolKey(t *testing.T) {
	assert.NotZero(t, stakePoolKey("scKey", "blobberID"))
}

func Test_stakePool_Encode_Decode(t *testing.T) {
	var spe, spd = newStakePool(), new(stakePool)
	require.NoError(t, spd.Decode(spe.Encode()))
	assert.EqualValues(t, spe, spd)
}

func Test_stakePool_save(t *testing.T) {
	const blobID = "blob_id"
	var (
		sp       = newStakePool()
		balances = newTestBalances(t, false)
	)
	require.NoError(t, sp.save(ADDRESS, blobID, balances))
	assert.NotZero(t, balances.tree[stakePoolKey(ADDRESS, blobID)])
}

type mockStakePool struct {
	zcnAmount float64
	MintAt    int64
}

const (
	blobberId        = "bob"
	transactionHash  = "12345678"
	clientId         = "sally"
	errDelta         = 6 // for testing values with rounding errors
	errStakePoolLock = "stake_pool_lock_failed: "
	errStakeTooSmall = "too small stake to lock"
)

type splResponse struct {
	Txn_hash    string
	To_pool     string
	Value       float64
	From_client string
	To_client   string
}

func TestStakePoolLock(t *testing.T) {
	var err error
	scYaml = &Config{
		MaxDelegates: 200,
		Minted:       zcnToBalance(0),
		MaxMint:      zcnToBalance(4000000.0),

		StakePool: &stakePoolConfig{
			MinLock: int64(zcnToBalance(0.1)),
		},
	}

	t.Run(errStakeTooSmall, func(t *testing.T) {
		var value = scYaml.StakePool.MinLock - 1
		creationDate = common.Timestamp(time.Second * 120)
		var delegates = []mockStakePool{{5, 0}}
		err = testStakePoolLock(t, value, value+1, delegates)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errStakePoolLock+errStakeTooSmall)
	})

	t.Run(errStakeTooSmall, func(t *testing.T) {
		scYaml.Minted = scYaml.MaxMint
		var value = scYaml.StakePool.MinLock - 1
		creationDate = common.Timestamp(time.Second * 120)
		var delegates = []mockStakePool{{5, 0}}
		err = testStakePoolLock(t, value, value+1, delegates)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errStakePoolLock))
	})
}

func testStakePoolLock(t *testing.T, value, clientBalance int64, delegates []mockStakePool) error {
	var f = formulaeStakePoolLock{
		value:         value,
		clientBalance: clientBalance,
		delegates:     delegates,
		scYaml:        *scYaml,
		now:           creationDate,
	}

	var txn = &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: datastore.Key(transactionHash),
		},

		ClientID:     clientId,
		ToClientID:   storageScId,
		Value:        value,
		CreationDate: creationDate,
	}
	var ctx = &mockStateContext{
		ctx: *cstate.NewStateContext(
			nil,
			&util.MerklePatriciaTrie{},
			txn,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		),
		clientBalance: state.Balance(clientBalance),
		store:         make(map[datastore.Key]util.MPTSerializable),
	}
	var ssc = &StorageSmartContract{
		&sci.SmartContract{
			ID: storageScId,
		},
	}
	_, err := ctx.InsertTrieNode(scConfigKey(ssc.ID), scYaml)
	require.NoError(t, err)
	var spr = &stakePoolRequest{
		BlobberID: blobberId,
		PoolID:    "paula",
	}
	input, err := json.Marshal(spr)
	require.NoError(t, err)
	var stakePool = newStakePool()
	for i, stake := range delegates {
		var id = strconv.Itoa(i)
		stakePool.Pools["pool"+id] = &stakepool.DelegatePool{
			Balance:      zcnToBalance(stake.zcnAmount),
			RoundCreated: stake.MintAt,
		}
	}
	var usp = stakepool.NewUserStakePools()
	require.NoError(t, usp.Save(spenum.Blobber, txn.ClientID, ctx))
	require.NoError(t, stakePool.save(ssc.ID, blobberId, ctx))

	resp, err := ssc.stakePoolLock(txn, input, ctx)
	if err != nil {
		return err
	}

	newStakePool, err := ssc.getStakePool(blobberId, ctx)
	require.NoError(t, err)
	var newUsp *stakepool.UserStakePools
	newUsp, err = stakepool.GetUserStakePool(spenum.Blobber, txn.ClientID, ctx)
	require.NoError(t, err)

	confirmPoolLockResult(t, f, resp, *newStakePool, *newUsp, ctx)

	return nil
}

func confirmPoolLockResult(t *testing.T, f formulaeStakePoolLock, resp string, newStakePool stakePool,
	newUsp stakepool.UserStakePools, ctx cstate.StateContextI) {
	for _, transfer := range ctx.GetTransfers() {
		require.EqualValues(t, f.value, int64(transfer.Amount))
		require.EqualValues(t, storageScId, transfer.ToClientID)
		require.EqualValues(t, clientId, transfer.ClientID)
		txPool, ok := newStakePool.Pools[transactionHash]
		require.True(t, ok)
		require.EqualValues(t, f.now, txPool.RoundCreated)
	}

	pools, ok := newUsp.Pools[blobberId]
	require.True(t, ok)
	require.Len(t, pools, 1)
	require.EqualValues(t, transactionHash, pools[0])

	var respObj = &splResponse{}
	require.NoError(t, json.Unmarshal([]byte(resp), respObj))
	require.EqualValues(t, transactionHash, respObj.Txn_hash)
	require.EqualValues(t, transactionHash, respObj.To_pool)
	require.EqualValues(t, f.value, respObj.Value)
	require.EqualValues(t, storageScId, respObj.To_client)
}

type formulaeStakePoolLock struct {
	value         int64
	clientBalance int64
	delegates     []mockStakePool
	scYaml        Config
	now           common.Timestamp
}
