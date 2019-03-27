package config

import "github.com/spf13/viper"

/*DevConfig - these are to control features in development*/
type DevConfig struct {
	State         bool
	IsDkgEnabled  bool
	SmartContract bool
	IsFeeEnabled  bool
}

//DevConfiguration - for configuration of features in development
var DevConfiguration DevConfig

func setupDevConfig() {
	viper.SetDefault("development.state", false)
	viper.SetDefault("development.smart_contract", false)
	viper.SetDefault("development.txn_fee", false)
	DevConfiguration.State = viper.GetBool("development.state")
	DevConfiguration.IsDkgEnabled = viper.GetBool("development.dkg")
	DevConfiguration.SmartContract = viper.GetBool("development.smart_contract")
	DevConfiguration.IsFeeEnabled = viper.GetBool("development.txn_fee")
}
