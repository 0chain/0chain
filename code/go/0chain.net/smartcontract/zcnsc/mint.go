package zcnsc

import (
	"fmt"
	"math"
	"math/rand"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool/spenum"
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
	if err != nil {
		msg := fmt.Sprintf("error while retriving number of authorizers: %v, %s", err, info)
		err = common.NewError(code, msg)
		return
	}

	if numAuth == 0 {
		return "", common.NewError(code, "no authorizers found")
	}

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

	if err = PartitionWZCNMintedNonceAdd(ctx, payload.Nonce); err != nil {
		if partitions.ErrItemExist(err) {
			err = common.NewError(
				code,
				fmt.Sprintf(
					"nonce given (%v) for receiving client (%s) has already been minted for Node.ID: '%s', %s",
					payload.Nonce, payload.ReceivingClientID, trans.ClientID, info))
			return
		}

		err = common.NewError(code, err.Error())
		return
	}

	uniqueSignatures := payload.getUniqueSignatures()

	// verify signatures of authorizers
	err = payload.verifySignatures(uniqueSignatures, ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to verify signatures with error: %v, %s", err, info)
		err = common.NewError(code, msg)
		return
	}

	if len(uniqueSignatures) < threshold {
		err = common.NewError(
			code,
			"not enough valid signatures for minting",
		)
		return
	}

	var (
		amount currency.Coin
		share  currency.Coin
	)
	share, _, err = currency.DistributeCoin(gn.ZCNSConfig.MaxFee, int64(len(payload.Signatures)))
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, DistributeCoin operation, %s", code, info))
		return
	}

	amount, err = currency.MinusCoin(payload.Amount, share)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, payload.Amount - share, %s", code, info))
		return
	}
	payload.Amount = amount

	// record mint nonce for a certain user
	signers := make([]string, 0, len(payload.Signatures))
	for _, sig := range payload.Signatures {
		signers = append(signers, sig.ID)
	}

	ctx.EmitEvent(event.TypeStats, event.TagAddBridgeMint, trans.ClientID, &event.BridgeMint{
		UserID:    trans.ClientID,
		MintNonce: payload.Nonce,
		Amount:    payload.Amount,
		Signers:   signers,
	})

	rand.Seed(ctx.GetBlock().GetRoundRandomSeed())
	sig := payload.Signatures[rand.Intn(len(payload.Signatures))]

	sp, err := zcn.getStakePool(sig.ID, ctx)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("failed to retrieve stake pool for authorizer %s", sig.ID))
		return
	}

	err = sp.DistributeRewards(share, sig.ID, spenum.Authorizer, spenum.FeeRewardAuthorizer, ctx)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("failed to retrieve stake pool for authorizer %s", sig.ID))
		return
	}

	// mint the tokens
	err = ctx.AddMint(&state.Mint{
		Minter:     ADDRESS,
		ToClientID: trans.ClientID,
		Amount:     payload.Amount,
	})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s, Add mint operation, %s", code, info))
		return
	}

	if err = sp.save("", sig.ID, ctx); err != nil {
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
