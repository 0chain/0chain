package zcnsc_test

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
	. "0chain.net/smartcontract/zcnsc"
	"strconv"
)

const (
	clientId  = "fred"
	txHash    = "tx hash"
	startTime = common.Timestamp(100)
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

func UpdateStateContext(tr *transaction.Transaction) *cstate.StateContext {
	return cstate.NewStateContext(
		nil,
		&util.MerklePatriciaTrie{},
		&state.Deserializer{},
		tr,
		nil,
		nil,
		nil,
		nil,
	)
}

func CreateStateContext(fromClientId string) *cstate.StateContext {
	var txn = &transaction.Transaction{
		HashIDField:  datastore.HashIDField{Hash: txHash},
		ClientID:     fromClientId,
		ToClientID:   zcnAddressId,
		CreationDate: startTime,
		Value:        int64(zcnToBalance(tokens)),
	}

	return UpdateStateContext(txn)
}

var store map[datastore.Key]util.Serializable

func UpdateMockStateContext(tr *transaction.Transaction) cstate.StateContextI {
	if store == nil {
		return CreateMockStateContextFromTransaction(tr)
	}

	m := &mockStateContext{
		ctx:                *UpdateStateContext(tr),
		clientStartBalance: zcnToBalance(3),
		store:              store,
	}

	node := createUserNode(tr.ClientID, int64(0))

	_, err := GetUserNode(node.ID, m)
	if err != nil {
		err := node.Save(m)
		if err != nil {
			panic(err)
		}
	}

	return m
}

func CreateMockStateContextFromTransaction(tr *transaction.Transaction) cstate.StateContextI {
	store = make(map[datastore.Key]util.Serializable)

	m := &mockStateContext{
		ctx:                *UpdateStateContext(tr),
		clientStartBalance: zcnToBalance(3),
		store:              store,
	}

	node := createUserNode(tr.ClientID, int64(0))
	err := node.Save(m)
	if err != nil {
		panic(err)
	}

	for i := 1; i <= 5; i++ {
		node := createUserNode(strconv.Itoa(i), int64(i))
		err := node.Save(m)
		if err != nil {
			panic(err)
		}
	}

	return m
}

func CreateMockStateContext(clientId string) cstate.StateContextI {
	store = make(map[datastore.Key]util.Serializable)

	m := &mockStateContext{
		ctx:                *CreateStateContext(clientId),
		clientStartBalance: zcnToBalance(3),
		store:              store,
	}

	node := createUserNode(clientId, int64(0))
	err := node.Save(m)
	if err != nil {
		panic(err)
	}

	for i := 1; i <= 5; i++ {
		node := createUserNode(strconv.Itoa(i), int64(i))
		err := node.Save(m)
		if err != nil {
			panic(err)
		}
	}

	return m
}

func (sc *mockStateContext) GetLastestFinalizedMagicBlock() *block.Block           { return nil }
func (sc *mockStateContext) GetBlock() *block.Block                                { return nil }
func (sc *mockStateContext) SetMagicBlock(_ *block.MagicBlock)                     { return }
func (sc *mockStateContext) GetTransaction() *transaction.Transaction              { return sc.ctx.GetTransaction() }
func (sc *mockStateContext) GetSignedTransfers() []*state.SignedTransfer           { return nil }
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
func (sc *mockStateContext) GetState() util.MerklePatriciaTrieI   { return nil }

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

func (sc *mockStateContext) GetTransfers() []*state.Transfer {
	return sc.ctx.GetTransfers()
}

func (sc *mockStateContext) AddMint(m *state.Mint) error {
	return sc.ctx.AddMint(m)
}

func (sc *mockStateContext) GetMints() []*state.Mint {
	return sc.ctx.GetMints()
}
