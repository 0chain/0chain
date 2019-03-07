package feesc

import (
	"fmt"

	"0chain.net/chaincore/block"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

const (
	Seperator = smartcontractinterface.Seperator
	owner     = "c8a5e74c2f4fae2c1bed79fb2b78d3b88f844bbb6bf1db5fc43240711f23321f"
	ADDRESS   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d1"
)

type FeeSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (fsc *FeeSmartContract) SetSC(sc *smartcontractinterface.SmartContract) {
	fsc.SmartContract = sc
}

func (fsc *FeeSmartContract) sumFee(b *block.Block) state.Balance {
	var totalMaxFee int64
	for _, txn := range b.Txns {
		totalMaxFee += txn.Fee
	}
	return state.Balance(totalMaxFee)
}

func (fsc *FeeSmartContract) payFees(t *transaction.Transaction, inputData []byte, gn *globalNode, balances c_state.StateContextI) (string, error) {
	block := balances.GetBlock()
	if t.ClientID != block.MinerID {
		return common.NewError("failed to pay fees", "not block generator").Error(), nil
	}
	if block.Round <= gn.LastRound {
		return common.NewError("failed to pay fees", "jumped back in time?").Error(), nil
	}
	var resp string
	fee := fsc.sumFee(block)
	transfer := state.NewTransfer(ADDRESS, t.ClientID, fee)
	balances.AddTransfer(transfer)
	resp += string(transfer.Encode())
	sharders := balances.GetBlockSharders(block.PrevBlock)
	for _, sharder := range sharders {
		//TODO: the mint amount will be controlled by governance
		mint := state.NewMint(ADDRESS, sharder, fee/state.Balance(len(sharders)))
		err := balances.AddMint(mint)
		if err != nil {
			resp += common.NewError("failed to mint", fmt.Sprintf("errored while adding mint for sharder %v: %v", sharder, err.Error())).Error()
		}
	}
	gn.LastRound = block.Round
	fsc.DB.PutNode(gn.getKey(), gn.encode())
	return resp, nil
}

func (fsc *FeeSmartContract) getGlobalNode() *globalNode {
	gn := &globalNode{ID: ADDRESS}
	globalBytes, err := fsc.DB.GetNode(gn.getKey())
	if err == nil {
		err = gn.decode(globalBytes)
		if err == nil {
			return gn
		}
	}
	return gn
}

func (fsc *FeeSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	gn := fsc.getGlobalNode()
	switch funcName {
	case "payFees":
		return fsc.payFees(t, inputData, gn, balances)
	default:
		return common.NewError("failed execution", "no function with that name").Error(), nil
	}
}
