package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

// inputData - is a MintPayload
func (zcn *ZCNSmartContract) Mint(trans *transaction.Transaction, inputData []byte, balances cstate.StateContextI) (resp string, err error) {
	gn := GetGlobalNode(balances)

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
	un, err := GetUserNode(trans.ClientID, balances)
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
				"nonce given (%v) is more than 1 higher than current (%v) for Node.ID: '%s'",
				payload.Nonce,
				un.Nonce,
				un.ID,
			),
		)
		return
	}

	// get the authorizers
	ans, err := GetAuthorizerNodes(balances)
	if err != nil {
		return
	}

	// check number of authorizers
	signaturesNeeded := int(gn.PercentAuthorizers * float64(len(ans.NodeMap)))
	if signaturesNeeded > len(payload.Signatures) {
		err = common.NewError("failed to mint", fmt.Sprintf("number of authorizers(%v) is lower than need signatures (%v)", len(payload.Signatures), signaturesNeeded))
		return
	}

	// verify signatures of authorizers
	err = payload.verifySignatures(ans)
	if err != nil {
		err = common.NewError("failed to mint", "failed to verify signatures with error: "+err.Error())
		return
	}

	// increase the nonce
	un.Nonce++

	// mint the tokens
	err = balances.AddMint(
		&state.Mint{
			Minter:     gn.ID,
			ToClientID: trans.ClientID,
			Amount:     payload.Amount,
		})

	if err != nil {
		return
	}

	// Save the user node
	err = un.Save(balances)
	if err != nil {
		return
	}

	resp = string(payload.Encode())
	return
}
