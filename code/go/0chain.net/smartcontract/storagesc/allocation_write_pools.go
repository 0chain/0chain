package storagesc

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/currency"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

// Created using StorageAllocation.getAllocationPools
type allocationWritePools struct {
	// The indices for ids and writePools match
	ownerId         int
	ids             []string
	writePools      []*writePool
	allocationPools allocationPools
}

func (awp *allocationWritePools) activeAllocationPools(
	allocID string, now common.Timestamp,
) allocationPools {
	var cut = awp.allocationPools.allocationCut(allocID)
	cut = removeExpired(cut, now)
	return cut
}

func (awp *allocationWritePools) getOwnerWP() (*writePool, error) {
	if len(awp.writePools) == 0 {
		return nil, errors.New("no write pools")
	}
	if awp.ownerId < 0 || len(awp.writePools) <= awp.ownerId {
		return nil, errors.New("no owner write pool")
	}
	return awp.writePools[awp.ownerId], nil
}

func (awp *allocationWritePools) saveWritePools(
	sscId string, balances chainstate.StateContextI,
) error {
	for i, wp := range awp.writePools {
		err := wp.save(sscId, awp.ids[i], balances)
		if err != nil {
			return fmt.Errorf("cannot save write pool of %s", awp.ids[i])
		}
	}
	return nil
}

func (awp *allocationWritePools) moveToChallenge(
	allocID, blobID string,
	cp *challengePool,
	now common.Timestamp,
	value currency.Coin,
) (err error) {
	return awp.allocationPools.moveToChallenge(allocID, blobID, cp, now, value)
}

func (aps allocationWritePools) allocUntil(
	allocID string, until common.Timestamp,
) (value currency.Coin) {
	return aps.allocationPools.allocUntil(allocID, until)
}

func (awp *allocationWritePools) addOwnerWritePool(ap *allocationPool) error {
	if len(awp.writePools) == 0 {
		return errors.New("no write pools")
	}
	if awp.ownerId < 0 || len(awp.writePools) <= awp.ownerId {
		return errors.New("no owner write pool")
	}
	awp.writePools[awp.ownerId].Pools.add(ap)
	awp.allocationPools.add(ap)
	return nil
}
