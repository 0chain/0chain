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
func (zcn *ZCNSmartContract) Burn(
	trans *transaction.Transaction,
	inputData []byte,
	balances cstate.StateContextI,
) (resp string, err error) {
	const (
		code = "failed to burn"
	)

	var (
		info = fmt.Sprintf(
			"transaction Hash %s, clientID: %s, payload: %s",
			trans.Hash,
			trans.ClientID,
			string(inputData),
		)
	)

	gn, err := GetGlobalNode(balances)
	if err != nil {
		msg := fmt.Sprintf("failed to get global node error: %v, %s", err, info)
		return "", common.NewError(code, msg)
	}

	// check burn amount
	if trans.Value < gn.Config.MinBurnAmount {
		msg := fmt.Sprintf(
			"amount (value) requested (%v) is lower than min burn amount (%v), %s",
			trans.Value,
			gn.Config.MinBurnAmount,
			info,
		)
		err = common.NewError(code, msg)
		return
	}

	payload := &BurnPayload{}
	err = payload.Decode(inputData)
	if err != nil {
		msg := fmt.Sprintf("payload decode error: %v, %s", err, info)
		err = common.NewError(code, msg)
		return
	}

	if payload.EthereumAddress == "" {
		err = common.NewError(code, "ethereum address is required "+info)
		return
	}

	// get user node and update nonce
	un, err := GetUserNode(trans.ClientID, balances)
	if err != nil && payload.Nonce != 1 {
		msg := fmt.Sprintf("get user node error %v with nonce != 1 (%d), %s", err, payload.Nonce, info)
		err = common.NewError(code, msg)
		return
	}

	// check nonce is correct (current + 1)
	if un.Nonce+1 != payload.Nonce {
		msg := fmt.Sprintf(
			"the payload nonce (%v) should be 1 higher than the current nonce (%v), %s",
			payload.Nonce,
			un.Nonce,
			info,
		)
		err = common.NewError(code, msg)
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
	err = balances.AddTransfer(state.NewTransfer(trans.ClientID, gn.Config.BurnAddress, state.Balance(trans.Value)))
	if err != nil {
		return "", err
	}

	response := &BurnPayloadResponse{
		TxnID:           trans.Hash,
		Amount:          trans.Value,
		Nonce:           payload.Nonce,
		EthereumAddress: payload.EthereumAddress,
	}

	resp = string(response.Encode())
	return
}
