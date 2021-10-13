package storagesc

import (
	"fmt"

	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/util"
	"0chain.net/smartcontract/partitions"

	"0chain.net/chaincore/chain/state"
)

const (
	allAllocationsPartitionSize = 100
	allValidatorsPartitionSize  = 50
)

func getAllAllocationsList(balances state.StateContextI) (partitions.RandPartition, error) {
	all, err := partitions.GetRandomSelector(ALL_ALLOCATIONS_KEY, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		all = partitions.NewRandomSelector(
			ALL_ALLOCATIONS_KEY,
			allAllocationsPartitionSize,
			allocationChangedPartition,
		)
	}
	all.SetCallback(allocationChangedPartition)
	return all, nil
}

func allocationChangedPartition(
	item partitions.PartitionItem,
	from, to int,
	balances state.StateContextI,
) error {
	var ssc = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	alloc, err := ssc.getAllocation(item.Name(), balances)
	if err != nil {
		return fmt.Errorf("cannot get allocation: %v", err)
	}
	alloc.AllAllocationsPartition = to
	_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), alloc)
	if err != nil {
		return fmt.Errorf("saving allocation: %v", err)
	}

	return nil
}
