package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"go.uber.org/zap"
)

// AddAuthorizer sc API function
// Transaction must include ClientID, ToClientID, PublicKey, Hash, Value
// inputData is a publicKey in case public key in Tx is missing
// Either PK or inputData must be present
// balances have `GetTriedNode` implemented to get nodes
// ContractMap contains all the SC addresses
// ToClient is an SC address
func (zcn *ZCNSmartContract) AddAuthorizer(tran *transaction.Transaction, inputData []byte, balances cstate.StateContextI) (string, error) {
	// check for authorizer already there
	ans, err := GetAuthorizerNodes(balances)
	logging.Logger.Debug("getting authorizer nodes", zap.String("hash", tran.Hash), zap.Int("nodes count", len(ans.NodeMap)))
	if err != nil {
		return "", err
	}

	if ans.NodeMap[tran.ClientID] != nil {
		err = common.NewError("failed to add authorizer", fmt.Sprintf("authorizer(id: %v) already exists", tran.ClientID))
		return "", err
	}

	logging.Logger.Debug("trying to get global node", zap.String("hash", tran.Hash))
	gn, err := GetGlobalNode(balances)
	if err != nil {
		return "", common.NewError("failed to add authorizer", fmt.Sprintf("failed to get global node error: %s, authorizer(id: %v)", err.Error(), tran.ClientID))
	}
	logging.Logger.Debug("found global node", zap.String("hash", tran.Hash))

	//compare the global min of an Authorizer to that of the transaction amount
	if gn.MinStakeAmount > tran.Value {
		err = common.NewError("failed to add authorizer", fmt.Sprintf("amount to stake (%v) is lower than min amount (%v)", tran.Value, gn.MinStakeAmount))
		return "", err
	}

	authParam := AuthorizerParameter{}
	err = authParam.Decode(inputData)
	if err != nil {
		err = common.NewError("failed to add authorizer", "public key was not included with transaction")
		return "", err
	}

	var publicKey string
	if tran.PublicKey == "" {
		publicKey = authParam.PublicKey
	} else {
		publicKey = tran.PublicKey
	}

	logging.Logger.Debug("trying to add authorizer", zap.String("hash", tran.Hash))

	//Save authorizer
	an := GetNewAuthorizer(publicKey, tran.ClientID, authParam.URL) // tran.ClientID = authorizer node id
	err = ans.AddAuthorizer(an)
	if err != nil {
		return "", err
	}

	//err = an.Save(balances)
	//if err != nil {
	//	return "", err
	//}

	logging.Logger.Debug("trying to save state", zap.String("hash", tran.Hash))
	err = ans.Save(balances)
	if err != nil {
		return "", err
	}
	logging.Logger.Debug("saved the state", zap.String("hash", tran.Hash))

	//Dig pool for authorizer

	transfer, response, err := an.Staking.DigPool(tran.Hash, tran)
	if err != nil {
		err = common.NewError("failed to add authorizer", fmt.Sprintf("error digging pool(%v)", err.Error()))
		return "", err
	}

	logging.Logger.Debug("trying to add transfer", zap.String("hash", tran.Hash))
	err = balances.AddTransfer(transfer)
	if err != nil {
		err = common.NewError(
			"failed to add transfer",
			fmt.Sprintf(
				"Error: '%v', Trans.ClientId: '%s', Trans.ToClientId: '%s', transfer.ClientID: '%s', transfer.ToClientID: '%s'",
				err.Error(),
				tran.ClientID,
				tran.ToClientID,
				transfer.ClientID,
				transfer.ToClientID,
			),
		)
		return "", err
	}

	return response, err
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

	gn, err := GetGlobalNode(balances)
	if err != nil {
		return "", common.NewError("failed to delete authorizer", fmt.Sprintf("failed to get global node error: %s, authorizer(id: %v)", err.Error(), tran.ClientID))
	}

	//empty the authorizer's pool
	var transfer *state.Transfer
	pool := ans.NodeMap[tran.ClientID].Staking
	if pool == nil {
		return "", common.NewError("failed to delete authorizer", "pool is not created")
	}

	transfer, resp, err = pool.EmptyPool(gn.ID, tran.ClientID, tran)
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

	logging.Logger.Info("deleted authorizer", zap.String("hash", tran.Hash), zap.String("authorizer_id", tran.ClientID))

	return
}
