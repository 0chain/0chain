package faucetsc

import (
	"sync"
	"time"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/util"
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
	Cost
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
		"cost",
	}

	costFunctions = []string{
		"update-settings",
		"pour",
		"refill",
	}
)

type FaucetConfig struct {
	PourAmount      currency.Coin  `json:"pour_amount"`
	MaxPourAmount   currency.Coin  `json:"max_pour_amount"`
	PeriodicLimit   currency.Coin  `json:"periodic_limit"`
	GlobalLimit     currency.Coin  `json:"global_limit"`
	IndividualReset time.Duration  `json:"individual_reset"`
	GlobalReset     time.Duration  `json:"global_rest"`
	OwnerId         string         `json:"owner_id"`
	Cost            map[string]int `json:"cost"`
}

type cache struct {
	config *FaucetConfig
	l      sync.RWMutex
}

var c = &cache{
	l: sync.RWMutex{},
}

func (*cache) update(conf *FaucetConfig) {
	c.l.Lock()
	c.config = conf
	c.l.Unlock()
}

// configurations from sc.yaml
func getFaucetConfig() (conf *FaucetConfig, err error) {

	conf = new(FaucetConfig)
	conf.PourAmount, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64("smart_contracts.faucetsc.pour_amount"))
	if err != nil {
		return nil, err
	}
	conf.MaxPourAmount, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64("smart_contracts.faucetsc.max_pour_amount"))
	if err != nil {
		return nil, err
	}
	conf.PeriodicLimit, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64("smart_contracts.faucetsc.periodic_limit"))
	if err != nil {
		return nil, err
	}
	conf.GlobalLimit, err = currency.ParseZCN(config.SmartContractConfig.GetFloat64("smart_contracts.faucetsc.global_limit"))
	if err != nil {
		return nil, err
	}
	conf.IndividualReset = config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.individual_reset")
	conf.GlobalReset = config.SmartContractConfig.GetDuration("smart_contracts.faucetsc.global_reset")
	conf.OwnerId = config.SmartContractConfig.GetString("smart_contracts.faucetsc.owner_id")
	conf.Cost = config.SmartContractConfig.GetStringMapInt("smart_contracts.faucetsc.cost")
	return
}

func InitConfig(balances cstate.CommonStateContextI) error {
	c.l.Lock()
	defer c.l.Unlock()

	var gn = &GlobalNode{ID: ADDRESS}
	err := balances.GetTrieNode(globalNodeKey, gn)
	if err == util.ErrValueNotPresent {
		gn.FaucetConfig, err = getFaucetConfig()
		if err != nil {
			return err
		}
		_, err = balances.InsertTrieNode(globalNodeKey, gn)
		if err != nil {
			return err
		}
	}
	c.config = gn.FaucetConfig
	return err
}
