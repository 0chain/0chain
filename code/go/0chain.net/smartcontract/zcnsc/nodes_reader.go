package zcnsc

import (
	"0chain.net/smartcontract/common"
	"fmt"

	"0chain.net/core/config"
	"0chain.net/smartcontract/provider"

	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/util"
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
func GetAuthorizerNode(id string, ctx common.StateContextI) (*AuthorizerNode, error) {
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
func GetUserNode(id string, ctx common.StateContextI) (*UserNode, error) {
	node := NewUserNode(id)
	err := ctx.GetTrieNode(node.GetKey(), node)
	switch err {
	case nil, util.ErrValueNotPresent:
		return node, nil
	default:
		return nil, err
	}
}

func GetGlobalSavedNode(ctx common.CommonStateContextI) (*GlobalNode, error) {
	node := &GlobalNode{ID: ADDRESS}
	err := ctx.GetTrieNode(node.GetKey(), node)
	switch err {
	case nil, util.ErrValueNotPresent:
		if node.ZCNSConfig == nil {
			node.ZCNSConfig, err = getConfig()
			if err != nil {
				return nil, err
			}
		}
		return node, nil
	default:
		return nil, err
	}
}
