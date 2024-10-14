package storagesc

import (
	"sync"
	"testing"
	"time"

	cstate "0chain.net/chaincore/chain/state"
	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/dbs/event"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/statecache"
	"github.com/0chain/common/core/util"
)

//
// helper for tests implements chainState.StateContextI
//

type testBalances struct {
	sync.RWMutex
	balances  map[datastore.Key]currency.Coin
	txn       *transaction.Transaction
	transfers []*state.Transfer
	tree      map[datastore.Key]util.MPTSerializable
	block     *block.Block
	tc        *statecache.TransactionCache
	events    []event.Event

	mpts      *mptStore // use for benchmarks
	skipMerge bool      // don't merge for now
}

func newTestBalances(t testing.TB, mpts bool) (tb *testBalances) {
	tb = &testBalances{
		balances: make(map[datastore.Key]currency.Coin),
		tree:     make(map[datastore.Key]util.MPTSerializable),
		txn:      new(transaction.Transaction),
		block:    new(block.Block),
	}

	var scYaml = Config{
		StakePool:                    &stakePoolConfig{},
		BlobberSlash:                 0.1,
		ValidatorReward:              0.025,
		MaxChallengeCompletionRounds: 720,
		TimeUnit:                     720 * time.Hour,
		MaxStake:                     zcnToBalance(100000.0),
		CancellationCharge:           0.2,
		MaxReadPrice:                 1e12,
		MaxWritePrice:                1e12,
		MaxCharge:                    1,
		MaxDelegates:                 200,
		//ValidatorsPerChallenge:       3,
		BlockReward: &blockReward{
			BlockReward:             zcnToBalance(0.06),
			BlockRewardChangePeriod: 125000000,
			BlockRewardChangeRatio:  0.1,
			QualifyingStake:         1,
			TriggerPeriod:           30,
			Gamma: blockRewardGamma{
				Alpha: 0.2,
				A:     10,
				B:     9,
			},
			Zeta: blockRewardZeta{
				I:  1,
				K:  0.9,
				Mu: 0.2,
			},
		},
	}

	bc := statecache.NewBlockCache(statecache.NewStateCache(), statecache.Block{
		Round:    tb.block.Round,
		Hash:     tb.block.Hash,
		PrevHash: tb.block.PrevHash,
	})
	tb.tc = statecache.NewTransactionCache(bc)

	if mpts {
		tb.mpts = newMptStore(t)
	}

	err := InitPartitions(tb)
	require.NoError(t, err)

	_, err = tb.InsertTrieNode(scConfigKey(ADDRESS), &scYaml)
	require.NoError(t, err)

	h := cstate.NewHardFork("apollo", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = cstate.NewHardFork("ares", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = cstate.NewHardFork("artemis", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = cstate.NewHardFork("athena", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = cstate.NewHardFork("demeter", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = cstate.NewHardFork("electra", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	h = cstate.NewHardFork("hercules", 0)
	if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
		t.Fatal(err)
	}

	bk := &block.Block{}
	bk.Round = 2
	tb.setBlock(t, bk)

	return
}

func (tb *testBalances) setTransaction(t testing.TB,
	txn *transaction.Transaction) {

	tb.txn = txn

	if tb.mpts != nil && !tb.skipMerge {
		tb.mpts.merge(t)
	}
}

func (tb *testBalances) setBlock(t testing.TB, block *block.Block) {
	tb.block = block
}

func (tb *testBalances) GetTransaction() *transaction.Transaction {
	return tb.txn
}

// stubs
func (tb *testBalances) GetBlock() *block.Block { return tb.block }
func (tb *testBalances) GetState() util.MerklePatriciaTrieI {
	if tb.mpts == nil {
		return nil
	}
	return tb.mpts.mpt
}
func (tb *testBalances) Validate() error                             { return nil }
func (tb *testBalances) GetMints() []*state.Mint                     { return nil }
func (tb *testBalances) SetStateContext(*state.State) error          { return nil }
func (tb *testBalances) AddMint(*state.Mint) error                   { return nil }
func (tb *testBalances) GetTransfers() []*state.Transfer             { return nil }
func (tb *testBalances) GetMagicBlock(round int64) *block.MagicBlock { return nil }
func (tb *testBalances) SetMagicBlock(block *block.MagicBlock)       {}
func (tb *testBalances) AddSignedTransfer(st *state.SignedTransfer)  {}
func (tb *testBalances) GetSignedTransfers() []*state.SignedTransfer { return nil }
func (tb *testBalances) GetEventDB() *event.EventDb                  { return nil }
func (tb *testBalances) EmitEvent(eventType event.EventType, tag event.EventTag, index string, data interface{}, appenders ...cstate.Appender) {
	tb.EmitEventWithVersion(event.Version1, eventType, tag, index, data, appenders...)
}
func (tb *testBalances) EmitEventWithVersion(eventVersion event.EventVersion, eventType event.EventType, tag event.EventTag, index string, data interface{}, appenders ...cstate.Appender) {
	tb.RWMutex.Lock()
	defer tb.RWMutex.Unlock()
	e := event.Event{
		BlockNumber: tb.block.Round,
		TxHash:      tb.txn.Hash,
		Type:        eventType,
		Tag:         tag,
		Index:       index,
		Data:        data,
	}
	if len(appenders) != 0 {
		tb.events = appenders[0](tb.events, e)
	} else {
		tb.events = append(tb.events, e)
	}
}
func (tb *testBalances) EmitError(error)                              {}
func (tb *testBalances) GetEvents() []event.Event                     { return tb.events }
func (tb *testBalances) GetChainCurrentMagicBlock() *block.MagicBlock { return nil }
func (tb *testBalances) GetLatestFinalizedBlock() *block.Block        { return nil }
func (tb *testBalances) DeleteTrieNode(key datastore.Key) (datastore.Key, error) {

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
	b currency.Coin, err error) {

	var ok bool
	if b, ok = tb.balances[clientID]; !ok {
		return 0, util.ErrValueNotPresent
	}
	return
}

func (tb *testBalances) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
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
		var btkey, err = tb.mpts.mpt.Insert(util.Path(encryption.Hash(key)), node)
		return datastore.Key(btkey), err
	}

	tb.tree[key] = node
	return "", nil
}

func (tb *testBalances) Cache() *statecache.TransactionCache {
	return tb.tc
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

func (tb *testBalances) GetInvalidStateErrors() []error { return nil }

func (tb *testBalances) GetClientState(clientID datastore.Key) (*state.State, error) {
	return nil, nil
}

func (tb *testBalances) SetClientState(clientID datastore.Key, s *state.State) (util.Key, error) {
	return nil, nil
}

func (tb *testBalances) GetMissingNodeKeys() []util.Key { return nil }
