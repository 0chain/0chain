package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

// AddAuthorizer sc API function
// Transaction must include ClientID, ToClientID, PublicKey, Hash, Value
// inputData is a publicKey in case public key in Tx is missing.
// Either PK or inputData must be present
// balances have `GetTriedNode` implemented to get nodes
// ContractMap contains all the SC addresses
// ToClient is an SC address
func (zcn *ZCNSmartContract) AddAuthorizer(t *transaction.Transaction, inputData []byte, balances cstate.StateContextI) (resp string, err error) {
	// check for authorizer already there
	ans, err := GetAuthorizerNodes(balances)
	if err != nil {
		return resp, err
	}
	if ans.NodeMap[t.ClientID] != nil {
		err = common.NewError("failed to add authorizer", fmt.Sprintf("authorizer(id: %v) already exists", t.ClientID))
		return
	}

	gn := GetGlobalNode(balances)

	//compare the global min of an Authorizer to that of the transaction amount
	if gn.MinStakeAmount > t.Value {
		err = common.NewError("failed to add authorizer", fmt.Sprintf("amount to stake (%v) is lower than min amount (%v)", t.Value, gn.MinStakeAmount))
		return
	}

	authParam := AuthorizerParameter{}
	err = authParam.Decode(inputData)
	if err != nil {
		err = common.NewError("failed to add authorizer", "public key was not included with transaction")
		return
	}

	var publicKey string
	if t.PublicKey == "" {
		publicKey = authParam.PublicKey
	} else {
		publicKey = t.PublicKey
	}
	an := GetNewAuthorizer(publicKey, t.ClientID, authParam.URL) // t.ClientID = authorizer node id

	//dig pool for authorizer
	var transfer *state.Transfer
	transfer, resp, err = an.Staking.DigPool(t.Hash, t)
	if err != nil {
		err = common.NewError("failed to add authorizer", fmt.Sprintf("error digging pool(%v)", err.Error()))
		return
	}
	err = balances.AddTransfer(transfer)
	if err != nil {
		currTr := balances.GetTransaction()
		err = common.NewError(
			"failed to add transfer",
			fmt.Sprintf(
				"Error: '%v', Trans.ClientId: '%s', Trans.ToClientId: '%s', transfer.ClientID: '%s', transfer.ToClientID: '%s'",
				err.Error(),
				currTr.ClientID,
				currTr.ToClientID,
				transfer.ClientID,
				transfer.ToClientID,
			),
		)
		return
	}
	err = ans.AddAuthorizer(an)
	if err != nil {
		return
	}
	//Save authorizer
	err = ans.Save(balances)
	return
}

func (zcn *ZCNSmartContract) DeleteAuthorizer(tran *transaction.Transaction, _ []byte, balances cstate.StateContextI) (resp string, err error) {
	//check for authorizer
	ans, err := GetAuthorizerNodes(balances)
	if err != nil {
		return
	}

	if ans.NodeMap[tran.ClientID] == nil {
		err = common.NewError("failed to delete authorizer", fmt.Sprintf("authorizer (%v) doesn't exist", tran.ClientID))
		return
	}

	gn := GetGlobalNode(balances)

	//empty the authorizer's pool
	var transfer *state.Transfer
	transfer, resp, err = ans.NodeMap[tran.ClientID].Staking.EmptyPool(gn.ID, tran.ClientID, tran)
	if err != nil {
		err = common.NewError("failed to delete authorizer", fmt.Sprintf("error emptying pool(%v)", err.Error()))
		return
	}

	//transfer tokens back to authorizer account
	_ = balances.AddTransfer(transfer)

	//delete authorizer node
	err = ans.DeleteAuthorizer(tran.ClientID)
	if err != nil {
		return
	}
	err = ans.Save(balances)
	return
}
