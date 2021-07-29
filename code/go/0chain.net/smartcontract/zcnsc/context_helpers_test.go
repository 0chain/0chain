package zcnsc_test

// StateContextI implementation

import (
	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/mock"
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

func MakeMockStateContext() *mocks.StateContextI {
	globalNode := &GlobalNode{ID: ADDRESS, MinStakeAmount: 11}
	//userNode := createUserNode(clientId, int64(0))

	ctx := mocks.StateContextI{}

	ans := &AuthorizerNodes{}
	ans.NodeMap = make(map[string]*AuthorizerNode)

	for _, authorizer := range authorizers {
		err := ans.AddAuthorizer(CreateMockAuthorizer(authorizer))
		if err != nil {
			panic(err.Error())
		}
	}

	ctx.
		On("GetClientBalance", mock.AnythingOfType("string")).
		Return(5, nil)

	ctx.
		On("AddTransfer", mock.AnythingOfType("*state.Transfer")).
		Return(nil)

	/// GetTrieNode

	ctx.
		On("GetTrieNode", AllAuthorizerKey).
		Return(
			func(_ datastore.Key) util.Serializable {
				return ans
			},
			func(_ datastore.Key) error {
				return nil
			})

	ctx.
		On("GetTrieNode", globalNode.GetKey()).
		Return(
			func(_ datastore.Key) util.Serializable {
				return globalNode
			},
			func(_ datastore.Key) error {
				return nil
			})

	for _, client := range authorizers {
		userNode := createUserNode(client, int64(0))

		ctx.
			On("GetTrieNode", userNode.GetKey(ADDRESS)).
			Return(
				func(_ datastore.Key) util.Serializable {
					return userNode
				},
				func(_ datastore.Key) error {
					return nil
				})
	}

	/// InsertTrieNode

	ctx.
		On("InsertTrieNode", AllAuthorizerKey, mock.AnythingOfType("*zcnsc.AuthorizerNodes")).
		Return(
			func(_ datastore.Key, nodes util.Serializable) datastore.Key {
				ans = nodes.(*AuthorizerNodes)
				return ""
			},
			func(_ datastore.Key, _ util.Serializable) error {
				return nil
			})

	ctx.
		On("InsertTrieNode", globalNode.GetKey(), mock.AnythingOfType("*zcnsc.GlobalNode")).
		Return(
			func(_ datastore.Key, node util.Serializable) datastore.Key {
				globalNode = node.(*GlobalNode)
				return ""
			},
			func(_ datastore.Key, _ util.Serializable) error {
				return nil
			})

	ctx.
		On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*zcnsc.UserNode")).
		Return(
			func(_ datastore.Key, _ util.Serializable) datastore.Key {
				return ""
			},
			func(_ datastore.Key, _ util.Serializable) error {
				return nil
			})

	ctx.
		On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*zcnsc.AuthorizerNodes")).
		Return(
			func(_ datastore.Key, _ util.Serializable) datastore.Key {
				return ""
			},
			func(_ datastore.Key, _ util.Serializable) error {
				return nil
			})

	ctx.
		On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*zcnsc.AuthorizerNode")).
		Return(
			func(_ datastore.Key, _ util.Serializable) datastore.Key {
				return ""
			},
			func(_ datastore.Key, _ util.Serializable) error {
				return nil
			})

	////////////////////////////

	for _, authorizer := range authorizers {
		client := authorizer
		mintPayload, _, _ := CreateMintPayload(client, authorizers)

		mint := &state.Mint{
			Minter:     globalNode.ID,
			ToClientID: mintPayload.ReceivingClientID,
			Amount:     mintPayload.Amount,
		}

		ctx.
			On("AddMint", mint).
			Return(nil).Once()
	}

	return &ctx
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
