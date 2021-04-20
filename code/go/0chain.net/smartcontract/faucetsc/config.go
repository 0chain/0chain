package faucetsc

import (
	"0chain.net/smartcontract"
	"context"
	"net/url"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
)

type faucetConfig struct {
	PourAmount      state.Balance `json:"pour_amount"`
	MaxPourAmount   state.Balance `json:"max_pour_amount"`
	PeriodicLimit   state.Balance `json:"periodic_limit"`
	GlobalLimit     state.Balance `json:"global_limit"`
	IndividualReset time.Duration `json:"individual_reset"` //in hours
	GlobalReset     time.Duration `json:"global_rest"`      //in hours
}

// configurations from sc.yaml
func getConfig() (conf *faucetConfig, err error) {
	conf = new(faucetConfig)
	conf.PourAmount = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.pour_amount"))
	conf.MaxPourAmount = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.max_pour_amount"))
	conf.PeriodicLimit = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.periodic_limit"))
	conf.GlobalLimit = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.faucetsc.global_limit"))
	conf.IndividualReset = config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.individual_reset")
	conf.GlobalReset = config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.global_reset")
	return
}

//
// REST-handler
//

func (fc *FaucetSmartContract) getConfigHandler(context.Context,
	url.Values, chainstate.StateContextI) (interface{}, error) {
	res, err := getConfig()
	if err != nil {
		err := smartcontract.NewError(smartcontract.FailRetrievingConfigErr, err.Error())
		return nil, smartcontract.WrapErrNoResource(err)
	}
	return res, nil
}
