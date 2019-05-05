package setupsc

import (
	"fmt"

	"github.com/spf13/viper"

	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/feesc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/zrc20sc"
)

var scs = []sci.SmartContractInterface{
	&faucetsc.FaucetSmartContract{}, &storagesc.StorageSmartContract{},
	&zrc20sc.ZRC20SmartContract{}, &interestpoolsc.InterestPoolSmartContract{},
	&minersc.MinerSmartContract{}, &feesc.FeeSmartContract{},
}

//SetupSmartContracts initialize smartcontract addresses
func SetupSmartContracts() {
	for _, sc := range scs {
		if viper.GetBool(fmt.Sprintf("development.smart_contract.%v", sc.GetName())) {
			smartcontract.ContractMap[sc.GetAddress()] = sc
		}
	}
}
