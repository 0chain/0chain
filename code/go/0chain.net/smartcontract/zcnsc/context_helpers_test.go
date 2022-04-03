package zcnsc_test

// StateContextI implementation

import (
	"fmt"
	"reflect"
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

func mockSetValue(v interface{}) interface{} {
	return mock.MatchedBy(func(c interface{}) bool {
		cv := reflect.ValueOf(c)
		if cv.Kind() != reflect.Ptr {
			panic(fmt.Sprintf("%t must be a pointer, %v", v, cv.Kind()))
		}

		vv := reflect.ValueOf(v)
		if vv.Kind() == reflect.Ptr {
			if vv.Type() != cv.Type() {
				return false
			}
			cv.Elem().Set(vv.Elem())
		} else {
			if vv.Type() != cv.Elem().Type() {
				return false
			}

			cv.Elem().Set(vv)
		}
		return true
	})
}

type mockStateContext struct {
	*mocks.StateContextI
	userNodes   map[string]*UserNode
	authorizers map[string]*Authorizer
	globalNode  *GlobalNode
}

func (m *mockStateContext) GetTrieNode(key datastore.Key, v util.MPTSerializable) (node util.MPTSerializable, err error) {
	if strings.Contains(key, UserNodeType) {
		n, ok := m.userNodes[key]
		if !ok {
			return nil, util.ErrValueNotPresent
		}

		b, err := n.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		_, err = v.UnmarshalMsg(b)
		if err != nil {
			panic(err)
		}

		return v, nil
	}

	if strings.Contains(key, AuthorizerNodeType) {
		authorizer, ok := m.authorizers[key]
		if !ok {
			return nil, util.ErrValueNotPresent
		}

		b, err := authorizer.Node.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		_, err = v.UnmarshalMsg(b)
		if err != nil {
			panic(err)
		}

		return v, nil
	}

	if strings.Contains(key, AuthorizerNewNodeType) {
		b, err := createTestAuthorizer(m, key).Node.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		if _, err := v.UnmarshalMsg(b); err != nil {
			panic(err)
		}

		return v, nil
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
		return v, nil
	}

	return nil, util.ErrValueNotPresent
}

func MakeMockStateContext() *mockStateContext {
	ctx := &mockStateContext{
		StateContextI: &mocks.StateContextI{},
	}

	// GetSignatureScheme

	ctx.On("GetSignatureScheme").Return(
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

	/// DeleteTrieNode

	ctx.
		On("DeleteTrieNode", mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			key := args[0].(datastore.Key)
			if strings.Contains(key, AuthorizerNodeType) {
				delete(ctx.authorizers, key)
			}
		}).
		Return(
			func(_ datastore.Key) error {
				return nil
			})
	/// InsertTrieNode

	ctx.
		On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("util.MPTSerializable")).
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
		On("InsertTrieNode", ctx.globalNode.GetKey(), mock.AnythingOfType("*zcnsc.GlobalNode")).
		Run(func(args mock.Arguments) {
			node := args[1].(util.Serializable)
			ctx.globalNode = node.(*GlobalNode)
		}).
		Return(
			func(_ datastore.Key, _ util.Serializable) error {
				return nil
			})

	ctx.
		On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*zcnsc.UserNode")).
		Run(func(args mock.Arguments) {
			key := args[0].(datastore.Key)
			node := args[1].(util.Serializable)
			n := node.(*UserNode)
			ctx.userNodes[key] = n
		}).
		Return(
			func(_ datastore.Key, _ util.Serializable) error {
				return nil
			})

	ctx.
		On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*zcnsc.AuthorizerNode")).
		Run(func(args mock.Arguments) {
			key := args[0].(datastore.Key)
			node := args[1].(util.MPTSerializable)
			if strings.Contains(key, UserNodeType) {
				ctx.userNodes[key] = node.(*UserNode)
				return
			}
			if strings.Contains(key, AuthorizerNodeType) {
				authorizerNode := node.(*AuthorizerNode)
				ctx.authorizers[key] = &Authorizer{
					Scheme: nil,
					Node:   authorizerNode,
				}
			}
		}).
		Return(
			func(_ datastore.Key, _ util.MPTSerializable) error {
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

	ctx.On(
		"EmitEvent",
		event.TypeStats,
		event.TagUpdateAuthorizer,
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
	scheme := ctx.GetSignatureScheme()
	_ = scheme.GenerateKeys()

	node := NewAuthorizer(id, scheme.GetPublicKey(), fmt.Sprintf("https://%s", id))
	tr := CreateAddAuthorizerTransaction(defaultClient, ctx, 100)
	_, _, _ = node.Staking.DigPool(tr.Hash, tr)

	ctx.authorizers[node.GetKey()] = &Authorizer{
		Scheme: scheme,
		Node:   node,
	}

	return ctx.authorizers[node.GetKey()]
}
