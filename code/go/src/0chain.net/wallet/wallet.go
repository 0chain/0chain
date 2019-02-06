package wallet

import (
	"encoding/hex"

	"0chain.net/encryption"
)

/*Wallet - a struct representing the client's wallet */
type Wallet struct {
	SignatureScheme encryption.SignatureScheme
	PublicKeyBytes  []byte
	ClientID        string
	Balance         int64
}

/*Initialize - initialize a wallet with public/private keys */
func (w *Wallet) Initialize(clientSignatureScheme string) error {
	var sigScheme encryption.SignatureScheme = encryption.GetSignatureScheme(clientSignatureScheme)
	err := sigScheme.GenerateKeys()
	if err != nil {
		return err
	}
	return w.SetSignatureScheme(sigScheme)
}

/*SetSignatureScheme - sets the keys for the wallet */
func (w *Wallet) SetSignatureScheme(signatureScheme encryption.SignatureScheme) error {
	w.SignatureScheme = signatureScheme
	publicKeyBytes, err := hex.DecodeString(signatureScheme.GetPublicKey())
	if err != nil {
		return err
	}
	w.ClientID = encryption.Hash(publicKeyBytes)
	return nil
}
