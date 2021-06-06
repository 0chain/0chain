package zcnsc

// StateContextI implementation

import (
	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

const (
	clientId             = "fred"
	startMinted          = 10
	clientStartZCN       = 777
	txHash               = "tx hash"
	errLock              = "failed locking tokens: "
	errInsufficientFunds = "insufficient amount to dig an interest pool"
	errNoTokens          = "you have no tokens to your name"
	errLockGtBalance     = "lock amount is greater than balance"
	errDurationToLong    = "is longer than max lock period"
	errDurationToShort   = "is shorter than min lock period"
	errMaxMint           = "can't mint anymore"
	errUnlock            = "failed to unlock tokens"
	errEmptyingPool      = "error emptying pool"
	errPoolLocked        = "pool is still locked"
	errPoolNotExist      = "doesn't exist"
	startTime            = common.Timestamp(100)
	clientID0            = "client0_address"
	clientID1            = "client1_address"
	zrc20scAddress       = "zrc20sc_address"
)

const x10 = 10 * 1000 * 1000 * 1000

func zcnToInt64(token float64) int64 {
	return int64(token * float64(x10))
}

func zcnToBalance(token float64) state.Balance {
	return state.Balance(token * float64(x10))
}

type mockStateContext struct {
	ctx                cstate.StateContext
	clientStartBalance state.Balance
	store              map[datastore.Key]util.Serializable
}

func (sc *mockStateContext) GetLastestFinalizedMagicBlock() *block.Block           { return nil }
func (sc *mockStateContext) GetBlock() *block.Block                                { return nil }
func (sc *mockStateContext) SetMagicBlock(_ *block.MagicBlock)                     { return }
func (sc *mockStateContext) GetState() util.MerklePatriciaTrieI                    { return nil }
func (sc *mockStateContext) GetTransaction() *transaction.Transaction              { return nil }
func (sc *mockStateContext) GetTransfers() []*state.Transfer                       { return nil }
func (sc *mockStateContext) GetSignedTransfers() []*state.SignedTransfer           { return nil }
func (sc *mockStateContext) GetMints() []*state.Mint                               { return nil }
func (sc *mockStateContext) Validate() error                                       { return nil }
func (sc *mockStateContext) GetBlockSharders(_ *block.Block) []string              { return nil }
func (sc *mockStateContext) GetSignatureScheme() encryption.SignatureScheme        { return nil }
func (sc *mockStateContext) AddSignedTransfer(_ *state.SignedTransfer)             { return }
func (sc *mockStateContext) DeleteTrieNode(_ datastore.Key) (datastore.Key, error) { return "", nil }
func (sc *mockStateContext) GetChainCurrentMagicBlock() *block.MagicBlock          { return nil }

func (sc *mockStateContext) GetClientBalance(_ datastore.Key) (state.Balance, error) {
	if sc.clientStartBalance == 0 {
		return 0, util.ErrValueNotPresent
	}
	return sc.clientStartBalance, nil
}

func (sc *mockStateContext) SetStateContext(_ *state.State) error { return nil }

func (sc *mockStateContext) GetTrieNode(key datastore.Key) (util.Serializable, error) {
	var val, ok = sc.store[key]
	if !ok {
		return nil, util.ErrValueNotPresent
	}
	return val, nil
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