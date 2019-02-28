package state

import (
	"context"

	"0chain.net/chaincore/client"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
)

// A SignedTransfer is a balance transfer from one client to another that has
// been authorized with a signature by the sending client.
type SignedTransfer struct {
	Transfer
	sig string
}

func NewSignedTransfer(transfer Transfer, sig string) *SignedTransfer {
	return &SignedTransfer{Transfer: transfer, sig: sig}
}

// Verify that the signature on the transfer is correct. This is done by
// checking the signature against the senders public key.
func (st SignedTransfer) VerifySignature(ctx context.Context) error {
	sigScheme, err := st.getSignatureScheme(ctx)
	if err != nil {
		return err
	}

	return st.VerifySignatureWithScheme(sigScheme)
}

// Verify that the signature on the transfer is correct. May be done against an
// arbitrary public key.
func (st SignedTransfer) VerifySignatureWithScheme(sigScheme encryption.SignatureScheme) error {
	hash := st.computeTransferHash()

	correctSignature, err := sigScheme.Verify(st.sig, hash)
	if err != nil {
		return err
	}
	if !correctSignature {
		return common.NewError("invalid_transfer_signature", "Invalid signature on transfer")
	}

	return nil
}

func (st SignedTransfer) computeTransferHash() string {
	return encryption.Hash(st.Transfer.Encode())
}

func (st SignedTransfer) getSignatureScheme(ctx context.Context) (encryption.SignatureScheme, error) {
	co, err := client.GetClient(ctx, st.Transfer.ClientID)
	if err != nil {
		return nil, err
	}

	return co.GetSignatureScheme(), nil
}
