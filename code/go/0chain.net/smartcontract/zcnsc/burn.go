package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

// inputData - is a burnPayload
func (zcn *ZCNSmartContract) burn(trans *transaction.Transaction, inputData []byte, balances cstate.StateContextI) (resp string, err error) {
	config := getSmartContractConfig()

	payload := &burnPayload{}
	err = payload.Decode(inputData)
	if err != nil {
		return
	}

	payload.TxnID = trans.Hash

	// check burn amount
	if trans.Value < config.MinBurnAmount {
		err = common.NewError("failed to burn", fmt.Sprintf("amount requested(%v) is lower than min amount for burn (%v)", trans.Value, config.MinBurnAmount))
		return
	}

	payload.Amount = trans.Value

	// get user node
	un, err := getUserNode(trans.ClientID, balances)
	if err != nil && payload.Nonce != 1 {
		err = common.NewError("failed to burn", fmt.Sprintf("get user node error (%v)", err.Error()))
		return
	}

	// check nonce is correct (current + 1)
	if un.Nonce+1 != payload.Nonce {
		err = common.NewError("failed to burn", fmt.Sprintf("the payload nonce (%v) should be 1 higher than the current nonce (%v)", payload.Nonce, un.Nonce))
		return
	}

	if payload.EthereumAddress == "" {
		err = common.NewError("failed to burn", "ethereum address is required")
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
	err = balances.AddTransfer(state.NewTransfer(trans.ClientID, config.BurnAddress, state.Balance(trans.Value)))
	if err != nil {
		return "", err
	}
	resp = string(payload.Encode())
	return
}
