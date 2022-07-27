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

// Mint inputData - is a MintPayload
func (zcn *ZCNSmartContract) Mint(trans *transaction.Transaction, inputData []byte, ctx cstate.StateContextI) (resp string, err error) {
	const (
		code = "failed to mint"
	)

	var (
		info = fmt.Sprintf(
			"transaction hash %s, clientID: %s, payload: %s",
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

	payload := &MintPayload{}
	err = payload.Decode(inputData)
	if err != nil {
		msg := fmt.Sprintf("payload decode error: %v, %s", err, info)
		err = common.NewError(code, msg)
		return
	}

	// ClientID - is a client who broadcasts this transaction to mint token
	// ToClientID - is an address of the smart contract
	if payload.ReceivingClientID != trans.ClientID {
		msg := fmt.Sprintf("transaction made from different account who made burn,  Oririnal: %s, Current: %s",
			payload.ReceivingClientID, trans.ClientID)
		err = common.NewError(code, msg)
	}

	// check mint amount
	if payload.Amount < gn.MinMintAmount {
		msg := fmt.Sprintf(
			"amount requested (%v) is lower than min amount for mint (%v), %s",
			payload.Amount,
			gn.MinMintAmount,
			info,
		)
		err = common.NewError(code, msg)
		return
	}

	// get user node
	un, err := GetUserNode(trans.ClientID, ctx)
	if err != nil {
		err = common.NewError(code, fmt.Sprintf("get user node error (%v), %s", err, info))
		logging.Logger.Error(err.Error(), zap.Error(err))
		return
	}

	_, exists := gn.WZCNNonceMinted[payload.Nonce]
	if exists { // global nonce from ETH SC has already been minted
		err = common.NewError(
			code,
			fmt.Sprintf(
				"nonce given (%v) for receiving client (%s) has alredy been minted for Node.ID: '%s', %s",
				payload.Nonce, payload.ReceivingClientID, un.ID, info))
		return
	}

	// verify signatures of authorizers
	err = payload.verifySignatures(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to verify signatures with error: %v, %s", err, info)
		err = common.NewError(code, msg)
		return
	}

	// record the global nonce from solidity smart contract
	gn.WZCNNonceMinted[payload.Nonce] = true

	// mint the tokens
	err = ctx.AddMint(&state.Mint{
		Minter:     gn.ID,
		ToClientID: trans.ClientID,
		Amount:     payload.Amount,
	})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, add mint operation, %s", code, info))
		return
	}

	// Save the user node
	err = gn.Save(ctx)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, global node failed to be saved, %s", code, info))
		return
	}

	resp = string(payload.Encode())
	return
}
