package zcnsc

import (
	"fmt"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (zcn *ZCNSmartContract) mint(t *transaction.Transaction, inputData []byte, balances c_state.StateContextI) (resp string, err error) {
	// get global node
	gn := getGlobalNode(balances)

	// decode input to mint payload
	var payload *mintPayload
	payload.Decode(inputData)

	// check mint amount
	if payload.Amount < gn.MinMintAmount {
		err = common.NewError("failed to mint", fmt.Sprintf("amount requested(%v) is lower than min amount for mint (%v)", payload.Amount, gn.MinMintAmount))
		return
	}

	// get user node
	un, err := getUserNode(t.ClientID, balances)
	if err != nil && payload.Nonce != 1 {
		err = common.NewError("failed to mint", fmt.Sprintf("get user node error (%v)", err.Error()))
		return
	}

	// check nonce is correct (current + 1)
	if un.Nonce+1 != payload.Nonce {
		err = common.NewError("failed to mint", fmt.Sprintf("nonce given (%v) is more than 1 higher than current (%v)", payload.Nonce, un.Nonce))
		return
	}

	// get the authorizers
	ans := getAuthorizerNodes(balances)

	// check number of authorizers
	signaturesNeeded := int(gn.PercentAuthorizers * float64(len(ans.NodeMap)))
	if signaturesNeeded > len(payload.Signatures) {
		err = common.NewError("failed to mint", fmt.Sprintf("number of authorizers(%v) is lower than need signatures (%v)", len(payload.Signatures), signaturesNeeded))
		return
	}

	// verify signatures of authorizers
	if !payload.verifySignatures(ans) {
		err = common.NewError("failed to mint", "failed to verify signatures")
		return
	}

	// increase the nonce
	un.Nonce++

	// mint the tokens
	balances.AddMint(&state.Mint{
		Minter:     gn.ID,
		ToClientID: t.ClientID,
		Amount:     payload.Amount,
	})

	// save the user node
	err = un.save(balances)
	if err != nil {
		return
	}
	resp = string(payload.Encode())
	return
}
