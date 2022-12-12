package zcnsc

import (
	"0chain.net/chaincore/config"
	"github.com/0chain/common/core/util"

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
	node := NewUserNode(id)
	err := ctx.GetTrieNode(node.GetKey(), node)
	switch err {
	case nil, util.ErrValueNotPresent:
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
