package config

/*DevConfig - these are to control features in development*/
type DevConfig struct {
	// All dev configurations are moved to Global Config
	// The type is not removed for the cases we need to
	// add and test a new developmnet feature
}

//DevConfiguration - for configuration of features in development
var DevConfiguration DevConfig

func setupDevConfig() {
	// Sample code in case for new configuration:
	// viper.SetDefault("development.state", false)
	// DevConfiguration.State = viper.GetBool("development.state")
}
