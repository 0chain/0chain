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
	minted, err := stakepool.CollectReward(
		input,
		func(
			crr stakepool.CollectRewardRequest, balances cstate.StateContextI,
		) (currency.Coin, error) {
			var provider *MinerNode
			var err error
			switch crr.ProviderType {
			case spenum.Miner:
				provider, err = getMinerNode(crr.ProviderId, balances)
			case spenum.Sharder:
				provider, err = getSharderNode(crr.ProviderId, balances)
			default:
				err = fmt.Errorf("unsupported provider type %s", crr.ProviderType)
			}
			if err != nil {
				return 0, err
			}

			minted, err := provider.StakePool.MintRewards(
				txn.ClientID, crr.ProviderId, crr.ProviderType, balances)
			if err != nil {
				return 0, err
			}

			if err := provider.save(balances); err != nil {
				return 0, err
			}

			return minted, nil
		},
		balances,
	)
	if err != nil {
		return "", err
	}
	if minted > 0 {
		gn.Minted += minted
		if !gn.canMint() {
			return "", common.NewErrorf("collect_reward_failed",
				"max mint %v exceeded, %v", gn.MaxMint, gn.Minted)
		}
		if err = gn.save(balances); err != nil {
			return "", common.NewErrorf("collect_reward_failed",
				"saving global node: %v", err)
		}
	}
	return "", nil
}
