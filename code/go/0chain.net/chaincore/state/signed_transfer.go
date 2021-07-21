package state

import (
	"encoding/hex"

	"0chain.net/core/encryption"
	"github.com/0chain/gosdk/core/common/errors"
)

// A SignedTransfer is a balance transfer from one client to another that has
// been authorized with a signature by the sending client.
type SignedTransfer struct {
	Transfer
	SchemeName string
	PublicKey string
	Sig string
}

func (st *SignedTransfer) Sign(sigScheme encryption.SignatureScheme) error {
	hash := st.computeTransferHash()

	sig, err := sigScheme.Sign(hash)
	if err != nil {
		return err
	}

	st.Sig = sig

	return nil
}

// Verify that the signature on the transfer is correct.
func (st SignedTransfer) VerifySignature(requireSendersSignature bool) error {
	if !encryption.IsValidSignatureScheme(st.SchemeName) {
		return errors.New("invalid_signature_scheme", "invalid signature scheme")
	}

	if requireSendersSignature {
		err := st.verifyPublicKey()
		if err != nil {
			return err
		}
	}

	sigScheme := encryption.GetSignatureScheme(st.SchemeName)

	err := sigScheme.SetPublicKey(st.PublicKey)
	if err != nil {
		return errors.New("invalid_public_key", "invalid public key")
	}

	hash := st.computeTransferHash()

	correctSignature, err := sigScheme.Verify(st.Sig, hash)
	if err != nil {
		return err
	}
	if !correctSignature {
		return errors.New("invalid_transfer_signature", "Invalid signature on transfer")
	}

	return nil
}

func (st SignedTransfer) verifyPublicKey() error {
	publicKeyBytes, err := hex.DecodeString(st.PublicKey)
	if err != nil {
		return errors.New("invalid_public_key", "invalid public key format")
	}

	if encryption.Hash(publicKeyBytes) != st.Transfer.ClientID {
		return errors.New("wrong_public_key", "public key does not match client id")
	}

	return nil
}

func (st SignedTransfer) computeTransferHash() string {
	return encryption.Hash(st.Transfer.Encode())
}
