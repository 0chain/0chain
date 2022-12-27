package common

import (
	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
)

//ValidateDelegateWallet - Protects against using the provider's clientID (operational wallet ID) as DelegateWalletID. Checks that clientID and delegateWalletID are not the same
func ValidateDelegateWallet(publicKey, delegateWalletID string) *common.Error {
	if config.Development() {
		return nil
	}

	operationalWalletID, err := encryption.GetClientIDFromPublicKey(publicKey)
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
