package zcnsc

import (
	"fmt"
	"math"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Mint inputData - is a MintPayload
func (zcn *ZCNSmartContract) Mint(trans *transaction.Transaction, inputData []byte, ctx cstate.StateContextI) (resp string, err error) {
	const (
		code = "failed to mint"
	)

	info := fmt.Sprintf(
		"transaction hash %s, clientID: %s, payload: %s",
		trans.Hash,
		trans.ClientID,
		string(inputData),
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

	if len(payload.Signatures) == 0 {
		msg := fmt.Sprintf("payload doesn't contain signatures: %v, %s", err, info)
		err = common.NewError(code, msg)
		return
	}

	numAuth, err := getAuthorizerCount(ctx)
	threshold := int(math.RoundToEven(gn.PercentAuthorizers * float64(numAuth)))

	// if number of slices exceeds limits the check only withing required range
	if len(payload.Signatures) < threshold {
		msg := fmt.Sprintf("no of signatures lesser than threshold %d: %v, %s", threshold, err, info)
		err = common.NewError(code, msg)
		return
	}

	if len(payload.Signatures) > numAuth {
		logging.Logger.Info("no of signatures execeed the no of available authorizers", zap.Int("available", numAuth))
		payload.Signatures = payload.Signatures[0:numAuth]
	}

	// ClientID - is a client who broadcasts this transaction to mint token
	// ToClientID - is an address of the smart contract
	if payload.ReceivingClientID != trans.ClientID {
		msg := fmt.Sprintf("transaction made from different account who made burn,  Original: %s, Current: %s",
			payload.ReceivingClientID, trans.ClientID)
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

	_, exists := gn.WZCNNonceMinted[payload.Nonce]
	if exists { // global nonce from ETH SC has already been minted
		err = common.NewError(
			code,
			fmt.Sprintf(
				"nonce given (%v) for receiving client (%s) has alredy been minted for Node.ID: '%s', %s",
				payload.Nonce, payload.ReceivingClientID, trans.ClientID, info))
		return
	}

	uniqueSignatures := payload.getUniqueSignatures()

	// verify signatures of authorizers
	err = payload.verifySignatures(uniqueSignatures, ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to verify signatures with error: %v, %s", err, info)
		err = common.NewError(code, msg)
	}

	if len(uniqueSignatures) < threshold {
		err = common.NewError(
			code,
			"not enough valid signatures for minting",
		)
		return
	}

	// record the global nonce from solidity smart contract
	gn.WZCNNonceMinted[payload.Nonce] = true

	var (
		amount currency.Coin
		n      currency.Coin
		share  currency.Coin
	)
	share, _, err = currency.DistributeCoin(gn.ZCNSConfig.MaxFee, int64(len(payload.Signatures)))
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, DistributeCoin operation, %s", code, info))
		return
	}
	n, err = currency.Int64ToCoin(int64(len(payload.Signatures)))
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, convert len signatures to coin, %s", code, info))
		return
	}
	amount, err = currency.MinusCoin(payload.Amount, share*n)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, payload.Amount - share * len(signatures), %s", code, info))
		return
	}
	for _, sig := range payload.Signatures {
		err = ctx.AddTransfer(&state.Transfer{
			ClientID:   trans.ClientID,
			ToClientID: sig.ID,
			Amount:     share,
		})
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("%s, AddTransfer operation, %s", code, info))
			return
		}
	}

	// 123

	// mint the tokens
	err = ctx.AddMint(&state.Mint{
		Minter:     gn.ID,
		ToClientID: trans.ClientID,
		Amount:     amount,
	})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, add mint operation, %s", code, info))
		return
	}

	// Save the user node
	err = gn.Save(ctx)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, global node failed to be saved, %s", code, info))
		return
	}

	resp = string(payload.Encode())
	return
}
