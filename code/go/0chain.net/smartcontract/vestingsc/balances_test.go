package vestingsc

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
func (tb *testBalances) GetBlock() *block.Block                       { return nil }
func (tb *testBalances) GetState() util.MerklePatriciaTrieI           { return nil }
func (tb *testBalances) GetTransaction() *transaction.Transaction     { return nil }
func (tb *testBalances) GetBlockSharders(b *block.Block) []string     { return nil }
func (tb *testBalances) Validate() error                              { return nil }
func (tb *testBalances) GetMints() []*state.Mint                      { return nil }
func (tb *testBalances) SetStateContext(*state.State) error           { return nil }
func (tb *testBalances) AddMint(*state.Mint) error                    { return nil }
func (tb *testBalances) GetTransfers() []*state.Transfer              { return nil }
func (tb *testBalances) GetChainCurrentMagicBlock() *block.MagicBlock { return nil }
func (tb *testBalances) AddSignedTransfer(st *state.SignedTransfer) {

}
func (tb *testBalances) SetMagicBlock(block *block.MagicBlock) {}
func (tb *testBalances) GetLastestFinalizedMagicBlock() *block.Block {
	return nil
}

func (tb *testBalances) GetSignatureScheme() encryption.SignatureScheme {
	return encryption.NewBLS0ChainScheme()
}
func (tb *testBalances) GetSignedTransfers() []*state.SignedTransfer {
	return nil
}
func (tb *testBalances) DeleteTrieNode(key datastore.Key) (
	datastore.Key, error) {

	delete(tb.tree, key)
	return key, nil
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
	if t.ClientID != tb.txn.ClientID && t.ClientID != tb.txn.ToClientID {
		return state.ErrInvalidTransfer
	}
	tb.balances[t.ClientID] -= t.Amount
	tb.balances[t.ToClientID] += t.Amount
	tb.transfers = append(tb.transfers, t)
	return nil
}

func (tb *testBalances) GetClientState(clientID string) (*state.State, error) {
	//node, err := sc.GetClientTrieNode(clientID)
	//
	//if err != nil {
	//	if err != util.ErrValueNotPresent {
	//		return nil, err
	//	}
	//	return nil, err
	//}
	//s = sc.clientStateDeserializer.Deserialize(node).(*state.State)
	return &state.State{}, nil
}

func (tb *testBalances) GetClientTrieNode(key datastore.Key) (util.Serializable, error) {
	return tb.GetTrieNode(key)
}

func (tb *testBalances) InsertClientTrieNode(key datastore.Key, node util.Serializable) (datastore.Key, error) {
	return tb.InsertTrieNode(key, node)
}

func (tb *testBalances) DeleteClientTrieNode(key datastore.Key) (datastore.Key, error) {
	return tb.DeleteTrieNode(key)
}

func (tb *testBalances) GetRWSets() (rset map[string]bool, wset map[string]bool) {
	return map[string]bool{}, map[string]bool{}
}

func (tb *testBalances) GetVersion() util.Sequence {
	return 0
}

func (tb *testBalances) PrintStates() {
}
