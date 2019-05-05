package config

import "github.com/spf13/viper"

/*DevConfig - these are to control features in development*/
type DevConfig struct {
	State         bool
	IsDkgEnabled  bool
	FaucetEnabled bool
	IsFeeEnabled  bool
}

//DevConfiguration - for configuration of features in development
var DevConfiguration DevConfig

func setupDevConfig() {
	viper.SetDefault("development.state", false)
	viper.SetDefault("development.smart_contract.fee", false)
	viper.SetDefault("development.smart_contract.faucet", false)
	DevConfiguration.State = viper.GetBool("development.state")
	DevConfiguration.IsDkgEnabled = viper.GetBool("development.dkg")
	DevConfiguration.FaucetEnabled = viper.GetBool("development.smart_contract.faucet")
	DevConfiguration.IsFeeEnabled = viper.GetBool("development.smart_contract.fee")
}
