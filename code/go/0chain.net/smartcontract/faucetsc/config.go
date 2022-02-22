package faucetsc

import (
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
)

//go:generate msgp -io=false -tests=false -v

type Setting int

const (
	PourAmount Setting = iota
	MaxPourAmount
	PeriodicLimit
	GlobalLimit
	IndividualReset
	GlobalReset
	OwnerId
)

var (
	Settings = []string{
		"pour_amount",
		"max_pour_amount",
		"periodic_limit",
		"global_limit",
		"individual_reset",
		"global_rest",
		"owner_id",
	}
)

type FaucetConfig struct {
	PourAmount      state.Balance `json:"pour_amount"`
	MaxPourAmount   state.Balance `json:"max_pour_amount"`
	PeriodicLimit   state.Balance `json:"periodic_limit"`
	GlobalLimit     state.Balance `json:"global_limit"`
	IndividualReset int64         `json:"individual_reset"`
	GlobalReset     int64         `json:"global_rest"`
	OwnerId         string        `json:"owner_id"`
}

// configurations from sc.yaml
func getConfig() (conf *FaucetConfig) {
	conf = new(FaucetConfig)
	conf.PourAmount = state.Balance(config.SmartContractConfig.GetFloat64("smart_contracts.faucetsc.pour_amount") * 1e10)
	conf.MaxPourAmount = state.Balance(config.SmartContractConfig.GetFloat64("smart_contracts.faucetsc.max_pour_amount") * 1e10)
	conf.PeriodicLimit = state.Balance(config.SmartContractConfig.GetFloat64("smart_contracts.faucetsc.periodic_limit") * 1e10)
	conf.GlobalLimit = state.Balance(config.SmartContractConfig.GetFloat64("smart_contracts.faucetsc.global_limit") * 1e10)
	conf.IndividualReset = int64(config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.individual_reset"))
	conf.GlobalReset = int64(config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.global_reset"))
	conf.OwnerId = config.SmartContractConfig.GetString("smart_contracts.faucetsc.owner_id")
	return
}
