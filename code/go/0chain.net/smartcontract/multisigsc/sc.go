package multisigsc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"0chain.net/chaincore/smartcontract"

	"0chain.net/chaincore/chain/state"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
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

func NewMultiSigSmartContract() smartcontractinterface.SmartContractInterface {
	var msCopy = &MultiSigSmartContract{
		SmartContract: smartcontractinterface.NewSC(Address),
	}
	msCopy.setSC(msCopy.SmartContract, &smartcontract.BCContext{})
	return msCopy
}

func (ipsc *MultiSigSmartContract) GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	return ipsc.SmartContract.HandlerStats(ctx, params)
}

func (ipsc *MultiSigSmartContract) GetExecutionStats() map[string]interface{} {
	return ipsc.SmartContractExecutionStats
}

func (ms *MultiSigSmartContract) GetName() string {
	return name
}

func (ms *MultiSigSmartContract) GetAddress() string {
	return Address
}

func (ms *MultiSigSmartContract) setSC(sc *smartcontractinterface.SmartContract, bc smartcontractinterface.BCContextI) {
	ms.SmartContract = sc
}

func (ms *MultiSigSmartContract) GetCost(t *transaction.Transaction, funcName string, balances state.StateContextI) (int, error) {
	return 0, nil
}

func (ms *MultiSigSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances state.StateContextI) (string, error) {
	if LogTimingInfo {
		start := time.Now().UnixNano()
		defer printTimeTaken(start)
	}

	switch funcName {
	case RegisterFuncName:
		return ms.register(t.ClientID, inputData, balances)
	case VoteFuncName:
		return ms.vote(t.Hash, t.ClientID, balances.GetBlock().CreationDate, inputData, balances)
	default:
		return "err_execute_function_not_found: no multi sig smart contract function with that name: " + funcName, nil
	}
}

func printTimeTaken(start int64) {
	end := time.Now().UnixNano()
	duration := (end - start) / int64(time.Microsecond)

	Logger.Info("Multi-signature smart contract execution time", zap.Int64("µs duration", duration))
}

func (ms MultiSigSmartContract) register(registeringClientID string, inputData []byte, balances state.StateContextI) (string, error) {
	var w Wallet

	err := json.Unmarshal(inputData, &w)
	if err != nil {
		return "err_register_formatting: incorrect request format", err
	}

	// Check for silly parameters that don't make sense. Not a comprehensive
	// check so errors might still pop up down the line.
	isValid, err := w.valid(registeringClientID)

	if err != nil {
		return "", err
	}
	if !isValid {
		//if there are no errors, it should be valid
		return "err_register_invalid: invalid request", common.NewError("err_register_invalid", "invalid request")
	}

	// If you want to replace a multi-sig wallet, you have to delete the
	// previous one first.
	alreadyExisted, err := ms.walletExists(registeringClientID, balances)
	if err != nil {
		// I/O error.
		if err != util.ErrValueNotPresent && err != util.ErrNodeNotFound {
			return "", err
		} //else means no wallet exists
	}
	if alreadyExisted {
		return "err_register_exists: multi-sig wallet already exists", common.NewError("err_register_exists", "multi-sig wallet already exists")
	}

	err = ms.putWallet(w, balances)
	if err != nil {
		// I/O error.
		return "", err
	}

	return "success: multi-signature wallet registered", nil
}

