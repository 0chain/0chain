package zcnsc

import (
	. "0chain.net/chaincore/config"
	"0chain.net/core/util"

	cstate "0chain.net/chaincore/chain/state"
)

var (
	cfg = SmartContractConfig
)

// GetAuthorizerNode returns error if node not found
func GetAuthorizerNode(id string, ctx cstate.StateContextI) (*AuthorizerNode, error) {
	node := &AuthorizerNode{ID: id}
	err := ctx.GetTrieNode(node.GetKey(), node)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// GetUserNode returns error if node not found
func GetUserNode(id string, ctx cstate.StateContextI) (*UserNode, error) {
	node := NewUserNode(id, 0)
	err := ctx.GetTrieNode(node.GetKey(), node)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func GetGlobalSavedNode(balances cstate.StateContextI) (*GlobalNode, error) {
	node := &GlobalNode{ID: ADDRESS}
	err := balances.GetTrieNode(node.GetKey(), node)
	switch err {
	case nil, util.ErrValueNotPresent:
		return node, err
	default:
		return nil, err
	}
}

func GetGlobalNode(ctx cstate.StateContextI) (*GlobalNode, error) {
	gn, err := GetGlobalSavedNode(ctx)
	if err == nil {
		return gn, nil
	}

	if gn == nil {
		return nil, err
	}

	return gn, nil
}
