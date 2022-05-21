package minersc

import (
	"fmt"

	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool"
)

// collectReward mints tokens for miner or sharder delegate rewards.
// The minted tokens are transferred to the user's wallet.
func (ssc *MinerSmartContract) collectReward(
	txn *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	var prr stakepool.CollectRewardRequest
	if err := prr.Decode(input); err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"can't decode request: %v", err)
	}
	if prr.ProviderType != spenum.Miner && prr.ProviderType != spenum.Sharder {
		return "", common.NewErrorf("collect_reward_failed",
			"invalid provider type: %s", prr.ProviderType.String())
	}

	var err error
	var usp *stakepool.UserStakePools
	var providerID = prr.ProviderId
	if len(prr.PoolId) > 0 {
		usp, err = stakepool.GetUserStakePools(prr.ProviderType, txn.ClientID, balances)
		if err != nil {
			return "", common.NewErrorf("collect_reward_failed",
				"can't get related user stake pools: %v", err)
		}

		if len(prr.ProviderId) == 0 {
			providerID = usp.FindProvider(prr.PoolId)
		}
	}

	var provider *MinerNode
	switch prr.ProviderType {
	case spenum.Miner:
		provider, err = GetMinerNode(providerID, balances)
	case spenum.Sharder:
		provider, err = ssc.getSharderNode(providerID, balances)
	default:
		err = fmt.Errorf("unsupported provider type %s", prr.ProviderType.String())
	}
	if err != nil {
		return "", common.NewError("collect_reward_failed", err.Error())
	}

	if providerID != txn.ClientID && provider.Settings.DelegateWallet != txn.ClientID {
		return "", common.NewErrorf("collect_reward_failed",
			"user %v does not own stake pool %v", txn.ClientID, prr.PoolId)
	}

	minted, err := provider.StakePool.MintRewards(
		txn.ClientID, prr.PoolId, providerID, prr.ProviderType, usp, balances)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"error emptying account, %v", err)
	}

	if err := provider.save(balances); err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"error saving stake pool, %v", err)
	}

	gn.Minted += minted
	if !gn.canMint() {
		return "", common.NewErrorf("collect_reward_failed",
			"max mint %v exceeded, %v", gn.MaxMint, gn.Minted)
	}
	if err = gn.save(balances); err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"saving global node: %v", err)
	}

	return "", nil
}
