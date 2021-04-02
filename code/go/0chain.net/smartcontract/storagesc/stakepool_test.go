package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_newStakePool(t *testing.T) {
	var sp = newStakePool()
	assert.NotNil(t, sp.Pools)
	assert.NotNil(t, sp.Offers)
}

func Test_stakePoolKey(t *testing.T) {
	assert.NotZero(t, stakePoolKey("scKey", "blobberID"))
}

func Test_stakePool_Encode_Decode(t *testing.T) {
	var spe, spd = newStakePool(), new(stakePool)
	require.NoError(t, spd.Decode(spe.Encode()))
	assert.EqualValues(t, spe, spd)
}

func Test_stakePool_offersStake(t *testing.T) {
	var (
		sp  = newStakePool()
		now = common.Now()
	)
	assert.Zero(t, sp.offersStake(now, false))
	sp.Offers["alloc_id"] = &offerPool{
		Lock:   90,
		Expire: now,
	}
	assert.Equal(t, state.Balance(90), sp.offersStake(now-1, false))
	assert.Equal(t, state.Balance(0), sp.offersStake(now, false))
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

func Test_stakePool_fill(t *testing.T) {
	const (
		clienID = "clien_id"
		txHash  = "tx_hash"
	)

	var (
		sp       = newStakePool()
		balances = newTestBalances(t, false)
		tx       = transaction.Transaction{
			ClientID:   clienID,
			ToClientID: ADDRESS,
			Value:      90,
		}
		err error
	)

	balances.setTransaction(t, &tx)
	balances.balances[clienID] = 100e10

	_, _, err = sp.dig(&tx, balances)
	require.NoError(t, err)
	assert.Equal(t, state.Balance(90), sp.stake())
}

type mockStakePool struct {
	zcnAmount float64
	MintAt    common.Timestamp
}

const (
	blobberId        = "bob"
	transactionHash  = "my tx hash"
	clientId         = "sally"
	errDelta         = 4 // for testing values with rounding errors
	offerId          = "offer"
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

var (
	creationDate    = common.Timestamp(100)
	approvedMinters = []string{
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9", // miner SC
		"cf8d0df9bd8cc637a4ff4e792ffe3686da6220c45f0e1103baa609f3f1751ef4", // interest SC
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7", // storage SC
	}
	storageScId = approvedMinters[2]
	scYaml      = &scConfig{
		MaxDelegates: 200,
		Minted:       zcnToBalance(0),
		MaxMint:      zcnToBalance(4000000.0),

		StakePool: &stakePoolConfig{
			InterestRate:     0.0000334,
			InterestInterval: 1 * time.Minute,
			MinLock:          int64(zcnToBalance(0.1)),
		},
	}
)

func TestStakePoolLock(t *testing.T) {
	var err error

	t.Run("stake pool lock", func(t *testing.T) {
		var value = 10 * scYaml.StakePool.MinLock

		var period = common.Timestamp(scYaml.StakePool.InterestInterval.Seconds())
		creationDate = period * 2
		var delegates = []mockStakePool{
			{2, creationDate - period - 1},
			{3, creationDate - period + 1},
			{5, 0},
			{3, creationDate - period},
		}
		var offers = []common.Timestamp{creationDate + 1, creationDate - 1, creationDate}
		err = testStakePoolLock(t, value, value+1, delegates, offers)
		require.NoError(t, err)
	})

	t.Run(errStakeTooSmall, func(t *testing.T) {
		var value = scYaml.StakePool.MinLock - 1

		var period = common.Timestamp(scYaml.StakePool.InterestInterval.Seconds())
		creationDate = period * 2
		var delegates = []mockStakePool{{5, 0}}
		var offers = []common.Timestamp{}
		err = testStakePoolLock(t, value, value+1, delegates, offers)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errStakePoolLock+errStakeTooSmall)
	})

	t.Run(errStakeTooSmall, func(t *testing.T) {
		t.Skip("no error returned when minted reaches max mint")
		scYaml.Minted = scYaml.MaxMint
		var value = 10 * scYaml.StakePool.MinLock
		var period = common.Timestamp(scYaml.StakePool.InterestInterval.Seconds())
		creationDate = period * 2
		var delegates = []mockStakePool{{5, 0}}
		var offers = []common.Timestamp{}
		err = testStakePoolLock(t, value, value+1, delegates, offers)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errStakePoolLock))
	})
}

func testStakePoolLock(t *testing.T, value, clientBalance int64, delegates []mockStakePool, offers []common.Timestamp) error {
	var f = formulae{
		value:         value,
		clientBalance: clientBalance,
		delegates:     delegates,
		offers:        offers,
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
			&state.Deserializer{},
			txn,
			nil,
			nil,
			nil,
		),
		clientBalance: state.Balance(clientBalance),
		store:         make(map[datastore.Key]util.Serializable),
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
		stakePool.Pools["pool"+id] = &delegatePool{
			DelegateID: strconv.Itoa(i),
			ZcnPool: tokenpool.ZcnPool{
				TokenPool: tokenpool.TokenPool{
					ID:      id,
					Balance: zcnToBalance(stake.zcnAmount),
				},
			},
			MintAt: stake.MintAt,
		}
	}
	for i, expires := range offers {
		var id = strconv.Itoa(i)
		stakePool.Offers[offerId+id] = &offerPool{
			//Lock:   zcnToBalance(offer.zcn),
			Expire: expires,
		}
	}

	var usp = newUserStakePools()
	require.NoError(t, usp.save(ssc.ID, txn.ClientID, ctx))
	require.NoError(t, stakePool.save(ssc.ID, blobberId, ctx))

	resp, err := ssc.stakePoolLock(txn, input, ctx)
	if err != nil {
		return err
	}

	newStakePool, err := ssc.getStakePool(blobberId, ctx)
	require.NoError(t, err)
	var newUsp *userStakePools
	newUsp, err = ssc.getOrCreateUserStakePool(txn.ClientID, ctx)
	require.NoError(t, err)

	confirmPoolLockResult(t, f, resp, *newStakePool, *newUsp, ctx)

	return nil
}
