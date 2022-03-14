package minersc

import (
	"fmt"

	"0chain.net/core/logging"
	"0chain.net/smartcontract/stakepool/spenum"
	"go.uber.org/zap"

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
	_ *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	var prr stakepool.CollectRewardRequest
	if err := prr.Decode(input); err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"can't decode request: %v", err)
	}
	logging.Logger.Info("piers miner sc collectReward",
		zap.Any("input", prr),
	)
	if prr.ProviderType != spenum.Miner && prr.ProviderType != spenum.Sharder {
		return "", common.NewErrorf("collect_reward_failed",
			"invalid provider type: %s", prr.ProviderType.String())
	}

	usp, err := stakepool.GetUserStakePool(prr.ProviderType, txn.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"can't get related user stake pools: %v", err)
	}

	var providerID = prr.ProviderId
	if len(prr.ProviderId) == 0 {
		providerID = usp.Find(prr.PoolId)
	}
	if len(providerID) == 0 {
		return "", common.NewErrorf("collect_reward_failed",
			"user %v does not own stake pool %v", txn.ClientID, prr.PoolId)
	}

	var provider *MinerNode
	switch prr.ProviderType {
	case spenum.Miner:
		provider, err = getMinerNode(providerID, balances)
	case spenum.Sharder:
		provider, err = ssc.getSharderNode(providerID, balances)
	default:
		err = fmt.Errorf("unsupported provider type %s", prr.ProviderType.String())
	}
	if err != nil {
		return "", common.NewError("collect_reward_failed", err.Error())
	}

	_, err = provider.StakePool.MintRewards(
		txn.ClientID, prr.PoolId, providerID, prr.ProviderType, usp, balances)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"error emptying account, %v", err)
	}

	if err := provider.save(balances); err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"error saving stake pool, %v", err)
	}
	return "", nil
}
