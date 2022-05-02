package multisigsc

import (
	"encoding/hex"
	"encoding/json"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

//msgp:ignore Vote
//go:generate msgp -io=false -tests=false -unexported -v

const (
	ExpirationTime = 60 * 60 * 24 * 7 // Proposals expire after one week.
	//ExpirationTime = 30 // Value in seconds that is more appropriate for testing.
	MaxSigners   = 20
	MinSigners   = 2
	MaxFieldSize = 256
)

type Wallet struct {
	ClientID        string `json:"client_id"`
	SignatureScheme string `json:"signature_scheme"`
	PublicKey       string `json:"public_key"`

	SignerThresholdIDs []string `json:"signer_threshold_ids"`
	SignerPublicKeys   []string `json:"signer_public_keys"`

	NumRequired int `json:"num_required"`
}

func (w Wallet) Encode() []byte {
	buff, _ := json.Marshal(w)
	return buff
}

func (w Wallet) Decode(input []byte) error {
	err := json.Unmarshal(input, &w)
	return err
}

func (w Wallet) isEmpty() bool {
	return w.ClientID == ""
}

func (w Wallet) getKey() datastore.Key {
	return getWalletKey(w.ClientID)
}

func getWalletKey(clientID string) datastore.Key {
	return datastore.Key(Address + clientID)
}

func (w Wallet) valid(forClientID string) (bool, error) {
	if w.ClientID != forClientID {
		return false, common.NewError("client_id_doesnot_match", "Multisig Wallet client ID is different than the requesting client ID")
	}

	if !isPublicKeyForClientID(w.PublicKey, w.ClientID) {
		return false, common.NewError("client_id_public_key_no_match", "the client id and the public key in the wallet do not match")
	}

	numIds := len(w.SignerThresholdIDs)
	numKeys := len(w.SignerPublicKeys)

	if numIds != numKeys {
		return false, common.NewError("signers_id_and_signer_public_key_no_match", "number of signer client ids and the the signer public keys do not match")
	}
	if numIds > MaxSigners {
		return false, common.NewError("num_ids_too-many", "number of signer client ids is more than the maximum number of signers")
	}

	if w.NumRequired < MinSigners {
		return false, common.NewError("signers_required_too_less", "number of signers required is less than 2")
	}
	if w.NumRequired > numIds {
		return false, common.NewError("too_many_signers_required", "number of signers required is less than 2")
	}

	if hasDuplicates(w.SignerThresholdIDs) {
		return false, common.NewError("duplicate_signer_ids", "duplicate threshold ids present")
	}
	if hasDuplicates(w.SignerPublicKeys) {
		return false, common.NewError("duplicate_signers", "duplicate signers are present")
	}

	if !encryption.IsValidSignatureScheme(w.SignatureScheme) ||
		!encryption.IsValidThresholdSignatureScheme(w.SignatureScheme) ||
		!encryption.IsValidReconstructSignatureScheme(w.SignatureScheme) {
		return false, common.NewError("signature_scheme_not_supported", "signature scheme of the wallet does not support multisig")
	}

	for _, key := range w.SignerPublicKeys {
		scheme := encryption.GetSignatureScheme(w.SignatureScheme)
		err := scheme.SetPublicKey(key)
		if err != nil {
			return false, err
		}
	}

	if len(w.ClientID) > MaxFieldSize ||
		len(w.SignatureScheme) > MaxFieldSize ||
		len(w.PublicKey) > MaxFieldSize {
		return false, common.NewError("too_many_signers", "Wallet has more than 256 ClientIDs or signature schemes or public keys")
	}
	for _, id := range w.SignerThresholdIDs {
		if len(id) > MaxFieldSize {
			return false, common.NewError("too_many_threshold_ids", "wallet has more than 256 threshold id fields")
		}
	}
	for _, key := range w.SignerPublicKeys {
		if len(key) > MaxFieldSize {
			return false, common.NewError("too_many_signer_keys", "wallet has more than 256 signer key fields")
		}
	}

	return true, nil
}

func isPublicKeyForClientID(publicKey, clientID string) bool {
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		return false
	}

	if encryption.Hash(publicKeyBytes) != clientID {
		return false
	}

	return true
}

func hasDuplicates(ss []string) bool {
	exists := make(map[string]bool, len(ss))
	for _, s := range ss {
		if exists[s] {
			return true
		}
		exists[s] = true
	}
	return false
}

func (w Wallet) isVoteAuthorized(signingClientID string, v Vote) bool {
	publicKey := w.publicKeyForSigner(signingClientID)
	if publicKey == "" {
		// Not a registered signer for this wallet.
		return false
	}

	err := w.makeSignedTransferForVote(publicKey, v).VerifySignature(false)
	return err == nil
}

func (w Wallet) makeSignedTransferForVote(signingPublicKey string, v Vote) state.SignedTransfer {
	return state.SignedTransfer{
		Transfer:   v.Transfer,
		SchemeName: w.SignatureScheme,
		PublicKey:  signingPublicKey,
		Sig:        v.Signature,
	}
}

func (w Wallet) makeSignedTransferForProposal(p proposal) state.SignedTransfer {
	return state.SignedTransfer{
		Transfer:   p.Transfer,
		SchemeName: w.SignatureScheme,
		PublicKey:  w.PublicKey,
		Sig:        p.ClientSignature,
	}
}

func (w Wallet) thresholdIdForSigner(signingClientID string) string {
	for i, key := range w.SignerPublicKeys {
		b, err := hex.DecodeString(key)
		if err != nil {
			// Unfortunate.
			continue
		}
		clientID := encryption.Hash(b)
		if clientID == signingClientID {
			return w.SignerThresholdIDs[i]
		}
	}
	return ""
}