func (ms MultiSigSmartContract) vote(currentTxnHash, signingClientID string, now common.Timestamp, inputData []byte, balances state.StateContextI) (string, error) {
	// Garbage collection of old proposals happens incrementally with every
	// incoming vote.
	err := ms.pruneExpirationQueue(now, balances)
	if err != nil {
		// I/O error.
		if err != util.ErrValueNotPresent && err != util.ErrNodeNotFound {
			return "", err
		} //else there are no expiration queue.
	}

	var v Vote

	err = json.Unmarshal(inputData, &v)
	if err != nil {
		return "", err
	}

	// Play nice.
	if !v.notTooBig() {
		return "", common.NewError("err_vote_too_big", "an input field exceeded allowable length")
	}
	if !v.hasValidAmount() {
		return "", common.NewError("err_vote_invalid_tokens", "invalid number of tokens to send")
	}
	if !v.hasSignature() {
		return "", common.NewError("err_vote_no_signature", " must sign vote")
	}

	// Every vote is associated with a proposal. If an appropriate proposal does
	// not exist yet, create one.
	p, err := ms.findOrCreateProposal(now, v, balances)
	if err != nil {
		// I/O error.
		return "", err
	}

	// Ensure all voters are on the same page.
	if !v.isCompatibleWithProposal(p) {
		return "", common.NewError("err_vote_not_compatible", " previous votes for same proposal differed")
	}

	// Check if the proposal was already finished, making this vote unnecessary.
	if p.ExecutedInTxnHash != "" {
		return "success 0: proposal previously executed in transaction hash " + p.ExecutedInTxnHash, nil
	}

	// Check that the multi-sig wallet is registered.
	w, err := ms.getWallet(v.Transfer.ClientID, balances)
	if err != nil {
		// I/O error.
		return "", err
	}
	if w.isEmpty() {
		return "", common.NewError("err_vote_wallet_not_registered", " wallet not registered")
	}

	// Check that the voter is registered on the wallet and that the signature
	// is valid.
	signerThresholdID := w.thresholdIdForSigner(signingClientID)
	if signerThresholdID == "" {
		return "", common.NewError("err_vote_auth", " authorization failure")
	}
	if !w.isVoteAuthorized(signingClientID, v) {
		return "", common.NewError("err_vote_auth", " authorization failure")
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
	err = ms.putProposal(&p, balances)
	if err != nil {
		// I/O error.
		return "", err
	}

	remaining--

	// If more votes are still needed we must wait for them. Nothing more to do.
	if remaining > 0 {
		msg := fmt.Sprintf("success %d: need %d more votes", remaining, remaining)
		return msg, nil
	}

	// Otherwise we can recover the threshold signature on the transfer and
	// execute it.
	thresholdSignature, err := w.constructTransferSignature(p)
	if err != nil {
		return "", common.NewError("err_vote_recover", " in signature recovery: "+err.Error())
	}

	p.ClientSignature = thresholdSignature

	// Request the transfer. The blockchain will validate the signature and
	// execute the transfer soon. If the signature is found to be invalid,
	// this vote transaction will fail.
	signedTransfer := w.makeSignedTransferForProposal(p)
	balances.AddSignedTransfer(&signedTransfer)

	// Save the proposal again.
	p.ExecutedInTxnHash = currentTxnHash

	err = ms.putProposal(&p, balances)
	if err != nil {
		// I/O error.
		return "", err
	}

	msg := "success 0: transfer executed with signature " + p.ClientSignature
	return msg, nil
}

// Prune the oldest proposal if it has expired.
func (ms MultiSigSmartContract) pruneExpirationQueue(now common.Timestamp, balances state.StateContextI) error {
	q, err := ms.getOrCreateExpirationQueue(balances)
	if err != nil {
		return err
	}

	// Reference to oldest proposal.
	ref := q.Head

	if ref == (proposalRef{}) {
		// No proposals currently exist.
		return nil
	}

	p, err := ms.getProposal(ref, balances)
	if err != nil {
		return err
	}

	if p.isExpired(now) {
		return ms.prune(ref, balances)
	}

	return nil
}

func (ms MultiSigSmartContract) prune(ref proposalRef, balances c_state.StateContextI) error {
	// Before we prune this proposal, fetch it one last time.
	p, err := ms.getProposal(ref, balances)
	if err != nil {
		return err
	}

	// Update expiry queue.
	q, err := ms.getOrCreateExpirationQueue(balances)
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
		err = ms.putExpirationQueue(&q, balances)
		if err != nil {
			return err
		}
	}

	// Update links.
	if p.Next != (proposalRef{}) {
		next, err := ms.getProposal(p.Next, balances)
		if err != nil {
			return err
		}

		next.Prev = p.Prev

		err = ms.putProposal(&next, balances)
		if err != nil {
			return err
		}
	}

	if p.Prev != (proposalRef{}) {
		prev, err := ms.getProposal(p.Prev, balances)
		if err != nil {
			return err
		}

		prev.Next = p.Next

		err = ms.putProposal(&prev, balances)
		if err != nil {
			return err
		}
	}

	// Now we can delete the pruned proposal.
	_, err = balances.DeleteTrieNode(p.getKey())
	if err != nil {
		return err
	}

	return nil
}

