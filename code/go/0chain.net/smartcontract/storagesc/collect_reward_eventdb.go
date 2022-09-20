package storagesc

import (
	"0chain.net/chaincore/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
)

func emitAddOrOverwriteReward(amount currency.Coin, providerID string, prr stakepool.CollectRewardRequest, balances cstate.StateContextI, t *transaction.Transaction) error {
	data := event.Reward{
		Amount:       int64(amount),
		BlockNumber:  balances.GetBlock().Round,
		ClientID:     t.ClientID,
		PoolID:       t.ClientID,
		ProviderType: prr.ProviderType.String(),
		ProviderID:   providerID,
	}

	balances.EmitEvent(event.TypeSmartContract, event.TagAddReward, t.Hash, data)

	return nil
}
