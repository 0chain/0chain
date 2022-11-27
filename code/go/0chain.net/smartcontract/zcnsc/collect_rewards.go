package zcnsc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool"
	"github.com/0chain/common/core/currency"
)

func (zcn *ZCNSmartContract) CollectRewards(
	tran *transaction.Transaction,
	input []byte,
	ctx cstate.StateContextI,
) (string, error) {
	const code = "pay_reward_failed"
	_, err := stakepool.CollectReward(
		input, func(
			crr stakepool.CollectRewardRequest, balances cstate.StateContextI,
		) (currency.Coin, error) {
			sp, err := zcn.getStakePool(crr.ProviderId, ctx)
			if err != nil {
				return 0, fmt.Errorf("can't get related stake pool: %v", err)
			}

			minted, err := sp.MintRewards(
				tran.ClientID, crr.ProviderId, crr.ProviderType, balances)
			if err != nil {
				return 0, err
			}

			if err := sp.save(zcn.ID, crr.ProviderId, ctx); err != nil {
				return 0, fmt.Errorf("error saving stake pool, %v", err)
			}

			return minted, nil
		}, ctx,
	)
	if err != nil {
		return "", common.NewError(code, err.Error())
	}

	return "", nil
}
