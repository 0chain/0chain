package zcnsc_test

// StateContextI implementation

import (
	"fmt"
	"strings"
	"time"

	"0chain.net/chaincore/block"
	"gorm.io/gorm/clause"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/chain/state/mocks"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"0chain.net/smartcontract/storagesc"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/mock"
)

const (
	txHash    = "tx hash"
	startTime = common.Timestamp(100)
)

var (
	_ cstate.StateContextI = (*mocks.StateContextI)(nil)
)

type mockStateContext struct {
	*mocks.StateContextI
	userNodes    map[string]*UserNode
	authorizers  map[string]*Authorizer
	globalNode   *GlobalNode
	stakingPools map[string]*StakePool
	authCount    *AuthCount
	block        *block.Block
	data         map[string][]byte
	eventDb      *event.EventDb
}

func (ctx *mockStateContext) GetLatestFinalizedBlock() *block.Block {
	//TODO implement me
	panic("implement me")
}

func (ctx *mockStateContext) SetEventDb(eventDb *event.EventDb) {
	ctx.eventDb = eventDb
}

func MakeMockStateContext() *mockStateContext {
	ctx := MakeMockStateContextWithoutAutorizers()

	// AuthorizerNodes & StakePools
	ctx.authorizers = make(map[string]*Authorizer, len(authorizersID))
	ctx.stakingPools = make(map[string]*StakePool, len(authorizersID))
	for _, id := range authorizersID {
		createTestAuthorizer(ctx, id)
		createTestStakingPools(ctx, id)
	}
	return ctx
}

