package zcnsc

import (
	"fmt"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
	"go.uber.org/zap"
)

// AddAuthorizer sc API function
// Transaction must include ClientID, ToClientID, PublicKey, Hash, Value
// ContractMap contains all the SC addresses
// ClientID is an authorizerID - used to search for authorizer
// ToClient is an SC address
func (zcn *ZCNSmartContract) AddAuthorizer(
	tran *transaction.Transaction,
	input []byte,
	ctx cstate.StateContextI,
) (response string, err error) {
	const (
		code = "failed to add authorizer"
	)

	var (
		authorizerID = tran.ClientID   // sender address
		recipientID  = tran.ToClientID // smart contract address
		authorizer   *AuthorizerNode
	)

	if authorizerID == "" {
		msg := "authorizerID is empty"
		err = common.NewError(code, msg)
		Logger.Error(msg, zap.Error(err))
		return "", err
	}

	if input == nil {
		msg := "input data is nil"
		err = common.NewError(code, msg)
		Logger.Error(msg, zap.Error(err))
		return "", err
	}

	// Decode input

	params := AuthorizerParameter{}
	err = params.Decode(input)
	if err != nil {
		err = common.NewError(code, "failed to decode AuthorizerParameter")
		Logger.Error("public key error", zap.Error(err))
		return "", err
	}

	if params.PublicKey == "" {
		err = common.NewError(code, "public key was not included with transaction")
		Logger.Error("public key error", zap.Error(err))
		return "", err
	}

	authorizerPublicKey := params.PublicKey
	authorizerURL := params.URL

	globalNode, err := GetGlobalNode(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get global node, authorizer(authorizerID: %v), err: %v", authorizerID, err)
		err = common.NewError(code, msg)
		Logger.Error("get global node", zap.Error(err))
		return "", err
	}

	// Check existing Authorizer

	authorizer, err = GetAuthorizerNode(authorizerID, ctx)
	if err == nil && authorizer != nil {
		err = fmt.Errorf("authorizer(authorizerID: %v) already exists", authorizerID)
		Logger.Error(code, zap.Error(err))
		return "", err
	}

	// compare the global min of authorizerNode Authorizer to that of the transaction amount
	if globalNode.MinStakeAmount > state.Balance(tran.Value*1e10) {
		msg := fmt.Sprintf("min stake amount '(%d)' > transaction value '(%d)'",
			globalNode.MinStakeAmount, tran.Value)
		err = common.NewError(code, msg)
		Logger.Error("min stake amount > transaction value", zap.Error(err))
		return "", err
	}

	// Create Authorizer instance

	authorizer = NewAuthorizer(authorizerID, authorizerPublicKey, authorizerURL)

	// Dig pool for authorizer

	var transfer *state.Transfer
	transfer, response, err = authorizer.LockingPool.DigPool(tran.Hash, tran)
	if err != nil {
		err = common.NewError(code, fmt.Sprintf("error digging pool, err: (%v)", err))
		return "", err
	}

	err = ctx.AddTransfer(transfer)
	if err != nil {
		msg := "Error: '%v', transaction.ClientId: '%s', transaction.ToClientId: '%s', transfer.ClientID: '%s', transfer.ToClientID: '%s'"
		err = common.NewError(
			code,
			fmt.Sprintf(
				msg,
				err,
				authorizerID,
				recipientID,
				transfer.ClientID,
				transfer.ToClientID,
			),
		)
		return "", err
	}

	err = authorizer.Save(ctx)
	if err != nil {
		msg := fmt.Sprintf("error saving authorizer(authorizerID: %v), err: %v", authorizerID, err)
		err = common.NewError(code, msg)
		Logger.Error("saving authorizer node", zap.Error(err))
		return "", err
	}

	ev, err := authorizer.ToEvent()
	if err != nil {
		msg := fmt.Sprintf("error marshalling authorizer(authorizerID: %v) to event, err: %v", authorizerID, err)
		err = common.NewError(code, msg)
		Logger.Error("emitting event", zap.Error(err))
		return "", err
	}

	ctx.EmitEvent(event.TypeStats, event.TagAddAuthorizer, authorizerID, string(ev))

	return
}

