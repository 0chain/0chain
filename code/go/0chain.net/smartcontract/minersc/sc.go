package minersc

import (
	"0chain.net/chaincore/smartcontractinterface"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

const (
	//ADDRESS address of minersc
	ADDRESS = "CF9C03CD22C9C7B116EED04E4A909F95ABEC17E98FE631D6AC94D5D8420C5B20"
)

//MinerSmartContract Smartcontract that takes care of all miner related requests
type MinerSmartContract struct {
	*smartcontractinterface.SmartContract
}

//SetSC setting up smartcontract as per interface
func (msc *MinerSmartContract) SetSC(sc *smartcontractinterface.SmartContract) {
	msc.SmartContract = sc
}

//Execute implemetning the interface
func (msc *MinerSmartContract) Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error) {

	switch funcName {
	default:
		return common.NewError("failed execution", "no function with that name").Error(), nil
	
	}
}
