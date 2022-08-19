package minersc

import (
	"fmt"

	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/currency"
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

	usp, err := stakepool.GetUserStakePools(prr.ProviderType, txn.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"can't get related user stake pools: %v", err)
	}

	var providers []string
	if len(prr.ProviderId) == 0 {
		providers = usp.FindProvidersByType(prr.ProviderType)
	} else {
		providers = []string{prr.ProviderId}
	}

	var totalMinted currency.Coin
	for _, providerID := range providers {
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

		if providerID == txn.ClientID || provider.Settings.DelegateWallet == txn.ClientID {
			minted, err := provider.StakePool.MintRewards(
				txn.ClientID, providerID, prr.ProviderType, usp, balances)
			if err != nil {
				return "", common.NewErrorf("collect_reward_failed",
					"error emptying account, %v", err)
			}

			if err := provider.save(balances); err != nil {
				return "", common.NewErrorf("collect_reward_failed",
					"error saving stake pool, %v", err)
			}

			tm, err := currency.AddCoin(totalMinted, minted)
			if err != nil {
				return "", common.NewErrorf("collect_reward_failed", "error adding total minted token: %v", err)
			}

			totalMinted = tm
		}
	}

	if totalMinted == 0 {
		return "", common.NewErrorf("collect_reward_failed",
			"user %v does not own stake pool of type %s", txn.ClientID, prr.ProviderType)
	}

	gnMinted, err := currency.AddCoin(gn.Minted, totalMinted)
	if err != nil {
		return "", common.NewErrorf("collect_reward_failed",
			"error adding minted to global node, %v", err)
	}
	gn.Minted = gnMinted
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
