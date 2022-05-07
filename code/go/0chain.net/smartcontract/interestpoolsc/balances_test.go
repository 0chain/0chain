package interestpoolsc

import (
	"fmt"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

// testBalance implements state.StateContextI  intended for using
// fake test balance structure
type testBalances struct {
	balances      map[datastore.Key]state.Balance
	txn           *transaction.Transaction
	transfers     []*state.Transfer
	tree          map[datastore.Key]util.MPTSerializable
	block         *block.Block
	blockSharders []string
	lfmb          *block.Block
}

func (tb *testBalances) setBalance(key datastore.Key, b state.Balance) {
	tb.balances[key] = b
}

func (tb *testBalances) GetBlock() *block.Block {
	return tb.block
}

func (sc *testBalances) GetChainCurrentMagicBlock() *block.MagicBlock {
	return nil
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
func (tb *testBalances) GetLatestFinalizedBlock() *block.Block    { return nil }
func (tb *testBalances) AddSignedTransfer(_ *state.SignedTransfer) {
}
func (tb *testBalances) GetEventDB() *event.EventDb { return nil }
func (tb *testBalances) GetSignedTransfers() []*state.SignedTransfer {
	return nil
}
func (tb *testBalances) DeleteTrieNode(datastore.Key) (datastore.Key, error) {
	return "", nil
}
func (tb *testBalances) GetLastestFinalizedMagicBlock() *block.Block {
	return tb.lfmb
}
func (tb *testBalances) EmitEvent(event.EventType, event.EventTag, string, string) {}
func (tb *testBalances) EmitError(error)                                           {}
func (tb *testBalances) GetEvents() []event.Event                                  { return nil }
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
	nd, ok := tb.tree[key]
	if !ok {
		return util.ErrValueNotPresent
	}

	b, err := nd.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}

	_, err = v.UnmarshalMsg(b)
	if err != nil {
		panic(err)
	}

	return nil
}

func (tb *testBalances) InsertTrieNode(key datastore.Key,
	node util.MPTSerializable) (_ datastore.Key, _ error) {
	//@TODO add mutex to secure reading and writing into the map
	if encryption.IsHash(key) {
		return "", common.NewError("failed to insert trie node",
			"key is too short")
	}
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

func (tb *testBalances) AddMint(mint *state.Mint) error {
	if mint.Minter != ADDRESS {
		return fmt.Errorf("invalid miner: %v", mint.Minter)
	}
	tb.balances[mint.ToClientID] += mint.Amount // mint!
	return nil
}
