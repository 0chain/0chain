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

type mockStateContext struct {
	*mocks.StateContextI
	userNodes   map[string]*UserNode
	authorizers map[string]*Authorizer
	globalNode  *GlobalNode
}

func (m *mockStateContext) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
	if strings.Contains(key, UserNodeType) {
		n, ok := m.userNodes[key]
		if !ok {
			return util.ErrValueNotPresent
		}

		b, err := n.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		_, err = v.UnmarshalMsg(b)
		if err != nil {
			panic(err)
		}

		return nil
	}

	if strings.Contains(key, AuthorizerNodeType) {
		authorizer, ok := m.authorizers[key]
		if !ok {
			return util.ErrValueNotPresent
		}

		b, err := authorizer.Node.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		_, err = v.UnmarshalMsg(b)
		if err != nil {
			panic(err)
		}

		return nil
	}

	if strings.Contains(key, AuthorizerNewNodeType) {
		b, err := createTestAuthorizer(m, key).Node.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		if _, err := v.UnmarshalMsg(b); err != nil {
			panic(err)
		}

		return nil
	}

	if strings.Contains(key, GlobalNodeType) {
		b, err := m.globalNode.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		_, err = v.UnmarshalMsg(b)
		if err != nil {
			panic(err)
		}
		return nil
	}

	return util.ErrValueNotPresent
}

