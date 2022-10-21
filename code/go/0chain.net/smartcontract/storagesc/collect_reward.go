package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
)

// collectReward mints tokens for delegate rewards.
// The minted tokens are transferred the user's wallet.
func (ssc *StorageSmartContract) collectReward(
	txn *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	var prr stakepool.CollectRewardRequest
	if err := prr.Decode(input); err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"can't decode request: %v", err)
	}
	if prr.ProviderType != spenum.Blobber && prr.ProviderType != spenum.Validator {
		return "", common.NewErrorf("collect_reward_failed",
			"invalid provider type: %s", prr.ProviderType.String())
	}

	usp, err := stakepool.GetUserStakePools(prr.ProviderType, txn.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"can't get related user stake pools: %v", err)
	}

	var providers []string
	if len(prr.ProviderId) == 0 {
		providers = usp.Providers
	} else {
		providers = []string{prr.ProviderId}
	}

	conf, err := ssc.getConfig(balances, true)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"can't get config: %v", err)
	}

	totalMinted := conf.Minted
	for _, providerID := range providers {
		sp, err := ssc.getStakePool(prr.ProviderType, providerID, balances)
		if err != nil {
			return "", common.NewErrorf("collect_reward_failed",
				"can't get related stake pool: %v", err)
		}

		reward, err := sp.MintRewards(txn.ClientID, providerID, prr.ProviderType, usp, balances)
		if err != nil {
			return "", common.NewErrorf("collect_reward_failed",
				"error emptying account, %v", err)
		}

		tm, err := currency.AddCoin(totalMinted, reward)
		if err != nil {
			return "", common.NewErrorf("collect_reward_failed", "error adding reward: %v", err)
		}

		if tm > conf.MaxMint {
			return "", common.NewErrorf("collect_reward_failed",
				"max min %v exceeded: %v", conf.MaxMint, conf.Minted)
		}

		totalMinted = tm

		if err := sp.save(prr.ProviderType, providerID, balances); err != nil {
			return "", common.NewErrorf("collect_reward_failed",
				"error saving stake pool, %v", err)
		}

		staked, err := sp.stake()
		if err != nil {
			return "", common.NewErrorf("collect_reward_failed",
				"can't get stake: %v", err)
		}

		tag, data := event.NewUpdateBlobberTotalStakeEvent(providerID, staked)
		balances.EmitEvent(event.TypeStats, tag, providerID, data)

		switch prr.ProviderType {
		case spenum.Blobber:
			tag, data := event.NewUpdateBlobberTotalStakeEvent(providerID, staked)
			balances.EmitEvent(event.TypeStats, tag, providerID, data)
		case spenum.Validator:
			// TODO: implement validator stake update events
		}

		err = emitAddOrOverwriteReward(reward, providerID, prr, balances, txn)
		if err != nil {
			return "", common.NewErrorf("pay_reward_failed",
				"emitting reward event: %v", err)
		}
	}

	if err := usp.Save(prr.ProviderType, txn.ClientID, balances); err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"error saving user stake pool, %v", err)
	}

	if totalMinted-conf.Minted == 0 {
		return "", nil
	}

	conf.Minted = totalMinted

	_, err = balances.InsertTrieNode(scConfigKey(ADDRESS), conf)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"cannot save config: %v", err)
	}

	return "", nil
}
