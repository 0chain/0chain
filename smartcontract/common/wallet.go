package common

import (
	"errors"

	"0chain.net/chaincore/config"
	"0chain.net/core/encryption"
)

// ValidateDelegateWallet - Protects against using the provider's clientID (operational wallet ID) as DelegateWalletID. Checks that clientID and delegateWalletID are not the same
func ValidateDelegateWallet(publicKey, delegateWalletID string) error {
	if config.Development() {
		return nil
	}

	operationalWalletID, err := encryption.GetClientIDFromPublicKey(publicKey)
	if err != nil {
		return errors.New("could not decode public key to compare to delegate wallet")
	}

	if operationalWalletID == delegateWalletID {
		return errors.New("could not use the same wallet as both operational and delegate")
	}

	return nil
}
