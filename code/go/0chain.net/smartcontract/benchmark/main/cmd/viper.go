package cmd

import (
	"fmt"

	"0chain.net/smartcontract/benchmark/main/cmd/log"

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
	config.SmartContractConfig = cviper.GetViper()
	validateConfig()
}

func validateConfig() {
	if 0 >= viper.GetInt(bk.AvailableKeys) {
		log.Fatal(fmt.Errorf("avalable keys %d must be grater than zero",
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

	if viper.GetInt(bk.NumClients) > viper.GetInt(bk.NumAllocations) {
		log.Fatal(fmt.Errorf("number of clients %d must be alt lest than the number of allocations %d",
			viper.GetInt(bk.NumClients), viper.GetInt(bk.NumAllocations)))
	}

	if viper.GetInt(bk.NumClients) < viper.GetInt(bk.NumSharders) {
		log.Fatal(fmt.Errorf("number of clients %d must be alt lest the number of miners %d",
			viper.GetInt(bk.NumClients), viper.GetInt(bk.NumMiners)))
	}

	if viper.GetInt(bk.NumClients) < viper.GetInt(bk.NumSharders) {
		log.Fatal(fmt.Errorf("number of clients %d must be alt lest the number of sharders %d",
			viper.GetInt(bk.NumClients), viper.GetInt(bk.NumMiners)))
	}
}
