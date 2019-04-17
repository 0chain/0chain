package feesc

import (
	"fmt"

	"0chain.net/chaincore/block"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	metrics "github.com/rcrowley/go-metrics"
)

var (
	Seperator                           = smartcontractinterface.Seperator
	owner                               = "c8a5e74c2f4fae2c1bed79fb2b78d3b88f844bbb6bf1db5fc43240711f23321f"
	ADDRESS                             = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d1"
	sharderMintAllocation state.Balance = 200
	minerMintAllocation   state.Balance = 100
	charity                             = .2
)

type FeeSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (fsc *FeeSmartContract) GetRestPoints() map[string]smartcontractinterface.SmartContractRestHandler {
	return fsc.RestHandlers
}

func (fsc *FeeSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	fsc.SmartContract = sc
	fsc.SmartContractExecutionStats["payFees"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", fsc.ID, "payFees"), nil)
	fsc.SmartContractExecutionStats["feesPaid"] = metrics.GetOrRegisterHistogram(fmt.Sprintf("sc:%v:func:%v", fsc.ID, "feesPaid"), nil, metrics.NewUniformSample(1024))
	fsc.SmartContractExecutionStats["mintedTokens"] = metrics.GetOrRegisterHistogram(fmt.Sprintf("sc:%v:func:%v", fsc.ID, "mintedTokens"), nil, metrics.NewUniformSample(1024))
}

func (fsc *FeeSmartContract) sumFee(b *block.Block, updateStats bool) state.Balance {
	var totalMaxFee int64
	for _, txn := range b.Txns {
		totalMaxFee += txn.Fee
		if updateStats {
			feeStats := fsc.SmartContractExecutionStats["feesPaid"].(metrics.Histogram)
			feeStats.Update(txn.Fee)
		}
	}
	return state.Balance(totalMaxFee)
}

func (fsc *FeeSmartContract) payFees(t *transaction.Transaction, inputData []byte, gn *globalNode, balances c_state.StateContextI) (string, error) {
	block := balances.GetBlock()
	if t.ClientID != block.MinerID {
		return "", common.NewError("failed to pay fees", "not block generator")
	}
	if block.Round <= gn.LastRound {
		return "", common.NewError("failed to pay fees", "jumped back in time?")
	}
	var resp string
	fee := fsc.sumFee(block, false)
	// feeHalved := fee / state.Balance(2)
	// totalForReplicators := state.Balance(float64(sharderMintAllocation+feeHalved) * (1.0 - charity))
	// totalForOtherSharders := state.Balance(float64(sharderMintAllocation+feeHalved) * charity)
	// totalForGenerator := state.Balance(float64(minerMintAllocation+feeHalved) * (1.0 - charity))
	// totalForOtherMiners := state.Balance(float64(minerMintAllocation+feeHalved) * charity)
	transfer := state.NewTransfer(ADDRESS, t.ClientID, fee)
	balances.AddTransfer(transfer)
	resp += string(transfer.Encode())
	sharders := balances.GetBlockSharders(block.PrevBlock)
	for _, sharder := range sharders {
		//TODO: the mint amount will be controlled by governance
		mint := state.NewMint(ADDRESS, sharder, fee/state.Balance(len(sharders)))
		mintStats := fsc.SmartContractExecutionStats["mintedTokens"].(metrics.Histogram)
		mintStats.Update(int64(mint.Amount))
		err := balances.AddMint(mint)
		if err != nil {
			resp += common.NewError("failed to mint", fmt.Sprintf("errored while adding mint for sharder %v: %v", sharder, err.Error())).Error()
		}
	}
	gn.LastRound = block.Round
	_, err := balances.InsertTrieNode(gn.GetKey(), gn)
	if err != nil {
		return "", err
	}
	fsc.sumFee(block, true)
	return resp, nil
}

func (fsc *FeeSmartContract) getGlobalNode(balances c_state.StateContextI) (*globalNode, error) {
	gn := &globalNode{ID: fsc.ID}
	gv, err := balances.GetTrieNode(gn.GetKey())
	if err != nil {
		return gn, err
	}
	gn.Decode(gv.Encode())
	return gn, err
}

func (fsc *FeeSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	gn, _ := fsc.getGlobalNode(balances)
	switch funcName {
	case "payFees":
		return fsc.payFees(t, inputData, gn, balances)
	default:
		return "", common.NewError("failed execution", "no function with that name")
	}
}
