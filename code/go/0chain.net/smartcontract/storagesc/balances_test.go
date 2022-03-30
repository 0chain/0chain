package storagesc

import (
	"testing"

	"0chain.net/smartcontract/dbs/event"

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
	balances  map[datastore.Key]state.Balance
	txn       *transaction.Transaction
	transfers []*state.Transfer
	tree      map[datastore.Key]util.MPTSerializable

	mpts      *mptStore // use for benchmarks
	skipMerge bool      // don't merge for now
}

func newTestBalances(t testing.TB, mpts bool) (tb *testBalances) {
	tb = &testBalances{
		balances: make(map[datastore.Key]state.Balance),
		tree:     make(map[datastore.Key]util.MPTSerializable),
	}

	if mpts {
		tb.mpts = newMptStore(t)
	}

	return
}

func (tb *testBalances) setTransaction(t testing.TB,
	txn *transaction.Transaction) {

	tb.txn = txn

	if tb.mpts != nil && !tb.skipMerge {
		tb.mpts.merge(t)
	}
}

// stubs
func (tb *testBalances) GetBlock() *block.Block                                    { return &block.Block{} }
func (tb *testBalances) GetState() util.MerklePatriciaTrieI                        { return nil }
func (tb *testBalances) GetTransaction() *transaction.Transaction                  { return nil }
func (tb *testBalances) GetBlockSharders(b *block.Block) []string                  { return nil }
func (tb *testBalances) Validate() error                                           { return nil }
func (tb *testBalances) GetMints() []*state.Mint                                   { return nil }
func (tb *testBalances) SetStateContext(*state.State) error                        { return nil }
func (tb *testBalances) AddMint(*state.Mint) error                                 { return nil }
func (tb *testBalances) GetTransfers() []*state.Transfer                           { return nil }
func (tb *testBalances) SetMagicBlock(block *block.MagicBlock)                     {}
func (tb *testBalances) AddSignedTransfer(st *state.SignedTransfer)                {}
func (tb *testBalances) GetSignedTransfers() []*state.SignedTransfer               { return nil }
func (tb *testBalances) GetEventDB() *event.EventDb                                { return nil }
func (tb *testBalances) EmitEvent(event.EventType, event.EventTag, string, string) {}
func (tb *testBalances) EmitError(error)                                           {}
func (tb *testBalances) GetEvents() []event.Event                                  { return nil }
func (tb *testBalances) GetChainCurrentMagicBlock() *block.MagicBlock              { return nil }
func (tb *testBalances) DeleteTrieNode(key datastore.Key) (
	datastore.Key, error) {

	if tb.mpts != nil {
		if encryption.IsHash(key) {
			return "", common.NewError("failed to get trie node",
				"key is too short")
		}
		var btkey, err = tb.mpts.mpt.Delete(util.Path(encryption.Hash(key)))
		return datastore.Key(btkey), err
	}

	delete(tb.tree, key)
	return "", nil
}
func (tb *testBalances) GetLastestFinalizedMagicBlock() *block.Block {
	return nil
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

func (tb *testBalances) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {

	if encryption.IsHash(key) {
		return common.NewError("failed to get trie node",
			"key is too short")
	}

	if tb.mpts != nil {
		return tb.mpts.mpt.GetNodeValue(util.Path(encryption.Hash(key)), v)
	}

	nd, ok := tb.tree[key]
	if !ok {
		return util.ErrValueNotPresent
	}

	d, err := nd.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}

	_, err = v.UnmarshalMsg(d)
	return err
}

func (tb *testBalances) InsertTrieNode(key datastore.Key,
	node util.MPTSerializable) (datastore.Key, error) {

	if tb.mpts != nil {
		if encryption.IsHash(key) {
			return "", common.NewError("failed to get trie node",
				"key is too short")
		}
		var btkey, err = tb.mpts.mpt.Insert(util.Path(encryption.Hash(key)), node)
		return datastore.Key(btkey), err
	}

	tb.tree[key] = node
	return "", nil
}

func (tb *testBalances) AddTransfer(t *state.Transfer) error {
	if t.ClientID != tb.txn.ClientID && t.ClientID != tb.txn.ToClientID {
		return state.ErrInvalidTransfer
	}
	tb.balances[t.ClientID] -= t.Amount
	tb.balances[t.ToClientID] += t.Amount
	tb.transfers = append(tb.transfers, t)
	return nil
}
