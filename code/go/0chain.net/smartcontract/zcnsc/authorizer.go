package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// AddAuthorizer sc API function
// Transaction must include ClientID, ToClientID, PublicKey, Hash, Value
// ContractMap contains all the SC addresses
// ClientID is an authorizerID - used to search for authorizer
// ToClient is an SC address
func (zcn *ZCNSmartContract) AddAuthorizer(
	tran *transaction.Transaction,
	inputData []byte,
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

	if inputData == nil {
		msg := "input data is nil"
		err = common.NewError(code, msg)
		Logger.Error(msg, zap.Error(err))
		return "", err
	}

	// Decode input

	params := AuthorizerParameter{}
	err = params.Decode(inputData)
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
	authorizerStakingPoolSettings := params.StakePoolSettings

	// Check existing Authorizer

	authorizer, err = GetAuthorizerNode(authorizerID, ctx)
	if err == nil && authorizer != nil {
		msg := fmt.Sprintf("authorizer(authorizerID: %v) already exists: %v", authorizerID, err)
		err = common.NewError(code, msg)
		Logger.Warn("get authorizer node", zap.Error(err))
		return "", err
	}

	if err != nil && err == util.ErrNodeNotFound {
		Logger.Error("get authorizer node", zap.Error(err))
		return "", err
	}

	globalNode, err := GetGlobalNode(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get global node, authorizer(authorizerID: %v), err: %v", authorizerID, err)
		err = common.NewError(code, msg)
		Logger.Error("get global node", zap.Error(err))
		return "", err
	}

	// create stake pool for the validator to count its rewards'

	var createOrUpdateStakePool = func() (err error) {
		var sp *StakePool
		sp, err = zcn.getOrUpdateStakePool(globalNode, authorizerID, spenum.Authorizer, authorizerStakingPoolSettings, ctx)
		if err != nil {
			return common.NewError(code, "failed to get or create stake pool: "+err.Error())
		}
		if err = sp.save(zcn.ID, authorizerID, ctx); err != nil {
			return common.NewError(code, "failed to save stake pool: "+err.Error())
		}
		return err
	}

	// Check existing Authorizer

	authorizer, err = GetAuthorizerNode(authorizerID, ctx)
	if err == nil && authorizer != nil {
		errs := fmt.Errorf("authorizer(authorizerID: %v) already exists", authorizerID)
		err = createOrUpdateStakePool()
		if err != nil {
			errs = multierror.Append(errs, errors.Wrap(err, "failed to get or create stake pool"))
		} else {
			Logger.Info("create or update stake pool completed successfully")
		}

		if authorizer.UpdateStakePoolSettings(&authorizerStakingPoolSettings) {
			err = authorizer.Save(ctx)
			if err != nil {
				errs = multierror.Append(errs, errors.Wrap(err, "failed to update stake pool settings"))
			} else {
				Logger.Info("update pool settings completed successfully")
			}
		}

		Logger.Error(code, zap.Error(errs))
		return "", errs
	} else {
		// compare the global min of authorizerNode Authorizer to that of the transaction amount
		if globalNode.MinStakeAmount > state.Balance(tran.Value*1e10) {
			msg := fmt.Sprintf("min stake amount '(%d)' > transaction value '(%d)'",
				globalNode.MinStakeAmount, tran.Value)
			err = common.NewError(code, msg)
			Logger.Error("min stake amount > transaction value", zap.Error(err))
			return "", err
		}

		// Create Authorizer instance

		authorizer = NewAuthorizer(authorizerID, authorizerPublicKey, authorizerURL, &authorizerStakingPoolSettings)

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

		err = createOrUpdateStakePool()
		if err != nil {
			err = common.NewError(code, "failed to create or update stake pool")
			Logger.Error("saving or creating stake pool", zap.Error(err))
			return "", err
		}

		ctx.EmitEvent(event.TypeStats, event.TagAddAuthorizer, authorizerID, string(ev))
	}

	return
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
