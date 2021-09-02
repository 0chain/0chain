package cmd

import (
	"fmt"
	"log"

	"0chain.net/smartcontract/multisigsc"

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

	if viper.GetInt(bk.NumClients) <= multisigsc.MaxSigners {
		log.Fatal(fmt.Errorf("number of clients %d must be greater than multi sig max singers %d",
			viper.GetInt(bk.NumClients), multisigsc.MaxSigners))
	}
	if viper.GetInt(bk.NumClients) >= viper.GetInt(bk.NumAllocations) {
		log.Fatal(fmt.Errorf("number of clients %d must be alt esst than the number of allocations %d",
			viper.GetInt(bk.NumClients), viper.GetInt(bk.NumAllocations)))
	}
}
