package zcnsc

import (
	"fmt"

	"0chain.net/smartcontract/provider"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/util"

	"0chain.net/chaincore/chain/state"
)

var (
	cfg = config.SmartContractConfig
)

func NewAuthorizerNode(id string) *AuthorizerNode {
	return &AuthorizerNode{
		Provider: provider.Provider{
			ID:           id,
			ProviderType: spenum.Authorizer,
		},
	}
}

// GetAuthorizerNode returns error if node not found
func GetAuthorizerNode(id string, ctx state.StateContextI) (*AuthorizerNode, error) {
	var node = NewAuthorizerNode(id)
	err := ctx.GetTrieNode(node.GetKey(), node)
	if err != nil {
		return nil, err
	}
	if node.ProviderType != spenum.Authorizer {
		return nil, fmt.Errorf("provider is %s should be %s", node.ProviderType, spenum.Blobber)
	}
	return node, nil
}

// GetUserNode returns error if node not found
func GetUserNode(id string, ctx state.StateContextI) (*UserNode, error) {
	node := NewUserNode(id)
	err := ctx.GetTrieNode(node.GetKey(), node)
	switch err {
	case nil, util.ErrValueNotPresent:
		if node.BurnTickets == nil {
			node.BurnTickets = make(map[string][][]byte)
		}
		return node, nil
	default:
		return nil, err
	}
}

func GetGlobalSavedNode(ctx state.CommonStateContextI) (*GlobalNode, error) {
	node := &GlobalNode{ID: ADDRESS}
	err := ctx.GetTrieNode(node.GetKey(), node)
	switch err {
	case nil, util.ErrValueNotPresent:
		if node.ZCNSConfig == nil {
			node.ZCNSConfig = getConfig()
		}
		if node.WZCNNonceMinted == nil {
			node.WZCNNonceMinted = make(map[int64]bool)
		}
		return node, nil
	default:
		return nil, err
	}
}
