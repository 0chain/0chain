package config

import "github.com/spf13/viper"

/*DevConfig - these are to control features in development*/
type DevConfig struct {
	State         bool
	IsDkgEnabled  bool
	FaucetEnabled bool
	IsFeeEnabled  bool
	ViewChange    bool
}

//DevConfiguration - for configuration of features in development
var DevConfiguration DevConfig

func setupDevConfig() {
	viper.SetDefault("development.state", false)
	viper.SetDefault("development.smart_contract.miner", false)
	viper.SetDefault("development.smart_contract.faucet", false)
	viper.SetDefault("development.view_change", false)
	DevConfiguration.State = viper.GetBool("development.state")
	DevConfiguration.IsDkgEnabled = viper.GetBool("development.dkg")
	DevConfiguration.FaucetEnabled = viper.GetBool("development.smart_contract.faucet")
	DevConfiguration.IsFeeEnabled = viper.GetBool("development.smart_contract.miner")
	DevConfiguration.ViewChange = viper.GetBool("development.view_change")
}
