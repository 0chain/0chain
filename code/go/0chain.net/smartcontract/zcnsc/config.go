package zcnsc

import (
	"fmt"
	"strings"

	"0chain.net/chaincore/currency"

	"0chain.net/core/common"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract"
	"github.com/pkg/errors"
)

const (
	SmartContract = "smart_contracts"
	ZcnSc         = "zcnsc"
)

const (
	MinMintAmount      = "min_mint"
	PercentAuthorizers = "percent_authorizers"
	MinAuthorizers     = "min_authorizers"
	MinBurnAmount      = "min_burn"
	MinStakeAmount     = "min_stake"
	MinLockAmount      = "min_lock"
	BurnAddress        = "burn_address"
	MaxFee             = "max_fee"
	OwnerID            = "owner_id"
	Cost               = "cost"
	MaxDelegates       = "max_delegates"
)

var CostFunctions = []string{
	MintFunc,
	BurnFunc,
	DeleteAuthorizerFunc,
	AddAuthorizerFunc,
}

func (zcn *ZCNSmartContract) UpdateGlobalConfig(t *transaction.Transaction, inputData []byte, ctx chain.StateContextI) (string, error) {
	const (
		Code     = "failed to update configuration"
		FuncName = "UpdateGlobalConfig"
	)

	gn, err := GetGlobalNode(ctx)
	if err != nil {
		return "", errors.Wrap(err, Code)
	}

	if err := smartcontractinterface.AuthorizeWithOwner(FuncName, func() (bool, error) {
		return gn.OwnerId == t.ClientID, nil
	}); err != nil {
		return "", errors.Wrap(err, Code)
	}

	var input smartcontract.StringMap
	err = input.Decode(inputData)
	if err != nil {
		return "", errors.Wrap(err, Code)
	}

	if err := gn.UpdateConfig(&input); err != nil {
		return "", errors.Wrap(err, Code)
	}

	if err = gn.Validate(); err != nil {
		return "", common.NewError(Code, "cannot validate changes: "+err.Error())
	}

	_, err = ctx.InsertTrieNode(gn.GetKey(), gn)
	if err != nil {
		return "", common.NewError(Code, "saving global node: "+err.Error())
	}

	return string(gn.Encode()), nil
}

func (gn *GlobalNode) ToStringMap() smartcontract.StringMap {
	fields := map[string]string{
		MinMintAmount:      fmt.Sprintf("%v", gn.MinMintAmount),
		MinBurnAmount:      fmt.Sprintf("%v", gn.MinBurnAmount),
		MinStakeAmount:     fmt.Sprintf("%v", gn.MinStakeAmount),
		PercentAuthorizers: fmt.Sprintf("%v", gn.PercentAuthorizers),
		MinAuthorizers:     fmt.Sprintf("%v", gn.MinAuthorizers),
		MinLockAmount:      fmt.Sprintf("%v", gn.MinLockAmount),
		MaxFee:             fmt.Sprintf("%v", gn.MaxFee),
		BurnAddress:        fmt.Sprintf("%v", gn.BurnAddress),
		OwnerID:            fmt.Sprintf("%v", gn.OwnerId),
		MaxDelegates:       fmt.Sprintf("%v", gn.MaxDelegates),
	}

	for _, key := range CostFunctions {
		fields[fmt.Sprintf("cost.%s", key)] = fmt.Sprintf("%0v", gn.Cost[strings.ToLower(key)])
	}

	return smartcontract.StringMap{
		Fields: fields,
	}
}

func postfix(section string) string {
	return fmt.Sprintf("%s.%s.%s", SmartContract, ZcnSc, section)
}

func loadGlobalNode() (conf *ZCNSConfig) {
	conf = new(ZCNSConfig)
	conf.MinMintAmount = currency.Coin(cfg.GetInt(postfix(MinMintAmount)))
	conf.MinBurnAmount = currency.Coin(cfg.GetInt64(postfix(MinBurnAmount)))
	conf.MinStakeAmount = currency.Coin(cfg.GetInt64(postfix(MinStakeAmount)))
	conf.PercentAuthorizers = cfg.GetFloat64(postfix(PercentAuthorizers))
	conf.MinAuthorizers = cfg.GetInt64(postfix(MinAuthorizers))
	conf.MinLockAmount = cfg.GetInt64(postfix(MinLockAmount))
	conf.MaxFee = currency.Coin(cfg.GetInt64(postfix(MaxFee)))
	conf.BurnAddress = cfg.GetString(postfix(BurnAddress))
	conf.OwnerId = cfg.GetString(postfix(OwnerID))
	conf.Cost = cfg.GetStringMapInt(postfix(Cost))
	conf.MaxDelegates = cfg.GetInt(postfix(MaxDelegates))

	return conf
}
