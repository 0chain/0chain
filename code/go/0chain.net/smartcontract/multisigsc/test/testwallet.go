package main

import (
	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/state"
	mptwallet "0chain.net/chaincore/wallet"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/multisigsc"
	. "github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// Client data necessary for a multi-sig wallet. Contains the private keys for
// every sub-key because we are just testing here.
type testWallet struct {
	id int

	signatureScheme string

	groupClientID string
	groupKey      encryption.SignatureScheme

	signerClientIDs []string
	signerKeys      []encryption.ThresholdSignatureScheme

	t, n int
}

type testProposal struct {
	votes map[string]multisigsc.Vote
}

func newTestWallet(id int, signatureScheme string, t, n int) testWallet {
	groupKey := encryption.GetSignatureScheme(signatureScheme)

	err := groupKey.GenerateKeys()
	if err != nil {
		panic(err)
	}

	groupClientID := clientIDForKey(groupKey)

	signerKeys, err := encryption.GenerateThresholdKeyShares(signatureScheme, t, n, groupKey)
	if err != nil {
		panic(err)
	}

	var signerClientIDs []string
	for _, key := range signerKeys {
		signerClientIDs = append(signerClientIDs, clientIDForKey(key))
	}

	return testWallet{
		id: id,

		signatureScheme: signatureScheme,

		groupClientID: groupClientID,
		groupKey:      groupKey,

		signerClientIDs: signerClientIDs,
		signerKeys:      signerKeys,

		t: t,
		n: n,
	}
}

func (t testWallet) registerMPTWallets() {
	// Register MPT wallets for everyone in our group.
	registerMPTWallet(t.getGroupMPTWallet())

	for _, mptWallet := range t.getSignerMPTWallets() {
		registerMPTWallet(mptWallet)
	}

	// Give the group and its sub-keys some tokens to play with.
	owner := getOwnerWallet(c.signatureScheme, c.ownerKeysFile)

	Logger.Info("Requesting airdrop for group wallet...", zap.Int("multi-sig wallet#", t.id))
	airdrop(owner, t.groupClientID)
	Logger.Info("Success on airdrop for group wallet", zap.Int("multi-sig wallet#", t.id))

	for i, signerClientID := range t.signerClientIDs {
		Logger.Info("Requesting airdrop for signer wallet...", zap.Int("multi-sig wallet#", t.id), zap.Int("signer#", i))
		airdrop(owner, signerClientID)
		Logger.Info("Success on airdrop for signer wallet", zap.Int("multi-sig wallet#", t.id), zap.Int("signer#", i))
	}
}

func (t testWallet) registerSCWallet() string {
	var signerThresholdIDs []string
	var signerPublicKeys []string

	for _, scheme := range t.signerKeys {
		signerThresholdIDs = append(signerThresholdIDs, scheme.GetID())
		signerPublicKeys = append(signerPublicKeys, scheme.GetPublicKey())
	}

	data := httpclientutil.SmartContractTxnData{
		Name: multisigsc.RegisterFuncName,
		InputArgs: multisigsc.Wallet{
			ClientID:        t.groupClientID,
			SignatureScheme: c.signatureScheme,
			PublicKey:       t.groupKey.GetPublicKey(),

			SignerThresholdIDs: signerThresholdIDs,
			SignerPublicKeys:   signerPublicKeys,

			NumRequired: t.t,
		},
	}

	Logger.Info("Requesting SC:Register...", zap.Int("multi-sig wallet#", t.id), zap.Any("args", data.InputArgs))

	txn := t.groupTransaction(0, &data)

	Logger.Info("Response received for SC:Register", zap.Int("multi-sig wallet#", t.id), zap.String("txn hash", txn.Hash), zap.String("txn output", txn.TransactionOutput))

	return txn.TransactionOutput
}

func (t testWallet) groupTransaction(value int64, data interface{}) httpclientutil.Transaction {
	return executeSCTransaction(t.getGroupMPTWallet(), multisigsc.Address, value, data)
}

func (t testWallet) newProposal(proposalID string, toClientID string, value int64) testProposal {
	transfer := state.Transfer{
		ClientID:   t.groupClientID,
		ToClientID: toClientID,
		Amount:     currency.Coin(value),
	}

	votes := make(map[string]multisigsc.Vote)

	for i, signer := range t.signerKeys {
		signedTransfer := state.SignedTransfer{
			Transfer: transfer,
		}

		err := signedTransfer.Sign(signer.(encryption.SignatureScheme))
		if err != nil {
			Logger.Fatal("Failed to sign transfer", zap.Error(err))
		}

		votes[t.signerClientIDs[i]] = multisigsc.Vote{
			ProposalID: proposalID,
			Transfer:   transfer,
			Signature:  signedTransfer.Sig,
		}
	}

	return testProposal{
		votes: votes,
	}
}

func (t testWallet) registerVote(p testProposal, signerClientID string) string {
	data := httpclientutil.SmartContractTxnData{
		Name:      multisigsc.VoteFuncName,
		InputArgs: p.votes[signerClientID],
	}

	Logger.Info("Requesting SC:Vote...", zap.Int("multi-sig wallet#", t.id), zap.Any("args", data.InputArgs))

	txn := t.signerTransaction(signerClientID, 0, &data)

	Logger.Info("Response received for SC:Vote", zap.Int("multi-sig wallet#", t.id), zap.String("txn hash", txn.Hash), zap.String("txn output", txn.TransactionOutput))

	return txn.TransactionOutput
}

func (t testWallet) signerTransaction(signerClientID string, value int64, data interface{}) httpclientutil.Transaction {
	signerMPTWallet := t.getMPTWalletForSigner(signerClientID)

	return executeSCTransaction(signerMPTWallet, multisigsc.Address, value, data)
}

func (t testWallet) getGroupMPTWallet() mptwallet.Wallet {
	return mptwallet.Wallet{
		SignatureScheme: t.groupKey,
		PublicKey:       t.groupKey.GetPublicKey(),
		ClientID:        t.groupClientID,
	}
}

func (t testWallet) getSignerMPTWallets() []mptwallet.Wallet {
	var ws []mptwallet.Wallet

	for i := range t.signerClientIDs {
		w := mptwallet.Wallet{
			SignatureScheme: t.signerKeys[i],
			PublicKey:       t.signerKeys[i].GetPublicKey(),
			ClientID:        t.signerClientIDs[i],
		}
		ws = append(ws, w)
	}

	return ws
}

func (t testWallet) getMPTWalletForSigner(signerClientID string) mptwallet.Wallet {
	for i, id := range t.signerClientIDs {
		if id == signerClientID {
			return mptwallet.Wallet{
				SignatureScheme: t.signerKeys[i],
				PublicKey:       t.signerKeys[i].GetPublicKey(),
				ClientID:        t.signerClientIDs[i],
			}
		}
	}

	Logger.Fatal("Logic error in asking for signer wallet", zap.String("signerClientID", signerClientID))
	return mptwallet.Wallet{} // Never reached.
}

func (t testWallet) toWallet() multisigsc.Wallet { //nolint
	var signerThresholdIDs []string
	var signerPublicKeys []string

	for _, signer := range t.signerKeys {
		signerThresholdIDs = append(signerThresholdIDs, signer.GetID())
		signerPublicKeys = append(signerPublicKeys, signer.GetPublicKey())
	}

	return multisigsc.Wallet{
		ClientID:           t.groupClientID,
		SignerThresholdIDs: signerThresholdIDs,
		SignerPublicKeys:   signerPublicKeys,
		NumRequired:        t.t,
		SignatureScheme:    t.signatureScheme,
	}
}
