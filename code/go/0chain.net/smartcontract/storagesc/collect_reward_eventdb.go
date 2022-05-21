package storagesc

import (
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
)

func emitAddOrOverwriteReward(amount currency.Coin, providerID string, prr stakepool.CollectRewardRequest, balances cstate.StateContextI, t *transaction.Transaction) error {
	data, err := json.Marshal(event.Reward{
		Amount:       int64(amount),
		BlockNumber:  balances.GetBlock().Round,
		ClientID:     t.ClientID,
		PoolID:       prr.PoolId,
		ProviderType: prr.ProviderType.String(),
		ProviderID:   providerID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal reward: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddReward, t.Hash, string(data))

	return nil
}
