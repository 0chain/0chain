package minersc

import (
	"fmt"
	"sort"

	"0chain.net/chaincore/block"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) payFees(t *transaction.Transaction, inputData []byte, gn *globalNode, balances c_state.StateContextI) (string, error) {
	pn, err := msc.getPhaseNode(balances)
	if err != nil {
		return "", err
	}
	err = msc.setPhaseNode(balances, pn, gn)
	if err != nil {
		return "", common.NewError("pay_fees_failed", fmt.Sprintf("error insterting phase node: %v", err))
	}
	block := balances.GetBlock()
	if block.Round == gn.ViewChange && !msc.SetMagicBlock(balances) {
		return "", common.NewError("pay_fees_failed", "can't set magic block")
	}

	if t.ClientID != block.MinerID {
		return "", common.NewError("failed to pay fees", "not block generator")
	}
	if block.Round <= gn.LastRound {
		return "", common.NewError("failed to pay fees", "jumped back in time?")
	}
	fee := msc.sumFee(block, true)
	resp := msc.paySharders(fee, block, balances, "")
	gn.LastRound = block.Round
	_, err = balances.InsertTrieNode(GlobalNodeKey, gn)
	if err != nil {
		return "", common.NewError("pay_fees_failed", fmt.Sprintf("error insterting global node: %v", err))
	}
	return resp, nil
}

func (msc *MinerSmartContract) sumFee(b *block.Block, updateStats bool) state.Balance {
	var totalMaxFee int64
	feeStats := msc.SmartContractExecutionStats["feesPaid"].(metrics.Histogram)
	for _, txn := range b.Txns {
		totalMaxFee += txn.Fee
		if updateStats {
			feeStats.Update(txn.Fee)
		}
	}
	return state.Balance(totalMaxFee)
}

func (msc *MinerSmartContract) payMiners(fee state.Balance, mn *MinerNode, balances c_state.StateContextI, t *transaction.Transaction) string {
	var resp string
	minerFee := state.Balance(float64(fee) * mn.Percentage)
	transfer := state.NewTransfer(ADDRESS, t.ClientID, minerFee)
	balances.AddTransfer(transfer)
	resp += string(transfer.Encode())

	restFee := fee - minerFee
	totalStaked := mn.TotalStaked
	var keys []string
	for k := range mn.Active {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		pool := mn.Active[key]
		userPercent := float64(pool.Balance) / float64(totalStaked)
		userFee := state.Balance(float64(restFee) * userPercent)
		Logger.Info("pay delegate", zap.Any("pool", pool), zap.Any("fee", userFee))
		transfer := state.NewTransfer(ADDRESS, pool.DelegateID, userFee)
		balances.AddTransfer(transfer)
		pool.TotalPaid += transfer.Amount
		pool.NumRounds++
		if pool.High < transfer.Amount {
			pool.High = transfer.Amount
		}
		if pool.Low == -1 || pool.Low > transfer.Amount {
			pool.Low = transfer.Amount
		}
		resp += string(transfer.Encode())
	}
	return resp
}

func (msc *MinerSmartContract) paySharders(fee state.Balance, block *block.Block, balances c_state.StateContextI, resp string) string {
	sharders := balances.GetBlockSharders(block.PrevBlock)
	sort.Strings(sharders)
	for _, sharder := range sharders {
		//TODO: the mint amount will be controlled by governance
		mint := state.NewMint(ADDRESS, sharder, fee/state.Balance(len(sharders)))
		mintStats := msc.SmartContractExecutionStats["mintedTokens"].(metrics.Histogram)
		mintStats.Update(int64(mint.Amount))
		err := balances.AddMint(mint)
		if err != nil {
			resp += common.NewError("failed to mint", fmt.Sprintf("errored while adding mint for sharder %v: %v", sharder, err.Error())).Error()
		}
	}
	return resp
}
