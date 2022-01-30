package zcnsc

import (
	"fmt"

	"0chain.net/chaincore/state"
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
)

type ZCNSConfig struct {
	MinMintAmount      state.Balance `json:"min_mint_amount"`
	PercentAuthorizers float64       `json:"percent_authorizers"`
	MinAuthorizers     int64         `json:"min_authorizers"`
	MinBurnAmount      int64         `json:"min_burn_amount"`
	MinStakeAmount     int64         `json:"min_stake_amount"`
	BurnAddress        string        `json:"burn_address"`
	MaxFee             int64         `json:"max_fee"`
}

func Section(section string) string {
	return fmt.Sprintf("%s.%s.%s", SmartContract, ZcnSc, section)
}

func loadSettings() (conf *ZCNSConfig) {
	conf = new(ZCNSConfig)
	conf.MinMintAmount = state.Balance(cfg.GetInt(Section(MinMintAmount)))
	conf.PercentAuthorizers = cfg.GetFloat64(Section(PercentAuthorizers))
	conf.MinAuthorizers = cfg.GetInt64(Section(MinAuthorizers))
	conf.MinBurnAmount = cfg.GetInt64(Section(MinBurnAmount))
	conf.MinStakeAmount = cfg.GetInt64(Section(MinStakeAmount))
	conf.BurnAddress = cfg.GetString(Section(BurnAddress))
	conf.MaxFee = cfg.GetInt64(Section(MaxFee))

	return conf
}
