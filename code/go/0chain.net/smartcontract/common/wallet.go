package common

import (
	"encoding/hex"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
)


func ValidateWallet(publicKey, delegateWalletID string) *common.Error {
	if config.Development() {
		return nil
	}

	operationalWalletID, err := GetWalletIdFromPublicKey(publicKey)
	if err != nil {
		return common.NewError("add_sharder",
			"couldn't decode publick key to compare to delegate wallet")
	}

	if operationalWalletID == delegateWalletID {
		logging.Logger.Error("couldn't use the same wallet as both operational and delegate")
		return common.NewError("add_sharder",
			"couldn't use the same wallet as both operational and delegate")
	}

	return nil
}

/*GetWalletIdFromPublicKey - given the PK of the provider, return the its operational walletID i.e. the key used to sign the txns */
func GetWalletIdFromPublicKey(pk string) (string, error) {
	publicKeyBytes, err := hex.DecodeString(pk)
	if err != nil {
		return "", err
	}
	operationalClientID := encryption.Hash(publicKeyBytes)
	return operationalClientID, nil
}