package storagesc

import (
	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

type mockStateContext struct {
	ctx           cstate.StateContext
	clientBalance state.Balance
	//block                      *block.Block
	store map[datastore.Key]util.Serializable
	//sharders                   []string
	//LastestFinalizedMagicBlock *block.Block
}

func (sc *mockStateContext) SetMagicBlock(_ *block.MagicBlock)                     { return }
func (sc *mockStateContext) GetState() util.MerklePatriciaTrieI                    { return nil }
func (sc *mockStateContext) GetTransaction() *transaction.Transaction              { return nil }
func (sc *mockStateContext) GetSignedTransfers() []*state.SignedTransfer           { return nil }
func (sc *mockStateContext) Validate() error                                       { return nil }
func (sc *mockStateContext) GetSignatureScheme() encryption.SignatureScheme        { return nil }
func (sc *mockStateContext) AddSignedTransfer(_ *state.SignedTransfer)             { return }
func (sc *mockStateContext) DeleteTrieNode(_ datastore.Key) (datastore.Key, error) { return "", nil }

func (sc *mockStateContext) GetClientBalance(_ datastore.Key) (state.Balance, error) {
	return sc.clientBalance, nil
}

func (sc *mockStateContext) GetTransfers() []*state.Transfer {
	return sc.ctx.GetTransfers()
}

func (sc *mockStateContext) GetMints() []*state.Mint {
	return sc.ctx.GetMints()
}

func (sc *mockStateContext) GetLastestFinalizedMagicBlock() *block.Block {
	return nil
}

func (sc *mockStateContext) GetBlockSharders(_ *block.Block) []string {
	return nil
}

func (sc *mockStateContext) GetBlock() *block.Block {
	return nil
}

func (sc *mockStateContext) SetStateContext(_ *state.State) error { return nil }

func (sc *mockStateContext) GetTrieNode(key datastore.Key) (util.Serializable, error) {
	return sc.store[key], nil
}

func (sc *mockStateContext) InsertTrieNode(key datastore.Key, node util.Serializable) (datastore.Key, error) {
	sc.store[key] = node
	return key, nil
}

func (sc *mockStateContext) AddTransfer(t *state.Transfer) error {
	return sc.ctx.AddTransfer(t)
}

func (sc *mockStateContext) AddMint(m *state.Mint) error {
	return sc.ctx.AddMint(m)
}

func zcnToInt64(token float64) int64 {
	return int64(token * float64(x10))
}

func zcnToBalance(token float64) state.Balance {
	return state.Balance(token * float64(x10))
}

func confirmPoolLockResult(t *testing.T, f formulae, resp string, newStakePool stakePool,
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

	var minted = []bool{}
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
	require.EqualValues(t, transactionHash, respObj.Txn_hash)
	require.EqualValues(t, transactionHash, respObj.To_pool)
	require.EqualValues(t, f.value, respObj.Value)
	require.EqualValues(t, storageScId, respObj.To_client)
}

type formulae struct {
	value         int64
	clientBalance int64
	delegates     []mockStakePool
	offers        []common.Timestamp
	scYaml        scConfig
	now           common.Timestamp
}

func (f formulae) delegateInterest(delegate int) int64 {
	var interestRate = scYaml.StakePool.InterestRate
	var numberOfPayments = float64(f.numberOfInterestPayments(delegate))
	var stake = float64(zcnToInt64(f.delegates[delegate].zcnAmount))

	return int64(stake * numberOfPayments * interestRate)
}

func (f formulae) numberOfInterestPayments(delegate int) int64 {
	var activeTime = int64(f.now - f.delegates[delegate].MintAt)
	var period = int64(f.scYaml.StakePool.InterestInterval.Seconds())
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
