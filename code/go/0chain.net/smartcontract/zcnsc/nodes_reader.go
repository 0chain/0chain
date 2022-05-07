package zcnsc

import (
	"0chain.net/chaincore/config"
	"0chain.net/core/util"

	"0chain.net/chaincore/chain/state"
)

var (
	cfg = config.SmartContractConfig
)

// GetAuthorizerNode returns error if node not found
func GetAuthorizerNode(id string, ctx state.StateContextI) (*AuthorizerNode, error) {
	node := &AuthorizerNode{ID: id}
	err := ctx.GetTrieNode(node.GetKey(), node)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// GetUserNode returns error if node not found
func GetUserNode(id string, ctx state.StateContextI) (*UserNode, error) {
	node := NewUserNode(id, 0)
	err := ctx.GetTrieNode(node.GetKey(), node)
	return node, err
}

func GetGlobalNode(ctx state.StateContextI) (*GlobalNode, error) {
	node := &GlobalNode{ID: ADDRESS}
	err := ctx.GetTrieNode(node.GetKey(), node)
	switch err {
	case nil, util.ErrValueNotPresent:
		if node.ZCNSConfig == nil {
			node.ZCNSConfig = loadGlobalNode()
		}
		return node, err
	default:
		return nil, err
	}
}