func MakeMockStateContext() *mockStateContext { //nolint
	ctx := &mockStateContext{
		StateContextI: &mocks.StateContextI{},
	}

	// GetSignatureScheme

	ctx.On("GetSignatureScheme").Return( //nolint: typecheck
		func() encryption.SignatureScheme {
			return encryption.NewBLS0ChainScheme()
		},
	)

	// Global Node

	ctx.globalNode = &GlobalNode{ID: ADDRESS, MinStakeAmount: 11}

	// User Node

	ctx.userNodes = make(map[string]*UserNode)
	for _, client := range clients {
		userNode := createUserNode(client, int64(0))
		ctx.userNodes[userNode.GetKey()] = userNode
	}

	// AuthorizerNodes

	ctx.authorizers = make(map[string]*Authorizer, len(authorizersID))
	for _, id := range authorizersID {
		createTestAuthorizer(ctx, id)
	}

	// Transfers

	var transfers []*state.Transfer

	// EventsDB
	events = make(map[string]*AuthorizerNode, 100)

	/// GetClientBalance

	ctx.
		On("GetClientBalance", mock.AnythingOfType("string")). //nolint: typecheck
		Return(5, nil)

	/// AddTransfer

	ctx.
		On("AddTransfer", mock.AnythingOfType("*state.Transfer")). //nolint: typecheck
		Return(
			func(transfer *state.Transfer) error {
				transfers = append(transfers, transfer)
				return nil
			})

	/// GetTransfers

	ctx.
		On("GetTransfers"). //nolint: typecheck
		Return(func() []*state.Transfer {
			return transfers
		})

	/// GetTrieNode - specific authorizer

	//for _, an := range authorizers {
	//	ctx.On("GetTrieNode", an.Node.GetKey(),
	//		mockSetValue(an.Node)).Return(nil)
	//}

	//ctx.On("GetTrieNode", mock.AnythingOfType("string"), mock.Anything).Run(
	//	func(args mock.Arguments) {
	//		key := args.Get(0).(string)
	//
	//	}).Return(err)

	//ctx.
	//	On("GetTrieNode", mock.AnythingOfType("string")).
	//	Return(
	//		func(key datastore.Key) util.Serializable {
	//			if strings.Contains(key, UserNodeType) {
	//				return userNodes[key]
	//			}
	//			if strings.Contains(key, AuthorizerNodeType) {
	//				if authorizer, ok := authorizers[key]; ok {
	//					return authorizer.Node
	//				}
	//			}
	//			if strings.Contains(key, AuthorizerNewNodeType) {
	//				return createTestAuthorizer(ctx, key).Node
	//			}
	//			if strings.Contains(key, GlobalNodeType) {
	//				return globalNode
	//			}
	//
	//			return nil
	//		},
	//		func(_ datastore.Key) error {
	//			return nil
	//		})

	/// DeleteTrieNode

	ctx.
		On("DeleteTrieNode", mock.AnythingOfType("string")). //nolint: typecheck
		Return(
			func(key datastore.Key) datastore.Key {
				if strings.Contains(key, AuthorizerNodeType) {
					delete(ctx.authorizers, key)
					return key
				}
				return ""
			},
			func(_ datastore.Key) error {
				return nil
			})

	/// InsertTrieNode

	ctx.
		On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("util.MPTSerializable")). //nolint: typecheck
		Return(
			func(key datastore.Key, node util.MPTSerializable) util.MPTSerializable {
				if strings.Contains(key, UserNodeType) {
					ctx.userNodes[key] = node.(*UserNode)
					return node
				}
				if strings.Contains(key, AuthorizerNodeType) {
					authorizerNode := node.(*AuthorizerNode)
					ctx.authorizers[key] = &Authorizer{
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
		On("InsertTrieNode", ctx.globalNode.GetKey(), mock.AnythingOfType("*zcnsc.GlobalNode")). //nolint: typecheck
		Return(
			func(_ datastore.Key, node util.MPTSerializable) datastore.Key {
				ctx.globalNode = node.(*GlobalNode)
				return ""
			},
			func(_ datastore.Key, _ util.MPTSerializable) error {
				return nil
			})

	ctx.
		On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*zcnsc.UserNode")). //nolint: typecheck
		Return(
			func(key datastore.Key, node util.MPTSerializable) datastore.Key {
				n := node.(*UserNode)
				ctx.userNodes[key] = n
				return ""
			},
			func(_ datastore.Key, _ util.MPTSerializable) error {
				return nil
			})

	ctx.On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*zcnsc.AuthorizerNode")). //nolint: typecheck
														Return(
			func(key datastore.Key, node util.MPTSerializable) datastore.Key {
				if strings.Contains(key, UserNodeType) {
					ctx.userNodes[key] = node.(*UserNode)
					return key
				}
				if strings.Contains(key, AuthorizerNodeType) {
					authorizerNode := node.(*AuthorizerNode)
					ctx.authorizers[key] = &Authorizer{
						Scheme: nil,
						Node:   authorizerNode,
					}
				}

				return key
			},
			func(_ datastore.Key, _ util.MPTSerializable) error {
				return nil
			})

	ctx.On("AddMint", mock.AnythingOfType("*state.Mint")).Return(nil) //nolint: typecheck

	// EventsDB

	ctx.On( //nolint: typecheck
		"EmitEvent",
		mock.AnythingOfType("event.EventType"),
		mock.AnythingOfType("event.EventTag"),
		mock.AnythingOfType("string"), // authorizerID
		mock.AnythingOfType("string"), // authorizer payload
	).Return(
		func(_ event.EventType, _ event.EventTag, id string, body string) {
			fmt.Println(".")
		})

	ctx.On( //nolint: typecheck
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

func createTestAuthorizer(ctx *mockStateContext, id string) *Authorizer {
	scheme := ctx.GetSignatureScheme() //nolint: typecheck
	_ = scheme.GenerateKeys()

	node := CreateAuthorizer(id, scheme.GetPublicKey(), fmt.Sprintf("https://%s", id))
	tr := CreateAddAuthorizerTransaction(defaultClient, ctx, 100) //nolint: typecheck
	_, _, _ = node.Staking.DigPool(tr.Hash, tr)

	ctx.authorizers[node.GetKey()] = &Authorizer{
		Scheme: scheme,
		Node:   node,
	}

	return ctx.authorizers[node.GetKey()]
}
