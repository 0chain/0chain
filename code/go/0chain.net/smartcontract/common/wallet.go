package common

import (
	"encoding/hex"

	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
)


func ValidateWallet(publicKey, delegateWalletID string) *common.Error {
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		return common.NewError("add_sharder",
			"couldn't decode publick key to compare to delegate wallet")
	}
	operationalClientID := encryption.Hash(publicKeyBytes)

	if operationalClientID == delegateWalletID {
		logging.Logger.Error("couldn't use the same wallet as both operational and delegate")
		return common.NewError("add_sharder",
			"couldn't use the same wallet as both operational and delegate")
	}

	return nil
}