func (zcn *ZCNSmartContract) AddOrUpdateAuthorizerStakePool(
	tran *transaction.Transaction,
	input []byte,
	ctx cstate.StateContextI,
) (response string, err error) {
	const (
		code = "add_authorizer_staking_pool_failed"
	)

	var (
		authorizerID = tran.ClientID // sender address
		authorizer   *AuthorizerNode
	)

	if authorizerID == "" {
		msg := "tran.ClientID is empty"
		err = common.NewError(code, msg)
		Logger.Error(msg, zap.Error(err))
		return "", err
	}

	if input == nil {
		msg := "input data is nil"
		err = common.NewError(code, msg)
		Logger.Error(msg, zap.Error(err))
		return "", err
	}

	// Decode input

	params := AuthorizerStakePoolParameter{}
	err = params.Decode(input)
	if err != nil {
		err = common.NewError(code, "failed to decode AuthorizerParameter")
		Logger.Error("public key error", zap.Error(err))
		return "", err
	}

	poolSettings := params.StakePoolSettings

	if poolSettings.DelegateWallet == "" {
		return "", common.NewError(code, "authorizer's delegate_wallet not set")
	}

	if authorizerID != poolSettings.DelegateWallet {
		return "", common.NewError(code, "access denied, allowed for delegate_wallet owner only")
	}

	globalNode, err := GetGlobalNode(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get global node, authorizer(authorizerID: %v), err: %v", authorizerID, err)
		err = common.NewError(code, msg)
		Logger.Error("get global node", zap.Error(err))
		return "", err
	}

	authorizer, err = GetAuthorizerNode(authorizerID, ctx)
	if err == nil && authorizer != nil {
		var sp *StakePool
		sp, err = zcn.getOrUpdateStakePool(globalNode, authorizerID, spenum.Authorizer, poolSettings, ctx)
		if err != nil {
			return "", common.NewError(code, "failed to get or create stake pool: "+err.Error())
		}
		if err = sp.save(zcn.ID, authorizerID, ctx); err != nil {
			return "", common.NewError(code, "failed to save stake pool: "+err.Error())
		}

		Logger.Info("create or update stake pool completed successfully")

		return string(sp.Encode()), nil
	}

	return "", fmt.Errorf("authorizer(authorizerID: %v) not found", authorizerID)
}

func (zcn *ZCNSmartContract) CollectRewards(
	tran *transaction.Transaction,
	input []byte,
	ctx cstate.StateContextI,
) (response string, err error) {
	const code = "pay_reward_failed"

	var prr stakepool.CollectRewardRequest
	if err := prr.Decode(input); err != nil {
		return "", common.NewErrorf(code, "can't decode request: %v", err)
	}

	usp, err := stakepool.GetUserStakePool(prr.ProviderType, tran.ClientID, ctx)
	if err != nil {
		return "", common.NewErrorf(code, "can't get related user stake pools: %v", err)
	}

	providerId := usp.Find(prr.PoolId)
	if len(providerId) == 0 {
		return "", common.NewErrorf(code, "user %v does not own stake pool %v", tran.ClientID, prr.PoolId)
	}

	sp, err := zcn.getStakePool(providerId, ctx)
	if err != nil {
		return "", common.NewErrorf(code, "can't get related stake pool: %v", err)
	}

	_, err = sp.MintRewards(tran.ClientID, prr.PoolId, providerId, prr.ProviderType, usp, ctx)
	if err != nil {
		return "", common.NewErrorf(code, "error emptying account, %v", err)
	}

	if err := usp.Save(spenum.Blobber, tran.ClientID, ctx); err != nil {
		return "", common.NewErrorf("pay_reward_failed",
			"error saving user stake pool, %v", err)
	}

	if err := sp.save(zcn.ID, providerId, ctx); err != nil {
		return "", common.NewErrorf("pay_reward_failed",
			"error saving stake pool, %v", err)
	}

	return "", nil
}

