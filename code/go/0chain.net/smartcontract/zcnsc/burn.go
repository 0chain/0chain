package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

// Burn inputData - is a BurnPayload.
// EthereumAddress => required
// Nonce => required
func (zcn *ZCNSmartContract) Burn(trans *transaction.Transaction, inputData []byte, balances cstate.StateContextI) (resp string, err error) {
	gn := GetGlobalNode(balances)

	// check burn amount
	if trans.Value < gn.MinBurnAmount {
		err = common.NewError("failed to burn", fmt.Sprintf("amount requested(%v) is lower than min amount for burn (%v)", trans.Value, gn.MinBurnAmount))
		return
	}

	payload := &BurnPayload{}
	err = payload.Decode(inputData)
	if err != nil {
		return
	}

	if payload.EthereumAddress == "" {
		err = common.NewError("failed to burn", "ethereum address is required")
		return
	}

	payload.TxnID = trans.Hash
	payload.Amount = trans.Value

	// get user node and update nonce
	un, err := GetUserNode(trans.ClientID, balances)
	if err != nil && payload.Nonce != 1 {
		err = common.NewError("failed to burn", fmt.Sprintf("get user node error (%v) with nonce != 1", err.Error()))
		return
	}

	// check nonce is correct (current + 1)
	if un.Nonce+1 != payload.Nonce {
		err = common.NewError("failed to burn", fmt.Sprintf("the payload nonce (%v) should be 1 higher than the current nonce (%v)", payload.Nonce, un.Nonce))
		return
	}

	// increase the nonce
	un.Nonce++

	// Save the user node
	err = un.Save(balances)
	if err != nil {
		return
	}

	// burn the tokens
	err = balances.AddTransfer(state.NewTransfer(trans.ClientID, gn.BurnAddress, state.Balance(trans.Value)))
	if err != nil {
		return "", err
	}
	resp = string(payload.Encode())
	return
}
