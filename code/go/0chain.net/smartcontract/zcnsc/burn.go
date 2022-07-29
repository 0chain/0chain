package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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
			"transaction: %s, clientID: %s, payload: %s",
			trans.Hash,
			trans.ClientID,
			string(inputData),
		)
	)

	gn, err := GetGlobalNode(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get global node error: %v, %s", err, info)
		logging.Logger.Error(msg, zap.Error(err))
		return "", common.NewError(code, msg)
	}

	// get user node
	un, err := GetUserNode(trans.ClientID, ctx)
	if err != nil {
		err = common.NewError(code, fmt.Sprintf("get user node error (%v), %s", err, info))
		logging.Logger.Error(err.Error(), zap.Error(err))
		return
	}

	// check burn amount
	if trans.Value < gn.MinBurnAmount {
		msg := fmt.Sprintf(
			"amount (value) requested (%v) is lower than min burn amount (%v), %s",
			trans.Value,
			gn.MinBurnAmount,
			info,
		)
		err = common.NewError(code, msg)
		logging.Logger.Error(msg, zap.Error(err))
		return
	}

	payload := &BurnPayload{}
	err = payload.Decode(inputData)
	if err != nil {
		msg := fmt.Sprintf("payload decode error: %v, %s", err, info)
		err = common.NewError(code, msg)
		logging.Logger.Error(msg, zap.Error(err))
		return
	}

	if payload.EthereumAddress == "" {
		err = common.NewError(code, "ethereum address is required, "+info)
		logging.Logger.Error(err.Error(), zap.Error(err))
		return
	}

	// increase the nonce
	un.BurnNonce++

	// Save the user node
	err = un.Save(ctx)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, user node failed to be saved, %s", code, info))
		return
	}

	// burn the tokens
	err = ctx.AddTransfer(state.NewTransfer(trans.ClientID, gn.BurnAddress, trans.Value))
	if err != nil {
		return "", err
	}

	response := &BurnPayloadResponse{
		TxnID:           trans.Hash,
		Amount:          trans.Value,
		Nonce:           un.BurnNonce, // it can be just the nonce of this transaction
		EthereumAddress: payload.EthereumAddress,
	}

	resp = string(response.Encode())
	return
}
