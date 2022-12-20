package common

import (
	"encoding/hex"

	"0chain.net/core/encryption"
)

/*GetWalletIdFromPublicKey - given the PK of the provider, return the its operational walletID i.e. the key used to sign the txns */
func GetWalletIdFromPublicKey(pk string) (string, error) {
	publicKeyBytes, err := hex.DecodeString(pk)
	if err != nil {
		return "", err
	}
	operationalClientID := encryption.Hash(publicKeyBytes)
	return operationalClientID, nil
}
