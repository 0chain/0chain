package faucetsc

import (
	"context"
	"net/url"
	"time"

	"github.com/0chain/0chain/code/go/0chain.net/core/common"

	chainstate "github.com/0chain/0chain/code/go/0chain.net/chaincore/chain/state"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/config"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/state"
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

const cantGetConfig = "can't get config"

func (fc *FaucetSmartContract) getConfigHandler(context.Context,
	url.Values, chainstate.StateContextI) (interface{}, error) {
	res, err := getConfig()
	if err != nil {
		return nil, common.NewErrNoResource(cantGetConfig, err.Error())
	}
	return res, nil
}
