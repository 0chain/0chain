package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	. "github.com/0chain/common/core/logging"
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
		authorizerID = tran.ClientID // sender address
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

	params := AddAuthorizerPayload{}
	err = params.Decode(input)
	if err != nil {
		err = common.NewError(code, "failed to decode AddAuthorizerPayload")
		Logger.Error("public key error", zap.Error(err))
		return "", err
	}

	if params.PublicKey == "" {
		err = common.NewError(code, "public key was not included with transaction")
		Logger.Error("public key error", zap.Error(err))
		return "", err
	}

	if params.StakePoolSettings.DelegateWallet == "" {
		return "", common.NewError(code, "authorizer's delegate_wallet not set")
	}

	if authorizerID != params.StakePoolSettings.DelegateWallet {
		return "", common.NewError(code, "access denied, allowed for delegate_wallet owner only")
	}

	globalNode, err := GetGlobalNode(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get global node, authorizer(authorizerID: %v), err: %v", authorizerID, err)
		err = common.NewError(code, msg)
		Logger.Error("get global node", zap.Error(err))
		return "", err
	}

	// Validating StakePoolSettings against GlobalNode settings

	// Check existing Authorizer

	authorizer, err := GetAuthorizerNode(authorizerID, ctx)
	if err == nil && authorizer != nil {
		err = fmt.Errorf("authorizer(authorizerID: %v) already exists", authorizerID)
		Logger.Error(code, zap.Error(err))
		return "", err
	}

	// Create Authorizer instance

	authorizerPublicKey := params.PublicKey
	authorizerURL := params.URL

	authorizer = NewAuthorizer(authorizerID, authorizerPublicKey, authorizerURL)
	err = authorizer.Save(ctx)
	if err != nil {
		msg := fmt.Sprintf("error saving authorizer(authorizerID: %v), err: %v", authorizerID, err)
		err = common.NewError(code, msg)
		Logger.Error("saving authorizer node", zap.Error(err))
		return "", err
	}

	// Creating StakePool

	var sp *StakePool
	sp, err = zcn.getOrUpdateStakePool(globalNode, authorizerID, params.StakePoolSettings, ctx)
	if err != nil {
		return "", common.NewError(code, "failed to get or create stake pool: "+err.Error())
	}
	if err = sp.save(zcn.ID, authorizerID, ctx); err != nil {
		return "", common.NewError(code, "failed to save stake pool: "+err.Error())
	}

	// Events emission
	ctx.EmitEvent(event.TypeStats, event.TagAddAuthorizer, authorizerID, authorizer.ToEvent())

	return string(authorizer.Encode()), nil
}

func (zcn *ZCNSmartContract) UpdateAuthorizerStakePool(
	tran *transaction.Transaction,
	input []byte,
	ctx cstate.StateContextI,
) (response string, err error) {
	const (
		code = "update_authorizer_staking_pool_failed"
	)

	var (
		authorizerID = tran.ClientID // sender address
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

	params := UpdateAuthorizerStakePoolPayload{}
	err = params.Decode(input)
	if err != nil {
		err = common.NewError(code, "failed to decode AddAuthorizerPayload")
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

	// StakePool may be updated only if authorizer exists/not deleted

	authorizer, err := GetAuthorizerNode(authorizerID, ctx)
	if err == nil && authorizer != nil {
		var sp *StakePool
		sp, err = zcn.getOrUpdateStakePool(globalNode, authorizerID, poolSettings, ctx)
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

	usp, err := stakepool.GetUserStakePools(prr.ProviderType, tran.ClientID, ctx)
	if err != nil {
		return "", common.NewErrorf(code, "can't get related user stake pools: %v", err)
	}

	providerId := usp.FindProvider(prr.PoolId)
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

	if err := usp.Save(spenum.Authorizer, tran.ClientID, ctx); err != nil {
		return "", common.NewErrorf(code, "error saving user stake pool, %v", err)
	}

	if err := sp.save(zcn.ID, providerId, ctx); err != nil {
		return "", common.NewErrorf(code, "error saving stake pool, %v", err)
	}

	return "", nil
}

func (zcn *ZCNSmartContract) DeleteAuthorizer(tran *transaction.Transaction, input []byte, ctx cstate.StateContextI) (string, error) {
	var (
		authorizerID = tran.ClientID
		errorCode    = "failed to delete authorizer"
		err          error
	)

	params := DeleteAuthorizerPayload{}
	err = params.Decode(input)
	if err != nil {
		err = common.NewError(errorCode, "failed to decode DeleteAuthorizerPayload")
		Logger.Error("payload decoding error", zap.Error(err))
		return "", err
	}

	if params.ID != tran.ClientID {
		err = common.NewError(errorCode, "delete Authorizer is not allowed")
		Logger.Error("delete authorizer", zap.Error(err))
		return "", err
	}

	authorizer, err := GetAuthorizerNode(authorizerID, ctx)
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

	// Mark StakePool as Deleted but not delete it

	sp, err := zcn.getStakePool(authorizerID, ctx)
	if err != nil {
		return "", common.NewError(errorCode, "failed to get stake pool: "+err.Error())
	}
	for _, v := range sp.Pools {
		v.Status = spenum.Deleted
	}
	if err = sp.save(zcn.ID, authorizerID, ctx); err != nil {
		return "", common.NewError(errorCode, "failed to save stake pool: "+err.Error())
	}

	// Delete authorizer node

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
		"Successfully deleted authorizer",
		zap.String("hash", tran.Hash),
		zap.String("authorizerID", authorizerID),
	)

	return string(sp.Encode()), nil
}

func (zcn *ZCNSmartContract) UpdateAuthorizerConfig(
	_ *transaction.Transaction,
	input []byte,
	ctx cstate.StateContextI,
) (string, error) {
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
	if err = in.Decode(input); err != nil {
		msg := fmt.Sprintf("decoding request: %v", err)
		Logger.Error(msg, zap.Error(err))
		err = common.NewError(code, msg)
		return "", err
	}

	if in.Config.Fee > gn.MaxFee {
		msg := fmt.Sprintf("authorizer fee (%v) is greater than allowed by SC (%v)", in.Config.Fee, gn.MaxFee)
		err = common.NewErrorf(code, msg)
		Logger.Error(msg, zap.Error(err))
		return "", err
	}

	authorizer, err := GetAuthorizerNode(in.ID, ctx)
	if err != nil {
		return "", common.NewError(code, err.Error())
	}

	err = authorizer.UpdateConfig(in.Config)
	if err != nil {
		msg := fmt.Sprintf("error updating config for authorizer(authorizerID: %v), err: %v", authorizer.ID, err)
		err = common.NewError(code, msg)
		Logger.Error("updating settings", zap.Error(err))
		return "", err
	}

	err = authorizer.Save(ctx)
	if err != nil {
		msg := fmt.Sprintf("error saving authorizer(authorizerID: %v), err: %v", authorizer.ID, err)
		err = common.NewError(code, msg)
		Logger.Error("saving authorizer node", zap.Error(err))
		return "", err
	}

	ctx.EmitEvent(event.TypeStats, event.TagUpdateAuthorizer, authorizer.ID, authorizer.ToEvent())

	return string(authorizer.Encode()), nil
}
