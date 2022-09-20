package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
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
				"id %v can't get related stake pool: %v", providerID, err)
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

		if err := sp.save(spenum.Blobber, providerID, balances); err != nil {
			return "", common.NewErrorf("collect_reward_failed",
				"error saving stake pool, %v", err)
		}

		//TODO sort out this code, we cant simply update here for validator and for blobber at the same time, also we need write price to calculate staked capacity change
	//staked, err := sp.stake()
		//if err != nil {
		//	return "", common.NewErrorf("collect_reward_failed",
		//		"can't get stake: %v", err)
		//}
	//
		//data := dbs.DbUpdates{
		//	Id: providerID,
		//	Updates: map[string]interface{}{
		//		"total_stake": int64(staked),
		//	},
		//}
		//balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, providerID, data)
	//balances.EmitEvent(event.TypeSmartContract, event.TagAllocBlobberValueChange, providerID, event.AllocationBlobberValueChanged{
	//	FieldType:    event.Staked,
	//	AllocationId: "",
	//	BlobberId:    providerID,
	//	Delta:        int64((sp.stake() - before) ),
	//})

		err = emitAddOrOverwriteReward(reward, providerID, prr, balances, txn)
		if err != nil {
			return "", common.NewErrorf("pay_reward_failed",
				"emitting reward event: %v", err)
		}
	}

	if err := usp.Save(spenum.Blobber, txn.ClientID, balances); err != nil {
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
