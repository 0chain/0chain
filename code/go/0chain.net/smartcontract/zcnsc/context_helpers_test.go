package zcnsc_test

// StateContextI implementation

import (
	"0chain.net/chaincore/mocks"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/mock"
)

const (
	clientId  = "fred"
	txHash    = "tx hash"
	startTime = common.Timestamp(100)
)

const x10 = 10 * 1000 * 1000 * 1000

func zcnToBalance(token float64) state.Balance {
	return state.Balance(token * float64(x10))
}

func MakeMockStateContext() *mocks.StateContextI {
	ctx := mocks.StateContextI{}

	// Global Node
	globalNode := &GlobalNode{ID: ADDRESS, MinStakeAmount: 11}

	// User Node
	userNodes := make(map[string]*UserNode)
	for _, client := range authorizers {
		userNode := createUserNode(client, int64(0))
		userNodes[userNode.GetKey(ADDRESS)] = userNode
	}

	// AuthorizerNodes
	ans := &AuthorizerNodes{}
	ans.NodeMap = make(map[string]*AuthorizerNode)
	for _, authorizer := range authorizers {
		err := ans.AddAuthorizer(CreateMockAuthorizer(authorizer))
		if err != nil {
			panic(err.Error())
		}
	}

	// Transfers
	var transfers []*state.Transfer

	/// GetClientBalance

	ctx.
		On("GetClientBalance", mock.AnythingOfType("string")).
		Return(5, nil)

	/// AddTransfer

	ctx.
		On("AddTransfer", mock.AnythingOfType("*state.Transfer")).
		Return(
			func(transfer *state.Transfer) error {
				transfers = append(transfers, transfer)
				return nil
			})

	/// GetTransfers

	ctx.
		On("GetTransfers").
		Return(
			func() []*state.Transfer {
				return transfers
			})

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

	for _, un := range userNodes {
		ctx.
			On("GetTrieNode", un.GetKey(ADDRESS)).
			Return(
				func(key datastore.Key) util.Serializable {
					return userNodes[key]
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
			func(key datastore.Key, node util.Serializable) datastore.Key {
				un := node.(*UserNode)
				userNodes[key] = un
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

	/// AddMint

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
