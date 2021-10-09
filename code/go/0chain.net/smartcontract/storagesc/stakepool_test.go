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
		clientID = "client_id"
	)

	var (
		sp       = newStakePool()
		balances = newTestBalances(t, false)
		tx       = transaction.Transaction{
			ClientID:   clientID,
			ToClientID: ADDRESS,
			Value:      90,
		}
		err error
	)

	balances.setTransaction(t, &tx)
	balances.balances[clientID] = 100e10

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
	transactionHash  = "12345678"
	clientId         = "sally"
	errDelta         = 0 // for testing values with rounding errors
	offerId          = "offer"
	errStakePoolLock = "stake_pool_lock_failed: "
	errStakeTooSmall = "too small stake to lock"
)

type splResponse struct {
	TxnHash string
	ToPool  string
	Value      float64
	FromClient string
	ToClient   string
}

func TestStakePoolLock(t *testing.T) {
	var err error
	scYaml = &scConfig{
		MaxDelegates: 200,
		Minted:       zcnToBalance(0),
		MaxMint:      zcnToBalance(4000000.0),
		StakePool: &stakePoolConfig{
			MinLock:          int64(zcnToBalance(0.1)),
		},
	}

	t.Run("stake pool lock", func(t *testing.T) {
		var value = 10 * scYaml.StakePool.MinLock

		var period = common.Timestamp(3600)
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

		var period = common.Timestamp(3600)
		creationDate = period * 2
		var delegates = []mockStakePool{{5, 0}}
		var offers []common.Timestamp
		err = testStakePoolLock(t, value, value+1, delegates, offers)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errStakePoolLock+errStakeTooSmall)
	})

	t.Run(errStakeTooSmall, func(t *testing.T) {
		scYaml.Minted = scYaml.MaxMint
		var value = 10 * scYaml.StakePool.MinLock
		var period = common.Timestamp(3600)
		creationDate = period * 2
		var delegates = []mockStakePool{{5, 0}}
		var offers []common.Timestamp
		err = testStakePoolLock(t, value, value+1, delegates, offers)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errStakePoolLock))
	})
}

func testStakePoolLock(t *testing.T, value, clientBalance int64, delegates []mockStakePool, offers []common.Timestamp) error {
	var f = formulaeStakePoolLock{
		value:         value,
		clientBalance: clientBalance,
		delegates:     delegates,
		offers:        offers,
		scYaml:        *scYaml,
		now:           creationDate,
	}

	var txn = &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: transactionHash,
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

func confirmPoolLockResult(t *testing.T, f formulaeStakePoolLock, resp string, newStakePool stakePool,
	newUsp userStakePools, ctx cstate.StateContextI) {
	for _, transfer := range ctx.GetTransfers() {
		require.EqualValues(t, f.value, int64(transfer.Amount))
		require.EqualValues(t, storageScId, transfer.ToClientID)
		require.EqualValues(t, clientId, transfer.ClientID)
		txPool, ok := newStakePool.Pools[transactionHash]
		require.True(t, ok)
		require.EqualValues(t, clientId, txPool.DelegateID)
		require.EqualValues(t, f.now, txPool.MintAt)
	}

	var minted []bool
	for range f.delegates {
		minted = append(minted, false)
	}
	for _, mint := range ctx.GetMints() {
		index, err := strconv.Atoi(mint.ToClientID)
		require.NoError(t, err)
		require.InDelta(t, f.delegateInterest(index), int64(mint.Amount), errDelta)
		require.EqualValues(t, storageScId, mint.Minter)
		minted[index] = true
	}
	for delegate, wasMinted := range minted {
		if !wasMinted {
			require.EqualValues(t, f.delegateInterest(delegate), 0, errDelta)
		}
	}

	for offer, expires := range f.offers {
		var key = offerId + strconv.Itoa(offer)
		_, ok := newStakePool.Offers[key]
		require.EqualValues(t, expires > f.now, ok)
	}
	pools, ok := newUsp.Pools[blobberId]
	require.True(t, ok)
	require.Len(t, pools, 1)
	require.EqualValues(t, transactionHash, pools[0])

	var respObj = &splResponse{}
	require.NoError(t, json.Unmarshal([]byte(resp), respObj))
	require.EqualValues(t, transactionHash, respObj.TxnHash)
	require.EqualValues(t, transactionHash, respObj.ToPool)
	require.EqualValues(t, f.value, respObj.Value)
	require.EqualValues(t, storageScId, respObj.ToClient)
}

type formulaeStakePoolLock struct {
	value         int64
	clientBalance int64
	delegates     []mockStakePool
	offers        []common.Timestamp
	scYaml        scConfig
	now           common.Timestamp
}

func (f formulaeStakePoolLock) delegateInterest(delegate int) int64 {
	var numberOfPayments = float64(f.numberOfInterestPayments(delegate))
	var stake = float64(zcnToInt64(f.delegates[delegate].zcnAmount))
	return int64(stake * numberOfPayments)
}

func (f formulaeStakePoolLock) numberOfInterestPayments(delegate int) int64 {
	var activeTime = int64(f.now - f.delegates[delegate].MintAt)
	var period = int64(3600)
	var periods = activeTime / period

	// round down to previous integer
	if activeTime%period == 0 {
		if periods-1 >= 0 {
			return periods - 1
		} else {
			return 0
		}
	} else {
		return periods
	}
}
