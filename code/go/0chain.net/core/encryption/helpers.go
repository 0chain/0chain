package encryption

import "encoding/hex"

// GetClientIDFromPublickKey - given the PK of the provider, return the its operational walletID i.e. the key used to sign the txns
func GetClientIDFromPublickKey(pk string) (string, error) {
	publicKeyBytes, err := hex.DecodeString(pk)
	if err != nil {
		return "", err
	}
	operationalClientID := Hash(publicKeyBytes)
	return operationalClientID, nil
}