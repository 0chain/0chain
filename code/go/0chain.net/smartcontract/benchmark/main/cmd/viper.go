package cmd

import (
	"0chain.net/chaincore/config"
	cviper "0chain.net/core/viper"
	"github.com/spf13/viper"
)

func GetViper(path string) {
	viper.SetConfigType("yaml")
	viper.SetConfigName("benchmark")
	viper.AddConfigPath("../config/")
	viper.AddConfigPath("./testdata/")
	viper.AddConfigPath("./config/")
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	aa := config.SmartContractConfig
	aa = aa

	config.SmartContractConfig = cviper.GetViper()
}
