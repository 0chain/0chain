package scfactory

import (
	"0chain.net/chaincore/smartcontract"
	"0chain.net/smartcontract/setupsc"
)

func SetUpSmartContractFactory() {
	smartcontract.SmartContractFactory = setupsc.NewSmartContractFactory()
}
