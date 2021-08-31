package cmd

import (
	"fmt"
	"log"

	"0chain.net/chaincore/config"
	cviper "0chain.net/core/viper"
	sc "0chain.net/smartcontract/benchmark"
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
	validateConfig()
}

func validateConfig() {
	if 0 >= viper.GetInt(sc.AvailableKeys) {
		log.Fatalln(fmt.Errorf("avalable keys %d must be grater than zero",
			viper.GetInt(sc.AvailableKeys)))
	}
	if viper.GetInt(sc.NumClients) < viper.GetInt(sc.AvailableKeys) {
		log.Fatal(fmt.Errorf("number of clients %d less than avalable keys %d",
			viper.GetInt(sc.NumClients), viper.GetInt(sc.AvailableKeys)))
	}
}
