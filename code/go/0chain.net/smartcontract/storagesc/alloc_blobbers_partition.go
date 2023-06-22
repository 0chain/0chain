package storagesc

import (
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
)

//go:generate msgp -io=false -tests=false -v

const allocBlobbersPartitionSize = 5

type AllocBlobbersNode struct {
	ID string
}

func (z *AllocBlobbersNode) GetID() string {
	return z.ID
}

// func getAllocBlobbersKey() string {
var allocBlobbersKey = encryption.Hash(ADDRESS + ":alloc_blobbers")

//}

func partitionsAllocBlobbers(balances state.StateContextI) (*partitions.Partitions, error) {
	return partitions.CreateIfNotExists(balances, allocBlobbersKey, allocBlobbersPartitionSize)
}
func partitionsAllocBlobbersAdd(state state.StateContextI, allocID string) (*partitions.Partitions, error) {
	allocBlobbersParts, err := partitionsAllocBlobbers(state)
	if err != nil {
		return nil, fmt.Errorf("error fetching alloc blobbers partition, %v", err)
	}

	err = allocBlobbersParts.Add(state, &AllocBlobbersNode{ID: allocID})
	if err != nil {
		return nil, err
	}

	if err := allocBlobbersParts.Save(state); err != nil {
		return nil, fmt.Errorf("could not update alloc blobbers partitions: %v", err)
	}

	return allocBlobbersParts, nil
}
