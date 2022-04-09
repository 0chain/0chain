package storagesc

import (
	"encoding/json"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
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

	var err error
	var usp *stakepool.UserStakePools
	var providerID = prr.ProviderId
	if len(prr.PoolId) > 0 {
		usp, err = stakepool.GetUserStakePool(prr.ProviderType, txn.ClientID, balances)
		if err != nil {
			return "", common.NewErrorf("collect_reward_failed",
				"can't get related user stake pools: %v", err)
		}

		if len(prr.ProviderId) == 0 {
			providerID = usp.Find(prr.PoolId)
		}
	}

	if len(providerID) == 0 {
		return "", common.NewErrorf("collect_reward_failed",
			"user %v does not own stake pool %v", txn.ClientID, prr.PoolId)
	}

	sp, err := ssc.getStakePool(providerID, balances)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"can't get related stake pool: %v", err)
	}

	minted, err := sp.MintRewards(
		txn.ClientID, prr.PoolId, providerID, prr.ProviderType, usp, balances)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"error emptying account, %v", err)
	}

	if err := usp.Save(spenum.Blobber, txn.ClientID, balances); err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"error saving user stake pool, %v", err)
	}

	if err := sp.save(ssc.ID, providerID, balances); err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"error saving stake pool, %v", err)
	}
	if minted == 0 {
		return "", nil
	}

	conf, err := ssc.getConfig(balances, true)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"can't get config: %v", err)
	}
	conf.Minted += minted
	if conf.Minted > conf.MaxMint {
		return "", common.NewErrorf("collect_reward_failed",
			"max min %v exceeded: %v", conf.MaxMint, conf.Minted)
	}
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"cannot save config: %v", err)
	}

	data, _ := json.Marshal(dbs.DbUpdates{
		Id: providerID,
		Updates: map[string]interface{}{
			"total_stake": int64(sp.stake()),
		},
	})
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, providerID, string(data))

	return "", nil
}
