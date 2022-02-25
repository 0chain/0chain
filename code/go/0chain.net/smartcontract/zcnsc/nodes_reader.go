package zcnsc

import (
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/core/util"

	cstate "0chain.net/chaincore/chain/state"
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
	node := &UserNode{ID: id}
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

	//if err != nil {
	//	if err != util.ErrValueNotPresent {
	//		return nil, err
	//	} else {
	//		return node, err
	//	}
	//}
	//if err := node.Decode(serializable.Encode()); err != nil {
	//	return nil, fmt.Errorf("%w: %v", common.ErrDecoding, err)
	//}
	//return node, nil
}

func GetGlobalNode(ctx cstate.StateContextI) (*GlobalNode, error) {
	gn := &GlobalNode{ID: ADDRESS}
	err := ctx.GetTrieNode(gn.GetKey(), gn)
	switch err {
	case nil:
		return gn, nil
	case util.ErrValueNotPresent:
		gn.MinMintAmount = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.zcn.min_mint_amount"))
		gn.PercentAuthorizers = config.SmartContractConfig.GetFloat64("smart_contracts.zcn.percent_authorizers")
		gn.MinAuthorizers = config.SmartContractConfig.GetInt64("smart_contracts.zcn.min_authorizers")
		gn.MinBurnAmount = config.SmartContractConfig.GetInt64("smart_contracts.zcn.min_burn_amount")
		gn.MinStakeAmount = config.SmartContractConfig.GetInt64("smart_contracts.zcn.min_stake_amount")
		gn.BurnAddress = config.SmartContractConfig.GetString("smart_contracts.zcn.burn_address")
		gn.MaxFee = config.SmartContractConfig.GetInt64("smart_contracts.zcn.max_fee")

		return gn, nil
	default:
		return nil, err
	}
}
