package storagesc

import (
	"testing"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_newStakePool(t *testing.T) {
	var sp = newStakePool()
	assert.NotNil(t, sp.Locked)
	assert.NotNil(t, sp.Unlocked)
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
	assert.Zero(t, sp.offersStake(now))
	sp.Offers["alloc_id"] = &offerPool{
		Lock:   90,
		Expire: now,
	}
	assert.Equal(t, state.Balance(90), sp.offersStake(now))
}

func Test_stakePool_save(t *testing.T) {
	const blobID = "blob_id"
	var (
		sp       = newStakePool()
		balances = newTestBalances()
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
		balances = newTestBalances()
		tx       = transaction.Transaction{
			ClientID:   clienID,
			ToClientID: ADDRESS,
			Value:      90,
		}
		err error
	)

	balances.txn = &tx

	_, _, err = sp.fill(&tx, balances)
	require.NoError(t, err)
	assert.Equal(t, state.Balance(90), sp.Locked.Balance)
}

func Test_stakePool_addOffer(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_stakePool_findOffer(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_stakePool_extendOffer(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_maxBalance(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_minBalance(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_stakePool_update(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_stakePool_dryUpdate(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_stakePool_stat(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func Test_stakePoolStat_encode_decode(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_getStakePoolBytes(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_getStakePool(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_newStakePool(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_stakePoolUnlock(t *testing.T) {
	// TODO (sfxdx): implements tests
}

func TestStorageSmartContract_getStakePoolStatHandler(t *testing.T) {
	// TODO (sfxdx): implements tests
}
