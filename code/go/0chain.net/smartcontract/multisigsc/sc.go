package multisigsc

import (
	"encoding/json"
	"fmt"
	"time"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

const (
	name             = "multisig"
	Address          = "27b5ef7120252b79f9dd9c05505dd28f328c80f6863ee446daede08a84d651a7"
	RegisterFuncName = "register"
	VoteFuncName     = "vote"
	LogTimingInfo    = false
)

type MultiSigSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (ms *MultiSigSmartContract) GetName() string {
	return name
}

func (ms *MultiSigSmartContract) GetAddress() string {
	return Address
}

func (ms *MultiSigSmartContract) GetRestPoints() map[string]smartcontractinterface.SmartContractRestHandler {
	return ms.SmartContract.RestHandlers
}

func (ms *MultiSigSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bc smartcontractinterface.BCContextI) {
	ms.SmartContract = sc
}

func (ms MultiSigSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances state.StateContextI) (string, error) {
	if LogTimingInfo {
		start := time.Now().UnixNano()
		defer printTimeTaken(start)
	}

	switch funcName {
	case RegisterFuncName:
		return ms.register(t.ClientID, inputData)
	case VoteFuncName:
		return ms.vote(t.Hash, t.ClientID, balances.GetBlock().CreationDate, inputData, balances)
	default:
		return "err_execute_function_not_found: no function with that name: " + funcName, nil
	}
}

func printTimeTaken(start int64) {
	end := time.Now().UnixNano()
	duration := (end - start) / int64(time.Microsecond)

	Logger.Info("Multi-signature smart contract execution time", zap.Int64("µs duration", duration))
}

func (ms MultiSigSmartContract) register(registeringClientID string, inputData []byte) (string, error) {
	var w Wallet

	err := json.Unmarshal(inputData, &w)
	if err != nil {
		return "err_register_formatting: incorrect request format", nil
	}

	// Check for silly parameters that don't make sense. Not a comprehensive
	// check so errors might still pop up down the line.
	if !w.valid(registeringClientID) {
		return "err_register_invalid: invalid request", nil
	}

	// If you want to replace a multi-sig wallet, you have to delete the
	// previous one first.
	alreadyExisted, err := ms.walletExists(registeringClientID)
	if err != nil {
		// I/O error.
		return "", err
	}
	if alreadyExisted {
		return "err_register_exists: multi-sig wallet already exists", nil
	}

	err = ms.putWallet(w)
	if err != nil {
		// I/O error.
		return "", err
	}

	return "success: multi-signature wallet registered", nil
}

func (ms MultiSigSmartContract) vote(currentTxnHash, signingClientID string, now common.Timestamp, inputData []byte, balances state.StateContextI) (string, error) {
	// Garbage collection of old proposals happens incrementally with every
	// incoming vote.
	err := ms.pruneExpirationQueue(now)
	if err != nil {
		// I/O error.
		return "", err
	}

	var v Vote

	err = json.Unmarshal(inputData, &v)
	if err != nil {
		return "err_vote_formatting: incorrect vote format", nil
	}

	// Play nice.
	if !v.notTooBig() {
		return "err_vote_too_big: an input field exceeded allowable length", nil
	}
	if !v.hasValidAmount() {
		return "err_vote_invalid_tokens: invalid number of tokens to send", nil
	}
	if !v.hasSignature() {
		return "err_vote_no_signature: must sign vote", nil
	}

	// Every vote is associated with a proposal. If an appropriate proposal does
	// not exist yet, create one.
	p, err := ms.findOrCreateProposal(now, v)
	if err != nil {
		// I/O error.
		return "", err
	}

	// Ensure all voters are on the same page.
	if !v.isCompatibleWithProposal(p) {
		return "err_vote_not_compatible: previous votes for same proposal differed", nil
	}

	// Check if the proposal was already finished, making this vote unnecessary.
	if p.ExecutedInTxnHash != "" {
		return "success 0: proposal previously executed in transaction hash " + p.ExecutedInTxnHash, nil
	}

	// Check that the multi-sig wallet is registered.
	w, err := ms.getWallet(v.Transfer.ClientID)
	if err != nil {
		// I/O error.
		return "", err
	}
	if w.isEmpty() {
		return "err_vote_wallet_not_registered: wallet not registered", nil
	}

	// Check that the voter is registered on the wallet and that the signature
	// is valid.
	signerThresholdID := w.thresholdIdForSigner(signingClientID)
	if signerThresholdID == "" {
		return "err_vote_auth: authorization failure", nil
	}
	if !w.isVoteAuthorized(signingClientID, v) {
		return "err_vote_auth: authorization failure", nil
	}

	remaining := w.NumRequired - len(p.SignerSignatures)

	// Check if this is a duplicate vote.
	for _, id := range p.SignerThresholdIDs {
		if id == signerThresholdID {
			return fmt.Sprintf("success %d: already voted, still need %d other votes", remaining, remaining), nil
		}
	}

	// Add the signature to the proposal. It is counted as a vote.
	p.SignerThresholdIDs = append(p.SignerThresholdIDs, signerThresholdID)
	p.SignerSignatures = append(p.SignerSignatures, v.Signature)

	// Save the proposal.
	err = ms.putProposal(p)
	if err != nil {
		// I/O error.
		return "", err
	}

	remaining -= 1

	// If more votes are still needed we must wait for them. Nothing more to do.
	if remaining > 0 {
		return fmt.Sprintf("success %d: need %d more votes", remaining, remaining), nil
	}

	// Otherwise we can recover the threshold signature on the transfer and
	// execute it.
	thresholdSignature, err := w.constructTransferSignature(p)
	if err != nil {
		return "err_vote_recover: in signature recovery: " + err.Error(), nil
	}

	p.ClientSignature = thresholdSignature

	// Request the transfer. The blockchain will validate the signature and
	// execute the transfer soon. If the signature is found to be invalid,
	// this vote transaction will fail.
	signedTransfer := w.makeSignedTransferForProposal(p)
	balances.AddSignedTransfer(&signedTransfer)

	// Save the proposal again.
	p.ExecutedInTxnHash = currentTxnHash

	err = ms.putProposal(p)
	if err != nil {
		// I/O error.
		return "", err
	}

	return "success 0: transfer executed with signature " + p.ClientSignature, nil
}

// Prune the oldest proposal if it has expired.
func (ms MultiSigSmartContract) pruneExpirationQueue(now common.Timestamp) error {
	q, err := ms.getOrCreateExpirationQueue()
	if err != nil {
		return err
	}

	// Reference to oldest proposal.
	ref := q.Head

	if ref == (proposalRef{}) {
		// No proposals currently exist.
		return nil
	}

	p, err := ms.getProposal(ref)
	if err != nil {
		return err
	}

	if p.isExpired(now) {
		return ms.prune(ref)
	}

	return nil
}

func (ms MultiSigSmartContract) prune(ref proposalRef) error {
	// Before we prune this proposal, fetch it one last time.
	p, err := ms.getProposal(ref)
	if err != nil {
		return err
	}

	// Update expiry queue.
	q, err := ms.getOrCreateExpirationQueue()
	if err != nil {
		return err
	}

	qChanged := false

	if q.Head == ref {
		q.Head = p.Next
		qChanged = true
	}
	if q.Tail == ref {
		q.Tail = p.Prev
		qChanged = true
	}

	if qChanged {
		err = ms.putExpirationQueue(q)
		if err != nil {
			return err
		}
	}

	// Update links.
	if p.Next != (proposalRef{}) {
		next, err := ms.getProposal(p.Next)
		if err != nil {
			return err
		}

		next.Prev = p.Prev

		err = ms.putProposal(next)
		if err != nil {
			return err
		}
	}

	if p.Prev != (proposalRef{}) {
		prev, err := ms.getProposal(p.Prev)
		if err != nil {
			return err
		}

		prev.Next = p.Next

		err = ms.putProposal(prev)
		if err != nil {
			return err
		}
	}

	// Now we can delete the pruned proposal.
	err = ms.DB.DeleteNode(p.getKey())
	if err != nil {
		return err
	}

	return nil
}

func (ms MultiSigSmartContract) findOrCreateProposal(now common.Timestamp, v Vote) (proposal, error) {
	// Start by trying to find an existing proposal.
	p, err := ms.getProposal(v.getProposalRef())
	if err != nil {
		return proposal{}, err
	}

	// Treat expired-but-not-yet-pruned proposals identically to pruned
	// proposals. Do this by pruning them now.
	if !p.isEmpty() && p.isExpired(now) {
		err = ms.prune(p.ref())
		if err != nil {
			return proposal{}, err
		}

		p = proposal{}
	}

	// If it didn't exist or was expired, create it and update expiration queue.
	if p.isEmpty() {
		p, err = ms.createProposal(now, v)
		if err != nil {
			return proposal{}, err
		}
	}

	return p, nil
}

// Create a proposal and add it to the expiration queue. Performs I/O.
func (ms MultiSigSmartContract) createProposal(now common.Timestamp, v Vote) (proposal, error) {
	q, err := ms.getOrCreateExpirationQueue()
	if err != nil {
		return proposal{}, err
	}

	// Create proposal.
	p := proposal{
		ProposalID:     v.ProposalID,
		ExpirationDate: now + ExpirationTime,

		Next: proposalRef{},
		Prev: q.Tail,

		Transfer: v.Transfer,

		SignerThresholdIDs: []string{},
		SignerSignatures:   []string{},

		ClientSignature:   "",
		ExecutedInTxnHash: "",
	}

	err = ms.putProposal(p)
	if err != nil {
		return proposal{}, err
	}

	// Update links.
	if q.Tail != (proposalRef{}) {
		prev, err := ms.getProposal(q.Tail)
		if err != nil {
			return proposal{}, err
		}

		prev.Next = p.ref()

		err = ms.putProposal(prev)
		if err != nil {
			return proposal{}, err
		}
	}

	// Update expiration queue.
	if q.Head == (proposalRef{}) {
		// The queue was empty.
		q.Head = p.ref()
	}

	// Enqueue.
	q.Tail = p.ref()

	err = ms.putExpirationQueue(q)
	if err != nil {
		return proposal{}, err
	}

	return p, nil
}

func (ms MultiSigSmartContract) walletExists(clientID string) (bool, error) {
	walletBytes, err := ms.DB.GetNode(getWalletKey(clientID))
	if err != nil {
		return false, err
	}

	return walletBytes != nil, nil
}

func (ms MultiSigSmartContract) getWallet(clientID string) (Wallet, error) {
	walletBytes, err := ms.DB.GetNode(getWalletKey(clientID))
	if err != nil {
		// I/O error.
		return Wallet{}, err
	}
	if walletBytes == nil {
		// Not found.
		return Wallet{}, nil
	}

	w := Wallet{}
	err = json.Unmarshal(walletBytes, &w)
	if err != nil {
		// Decoding error.
		return Wallet{}, err
	}

	// Okay.
	return w, nil
}

func (ms MultiSigSmartContract) putWallet(w Wallet) error {
	walletBytes, err := json.Marshal(w)
	if err != nil {
		return err
	}

	return ms.DB.PutNode(w.getKey(), walletBytes)
}

func (ms MultiSigSmartContract) getProposal(ref proposalRef) (proposal, error) {
	proposalBytes, err := ms.DB.GetNode(getProposalKey(ref.ClientID, ref.ProposalID))
	if err != nil {
		// I/O error.
		return proposal{}, err
	}
	if proposalBytes == nil {
		// Not found.
		return proposal{}, nil
	}

	p := proposal{}
	err = json.Unmarshal(proposalBytes, &p)
	if err != nil {
		// Decoding error.
		return proposal{}, err
	}

	// Okay.
	return p, nil
}

func (ms MultiSigSmartContract) putProposal(p proposal) error {
	proposalBytes, err := json.Marshal(p)
	if err != nil {
		return err
	}

	return ms.DB.PutNode(p.getKey(), proposalBytes)
}

func (ms MultiSigSmartContract) getOrCreateExpirationQueue() (expirationQueue, error) {
	qBytes, err := ms.DB.GetNode(getExpirationQueueKey())
	if err != nil {
		// I/O error.
		return expirationQueue{}, err
	}
	if qBytes == nil {
		// Not found.
		return expirationQueue{}, nil
	}

	q := expirationQueue{}
	err = json.Unmarshal(qBytes, &q)
	if err != nil {
		// Decoding error.
		return expirationQueue{}, err
	}

	// Okay.
	return q, nil
}

func (ms MultiSigSmartContract) putExpirationQueue(q expirationQueue) error {
	qBytes, err := json.Marshal(q)
	if err != nil {
		return err
	}

	return ms.DB.PutNode(getExpirationQueueKey(), qBytes)
}
