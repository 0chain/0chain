package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool"
)

// collectReward mints tokens for delegate rewards.
// The minted tokens are transferred the user's wallet.
func (ssc *StorageSmartContract) collectReward(
	txn *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	conf, err := getConfig(balances)
	if err != nil {
		return "", common.NewError("collect reward", "can't get config: "+err.Error())
	}
	minted, err := stakepool.CollectReward(
		input, func(
			crr stakepool.CollectRewardRequest, balances cstate.StateContextI,
		) (currency.Coin, error) {
			sp, err := ssc.getStakePool(crr.ProviderType, crr.ProviderId, balances)
			if err != nil {
				return 0, err
			}

			minted, err := sp.MintRewards(
				txn.ClientID, crr.ProviderId, crr.ProviderType, balances)
			if err != nil {
				return 0, err
			}

			if err := sp.save(crr.ProviderType, crr.ProviderId, balances); err != nil {
				return 0, err
			}

			err = sp.stakeForProvider(crr.ProviderType, crr.ProviderId, balances)
			if err != nil {
				return 0, err
			}

			return minted, nil
		},
		balances,
	)
	if err != nil {
		return "", common.NewError("collect reward", err.Error())
	}

	if err := conf.saveMints(minted, balances); err != nil {
		return "", common.NewError("collect reward", "can't save config: "+err.Error())
	}
	return "", err
}
