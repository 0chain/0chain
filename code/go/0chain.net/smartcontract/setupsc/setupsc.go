package setupsc

import (
	"0chain.net/chaincore/smartcontract"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/zrc20sc"
)

func SetupSmartContracts() {
	smartcontract.ContractMap[faucetsc.ADDRESS] = &faucetsc.FaucetSmartContract{}
	smartcontract.ContractMap[storagesc.ADDRESS] = &storagesc.StorageSmartContract{}
	smartcontract.ContractMap[zrc20sc.ADDRESS] = &zrc20sc.ZRC20SmartContract{}
	smartcontract.ContractMap[interestpoolsc.ADDRESS] = &interestpoolsc.InterestPoolSmartContract{}
}
