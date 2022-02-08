package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/stakepool"
)

func (msc *MinerSmartContract) payoutReward(
	t *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	_, err := stakepool.PayoutReward(t.ClientID, input, balances)
	if err != nil {
		return "", err
	}
	return "", nil
}
