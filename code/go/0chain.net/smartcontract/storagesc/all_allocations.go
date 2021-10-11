package storagesc

import (
	"encoding/json"

	"0chain.net/core/util"
	"0chain.net/smartcontract/partitions"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

var allAllocationsPartitionSize = 100

func (_ *StorageSmartContract) getAllAllocationsList(
	balances state.StateContextI) (*Allocations, error) {

	allocationList := &Allocations{}

	allocationListBytes, err := balances.GetTrieNode(ALL_ALLOCATIONS_KEY)
	if allocationListBytes == nil {
		return allocationList, nil
	}
	err = json.Unmarshal(allocationListBytes.Encode(), allocationList)
	if err != nil {
		return nil, common.NewError("getAllAllocationsList_failed",
			"Failed to retrieve existing allocations list")
	}
	return allocationList, nil
}

func getAllAllocationsList(balances state.StateContextI) (partitions.RandPartition, error) {
	all, err := partitions.GetRandomSelector(ALL_ALLOCATIONS_KEY, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		all = partitions.NewRandomSelector(ALL_ALLOCATIONS_KEY, allAllocationsPartitionSize, allocationChangedPartition)
		return all, nil
	}
	return all, nil
}

func allocationChangedPartition(partitions.PartitionItem, int, int, state.StateContextI) error {
	return nil
}
