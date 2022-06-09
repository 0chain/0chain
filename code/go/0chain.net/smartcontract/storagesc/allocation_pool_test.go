package storagesc

import (
	"0chain.net/chaincore/currency"

	"0chain.net/core/common"
)

//
// test extension
//

func (aps allocationPools) gimmeAll() (total currency.Coin) {
	for _, ap := range aps.Pools {
		total += ap.Balance
	}
	return
}

func (aps allocationPools) allocTotal(allocID string, now int64) (
	total currency.Coin) {

	for _, ap := range aps.Pools {
		if ap.ExpireAt < common.Timestamp(now) {
			continue
		}
		total += ap.Balance
	}
	return
}
