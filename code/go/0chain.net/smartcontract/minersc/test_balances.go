package minersc

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"github.com/stretchr/testify/require"
	"testing"
)

type testBalances struct {
	balances      map[datastore.Key]state.Balance
	txn           *transaction.Transaction
	transfers     []*state.Transfer
	tree          map[datastore.Key]util.Serializable
	block         *block.Block
	blockSharders []string
	lfmb          *block.Block
}

func newTestBalances() *testBalances {
	return &testBalances{
		balances: make(map[datastore.Key]state.Balance),
		tree:     make(map[datastore.Key]util.Serializable),
	}
}

func (tb *testBalances) zeroize() {
	tb.balances = make(map[string]state.Balance)
}

func (tb *testBalances) setBalance(key datastore.Key, b state.Balance) {
	tb.balances[key] = b
}

func (tb *testBalances) setLFMB(lfmb *block.Block) {
	tb.lfmb = lfmb
}

func (tb *testBalances) requireAllBeZeros(t *testing.T) {
	for id, value := range tb.balances {
		if id == ADDRESS {
			continue
		}
		require.Zerof(t, value, "%s has non-zero balance: %d", id, value)
	}
}

func (tb *testBalances) requireSpecifiedBeEqual(t *testing.T,
	clients []*Client, value state.Balance, message string) {

	for _, client := range clients {
		require.EqualValues(t, value, tb.balances[client.id], message)
	}
}

func (tb *testBalances) requireTotalAmountBeEqual(t * testing.T,
	expected state.Balance) {

	var total state.Balance
	for id, value := range tb.balances {
		if id == ADDRESS {
			continue
		}
		total += value
	}

	require.EqualValues(t, expected, total, "total amount of tokens is wrong")
}

func (tb *testBalances) requireNodeAndStakersSumUpTo(t *testing.T,
	node *Client, stakers []*Client, expected state.Balance) {

	var total state.Balance
	for _, staker := range stakers {
		total += tb.balances[staker.id]
	}
	total += tb.balances[node.id]

	require.EqualValues(t, expected, total,
		"total amount distributed among node and its stakers is wrong")
}

func (tb *testBalances) GetBlock() *block.Block {
	return tb.block
}

func (tb *testBalances) SetMagicBlock(mb *block.MagicBlock) {
	if tb.block != nil {
		tb.block.MagicBlock = mb
	}
}

func (tb *testBalances) GetBlockSharders(*block.Block) []string {
	return tb.blockSharders
}

// stubs
func (tb *testBalances) GetState() util.MerklePatriciaTrieI       { return nil }
func (tb *testBalances) GetTransaction() *transaction.Transaction { return nil }
func (tb *testBalances) Validate() error                          { return nil }
func (tb *testBalances) GetMints() []*state.Mint                  { return nil }
func (tb *testBalances) SetStateContext(*state.State) error       { return nil }
func (tb *testBalances) GetTransfers() []*state.Transfer          { return nil }
func (tb *testBalances) AddSignedTransfer(st *state.SignedTransfer) {
}
func (tb *testBalances) GetSignedTransfers() []*state.SignedTransfer {
	return nil
}
func (tb *testBalances) DeleteTrieNode(datastore.Key) (datastore.Key, error) {
	return "", nil
}
func (tb *testBalances) GetLastestFinalizedMagicBlock() *block.Block {
	return tb.lfmb
}

func (tb *testBalances) GetSignatureScheme() encryption.SignatureScheme {
	return encryption.NewBLS0ChainScheme()
}

func (tb *testBalances) GetClientBalance(clientID datastore.Key) (
	b state.Balance, err error) {

	var ok bool
	if b, ok = tb.balances[clientID]; !ok {
		return 0, util.ErrValueNotPresent
	}
	return
}

func (tb *testBalances) GetTrieNode(key datastore.Key) (
	node util.Serializable, err error) {

	if encryption.IsHash(key) {
		return nil, common.NewError("failed to get trie node",
			"key is too short")
	}

	var ok bool
	if node, ok = tb.tree[key]; !ok {
		return nil, util.ErrValueNotPresent
	}
	return
}

func (tb *testBalances) InsertTrieNode(key datastore.Key,
	node util.Serializable) (_ datastore.Key, _ error) {

	tb.tree[key] = node
	return
}

func (tb *testBalances) AddTransfer(t *state.Transfer) error {
	if t.Sender != tb.txn.ClientID && t.Sender != tb.txn.ToClientID {
		return state.ErrInvalidTransfer
	}
	tb.balances[t.Sender] -= t.Amount
	tb.balances[t.Receiver] += t.Amount
	tb.transfers = append(tb.transfers, t)
	return nil
}

func (tb *testBalances) AddMint(mint *state.Mint) error {
	if mint.Minter != ADDRESS {
		panic("invalid miner: " + mint.Minter)
	}
	tb.balances[mint.Receiver] += mint.Amount // mint!
	return nil
}
