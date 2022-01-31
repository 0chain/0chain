package stakepool

import (
	"strconv"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs/event"
)

func emitRewardIncrement(
	provider Provider,
	providerId string,
	rewardIncrement state.Balance,
	balances cstate.StateContextI,
) error {
	balances.EmitEvent(
		event.TypeStats,
		event.TagStakePool,
		providerId,
		strconv.FormatInt(int64(rewardIncrement), 10),
	)
	return nil
}
