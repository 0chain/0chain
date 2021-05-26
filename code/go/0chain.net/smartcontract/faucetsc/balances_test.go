package faucetsc

import (
	"0chain.net/chaincore/block"
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
	txn  *transaction.Transaction
	tree map[datastore.Key]util.Serializable
}

func newTestBalances() *testBalances {
	return &testBalances{
		tree: make(map[datastore.Key]util.Serializable),
	}
}

// stubs
func (tb *testBalances) GetBlock() *block.Block                                               { return nil }
func (tb *testBalances) GetState() util.MerklePatriciaTrieI                                   { return nil }
func (tb *testBalances) GetTransaction() *transaction.Transaction                             { return nil }
func (tb *testBalances) GetBlockSharders(b *block.Block) []string                             { return nil }
func (tb *testBalances) Validate() error                                                      { return nil }
func (tb *testBalances) GetMints() []*state.Mint                                              { return nil }
func (tb *testBalances) SetStateContext(*state.State) error                                   { return nil }
func (tb *testBalances) AddMint(*state.Mint) error                                            { return nil }
func (tb *testBalances) GetTransfers() []*state.Transfer                                      { return nil }
func (tb *testBalances) AddSignedTransfer(st *state.SignedTransfer)                           {}
func (tb *testBalances) SetMagicBlock(block *block.MagicBlock)                                {}
func (tb *testBalances) GetLastestFinalizedMagicBlock() *block.Block                          { return nil }
func (tb *testBalances) GetSignatureScheme() encryption.SignatureScheme                       { return nil }
func (tb *testBalances) GetSignedTransfers() []*state.SignedTransfer                          { return nil }
func (tb *testBalances) DeleteTrieNode(key datastore.Key) (datastore.Key, error)              { return key, nil }
func (tb *testBalances) GetClientBalance(clientID datastore.Key) (b state.Balance, err error) { return }
func (tb *testBalances) AddTransfer(t *state.Transfer) error                                  { return nil }
func (tb *testBalances) GetChainCurrentMagicBlock() *block.MagicBlock                         { return nil }

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