func (ms MultiSigSmartContract) findOrCreateProposal(now common.Timestamp, v Vote, balances state.StateContextI) (proposal, error) {
	// Start by trying to find an existing proposal.
	p, err := ms.getProposal(v.getProposalRef(), balances)
	if err != nil {
		return proposal{}, err
	}

	// Treat expired-but-not-yet-pruned proposals identically to pruned
	// proposals. Do this by pruning them now.
	if !p.isEmpty() && p.isExpired(now) {
		err = ms.prune(p.ref(), balances)
		if err != nil {
			return proposal{}, err
		}
		return proposal{}, common.NewError("proposal_expired", "proposal is expired")
	}

	// If it didn't exist or was expired, create it and update expiration queue.
	if p.isEmpty() {
		p, err = ms.createProposal(now, v, balances)
		if err != nil {
			return proposal{}, err
		}
	}

	return p, nil
}

// Create a proposal and add it to the expiration queue. Performs I/O.
func (ms MultiSigSmartContract) createProposal(now common.Timestamp, v Vote, balances state.StateContextI) (proposal, error) {
	q, err := ms.getOrCreateExpirationQueue(balances)
	if err != nil {
		if err != util.ErrValueNotPresent && err != util.ErrNodeNotFound {
			return proposal{}, err
		} //else we will create a proposal
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

	err = ms.putProposal(&p, balances)
	if err != nil {
		return proposal{}, err
	}

	// Update links.
	if q.Tail != (proposalRef{}) {
		prev, err := ms.getProposal(q.Tail, balances)
		if err != nil {
			return proposal{}, err
		}

		prev.Next = p.ref()

		err = ms.putProposal(&prev, balances)
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

	err = ms.putExpirationQueue(&q, balances)
	if err != nil {
		return proposal{}, err
	}

	return p, nil
}

func (ms MultiSigSmartContract) walletExists(clientID string, balances c_state.StateContextI) (bool, error) {
	w := &Wallet{}
	err := balances.GetTrieNode(getWalletKey(clientID), w)
	switch err {
	case nil:
		return true, nil
	case util.ErrValueNotPresent:
		return false, nil
	default:
		return false, err
	}
}

func (ms MultiSigSmartContract) getWallet(clientID string, balances c_state.StateContextI) (Wallet, error) {

	w := Wallet{}
	err := balances.GetTrieNode(getWalletKey(clientID), &w)

	if err != nil {
		// I/O error.
		return Wallet{}, err
	}

	// Okay.
	return w, nil
}

func (ms MultiSigSmartContract) putWallet(w Wallet, balances c_state.StateContextI) error {
	//walletBytes, err := json.Marshal(w)
	//if err != nil {
	//	return err
	//}
	_, err := balances.InsertTrieNode(w.getKey(), &w)
	return err
}

func (ms MultiSigSmartContract) getProposal(ref proposalRef, balances c_state.StateContextI) (proposal, error) {
	p := proposal{}
	err := balances.GetTrieNode(getProposalKey(ref.ClientID, ref.ProposalID), &p)
	switch err {
	case nil, util.ErrValueNotPresent:
		return p, nil
	default:
		return proposal{}, err
	}
}

func (ms MultiSigSmartContract) putProposal(p *proposal, balances c_state.StateContextI) error {
	//proposalBytes, err := json.Marshal(p)
	//if err != nil {
	//	return err
	//}

	_, err := balances.InsertTrieNode(p.getKey(), p)
	return err
}

func (ms MultiSigSmartContract) getOrCreateExpirationQueue(balances c_state.StateContextI) (expirationQueue, error) {
	q := expirationQueue{}
	err := balances.GetTrieNode(getExpirationQueueKey(), &q)
	switch err {
	case nil, util.ErrValueNotPresent:
		return q, nil
	default:
		return q, err
	}
}

func (ms MultiSigSmartContract) putExpirationQueue(q *expirationQueue, balances c_state.StateContextI) error {
	//proposalBytes, err := json.Marshal(p)
	//if err != nil {
	//	return err
	//}

	_, err := balances.InsertTrieNode(getExpirationQueueKey(), q)
	return err
}
