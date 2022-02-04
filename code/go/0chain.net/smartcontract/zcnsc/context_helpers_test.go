package zcnsc_test

// StateContextI implementation

import (
	"fmt"
	"strings"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/encryption"

	"0chain.net/chaincore/mocks"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/mock"
)

const (
	txHash    = "tx hash"
	startTime = common.Timestamp(100)
)

const x10 = 10 * 1000 * 1000 * 1000

func zcnToBalance(token float64) state.Balance {
	return state.Balance(token * float64(x10))
}

func MakeMockStateContext() *mocks.StateContextI {
	ctx := &mocks.StateContextI{}

	// GetSignatureScheme

	ctx.On("GetSignatureScheme").Return(
		func() encryption.SignatureScheme {
			return encryption.NewBLS0ChainScheme()
		},
	)

	// Global Node

	globalNode := &GlobalNode{ID: ADDRESS, MinStakeAmount: 11}

	// User Node

	userNodes := make(map[string]*UserNode)
	for _, client := range clients {
		userNode := createUserNode(client, int64(0))
		userNodes[userNode.GetKey()] = userNode
	}

	// AuthorizerNodes

	authorizers = make(map[string]*Authorizer, len(authorizersID))
	for _, id := range authorizersID {
		createTestAuthorizer(ctx, id)
	}

	// Transfers

	var transfers []*state.Transfer

	// EventsDB
	events = make(map[string]*AuthorizerNode, 100)

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
		Return(func() []*state.Transfer {
			return transfers
		})

	/// GetTrieNode - specific authorizer

	for _, an := range authorizers {
		ctx.
			On("GetTrieNode", an.Node.GetKey()).
			Return(
				func(key datastore.Key) util.Serializable {
					if authorizer, ok := authorizers[key]; ok {
						return authorizer.Node
					}
					return nil
				},
				func(_ datastore.Key) error {
					return nil
				})
	}

	ctx.
		On("GetTrieNode", mock.AnythingOfType("string")).
		Return(
			func(key datastore.Key) util.Serializable {
				if strings.Contains(key, UserNodeType) {
					return userNodes[key]
				}
				if strings.Contains(key, AuthorizerNodeType) {
					if authorizer, ok := authorizers[key]; ok {
						return authorizer.Node
					}
				}
				if strings.Contains(key, AuthorizerNewNodeType) {
					return createTestAuthorizer(ctx, key).Node
				}
				if strings.Contains(key, GlobalNodeType) {
					return globalNode
				}

				return nil
			},
			func(_ datastore.Key) error {
				return nil
			})

	/// DeleteTrieNode

	ctx.
		On("DeleteTrieNode", mock.AnythingOfType("string")).
		Return(
			func(key datastore.Key) datastore.Key {
				if strings.Contains(key, AuthorizerNodeType) {
					delete(authorizers, key)
					return key
				}
				return ""
			},
			func(_ datastore.Key) error {
				return nil
			})

	/// InsertTrieNode

	ctx.
		On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("util.Serializable")).
		Return(
			func(key datastore.Key, node util.Serializable) util.Serializable {
				if strings.Contains(key, UserNodeType) {
					userNodes[key] = node.(*UserNode)
					return node
				}
				if strings.Contains(key, AuthorizerNodeType) {
					authorizerNode := node.(*AuthorizerNode)
					authorizers[key] = &Authorizer{
						Scheme: nil,
						Node:   authorizerNode,
					}
					return authorizerNode
				}

				return nil
			},
			func(_ datastore.Key) error {
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
				n := node.(*UserNode)
				userNodes[key] = n
				return ""
			},
			func(_ datastore.Key, _ util.Serializable) error {
				return nil
			})

	ctx.
		On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*zcnsc.AuthorizerNode")).
		Return(
			func(key datastore.Key, node util.Serializable) datastore.Key {
				if strings.Contains(key, UserNodeType) {
					userNodes[key] = node.(*UserNode)
					return key
				}
				if strings.Contains(key, AuthorizerNodeType) {
					authorizerNode := node.(*AuthorizerNode)
					authorizers[key] = &Authorizer{
						Scheme: nil,
						Node:   authorizerNode,
					}
				}

				return key
			},
			func(_ datastore.Key, _ util.Serializable) error {
				return nil
			})

	ctx.
		On("AddMint", mock.AnythingOfType("*state.Mint")).
		Return(nil)

	// EventsDB

	ctx.On(
		"EmitEvent",
		mock.AnythingOfType("event.EventType"),
		mock.AnythingOfType("event.EventTag"),
		mock.AnythingOfType("string"), // authorizerID
		mock.AnythingOfType("string"), // authorizer payload
	).Return(
		func(_ event.EventType, _ event.EventTag, id string, body string) {
			fmt.Println(".")
		})

	ctx.On(
		"EmitEvent",
		event.TypeStats,
		event.TagAddAuthorizer,
		mock.AnythingOfType("string"), // authorizerID
		mock.AnythingOfType("string"), // authorizer payload
	).Return(
		func(_ event.EventType, _ event.EventTag, id string, body string) {
			authorizerNode, err := AuthorizerFromEvent([]byte(body))
			if err != nil {
				panic(err)
			}
			if authorizerNode.ID != id {
				panic("authorizerID must be equal to ID")
			}
			events[id] = authorizerNode
		})

	return ctx
}

func createTestAuthorizer(ctx *mocks.StateContextI, id string) *Authorizer {
	scheme := ctx.GetSignatureScheme()
	_ = scheme.GenerateKeys()

	node := CreateAuthorizer(id, scheme.GetPublicKey(), fmt.Sprintf("https://%s", id))
	tr := CreateAddAuthorizerTransaction(defaultClient, ctx, 100)
	_, _, _ = node.Staking.DigPool(tr.Hash, tr)

	authorizers[node.GetKey()] = &Authorizer{
		Scheme: scheme,
		Node:   node,
	}

	return authorizers[node.GetKey()]
}
