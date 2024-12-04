package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"fmt"
	"github.com/0chain/common/core/logging"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"strings"
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

	if actErr := cstate.WithActivation(ctx, "hermes", func() error {
		return nil
	}, func() error {
		payload.EthereumAddress = ethcommon.HexToAddress(payload.EthereumAddress).Hex()
		return nil
	}); actErr != nil {
		return "", actErr
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

	// burn the tokens, transfer back to SC address
	err = ctx.AddTransfer(state.NewTransfer(trans.ClientID, ADDRESS, trans.Value))
	if err != nil {
		return "", err
	}

	response := &BurnPayloadResponse{
		TxnID:           trans.Hash,
		Amount:          trans.Value,
		Nonce:           un.BurnNonce, // it can be just the nonce of this transaction
		EthereumAddress: payload.EthereumAddress,
	}

	ctx.EmitEvent(event.TypeStats, event.TagAuthorizerBurn, trans.ClientID, state.Burn{
		Burner: trans.ClientID,
		Amount: trans.Value,
	})

	ctx.EmitEvent(event.TypeStats, event.TagAddBurnTicket, payload.EthereumAddress, &event.BurnTicket{
		EthereumAddress: payload.EthereumAddress,
		Hash:            trans.Hash,
		Amount:          trans.Value,
		Nonce:           un.BurnNonce,
	})

	resp = string(response.Encode())
	return
}

func (zcn *ZCNSmartContract) RepairEthAddressMerge(trans *transaction.Transaction,
	inputData []byte,
	ctx cstate.StateContextI) (resp string, err error) {
	const (
		code = "failed to repair"
	)

	var (
		info = fmt.Sprintf(
			"transaction: %s, clientID: %s, payload: %s",
			trans.Hash,
			trans.ClientID,
			string(inputData),
		)
	)

	payload := &RepairEthAddressPayload{}
	err = payload.Decode(inputData)
	if err != nil {
		msg := fmt.Sprintf("payload decode error: %v, %s", err, info)
		err = common.NewError(code, msg)
		logging.Logger.Error(msg, zap.Error(err))
		return
	}

	lowerCaseEthAddress := strings.ToLower(payload.EthereumAddress)
	if err = DeleteUserNodeIfExist(lowerCaseEthAddress, ctx); err != nil {
		err = common.NewError(code, fmt.Sprintf("delete user node error (%v), %s", err, info))
		logging.Logger.Error(err.Error(), zap.Error(err))
		return
	}

	correctedEthAddress := ethcommon.HexToAddress(payload.EthereumAddress).Hex()
	correctedUserNode, err := GetUserNode(correctedEthAddress, ctx)
	if err != nil {
		err = common.NewError(code, fmt.Sprintf("get user node error (%v), %s", err, info))
		logging.Logger.Error(err.Error(), zap.Error(err))
		return
	}
	correctedUserNode.BurnNonce = payload.Nonce

	err = correctedUserNode.Save(ctx)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, user node failed to be saved, %s", code, info))
		return
	}

	return "repair success", nil
}
