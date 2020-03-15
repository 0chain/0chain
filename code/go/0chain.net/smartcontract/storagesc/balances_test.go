package storagesc

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

//
// helper for tests implements chainState.StateContextI
//

type testBalances struct {
	balances  map[datastore.Key]state.Balance
	txn       *transaction.Transaction
	transfers []*state.Transfer
	tree      map[datastore.Key]util.Serializable
}

func newTestBalances() *testBalances {
	return &testBalances{
		balances: make(map[datastore.Key]state.Balance),
		tree:     make(map[datastore.Key]util.Serializable),
	}
}

func (tb *testBalances) setBalance(key datastore.Key, b state.Balance) {
	tb.balances[key] = b
}

// stubs
func (tb *testBalances) GetBlock() *block.Block                   { return nil }
func (tb *testBalances) GetState() util.MerklePatriciaTrieI       { return nil }
func (tb *testBalances) GetTransaction() *transaction.Transaction { return nil }
func (tb *testBalances) GetBlockSharders(b *block.Block) []string { return nil }
func (tb *testBalances) Validate() error                          { return nil }
func (tb *testBalances) GetMints() []*state.Mint                  { return nil }
func (tb *testBalances) SetStateContext(*state.State) error       { return nil }
func (tb *testBalances) AddMint(*state.Mint) error                { return nil }
func (tb *testBalances) GetTransfers() []*state.Transfer          { return nil }
func (tb *testBalances) AddSignedTransfer(st *state.SignedTransfer) {

}
func (tb *testBalances) GetSignedTransfers() []*state.SignedTransfer {
	return nil
}
func (tb *testBalances) DeleteTrieNode(datastore.Key) (datastore.Key, error) {
	return "", nil
}

func (tb *testBalances) GetClientBalance(clientID datastore.Key) (
	b state.Balance, err error) {

	var ok bool
	if b, ok = t.balances[clientID]; !ok {
		return 0, util.ErrValueNotPresent
	}
	return
}

func (tb *testBalances) GetTrieNode(key datastore.Key) (
	node util.Serializable, err error) {

	var ok bool
	if node, ok = t.tree[key]; !ok {
		return 0, util.ErrValueNotPresent
	}
	return
}

func (tb *testBalances) InsertTrieNode(key datastore.Key,
	node util.Serializable) (_ datastore.Key, _ error) {

	tb.tree[key] = node
	return
}

func (tb *testBalances) AddTransfer(t *state.Transfer) error {
	if t.ClientID != tb.txn.ClientID && t.ClientID != tb.txn.ToClientID {
		return state.ErrInvalidTransfer
	}
	tb.transfers = append(tb.transfers, t)
	return nil
}
