package zcnsc

import (
	"0chain.net/smartcontract/entity"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"github.com/0chain/common/core/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var BurnTickets map[string][]entity.BurnTicketDetails

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

	// get user node
	un, err := GetUserNode(payload.EthereumAddress, ctx)
	if err != nil {
		err = common.NewError(code, fmt.Sprintf("get user node error (%v), %s", err, info))
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

	BurnTickets[trans.ClientID] = append(BurnTickets[trans.ClientID], entity.BurnTicketDetails{
		Hash:  trans.Hash,
		Nonce: trans.Nonce,
	})

	response := &BurnPayloadResponse{
		TxnID:           trans.Hash,
		Amount:          trans.Value,
		Nonce:           un.BurnNonce, // it can be just the nonce of this transaction
		EthereumAddress: payload.EthereumAddress,
	}

	ctx.EmitEvent(event.TypeStats, event.TagBurn, trans.ClientID, trans.Value)

	resp = string(response.Encode())
	return
}
