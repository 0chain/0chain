package config

import "github.com/spf13/viper"

/*DevConfig - these are to control features in development*/
type DevConfig struct {
	State bool
}

//DevConfiguration - for configuration of features in development
var DevConfiguration DevConfig

func setupDevConfig() {
	viper.SetDefault("development.state", false)
	DevConfiguration.State = viper.GetBool("development.state")
}
