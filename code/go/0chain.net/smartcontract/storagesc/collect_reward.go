package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool"
	"github.com/0chain/common/core/currency"
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
		return "", common.NewError("collect_reward_failed", "can't get config: "+err.Error())
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

			if err := sp.Save(crr.ProviderType, crr.ProviderId, balances); err != nil {
				return 0, err
			}

			err = sp.EmitStakeEvent(crr.ProviderType, crr.ProviderId, balances)
			if err != nil {
				return 0, err
			}

			return minted, nil
		},
		balances,
	)
	if err != nil {
		return "", common.NewError("collect_reward_failed", err.Error())
	}

	if err := conf.saveMints(minted, balances); err != nil {
		return "", common.NewError("collect_reward_failed", "can't Save config: "+err.Error())
	}
	return "", err
}
