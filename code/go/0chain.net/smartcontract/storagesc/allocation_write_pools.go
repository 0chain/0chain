package storagesc

import (
	"fmt"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"github.com/0chain/errors"
)

// Created using StorageAllocation.getAllocationPools
type allocationWritePools struct {
	// The indices for ids and writePools match
	ownerId         int
	ids             []string
	writePools      []*writePool
	allocationPools allocationPools
}

func (awp *allocationWritePools) getOwnerWP() (*writePool, error) {
	if len(awp.writePools) == 0 {
		return nil, errors.New("", "no write pools")
	}
	if awp.ownerId < 0 || len(awp.writePools) <= awp.ownerId {
		return nil, errors.New("", "no owner write pool")
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
	value state.Balance,
) (err error) {
	return awp.allocationPools.moveToChallenge(allocID, blobID, cp, now, value)
}

func (aps allocationWritePools) allocUntil(
	allocID string, until common.Timestamp,
) (value state.Balance) {
	return aps.allocationPools.allocUntil(allocID, until)
}

func (awp *allocationWritePools) addOwnerWritePool(ap *allocationPool) error {
	if len(awp.writePools) == 0 {
		return errors.New("", "no write pools")
	}
	if awp.ownerId < 0 || len(awp.writePools) <= awp.ownerId {
		return errors.New("", "no owner write pool")
	}
	awp.writePools[awp.ownerId].Pools.add(ap)
	awp.allocationPools.add(ap)
	return nil
}
