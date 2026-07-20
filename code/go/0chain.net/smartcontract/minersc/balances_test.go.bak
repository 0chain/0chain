package minersc

import (
	"0chain.net/chaincore/threshold/bls"
	"testing"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/statecache"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

type testBalances struct {
	balances      map[datastore.Key]currency.Coin
	txn           *transaction.Transaction
	transfers     []*state.Transfer
	tree          map[datastore.Key]util.MPTSerializable
	block         *block.Block
	blockSharders []string
	lfmb          *block.Block
	magicBlock    *block.MagicBlock
	tc            *statecache.TransactionCache
}

func (tb *testBalances) LoadDKGSummary(magicBlockNum int64) (*bls.DKGSummary, error) {
	return nil, nil
}

func (tb *testBalances) SetDKG(dkg *bls.DKG) error {
	//TODO implement me
	panic("implement me")
}

func newTestBalances() *testBalances {
	bc := statecache.NewBlockCache(statecache.NewStateCache(), statecache.Block{})
	balances := &testBalances{
		balances: make(map[datastore.Key]currency.Coin),
		tree:     make(map[datastore.Key]util.MPTSerializable),
		magicBlock: &block.MagicBlock{
			Miners:   node.NewPool(node.NodeTypeMiner),
			Sharders: node.NewPool(node.NodeTypeSharder),
		},
		tc: statecache.NewTransactionCache(bc),
	}
	b := &block.Block{}
	b.Round = 100
	balances.block = b

	return balances
}

func (tb *testBalances) Cache() *statecache.TransactionCache {
	return tb.tc
}

func (tb *testBalances) zeroize() { //nolint
	tb.balances = make(map[string]currency.Coin)
}

func (tb *testBalances) setBalance(key datastore.Key, b currency.Coin) { //nolint
	tb.balances[key] = b
}

func (tb *testBalances) setLFMB(lfmb *block.Block) {
	tb.lfmb = lfmb
}

func (tb *testBalances) setMagicBlock(magicBlock *block.MagicBlock) {
	tb.magicBlock = magicBlock
}

func (tb *testBalances) requireAllBeZeros(t *testing.T) { //nolint
	for id, value := range tb.balances {
		if id == ADDRESS {
			continue
		}
		require.Zerof(t, value, "%s has non-zero balance: %d", id, value)
	}
}

func (tb *testBalances) GetBlock() *block.Block {
	return tb.block
}

func (tb *testBalances) GetMagicBlock(round int64) *block.MagicBlock {
	return tb.magicBlock
}

func (tb *testBalances) SetMagicBlock(mb *block.MagicBlock) {
	if tb.block != nil {
		tb.block.MagicBlock = mb
	}
}

// stubs
func (tb *testBalances) GetState() util.MerklePatriciaTrieI         { return nil }
func (tb *testBalances) GetTransaction() *transaction.Transaction   { return nil }
func (tb *testBalances) Validate() error                            { return nil }
func (tb *testBalances) GetMints() []*state.Mint                    { return nil }
func (tb *testBalances) SetStateContext(*state.State) error         { return nil }
func (tb *testBalances) GetTransfers() []*state.Transfer            { return nil }
func (tb *testBalances) AddSignedTransfer(st *state.SignedTransfer) {}
func (tb *testBalances) GetEventDB() *event.EventDb                 { return nil }
func (tb *testBalances) EmitEventWithVersion(eventVersion event.EventVersion, eventType event.EventType, tag event.EventTag, index string, data interface{}, appenders ...cstate.Appender) {
}
func (tb *testBalances) EmitEvent(event.EventType, event.EventTag, string, interface{}, ...cstate.Appender) {
}
func (tb *testBalances) EmitError(error)                       {}
func (tb *testBalances) GetEvents() []event.Event              { return nil }
func (tb *testBalances) GetLatestFinalizedBlock() *block.Block { return nil }
func (tb *testBalances) GetSignedTransfers() []*state.SignedTransfer {
	return nil
}
func (tb *testBalances) DeleteTrieNode(key datastore.Key) (datastore.Key, error) {
	delete(tb.tree, key)
	return "", nil
}
func (tb *testBalances) GetLastestFinalizedMagicBlock() *block.Block {
	return tb.lfmb
}

func (tb *testBalances) GetSignatureScheme() encryption.SignatureScheme {
	return encryption.NewBLS0ChainScheme()
}

func (tb *testBalances) GetClientBalance(clientID datastore.Key) (
	b currency.Coin, err error) {

	var ok bool
	if b, ok = tb.balances[clientID]; !ok {
		return 0, util.ErrValueNotPresent
	}
	return
}

func (tb *testBalances) GetInvalidStateErrors() []error {
	return nil
}

func (tb *testBalances) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
	if encryption.IsHash(key) {
		return common.NewError("failed to get trie node",
			"key is too short")
	}

	node, ok := tb.tree[key]
	if !ok {
		return util.ErrValueNotPresent
	}
	d, err := node.MarshalMsg(nil)
	if err != nil {
		return err
	}

	_, err = v.UnmarshalMsg(d)
	return err
}

func (tb *testBalances) InsertTrieNode(key datastore.Key,
	node util.MPTSerializable) (_ datastore.Key, _ error) {

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
		panic("invalid miner: " + mint.Minter)
	}
	tb.balances[mint.ToClientID] += mint.Amount // mint!
	return nil
}

func (tb *testBalances) GetChainCurrentMagicBlock() *block.MagicBlock {
	return tb.magicBlock
}

func (tb *testBalances) GetClientState(clientID datastore.Key) (*state.State, error) {
	return nil, nil
}

func (tb *testBalances) SetClientState(clientID datastore.Key, s *state.State) (util.Key, error) {
	return nil, nil
}

func (tb *testBalances) GetMissingNodeKeys() []util.Key { return nil }
