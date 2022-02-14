package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
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

	var prr stakepool.CollectRewardRequest
	if err := prr.Decode(input); err != nil {
		return "", common.NewErrorf("pay_reward_failed",
			"can't decode request: %v", err)
	}

	usp, err := stakepool.GetUserStakePool(prr.ProviderType, txn.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("pay_reward_failed",
			"can't get related user stake pools: %v", err)
	}

	providerId := usp.Find(prr.PoolId)
	if len(providerId) == 0 {
		return "", common.NewErrorf("pay_reward_failed",
			"user %v does not own stake pool %v", txn.ClientID, prr.PoolId)
	}

	sp, err := ssc.getStakePool(providerId, balances)
	if err != nil {
		return "", common.NewErrorf("pay_reward_failed",
			"can't get related stake pool: %v", err)
	}

	_, err = sp.MintRewards(
		txn.ClientID, prr.PoolId, providerId, prr.ProviderType, usp, balances)
	if err != nil {
		return "", common.NewErrorf("pay_reward_failed",
			"error emptying account, %v", err)
	}

	if err := usp.Save(stakepool.Blobber, txn.ClientID, balances); err != nil {
		return "", common.NewErrorf("pay_reward_failed",
			"error saving user stake pool, %v", err)
	}

	if err := sp.save(ssc.ID, providerId, balances); err != nil {
		return "", common.NewErrorf("pay_reward_failed",
			"error saving stake pool, %v", err)
	}
	return "", nil
}
