package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

// addAuthorizer sc API function
// Transaction must include ClientID, ToClientID, PublicKey, Hash, Value
// inputData is a publicKey in case public key in Tx is missing. Either PK or inputData must be present
// balances have `GetTriedNode` implemented to get nodes
// ContractMap contains all the SC addresses
// ToClient is a SC address
func (zcn *ZCNSmartContract) addAuthorizer(t *transaction.Transaction, inputData []byte, balances cstate.StateContextI) (resp string, err error) {
	// check for authorizer already there
	ans, err := getAuthorizerNodes(balances)
	if err != nil {
		return resp, err
	}
	if ans.NodeMap[t.ClientID] != nil {
		err = common.NewError("failed to add authorizer", fmt.Sprintf("authorizer(id: %v) already exists", t.ClientID))
		return
	}

	//get global node
	gn := getGlobalNode(balances)

	//compare the global min of an Authorizer to that of the transaction amount
	if gn.MinStakeAmount > t.Value {
		err = common.NewError("failed to add authorizer", fmt.Sprintf("amount to stake (%v) is lower than min amount (%v)", t.Value, gn.MinStakeAmount))
		return
	}

	// get public key
	var key string
	if t.PublicKey == "" {
		pk := PublicKey{}
		err = pk.Decode(inputData)
		if err != nil {
			err = common.NewError("failed to add authorizer", "public key was not included with transaction")
			return
		}
		key = pk.Key
	} else {
		key = t.PublicKey
	}
	an := getNewAuthorizer(key, t.ClientID)

	//dig pool for authorizer
	var transfer *state.Transfer
	transfer, resp, err = an.Staking.DigPool(t.Hash, t)
	if err != nil {
		err = common.NewError("failed to add authorizer", fmt.Sprintf("error digging pool(%v)", err.Error()))
		return
	}
	err = balances.AddTransfer(transfer)
	if err != nil {
		return
	}
	err = ans.addAuthorizer(an)
	if err != nil {
		return
	}
	//save authorizer
	err = ans.save(balances)
	return
}

func (zcn *ZCNSmartContract) deleteAuthorizer(t *transaction.Transaction, _ []byte, balances cstate.StateContextI) (resp string, err error) {
	//check for authorizer
	ans, err := getAuthorizerNodes(balances)
	if err != nil {
		return
	}

	if ans.NodeMap[t.ClientID] == nil {
		err = common.NewError("failed to delete authorizer", fmt.Sprintf("authorizer (%v) doesn't exist", t.ClientID))
		return
	}

	gn := getGlobalNode(balances)

	//empty the authorizer's pool
	var transfer *state.Transfer
	transfer, resp, err = ans.NodeMap[t.ClientID].Staking.EmptyPool(gn.ID, t.ClientID, t)
	if err != nil {
		err = common.NewError("failed to delete authorizer", fmt.Sprintf("error emptying pool(%v)", err.Error()))
		return
	}

	//transfer tokens back to authorizer account
	_ = balances.AddTransfer(transfer)

	//delete authorizer node
	err = ans.deleteAuthorizer(t.ClientID)
	if err != nil {
		return
	}
	err = ans.save(balances)
	return
}
