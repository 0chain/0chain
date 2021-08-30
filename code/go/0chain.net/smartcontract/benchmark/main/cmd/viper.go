package cmd

import (
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
}
