package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"github.com/pkg/errors"
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
	switch err {
	case nil:
		if un.Nonce+1 != payload.Nonce {
			err = common.NewError(
				code,
				fmt.Sprintf(
					"nonce given (%v) for receiving client (%s) must be greater by 1 than the current node nonce (%v) for Node.ID: '%s', %s",
					payload.Nonce,
					payload.ReceivingClientID,
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
	//if err != nil && payload.Nonce != 1 {
	//	err = common.NewError(code, fmt.Sprintf("get user node error (%v), %s", err, info))
	//	return
	//}

	//if un == nil {
	//	err = common.NewError(code, "user node is nil "+info)
	//	return
	//}

	// check nonce is correct (current + 1)
	//if un.Nonce+1 != payload.Nonce {
	//	err = common.NewError(
	//		code,
	//		fmt.Sprintf(
	//			"nonce given (%v) for receiving client (%s) must be greater by 1 than the current node nonce (%v) for Node.ID: '%s', %s",
	//			payload.Nonce,
	//			payload.ReceivingClientID,
	//			un.Nonce,
	//			un.ID,
	//			info,
	//		),
	//	)
	//	return
	//}

	// verify signatures of authorizers
	err = payload.verifySignatures(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to verify signatures with error: %v, %s", err, info)
		err = common.NewError(code, msg)
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
		err = errors.Wrap(err, fmt.Sprintf("%s, add mint operation, %s", code, info))
		return
	}

	// Save the user node
	err = un.Save(ctx)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, save MPR operation, %s", code, info))
		return
	}

	resp = string(payload.Encode())
	return
}
