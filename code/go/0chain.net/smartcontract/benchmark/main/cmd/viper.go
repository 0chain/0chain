package cmd

import (
	"fmt"
	"log"

	"0chain.net/chaincore/config"
	cviper "0chain.net/core/viper"
	bk "0chain.net/smartcontract/benchmark"
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
	if 0 >= viper.GetInt(bk.AvailableKeys) {
		log.Fatalln(fmt.Errorf("avalable keys %d must be grater than zero",
			viper.GetInt(bk.AvailableKeys)))
	}
	if viper.GetInt(bk.NumClients) < viper.GetInt(bk.AvailableKeys) {
		log.Fatal(fmt.Errorf("number of clients %d less than avalable keys %d",
			viper.GetInt(bk.NumClients), viper.GetInt(bk.AvailableKeys)))
	}
	if viper.GetInt(bk.NumMiners) < viper.GetInt(bk.AvailableKeys) {
		log.Fatal(fmt.Errorf("number of miners %d less than avalable keys %d",
			viper.GetInt(bk.NumMiners), viper.GetInt(bk.AvailableKeys)))
	}
	if viper.GetInt(bk.NumSharders) < viper.GetInt(bk.AvailableKeys) {
		log.Fatal(fmt.Errorf("number of sharders %d less than avalable keys %d",
			viper.GetInt(bk.NumSharders), viper.GetInt(bk.AvailableKeys)))
	}

	if viper.GetInt(bk.NumBlobbersPerAllocation) < viper.GetInt(bk.AvailableKeys) {
		log.Fatal(fmt.Errorf("number of blobber per allocation %d must be lestt than the avalable keys %d",
			viper.GetInt(bk.NumSharders), viper.GetInt(bk.AvailableKeys)))
	}

}
