package storagesc

import (
	"fmt"

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
	var part *stakePoolPartition
	switch prr.ProviderType {
	case spenum.Blobber:
		part = blobberStakePoolPartitions
	case spenum.Validator:
		part = validatorStakePoolPartitions
	default:
		return "", common.NewErrorf("collect_reward_failed",
			"invalid provider type: %s", prr.ProviderType.String())
	}

	spKeys := make([]string, 0, len(providers))
	for _, p := range providers {
		spKeys = append(spKeys, stakePoolKey(prr.ProviderType, p))
	}

	if err := part.updateArray(balances, spKeys, func(sps []*stakePool) error {
		for i, sp := range sps {
			reward, err := sp.MintRewards(txn.ClientID, providers[i], prr.ProviderType, usp, balances)
			if err != nil {
				return fmt.Errorf("error emptying account, %v", err)
			}

			tm, err := currency.AddCoin(totalMinted, reward)
			if err != nil {
				return fmt.Errorf("error adding reward: %v", err)
			}

			if tm > conf.MaxMint {
				return fmt.Errorf("max min %v exceeded: %v", conf.MaxMint, conf.Minted)
			}

			totalMinted = tm

			staked, err := sp.stake()
			if err != nil {
				return fmt.Errorf("can't get stake: %v", err)
			}

			tag, data := event.NewUpdateBlobberTotalStakeEvent(providers[i], staked)
			balances.EmitEvent(event.TypeStats, tag, providers[i], data)

			switch prr.ProviderType {
			case spenum.Blobber:
				tag, data := event.NewUpdateBlobberTotalStakeEvent(providers[i], staked)
				balances.EmitEvent(event.TypeStats, tag, providers[i], data)
			case spenum.Validator:
				// TODO: implement validator stake update events
			}

			err = emitAddOrOverwriteReward(reward, providers[i], prr, balances, txn)
			if err != nil {
				return fmt.Errorf("emitting reward event: %v", err)
			}
		}

		return nil
	}); err != nil {
		return "", common.NewErrorf("collect_reward_failed", err.Error())
	}

	if err := usp.Save(prr.ProviderType, txn.ClientID, balances); err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"error saving user stake pool, %v", err)
	}

	if totalMinted-conf.Minted == 0 {
		return "", nil
	}

	conf.Minted = totalMinted

	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"cannot save config: %v", err)
	}

	return "", nil
}
