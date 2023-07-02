package storagesc

import (
	"github.com/0chain/common/core/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
)

func emitChallengePoolEvent(alloc *StorageAllocation, balances cstate.StateContextI) {
	data := event.ChallengePool{
		ID:         alloc.ID,
		Balance:    int64(alloc.ChallengePool),
		StartTime:  int64(alloc.StartTime),
		Expiration: int64(alloc.Expiration),
		Finalized:  alloc.Finalized,
	}

	balances.EmitEvent(event.TypeStats, event.TagAddOrUpdateChallengePool, alloc.ID, data)

	return
}

func toChallengePoolStat(cp *event.ChallengePool) *challengePoolStat {
	stat := challengePoolStat{
		ID:         cp.ID,
		Balance:    currency.Coin(cp.Balance),
		StartTime:  common.Timestamp(cp.StartTime),
		Expiration: common.Timestamp(cp.Expiration),
		Finalized:  cp.Finalized,
	}

	return &stat
}

// swagger:model challengePoolStat
type challengePoolStat struct {
	ID         string           `json:"id"`
	Balance    currency.Coin    `json:"balance"`
	StartTime  common.Timestamp `json:"start_time"`
	Expiration common.Timestamp `json:"expiration"`
	Finalized  bool             `json:"finalized"`
}
