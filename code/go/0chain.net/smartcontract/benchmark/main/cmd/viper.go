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

func GetViper(loadPath string) {
	viper.SetConfigType("yaml")
	viper.SetConfigName("benchmark")
	viper.AddConfigPath(loadPath)
	viper.AddConfigPath("../config/")
	viper.AddConfigPath("./testdata/")
	viper.AddConfigPath("./config/")
	viper.AddConfigPath(loadPath)
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
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

	if viper.GetInt(bk.NumClients) < viper.GetInt(bk.NumActiveClients) {
		log.Fatal(fmt.Errorf("number of clients %d less than then number of active clients %d",
			viper.GetInt(bk.NumClients), viper.GetInt(bk.NumActiveClients)))
	}

	if viper.GetInt(bk.NumActiveClients) < viper.GetInt(bk.NumMiners) {
		log.Fatal(fmt.Errorf("number of active clients %d less than then number of miners %d",
			viper.GetInt(bk.NumActiveClients), viper.GetInt(bk.NumMiners)))
	}

	if viper.GetInt(bk.NumActiveClients) < viper.GetInt(bk.NumSharders) {
		log.Fatal(fmt.Errorf("number of active clients %d less than then number of sharders %d",
			viper.GetInt(bk.NumActiveClients), viper.GetInt(bk.NumSharders)))
	}

	if viper.GetInt(bk.NumActiveClients) < viper.GetInt(bk.NumMinerDelegates) {
		log.Fatal(fmt.Errorf("number of active clients %d less than then number of minter delegates %d",
			viper.GetInt(bk.NumActiveClients), viper.GetInt(bk.NumMinerDelegates)))
	}
	if viper.GetInt(bk.NumActiveClients) < viper.GetInt(bk.NumSharderDelegates) {
		log.Fatal(fmt.Errorf("number of active clients %d less than then number of sharder delegates %d",
			viper.GetInt(bk.NumActiveClients), viper.GetInt(bk.NumSharderDelegates)))
	}
	if viper.GetInt(bk.NumActiveClients) < viper.GetInt(bk.NumAllocationPayerPools) {
		log.Fatal(fmt.Errorf("number of active clients %d less than then number of allocation pools %d",
			viper.GetInt(bk.NumActiveClients), viper.GetInt(bk.NumAllocationPayerPools)))
	}
	if viper.GetInt(bk.NumActiveClients) < viper.GetInt(bk.NumAllocationPayer) {
		log.Fatal(fmt.Errorf("number of active clients %d less than then number of allocation pools %d",
			viper.GetInt(bk.NumActiveClients), viper.GetInt(bk.NumAllocationPayer)))
	}

	if viper.GetInt(bk.NumBlobbersPerAllocation) > viper.GetInt(bk.NumBlobbers) {
		log.Fatal(fmt.Errorf("number of blobber per allocation %d grater than avalable blobbers %d",
			viper.GetInt(bk.NumBlobbersPerAllocation), viper.GetInt(bk.NumBlobbers)))
	}

	if viper.GetInt(bk.NumClients) < viper.GetInt(bk.NumAuthorizers) {
		log.Fatal(fmt.Errorf("number of clients %d less than authorisers %d",
			viper.GetInt(bk.NumClients), viper.GetInt(bk.NumAuthorizers)))
	}

	if viper.GetInt(bk.NumClients) <= multisigsc.MaxSigners {
		log.Fatal(fmt.Errorf("number of clients %d must be greater than multi sig max singers %d",
			viper.GetInt(bk.NumClients), multisigsc.MaxSigners))
	}

	if viper.GetInt(bk.NumClients) > viper.GetInt(bk.NumAllocations) {
		log.Fatal(fmt.Errorf("number of clients %d must not exceed the number of allocations %d",
			viper.GetInt(bk.NumClients), viper.GetInt(bk.NumAllocations)))
	}

	if viper.GetInt(bk.NumClients) < viper.GetInt(bk.NumMiners) {
		log.Fatal(fmt.Errorf("number of clients %d must be at least the number of miners %d",
			viper.GetInt(bk.NumClients), viper.GetInt(bk.NumMiners)))
	}

	if viper.GetInt(bk.NumClients) < viper.GetInt(bk.NumSharders) {
		log.Fatal(fmt.Errorf("number of clients %d must be at least the number of sharders %d",
			viper.GetInt(bk.NumClients), viper.GetInt(bk.NumSharders)))
	}

	if viper.GetInt(bk.NumActiveMiners) > viper.GetInt(bk.NumMiners) {
		log.Fatal(fmt.Errorf("number of active miners %d cannot exceeed the number of miners %d",
			viper.GetInt(bk.NumActiveMiners), viper.GetInt(bk.NumMiners)))
	}

	if viper.GetInt(bk.NumActiveMiners) > viper.GetInt(bk.NumMiners) {
		log.Fatal(fmt.Errorf("number of active miners %d cannot exceed the number of miners %d",
			viper.GetInt(bk.NumActiveMiners), viper.GetInt(bk.NumMiners)))
	}

	if viper.GetInt(bk.NumActiveSharders) > viper.GetInt(bk.NumSharders) {
		log.Fatal(fmt.Errorf("number of active sharders %d cannot exceed the number of sharders %d",
			viper.GetInt(bk.NumActiveSharders), viper.GetInt(bk.NumSharders)))
	}
	if viper.GetInt(bk.BenchDataListLength) <= 0 {
		log.Fatal(fmt.Errorf("bench_data_list_length %d, must be greater than zero", viper.GetInt(bk.BenchDataListLength)))
	}
}
