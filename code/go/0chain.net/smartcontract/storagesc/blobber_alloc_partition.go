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
	return partitions.CreateIfNotExists(balances, getBlobberAllocationsKey(blobberID), blobberAllocationPartitionSize)
}

func partitionsBlobberAllocationsAdd(state state.StateContextI, blobberID, allocID string) error {
	blobAllocsParts, err := partitionsBlobberAllocations(blobberID, state)
	if err != nil {
		return fmt.Errorf("error fetching blobber challenge allocation partition, %v", err)
	}

	err = blobAllocsParts.Add(state, &BlobberAllocationNode{ID: allocID})
	if err != nil && !partitions.ErrItemExist(err) {
		return err
	}

	if err := blobAllocsParts.Save(state); err != nil {
		return fmt.Errorf("could not update blobber allocations partitions: %v", err)
	}

	return nil
}

func partitionsBlobberAllocationsRemove(state state.StateContextI, blobberID, allocID string, blobberAllocParts *partitions.Partitions) error {

	err := blobberAllocParts.Remove(state, allocID)
	if err != nil {
		return fmt.Errorf("could not remove allocation from blobber: %v", err)
	}

	allocNum, err := blobberAllocParts.Size(state)
	if err != nil {
		return fmt.Errorf("could not get challenge partition size: %v", err)
	}

	if allocNum == 0 {
		// remove blobber from challenge ready partition when there's no allocation bind to it
		err = partitionsChallengeReadyBlobbersRemove(state, blobberID)
		if err != nil && !partitions.ErrItemNotFound(err) {
			// it could be empty if we finalize the allocation before committing any read or write
			return fmt.Errorf("failed to remove blobber from challenge ready partitions: %v", err)
		}
	}
	return nil
}
