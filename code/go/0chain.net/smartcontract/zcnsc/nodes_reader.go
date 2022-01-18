package zcnsc

import (
	"fmt"
	"reflect"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/core/util"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
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
	blob, err := ctx.GetTrieNode(node.GetKey())
	if err != nil {
		return node, err
	}

	if isNil(blob) {
		return nil, fmt.Errorf("authorizer node (%s) not found", id)
	}

	if err := node.Decode(blob.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %v", common.ErrDecoding, err)
	}

	return node, err
}

// GetUserNode returns error if node not found
func GetUserNode(id string, ctx cstate.StateContextI) (*UserNode, error) {
	node := &UserNode{ID: id}
	blob, err := ctx.GetTrieNode(node.GetKey())
	if err != nil {
		return node, err
	}

	if isNil(blob) {
		return nil, fmt.Errorf("user node: %s not found", id)
	}

	if err := node.Decode(blob.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}

	return node, err
}

func GetGlobalSavedNode(balances cstate.StateContextI) (*GlobalNode, error) {
	node := &GlobalNode{ID: ADDRESS}
	serializable, err := balances.GetTrieNode(node.GetKey())
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		} else {
			return node, err
		}
	}
	if err := node.Decode(serializable.Encode()); err != nil {
		return nil, fmt.Errorf("%w: %v", common.ErrDecoding, err)
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
