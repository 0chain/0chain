package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

// Mint inputData - is a MintPayload
func (zcn *ZCNSmartContract) Mint(trans *transaction.Transaction, inputData []byte, ctx cstate.StateContextI) (resp string, err error) {
	gn, err := GetGlobalNode(ctx)
	if err != nil {
		return "", common.NewError("failed to burn", fmt.Sprintf("failed to get global node error: %s, Client ID: %s", err.Error(), trans.Hash))
	}

	payload := &MintPayload{}
	err = payload.Decode(inputData)
	if err != nil {
		return
	}

	// check mint amount
	if payload.Amount < gn.MinMintAmount {
		err = common.NewError("failed to mint", fmt.Sprintf("amount requested(%v) is lower than min amount for mint (%v)", payload.Amount, gn.MinMintAmount))
		return
	}

	// get user node
	un, err := GetUserNode(trans.ClientID, ctx)
	if err != nil && payload.Nonce != 1 {
		err = common.NewError("failed to mint", fmt.Sprintf("get user node error (%v)", err.Error()))
		return
	}

	if un == nil {
		err = common.NewError("failed to mint", "user node is nil")
		return
	}

	// check nonce is correct (current + 1)
	if un.Nonce+1 != payload.Nonce {
		err = common.NewError(
			"failed to mint",
			fmt.Sprintf(
				"nonce given (%v) for receiving client (%s) must be greater by 1 than the current node nonce (%v) for Node.ID: '%s'",
				payload.Nonce,
				payload.ReceivingClientID,
				un.Nonce,
				un.ID,
			),
		)
		return
	}

	// verify signatures of authorizers
	err = payload.verifySignatures(ctx)
	if err != nil {
		err = common.NewError("failed to mint", "failed to verify signatures with error: "+err.Error())
		return
	}

	// increase the nonce
	un.Nonce++

	// mint the tokens
	err = ctx.AddMint(&state.Mint{
		Minter:     gn.ID,
		ToClientID: trans.ClientID,
		Amount:     payload.Amount,
	})
	if err != nil {
		return
	}

	// Save the user node
	err = un.Save(ctx)
	if err != nil {
		return
	}

	resp = string(payload.Encode())
	return
}
