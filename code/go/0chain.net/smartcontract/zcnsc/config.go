package zcnsc

import (
	"fmt"
	"strings"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/config"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/util"

	"0chain.net/core/common"

	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"github.com/pkg/errors"
)

const (
	SmartContract = "smart_contracts"
	ZcnSc         = "zcnsc"
)

const (
	MinMintAmount       = "min_mint"
	PercentAuthorizers  = "percent_authorizers"
	MinAuthorizers      = "min_authorizers"
	MinBurnAmount       = "min_burn"
	MinStakeAmount      = "min_stake"
	MinStakePerDelegate = "min_stake_per_delegate"
	MaxStakeAmount      = "max_stake"
	MinLockAmount       = "min_lock"
	MaxFee              = "max_fee"
	OwnerID             = "owner_id"
	Cost                = "cost"
	MaxDelegates        = "max_delegates"
	HealthCheckPeriod   = "health_check_period"
)

var CostFunctions = []string{
	MintFunc,
	BurnFunc,
	DeleteAuthorizerFunc,
	AddAuthorizerFunc,
}

// InitConfig initializes global node config to MPT
func InitConfig(ctx state.StateContextI) error {
	node := &GlobalNode{ID: ADDRESS}
	err := ctx.GetTrieNode(node.GetKey(), node)
	if err == util.ErrValueNotPresent {
		node.ZCNSConfig, err = getConfig()
		if err != nil {
			return err
		}
		_, err := ctx.InsertTrieNode(node.GetKey(), node)
		return err
	}
	return err
}

func GetGlobalNode(ctx state.CommonStateContextI) (*GlobalNode, error) {
	return GetGlobalSavedNode(ctx)
}

func (zcn *ZCNSmartContract) UpdateGlobalConfig(t *transaction.Transaction, inputData []byte, ctx state.StateContextI) (string, error) {
	const (
		Code     = "failed to update configuration"
		FuncName = "UpdateGlobalConfig"
	)

	gn, err := GetGlobalNode(ctx)
	if err != nil {
		return "", errors.Wrap(err, Code)
	}

	if err := smartcontractinterface.AuthorizeWithOwner(FuncName, func() bool {
		return gn.OwnerId == t.ClientID
	}); err != nil {
		return "", errors.Wrap(err, Code)
	}

	var input config.StringMap
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

func (gn *GlobalNode) ToStringMap() config.StringMap {
	fields := map[string]string{
		MinMintAmount:       fmt.Sprintf("%v", gn.MinMintAmount),
		MinBurnAmount:       fmt.Sprintf("%v", gn.MinBurnAmount),
		MinStakeAmount:      fmt.Sprintf("%v", gn.MinStakeAmount),
		MinStakePerDelegate: fmt.Sprintf("%v", gn.MinStakePerDelegate),
		MaxStakeAmount:      fmt.Sprintf("%v", gn.MaxStakeAmount),
		PercentAuthorizers:  fmt.Sprintf("%v", gn.PercentAuthorizers),
		MinAuthorizers:      fmt.Sprintf("%v", gn.MinAuthorizers),
		MinLockAmount:       fmt.Sprintf("%v", gn.MinLockAmount),
		MaxFee:              fmt.Sprintf("%v", gn.MaxFee),
		OwnerID:             fmt.Sprintf("%v", gn.OwnerId),
		MaxDelegates:        fmt.Sprintf("%v", gn.MaxDelegates),
		HealthCheckPeriod:   fmt.Sprintf("%v", gn.HealthCheckPeriod),
	}

	for _, key := range CostFunctions {
		fields[fmt.Sprintf("cost.%s", key)] = fmt.Sprintf("%0v", gn.Cost[strings.ToLower(key)])
	}

	return config.StringMap{
		Fields: fields,
	}
}

func postfix(section string) string {
	return fmt.Sprintf("%s.%s.%s", SmartContract, ZcnSc, section)
}

func getConfig() (conf *ZCNSConfig, err error) {
	conf = new(ZCNSConfig)
	conf.MinMintAmount, err = currency.ParseZCN(cfg.GetFloat64(postfix(MinMintAmount)))
	if err != nil {
		return nil, err
	}
	conf.MinBurnAmount, err = currency.ParseZCN(cfg.GetFloat64(postfix(MinBurnAmount)))
	if err != nil {
		return nil, err
	}
	conf.MinStakeAmount, err = currency.ParseZCN(cfg.GetFloat64(postfix(MinStakeAmount)))
	if err != nil {
		return nil, err
	}
	conf.MinStakePerDelegate, err = currency.ParseZCN(cfg.GetFloat64(postfix(MinStakePerDelegate)))
	if err != nil {
		return nil, err
	}
	conf.MaxStakeAmount, err = currency.ParseZCN(cfg.GetFloat64(postfix(MaxStakeAmount)))
	if err != nil {
		return nil, err
	}
	conf.PercentAuthorizers = cfg.GetFloat64(postfix(PercentAuthorizers))
	conf.MinAuthorizers = cfg.GetInt64(postfix(MinAuthorizers))
	conf.MinLockAmount, err = currency.ParseZCN(cfg.GetFloat64(postfix(MinLockAmount)))
	if err != nil {
		return nil, err
	}
	conf.MaxFee = currency.Coin(cfg.GetFloat64(postfix(MaxFee)))
	conf.OwnerId = cfg.GetString(postfix(OwnerID))
	conf.Cost = cfg.GetStringMapInt(postfix(Cost))
	conf.MaxDelegates = cfg.GetInt(postfix(MaxDelegates))
	conf.HealthCheckPeriod = cfg.GetDuration(postfix(HealthCheckPeriod))

	return conf, nil
}
