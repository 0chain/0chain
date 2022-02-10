package zcnsc

import (
	"encoding/json"
	"fmt"

	"0chain.net/core/common"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract"
	"github.com/pkg/errors"
)

const (
	SmartContract = "smart_contracts"
	ZcnSc         = "zcnsc"
)

const (
	MinMintAmount      = "min_mint_amount"
	PercentAuthorizers = "percent_authorizers"
	MinAuthorizers     = "min_authorizers"
	MinBurnAmount      = "min_burn_amount"
	MinStakeAmount     = "min_stake_amount"
	BurnAddress        = "burn_address"
	MaxFee             = "max_fee"
	OwnerID            = "owner_id"
)

type AuthorizerConfig struct {
	Fee state.Balance `json:"fee"`
}

type ZCNSConfig struct {
	MinMintAmount      state.Balance `json:"min_mint_amount"`
	MinBurnAmount      state.Balance `json:"min_burn_amount"`
	MinStakeAmount     state.Balance `json:"min_stake_amount"`
	MaxFee             state.Balance `json:"max_fee"`
	PercentAuthorizers float64       `json:"percent_authorizers"`
	MinAuthorizers     int64         `json:"min_authorizers"`
	BurnAddress        string        `json:"burn_address"`
	OwnerId            datastore.Key `json:"owner_id"`
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

	if err := smartcontractinterface.AuthorizeWithOwner(FuncName, func() bool {
		return gn.Config.OwnerId == t.ClientID
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

func (cfg *ZCNSConfig) ToStringMap() (res *smartcontract.StringMap, err error) {
	bytes, err := json.Marshal(cfg)
	if err != nil {
		return res, errors.Wrap(err, "failed to convert config to StringMap")
	}

	var stringMap map[string]interface{}

	err = json.Unmarshal(bytes, &stringMap)
	if err != nil {
		return res, errors.Wrap(err, "failed to convert config to StringMap")
	}

	res = new(smartcontract.StringMap)
	res.Fields = make(map[string]string)

	for k, v := range stringMap {
		res.Fields[k] = fmt.Sprintf("%v", v)
	}

	return
}

func Section(section string) string {
	return fmt.Sprintf("%s.%s.%s", SmartContract, ZcnSc, section)
}

func loadSettings() (conf *ZCNSConfig) {
	conf = new(ZCNSConfig)
	conf.MinMintAmount = state.Balance(cfg.GetInt(Section(MinMintAmount)))
	conf.PercentAuthorizers = cfg.GetFloat64(Section(PercentAuthorizers))
	conf.MinAuthorizers = cfg.GetInt64(Section(MinAuthorizers))
	conf.MinBurnAmount = state.Balance(cfg.GetInt64(Section(MinBurnAmount)))
	conf.MinStakeAmount = state.Balance(cfg.GetInt64(Section(MinStakeAmount)))
	conf.BurnAddress = cfg.GetString(Section(BurnAddress))
	conf.MaxFee = state.Balance(cfg.GetInt64(Section(MaxFee)))
	conf.OwnerId = cfg.GetString(Section(OwnerID))

	return conf
}
