package setupsc

import (
	"fmt"

	"github.com/spf13/viper"

	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
)

var scs = []sci.SmartContractInterface{
	&faucetsc.FaucetSmartContract{}, &storagesc.StorageSmartContract{},
	&interestpoolsc.InterestPoolSmartContract{}, &multisigsc.MultiSigSmartContract{},
	&minersc.MinerSmartContract{}, &vestingsc.VestingSmartContract{},
}

//SetupSmartContracts initialize smartcontract addresses
func SetupSmartContracts() {
	for _, sc := range scs {
		if viper.GetBool(fmt.Sprintf("development.smart_contract.%v", sc.GetName())) {
			sc.InitSC()
			smartcontract.ContractMap[sc.GetAddress()] = sc
		}
	}
}
