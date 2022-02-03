package zcnsc

import (
	"fmt"
	"reflect"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/core/util"

	cstate "0chain.net/chaincore/chain/state"
)

type persistentNode interface {
	util.Serializable
	GetKey() string
}

func isNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}

// GetAuthorizerNode returns error if node not found
func GetAuthorizerNode(id string, ctx cstate.StateContextI) (*AuthorizerNode, error) {
	node := &AuthorizerNode{ID: id}
	raw, err := ctx.GetTrieNode(node.GetKey(), node)
	if err != nil {
		return node, err
	}

	if isNil(raw) {
		return nil, fmt.Errorf("authorizer node (%s) not found", id)
	}
	var ok bool
	if node, ok = raw.(*AuthorizerNode); !ok {
		return nil, fmt.Errorf("unexpected node type")
	}
	return node, err
}

// GetUserNode returns error if node not found
func GetUserNode(id string, ctx cstate.StateContextI) (*UserNode, error) {
	node := &UserNode{ID: id}
	raw, err := ctx.GetTrieNode(node.GetKey(), node)
	if err != nil {
		return node, err
	}

	if isNil(raw) {
		return nil, fmt.Errorf("user node: %s not found", id)
	}
	var ok bool
	if node, ok = raw.(*UserNode); !ok {
		return nil, fmt.Errorf("unexpected node type")
	}
	return node, err
}

func GetGlobalSavedNode(balances cstate.StateContextI) (*GlobalNode, error) {
	node := &GlobalNode{ID: ADDRESS}
	raw, err := balances.GetTrieNode(node.GetKey(), node)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		} else {
			return node, err
		}
	}
	var ok bool
	if node, ok = raw.(*GlobalNode); !ok {
		return nil, fmt.Errorf("unexpected node type")
	}
	return node, nil
}

func GetGlobalNode(ctx cstate.StateContextI) (*GlobalNode, error) {
	gn, err := GetGlobalSavedNode(ctx)
	if err == nil {
		return gn, nil
	}

	if gn == nil {
		return nil, err
	}

	gn.MinMintAmount = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.zcn.min_mint_amount"))
	gn.PercentAuthorizers = config.SmartContractConfig.GetFloat64("smart_contracts.zcn.percent_authorizers")
	gn.MinAuthorizers = config.SmartContractConfig.GetInt64("smart_contracts.zcn.min_authorizers")
	gn.MinBurnAmount = config.SmartContractConfig.GetInt64("smart_contracts.zcn.min_burn_amount")
	gn.MinStakeAmount = config.SmartContractConfig.GetInt64("smart_contracts.zcn.min_stake_amount")
	gn.BurnAddress = config.SmartContractConfig.GetString("smart_contracts.zcn.burn_address")
	gn.MaxFee = config.SmartContractConfig.GetInt64("smart_contracts.zcn.max_fee")

	return gn, nil
}
