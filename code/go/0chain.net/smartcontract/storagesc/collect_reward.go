package storagesc

import (
	"strconv"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"

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
	var req stakepool.CollectRewardRequest
	minted, err := stakepool.CollectReward(
		input, func(
			crr stakepool.CollectRewardRequest, balances cstate.StateContextI,
		) (currency.Coin, error) {
			req = crr
			sp, err := ssc.getStakePool(crr.ProviderType, crr.ProviderId, balances)
			if err != nil {
				return 0, err
			}

			// TODO: do for other provider types in storagesc
			if crr.ProviderType == spenum.Blobber {
				bil, err := getBlobbersInfoList(balances)
				if err != nil {
					return 0, err
				}

				b, err := getBlobber(crr.ProviderId, balances)
				if err != nil {
					return 0, err
				}

				if err := sp.DistributeRewards(
					bil[b.Index].Rewards,
					crr.ProviderId,
					crr.ProviderType,
					spenum.CancellationChargeReward, // TODO: use correct reward type
					balances); err != nil {
					return 0, err
				}

				bil[b.Index].Rewards = 0
				if err := bil.Save(balances); err != nil {
					return 0, err
				}
			}

			minted, err := sp.MintRewards(
				txn.ClientID, crr.ProviderId, crr.ProviderType, balances)
			if err != nil {
				return 0, err
			}

			if err := sp.Save(crr.ProviderType, crr.ProviderId, balances); err != nil {
				return 0, err
			}

			//err = sp.EmitStakeEvent(crr.ProviderType, crr.ProviderId, balances)
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

	return toJson(&event.RewardMint{
		Amount:       int64(minted),
		BlockNumber:  balances.GetBlock().Round,
		ClientID:     txn.ClientID,
		ProviderType: strconv.Itoa(int(req.ProviderType)),
		ProviderID:   req.ProviderId,
	}), err
}