func MakeMockStateContextWithoutAutorizers() *mockStateContext {
	ctx := &mockStateContext{
		StateContextI: &mocks.StateContextI{},
		data:          make(map[string][]byte),
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
			MaxStakeAmount: 111,
			OwnerId:        "8a15e216a3b4237330c1fff19c7b3916ece5b0f47341013ceb64d53595a4cebb",
			MaxFee:         100,
			MaxDelegates:   1000000000,
		},
	}

	// User Node

	ctx.userNodes = make(map[string]*UserNode)
	for _, client := range clients {
		userNode := createUserNode(client)
		ctx.userNodes[userNode.GetKey()] = userNode
	}

	ctx.block = block.NewBlock("", 0)

	// Transfers

	var transfers []*state.Transfer
	var mints []*state.Mint

	// EventsDB
	addAuthorizerEvents = make(map[string]*AuthorizerNode, 100)
	burnTicketEvents = make(map[string][]*event.BurnTicket, 100)

	ctx.On("GetEventDB").Return(
		func() *event.EventDb {
			return ctx.eventDb
		},
	)

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

	/// GetBlock

	ctx.On("GetBlock").Return(func() *block.Block {
		return ctx.block
	})

	/// DeleteTrieNode

	ctx.On("DeleteTrieNode", mock.AnythingOfType("string")).Return(
		func(key datastore.Key) datastore.Key {
			if strings.Contains(key, Porvider) {
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

	ctx.On("AddMint", mock.AnythingOfType("*state.Mint")).Return(func(m *state.Mint) error {
		mints = append(mints, m)
		return nil
	})
	ctx.On("GetMints").Return(func() []*state.Mint {
		return mints
	})

	// EventsDB

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
			addAuthorizerEvents[id] = authorizerNode
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
			addAuthorizerEvents[id] = authorizerNode
		})

	ctx.On("EmitEvent",
		event.TypeStats,
		event.TagAddBridgeMint,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("*event.BridgeMint"),
	).Run(
		func(args mock.Arguments) {
			userId, ok := args.Get(2).(string)
			if !ok {
				panic("failed to convert to user id")
			}
			bm, ok := args.Get(3).(*event.BridgeMint)
			if !ok {
				panic("failed to convert to get user")
			}
			user := &event.User{
				UserID:    bm.UserID,
				MintNonce: bm.MintNonce,
			}
			if user.UserID != userId {
				panic("user id must be equal to the id given as a param")
			}

			err := ctx.eventDb.Get().Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"mint_nonce"}),
			}).Create(user).Error
			if err != nil {
				panic(err)
			}
		},
	)

	ctx.On("EmitEvent",
		event.TypeStats,
		event.TagAddBurnTicket,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("*event.BurnTicket"),
	).Run(
		func(args mock.Arguments) {
			ethereumAdress, ok := args.Get(2).(string)
			if !ok {
				panic("failed to convert to user id")
			}
			burnTicket, ok := args.Get(3).(*event.BurnTicket)
			if !ok {
				panic("failed to convert to get user")
			}
			if burnTicket.EthereumAddress != ethereumAdress {
				panic("given ethereum address as index should be equal to the one given as a payload")
			}
			burnTicketEvents[ethereumAdress] = append(burnTicketEvents[ethereumAdress], burnTicket)
		})

	ctx.On("EmitEvent",
		event.TypeStats,
		event.TagStakePoolReward,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("*dbs.StakePoolReward"),
	).Run(func(args mock.Arguments) {
		stakePoolReward, ok := args.Get(3).(*dbs.StakePoolReward)
		if !ok {
			panic("failed to convert to get stake pool reward")
		}

		stakePool, ok := ctx.stakingPools[stakepool.StakePoolKey(spenum.Authorizer, stakePoolReward.ID)]
		if !ok {
			panic("failed to retreive a stake pool")
		}

		stakePool.Reward = stakePoolReward.Reward
	})

	ctx.On("EmitEvent",
		mock.AnythingOfType("event.EventType"),
		mock.AnythingOfType("event.EventTag"),
		mock.AnythingOfType("string"), // authorizerID
		mock.Anything,                 // authorizer payload
	).Return(
		func(_ event.EventType, _ event.EventTag, id string, body string) {
			fmt.Println(".")
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

	numAuth := &AuthCount{}
	err := ctx.GetTrieNode(storagesc.AUTHORIZERS_COUNT_KEY, numAuth)
	if err == util.ErrValueNotPresent {
		numAuth.Count = 0
	} else if err != nil {
		panic(err)
	}

	numAuth.Count++

	_, err = ctx.InsertTrieNode(storagesc.AUTHORIZERS_COUNT_KEY, numAuth)

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

	if strings.Contains(key, Porvider) {
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

	if strings.Contains(key, storagesc.AUTHORIZERS_COUNT_KEY) {
		if ctx.authCount == nil {
			return util.ErrValueNotPresent
		}
		b, err := ctx.authCount.MarshalMsg(nil)
		if err != nil {
			return err
		}
		_, err = node.UnmarshalMsg(b)
		if err != nil {
			panic(err)
		}
		return nil
	}

	if v, ok := ctx.data[key]; ok {
		if _, err := node.UnmarshalMsg(v); err != nil {
			return err
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

	if strings.Contains(key, Porvider) {
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

		return key, fmt.Errorf("failed to convert key: %s to Provider: %v", key, node)
	}

	if strings.Contains(key, storagesc.AUTHORIZERS_COUNT_KEY) {
		if authCount, ok := node.(*AuthCount); ok {
			ctx.authCount = authCount
			return key, nil
		}

		return key, fmt.Errorf("failed to convert key: %s to authCount: %v", key, node)
	}

	v, err := node.MarshalMsg(nil)
	if err != nil {
		return "", err
	}

	ctx.data[key] = v
	return "", nil
}

var (
	_ cstate.TimedQueryStateContextI = (*mocks.TimedQueryStateContextI)(nil)
)

type mockTimedQueryStateContext struct {
	*mockStateContext
}

func MakeMockTimedQueryStateContext() *mockTimedQueryStateContext {
	ctx := new(mockTimedQueryStateContext)

	ctx.mockStateContext = MakeMockStateContext()

	return ctx
}

func (ctx mockTimedQueryStateContext) Now() common.Timestamp {
	return common.Timestamp(time.Now().Unix())
}
