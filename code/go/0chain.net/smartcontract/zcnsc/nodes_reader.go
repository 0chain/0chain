package zcnsc

import (
	. "0chain.net/chaincore/config"
	"0chain.net/core/util"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
)

var (
	cfg = SmartContractConfig
)

// GetAuthorizerNode returns error if node not found
func GetAuthorizerNode(id string, ctx cstate.StateContextI) (*AuthorizerNode, error) {
	node := &AuthorizerNode{ID: id}
	raw, err := ctx.GetTrieNode(node.GetKey(), node)
	if err != nil {
		return nil, err
	}
	var ok bool
	if node, ok = raw.(*AuthorizerNode); !ok {
		return nil, fmt.Errorf("unexpected node type")
	}

	return node, nil
}

// GetUserNode returns error if node not found
func GetUserNode(id string, ctx cstate.StateContextI) (*UserNode, error) {
	node := NewUserNode(id, 0)
	raw, err := ctx.GetTrieNode(node.GetKey(), node)
	if err != nil {
		return nil, err
	}
	var ok bool
	if node, ok = raw.(*UserNode); !ok {
		return nil, fmt.Errorf("unexpected node type")
	}
	return node, nil
}

func GetGlobalSavedNode(balances cstate.StateContextI) (*GlobalNode, error) {
	node := &GlobalNode{ID: ADDRESS}
	raw, err := balances.GetTrieNode(node.GetKey(), node)
	switch err {
	case nil, util.ErrValueNotPresent:
		var ok bool
		if node, ok = raw.(*GlobalNode); !ok {
			return nil, fmt.Errorf("unexpected node type")
		}
		return node, nil
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
