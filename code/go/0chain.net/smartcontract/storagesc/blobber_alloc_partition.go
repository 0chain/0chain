package storagesc

import (
	"fmt"

	state "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
)

//go:generate msgp -io=false -tests=false -v

//------------------------------------------------------------------------------

// BlobberAllocationNode represents the allocation that belongs to a blobber,
// will be saved in blobber allocations partitions.
type BlobberAllocationNode struct {
	ID string `json:"id"` // allocation id
}

func (z *BlobberAllocationNode) GetID() string {
	return z.ID
}

func partitionsBlobberAllocations(blobberID string, balances state.StateContextI) (*partitions.Partitions, error) {
	return partitions.CreateIfNotExists(
		balances, getBlobberAllocationsKey(blobberID), blobberAllocationPartitionSize)
}

func partitionsBlobberAllocationsAdd(state state.StateContextI, blobberID, allocID string) (
	*partitions.Partitions, *partitions.PartitionLocation, error) {
	blobAllocsParts, err := partitionsBlobberAllocations(blobberID, state)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching blobber challenge allocation partition, %v", err)
	}

	loc, err := blobAllocsParts.AddItem(state, &BlobberAllocationNode{ID: allocID})
	if err != nil {
		if err == partitions.ErrPartitionItemAlreadyExist {
			return blobAllocsParts, &partitions.PartitionLocation{Location: loc}, nil
		}
		return nil, nil, fmt.Errorf("could not add blobber allocation node to partition, %v", err)
	}

	if err := blobAllocsParts.Save(state); err != nil {
		return nil, nil, fmt.Errorf("could not update blobber allocations partitions: %v", err)
	}

	return blobAllocsParts, &partitions.PartitionLocation{Location: loc}, nil
}