func (w Wallet) publicKeyForSigner(signingClientID string) string {
	for _, key := range w.SignerPublicKeys {
		b, err := hex.DecodeString(key)
		if err != nil {
			// Unfortunate.
			continue
		}
		clientID := encryption.Hash(b)
		if clientID == signingClientID {
			return key
		}
	}
	return ""
}

func (w Wallet) publicKeyForThresholdID(signerThresholdID string) string {
	for i, id := range w.SignerThresholdIDs {
		if id == signerThresholdID {
			return w.SignerPublicKeys[i]
		}
	}
	return ""
}

// Compute the Lagrange polynomial of Wallet.NumRequired signature shares. The
// y-intercept of this polynomial is the proposal's signature. (This process is
// called reconstruction in the literature.)
func (w Wallet) constructTransferSignature(p proposal) (string, error) {
	t := w.NumRequired
	n := len(w.SignerThresholdIDs)
	rec := encryption.GetReconstructSignatureScheme(w.SignatureScheme, t, n)

	for i, id := range p.SignerThresholdIDs {
		publicKey := w.publicKeyForThresholdID(id)
		if publicKey == "" {
			// Logic error?
			return "", common.NewError("wallet_sc_signature_reconstruction", "couldn't find public key for id")
		}

		tss := encryption.GetThresholdSignatureScheme(w.SignatureScheme)

		err := tss.SetPublicKey(publicKey)
		if err != nil {
			return "", err
		}

		err = tss.SetID(id)
		if err != nil {
			return "", err
		}

		sig := p.SignerSignatures[i]

		err = rec.Add(tss, sig)
		if err != nil {
			return "", err
		}
	}

	// All of the SignerSignatures are signatures on the transfer, which means
	// this reconstructed signature will be, too.
	return rec.Reconstruct()
}

type Vote struct {
	ProposalID string `json:"proposal_id"`

	// Client ID in transfer is that of the multi-sig wallet, not the signer.
	Transfer state.Transfer `json:"transfer"`

	Signature string `json:"signature"`
}

func (v Vote) notTooBig() bool {
	return len(v.ProposalID) <= MaxFieldSize &&
		len(v.Transfer.ClientID) <= MaxFieldSize &&
		len(v.Transfer.ToClientID) <= MaxFieldSize &&
		len(v.Signature) <= MaxFieldSize
}

func (v Vote) hasValidAmount() bool {
	return v.Transfer.Amount > 0
}

func (v Vote) hasSignature() bool {
	return v.Signature != ""
}

func (v Vote) getProposalRef() proposalRef {
	return proposalRef{
		ClientID:   v.Transfer.ClientID,
		ProposalID: v.ProposalID,
	}
}

func (v Vote) isCompatibleWithProposal(p proposal) bool {
	return v.Transfer == p.Transfer
}

// Uniquely identifies a proposal. Can be used to refer to one.
type proposalRef struct {
	ClientID   string `json:"client_id"`
	ProposalID string `json:"proposal_id"`
}

func (pr *proposalRef) Encode() []byte {
	buff, _ := json.Marshal(pr)
	return buff
}

func (pr *proposalRef) Decode(input []byte) error {
	err := json.Unmarshal(input, pr)
	return err
}

// Proposal to transfer tokens out of the multi-sig wallet. Built up from T
// different votes.
type proposal struct {
	// Proposal ID is unique only within a single multi-sig wallet. Globally, a
	// proposal may be referred to by a wallet ID / proposal ID pair.
	ProposalID     string           `json:"proposal_id"`
	ExpirationDate common.Timestamp `json:"expiration_date"`

	// Intrusive queue sorted by expiration date for garbage collection.
	Next proposalRef `json:"next"`
	Prev proposalRef `json:"prev"`

	Transfer state.Transfer `json:"transfer"`

	// Pertinent data from votes.
	SignerThresholdIDs []string `json:"signer_threshold_ids"`
	SignerSignatures   []string `json:"signer_signatures"`

	// Filled upon completing a proposal.
	ClientSignature   string `json:"client_signature"`
	ExecutedInTxnHash string `json:"executed_in_txn_hash"`
}

func (p *proposal) Encode() []byte {
	buff, _ := json.Marshal(p)
	return buff
}

func (p *proposal) Decode(input []byte) error {
	err := json.Unmarshal(input, p)
	return err
}

func (p proposal) isEmpty() bool {
	return p.Transfer.ClientID == ""
}

func (p proposal) isExpired(now common.Timestamp) bool {
	return now >= p.ExpirationDate
}

func (p proposal) ref() proposalRef {
	return proposalRef{
		ClientID:   p.Transfer.ClientID,
		ProposalID: p.ProposalID,
	}
}

func (p proposal) getKey() datastore.Key {
	return getProposalKey(p.Transfer.ClientID, p.ProposalID)
}

func getProposalKey(clientID, proposalID string) datastore.Key {
	return datastore.Key(Address + clientID + encryption.Hash(proposalID))
}

// Queue of all proposals across all wallets sorted by expiration date.
type expirationQueue struct {
	Head proposalRef `json:"head"`
	Tail proposalRef `json:"tail"`
}

func (q *expirationQueue) Encode() []byte {
	buff, _ := json.Marshal(q)
	return buff
}

func (q *expirationQueue) Decode(input []byte) error {
	err := json.Unmarshal(input, &q)
	return err
}

func getExpirationQueueKey() datastore.Key {
	return datastore.Key(Address + encryption.Hash("queue"))
}