func (zcn *ZCNSmartContract) DeleteAuthorizer(tran *transaction.Transaction, _ []byte, ctx cstate.StateContextI) (string, error) {
	var (
		authorizerID = tran.ClientID
		authorizer   *AuthorizerNode
		transfer     *state.Transfer
		response     string
		err          error
		errorCode    = "failed to delete authorizer"
	)

	authorizer, err = GetAuthorizerNode(authorizerID, ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get authorizer (authorizerID: %v), err: %v", authorizerID, err)
		err = common.NewError(errorCode, msg)
		Logger.Error("get authorizer node", zap.Error(err))
		return "", err
	}

	if authorizer == nil {
		msg := fmt.Sprintf("authorizer (authorizerID: %v) not found, err: %v", authorizerID, err)
		err = common.NewError(errorCode, msg)
		Logger.Error("authorizer node not found", zap.Error(err))
		return "", err
	}

	globalNode, err := GetGlobalNode(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get global node (authorizerID: %v), err: %v", authorizerID, err)
		err = common.NewError(errorCode, msg)
		Logger.Error("get global node", zap.Error(err))
		return "", err
	}

	// empty the authorizer's pool
	pool := authorizer.LockingPool
	if pool == nil {
		msg := "pool is not created"
		err := common.NewError(errorCode, msg)
		Logger.Error("node staking pool", zap.Error(err))
		return "", err
	}

	transfer, response, err = pool.EmptyPool(globalNode.ID, tran.ClientID, tran)
	if err != nil {
		msg := fmt.Sprintf("error emptying pool, err: (%v)", err)
		err = common.NewError(errorCode, msg)
		Logger.Error("empty pool", zap.Error(err))
		return response, err
	}

	// transfer tokens back to authorizer account
	err = ctx.AddTransfer(transfer)
	if err != nil {
		msg := fmt.Sprintf("error adding transfer: (%v)", err)
		err = common.NewError(errorCode, msg)
		Logger.Error("add transfer", zap.Error(err))
		return response, err
	}

	// delete authorizer node
	_, err = ctx.DeleteTrieNode(authorizer.GetKey())
	if err != nil {
		msg := fmt.Sprintf(
			"failed to delete authorizerID: (%v), node key: (%v), err: %v",
			authorizerID,
			authorizer.GetKey(),
			err,
		)
		err = common.NewError(errorCode, msg)
		Logger.Error("delete trie node", zap.Error(err))
		return "", err
	}

	ctx.EmitEvent(event.TypeStats, event.TagDeleteAuthorizer, authorizerID, authorizerID)

	Logger.Info(
		"deleted authorizer",
		zap.String("hash", tran.Hash),
		zap.String("authorizerID", authorizerID),
	)

	return response, err
}

func (zcn *ZCNSmartContract) UpdateAuthorizerConfig(_ *transaction.Transaction, inputData []byte, ctx cstate.StateContextI) (string, error) {
	const (
		code = "update_authorizer_settings"
	)

	gn, err := GetGlobalNode(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get global node, err: %v", err)
		err = common.NewError(code, msg)
		Logger.Error("get global node", zap.Error(err))
		return "", err
	}

	in := &AuthorizerNode{}
	if err = in.Decode(inputData); err != nil {
		msg := fmt.Sprintf("decoding request: %v", err)
		Logger.Error(msg, zap.Error(err))
		err = common.NewError(code, msg)
		return "", err
	}

	if in.Config.Fee < 0 || gn.MaxFee < 0 {
		msg := fmt.Sprintf("invalid negative Auth Config Fee: %v or GN Config MaxFee: %v", in.Config.Fee, gn.MaxFee)
		err = common.NewErrorf(code, msg)
		Logger.Error(msg, zap.Error(err))
		return "", err
	}

	if in.Config.Fee > gn.MaxFee {
		msg := fmt.Sprintf("authorizer fee (%v) is greater than allowed by SC (%v)", in.Config.Fee, gn.MaxFee)
		err = common.NewErrorf(code, msg)
		Logger.Error(msg, zap.Error(err))
		return "", err
	}

	var an *AuthorizerNode
	an, err = GetAuthorizerNode(in.ID, ctx)
	if err != nil {
		return "", common.NewError(code, err.Error())
	}

	err = an.UpdateConfig(in.Config)
	if err != nil {
		msg := fmt.Sprintf("error updating config for authorizer(authorizerID: %v), err: %v", an.ID, err)
		err = common.NewError(code, msg)
		Logger.Error("updating settings", zap.Error(err))
		return "", err
	}

	err = an.Save(ctx)
	if err != nil {
		msg := fmt.Sprintf("error saving authorizer(authorizerID: %v), err: %v", an.ID, err)
		err = common.NewError(code, msg)
		Logger.Error("saving authorizer node", zap.Error(err))
		return "", err
	}

	ev, err := an.ToEvent()
	if err != nil {
		msg := fmt.Sprintf("error marshalling authorizer (authorizerID: %v) to event, err: %v", an.ID, err)
		err = common.NewError(code, msg)
		Logger.Error("emitting event", zap.Error(err))
		return "", err
	}

	ctx.EmitEvent(event.TypeStats, event.TagAddAuthorizer, an.ID, string(ev))

	return string(an.Encode()), nil
}
