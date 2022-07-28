package zcnsc_test

// StateContextI implementation

import (
	"fmt"
	"strings"

	"0chain.net/chaincore/block"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/chain/state/mocks"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs/event"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/mock"
)

const (
	txHash    = "tx hash"
	startTime = common.Timestamp(100)
)

type mockStateContext struct {
	*mocks.StateContextI
	userNodes    map[string]*UserNode
	authorizers  map[string]*Authorizer
	globalNode   *GlobalNode
	stakingPools map[string]*StakePool
}

func (ctx *mockStateContext) GetLatestFinalizedBlock() *block.Block {
	//TODO implement me
	panic("implement me")
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

	ctx.globalNode = &GlobalNode{
		ID: ADDRESS,
		ZCNSConfig: &ZCNSConfig{
			MinStakeAmount: 11,
		},
	}

	// User Node

	ctx.userNodes = make(map[string]*UserNode)
	for _, client := range clients {
		userNode := createUserNode(client)
		ctx.userNodes[userNode.GetKey()] = userNode
	}

	// AuthorizerNodes & StakePools

	ctx.authorizers = make(map[string]*Authorizer, len(authorizersID))
	ctx.stakingPools = make(map[string]*StakePool, len(authorizersID))
	for _, id := range authorizersID {
		createTestAuthorizer(ctx, id)
		createTestStakingPools(ctx, id)
	}

	// Transfers

	var transfers []*state.Transfer

	// EventsDB
	events = make(map[string]*AuthorizerNode, 100)

	/// GetClientBalance

	ctx.On("GetClientBalance", mock.AnythingOfType("string")).Return(5, nil)

	/// AddTransfer

	ctx.On("AddTransfer", mock.AnythingOfType("*state.Transfer")).Return(
		func(transfer *state.Transfer) error {
			transfers = append(transfers, transfer)
			return nil
		})

	/// GetTransfers

	ctx.On("GetTransfers").Return(func() []*state.Transfer {
		return transfers
	})

	/// DeleteTrieNode

	ctx.On("DeleteTrieNode", mock.AnythingOfType("string")).Return(
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

	ctx.On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("util.MPTSerializable")).Return(
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

	ctx.On("InsertTrieNode", ctx.globalNode.GetKey(),
		mock.AnythingOfType("*zcnsc.GlobalNode")).Return(
		func(_ datastore.Key, node util.MPTSerializable) datastore.Key {
			ctx.globalNode = node.(*GlobalNode)
			return ""
		},
		func(_ datastore.Key, _ util.MPTSerializable) error {
			return nil
		})

	ctx.On("InsertTrieNode", mock.AnythingOfType("string"),
		mock.AnythingOfType("*zcnsc.UserNode")).Return(
		func(key datastore.Key, node util.MPTSerializable) datastore.Key {
			n := node.(*UserNode)
			ctx.userNodes[key] = n
			return ""
		},
		func(_ datastore.Key, _ util.MPTSerializable) error {
			return nil
		})

	ctx.On("InsertTrieNode", mock.AnythingOfType("string"),
		mock.AnythingOfType("*zcnsc.AuthorizerNode")).Return(
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

	ctx.On("AddMint", mock.AnythingOfType("*state.Mint")).Return(nil)

	// EventsDB

	ctx.On("EmitEvent",
		mock.AnythingOfType("event.EventType"),
		mock.AnythingOfType("event.EventTag"),
		mock.AnythingOfType("string"), // authorizerID
		mock.Anything,                 // authorizer payload
	).Return(
		func(_ event.EventType, _ event.EventTag, id string, body string) {
			fmt.Println(".")
		})

	ctx.On("EmitEvent",
		event.TypeStats,
		event.TagAddAuthorizer,
		mock.AnythingOfType("string"), // authorizerID
		mock.Anything,                 // authorizer payload
	).Return(
		func(_ event.EventType, _ event.EventTag, id string, ev *event.Authorizer) {
			authorizerNode, err := AuthorizerFromEvent(ev)
			if err != nil {
				panic(err)
			}
			if authorizerNode.ID != id {
				panic("authorizerID must be equal to ID")
			}
			events[id] = authorizerNode
		})

	ctx.On("EmitEvent",
		event.TypeStats,
		event.TagUpdateAuthorizer,
		mock.AnythingOfType("string"), // authorizerID
		mock.Anything,                 // authorizer payload
	).Return(
		func(_ event.EventType, _ event.EventTag, id string, ev *event.Authorizer) {
			authorizerNode, err := AuthorizerFromEvent(ev)
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

func createTestStakingPools(ctx *mockStateContext, delegateID string) *StakePool {
	sp := NewStakePool()
	sp.Minter = cstate.MinterStorage
	sp.Settings.DelegateWallet = delegateID

	ctx.stakingPools[sp.GetKey()] = sp

	return sp
}

func createTestAuthorizer(ctx *mockStateContext, id string) *Authorizer {
	scheme := ctx.GetSignatureScheme()
	_ = scheme.GenerateKeys()

	node := NewAuthorizer(id, scheme.GetPublicKey(), fmt.Sprintf("https://%s", id))

	ctx.authorizers[node.GetKey()] = &Authorizer{
		Scheme: scheme,
		Node:   node,
	}

	return ctx.authorizers[node.GetKey()]
}

func (ctx *mockStateContext) GetTrieNode(key datastore.Key, node util.MPTSerializable) error {
	if strings.Contains(key, UserNodeType) {
		n, ok := ctx.userNodes[key]
		if !ok {
			return util.ErrValueNotPresent
		}

		b, err := n.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		_, err = node.UnmarshalMsg(b)
		if err != nil {
			panic(err)
		}

		return nil
	}

	if strings.Contains(key, AuthorizerNodeType) {
		authorizer, ok := ctx.authorizers[key]
		if !ok {
			return util.ErrValueNotPresent
		}

		b, err := authorizer.Node.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		_, err = node.UnmarshalMsg(b)
		if err != nil {
			panic(err)
		}

		return nil
	}

	if strings.Contains(key, AuthorizerNewNodeType) {
		b, err := createTestAuthorizer(ctx, key).Node.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		if _, err := node.UnmarshalMsg(b); err != nil {
			panic(err)
		}

		return nil
	}

	if strings.Contains(key, GlobalNodeType) {
		b, err := ctx.globalNode.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		_, err = node.UnmarshalMsg(b)
		if err != nil {
			panic(err)
		}
		return nil
	}

	if strings.Contains(key, StakePoolNodeType) {
		n, ok := ctx.stakingPools[key]
		if !ok {
			return util.ErrValueNotPresent
		}

		b, err := n.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}

		_, err = node.UnmarshalMsg(b)
		if err != nil {
			panic(err)
		}

		return nil
	}

	return util.ErrValueNotPresent
}

func (ctx *mockStateContext) InsertTrieNode(key datastore.Key, node util.MPTSerializable) (datastore.Key, error) {
	if strings.Contains(key, UserNodeType) {
		if userNode, ok := node.(*UserNode); ok {
			ctx.userNodes[key] = userNode
			return key, nil
		}

		return key, fmt.Errorf("failed to convert key: %s to UserNode: %v", key, node)
	}

	if strings.Contains(key, AuthorizerNodeType) {
		if authorizer, ok := node.(*AuthorizerNode); ok {
			ctx.authorizers[key] = &Authorizer{
				Scheme: nil,
				Node:   authorizer,
			}
			return key, nil
		}

		return key, fmt.Errorf("failed to convert key: %s to AuthorizerNode: %v", key, node)
	}

	if strings.Contains(key, GlobalNodeType) {
		ctx.globalNode = node.(*GlobalNode)
		return key, nil
	}

	if strings.Contains(key, StakePoolNodeType) {
		if stakePool, ok := node.(*StakePool); ok {
			ctx.stakingPools[key] = stakePool
			return key, nil
		}

		return key, fmt.Errorf("failed to convert key: %s to StakePool: %v", key, node)
	}

	return "", fmt.Errorf("node with key: %s is not supported", key)
}
