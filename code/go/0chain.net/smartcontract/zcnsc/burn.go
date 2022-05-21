package zcnsc

import (
	"fmt"

	"0chain.net/chaincore/currency"

	"0chain.net/core/util"

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
	ctx cstate.StateContextI,
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

	gn, err := GetGlobalNode(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get global node error: %v, %s", err, info)
		return "", common.NewError(code, msg)
	}

	// check burn amount
	if currency.Coin(trans.Value*1e10) < gn.MinBurnAmount {
		msg := fmt.Sprintf(
			"amount (value) requested (%v) is lower than min burn amount (%v), %s",
			trans.Value,
			gn.MinBurnAmount,
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

	// get user node
	un, err := GetUserNode(trans.ClientID, ctx)
	switch err {
	case nil:
		if un.Nonce+1 != payload.Nonce {
			err = common.NewError(
				code,
				fmt.Sprintf(
					"nonce given (%v) for burning client (%s) must be greater by 1 than the current node nonce (%v) for Node.ID: '%s', %s",
					payload.Nonce,
					trans.ClientID,
					un.Nonce,
					un.ID,
					info,
				),
			)
			return
		}
	case util.ErrValueNotPresent:
		err = common.NewError(code, "user node is nil "+info)
		return
	default:
		err = common.NewError(code, fmt.Sprintf("get user node error (%v), %s", err, info))
		return
	}

	// increase the nonce
	un.Nonce++

	// Save the user node
	err = un.Save(ctx)
	if err != nil {
		return
	}

	// burn the tokens
	err = ctx.AddTransfer(state.NewTransfer(trans.ClientID, gn.BurnAddress, currency.Coin(trans.Value)))
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
