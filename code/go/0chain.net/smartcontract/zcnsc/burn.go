package zcnsc

import (
	"fmt"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

func (zcn *ZCNSmartContract) burn(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI) (resp string, err error) {
	// get global node
	gn := getGlobalNode(balances)

	// decode input to burn payload
	payload := &burnPayload{}
	payload.Decode(inputData)
	payload.TxnID = t.Hash

	// check burn amount
	if t.Value < gn.MinBurnAmount {
		err = common.NewError("failed to burn", fmt.Sprintf("amount requested(%v) is lower than min amount for burn (%v)", t.Value, gn.MinBurnAmount))
		return
	}
	payload.Amount = t.Value
	Logger.Info("burn ticket", zap.Any("payload", payload), zap.Any("input", string(inputData)))

	// get user node
	un, err := getUserNode(t.ClientID, balances)
	if err != nil && payload.Nonce != 1 {
		err = common.NewError("failed to burn", fmt.Sprintf("get user node error (%v)", err.Error()))
		return
	}

	// check nonce is correct (current + 1)
	if un.Nonce+1 != payload.Nonce {
		err = common.NewError("failed to burn", fmt.Sprintf("nonce given (%v) is more than 1 higher than current (%v)", payload.Nonce, un.Nonce))
		return
	}

	if payload.EthereumAddress == "" {
		err = common.NewError("failed to burn", "Ethereum address is required")
		return
	}

	// increase the nonce
	un.Nonce++

	// save the user node
	err = un.save(balances)
	if err != nil {
		return
	}
	// burn the tokens
	balances.AddTransfer(state.NewTransfer(t.ClientID, gn.BurnAddress, state.Balance(t.Value)))
	resp = string(payload.Encode())
	return
}
