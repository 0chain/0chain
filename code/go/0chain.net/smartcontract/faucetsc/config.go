package faucetsc

import (
	"encoding/json"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
)

type Setting int

const (
	PourAmount Setting = iota
	MaxPourAmount
	PeriodicLimit
	GlobalLimit
	IndividualReset
	GlobalReset
)

var (
	Settings = []string{
		"pour_amount",
		"max_pour_amount",
		"periodic_limit",
		"global_limit",
		"individual_reset",
		"global_rest",
	}
)

type inputMap struct {
	Fields map[string]interface{} `json:"fields"`
}

func (im *inputMap) Decode(input []byte) error {
	err := json.Unmarshal(input, im)
	if err != nil {
		return err
	}
	return nil
}

type FaucetConfig struct {
	PourAmount      state.Balance `json:"pour_amount"`
	MaxPourAmount   state.Balance `json:"max_pour_amount"`
	PeriodicLimit   state.Balance `json:"periodic_limit"`
	GlobalLimit     state.Balance `json:"global_limit"`
	IndividualReset time.Duration `json:"individual_reset"` //in hours
	GlobalReset     time.Duration `json:"global_rest"`      //in hours
}

// configurations from sc.yaml
func getConfig() (conf *FaucetConfig) {
	conf = new(FaucetConfig)
	conf.PourAmount = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.pour_amount"))
	conf.MaxPourAmount = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.max_pour_amount"))
	conf.PeriodicLimit = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.periodic_limit"))
	conf.GlobalLimit = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.global_limit"))
	conf.IndividualReset = config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.individual_reset")
	conf.GlobalReset = config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.global_reset")
	return
}
