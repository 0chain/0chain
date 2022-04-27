package storagesc

import (
	state "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
)

//go:generate msgp -io=false -tests=false -v

//------------------------------------------------------------------------------

type BlobberAllocationNode struct {
	ID string `json:"id"` // allocation id
}

func (z *BlobberAllocationNode) GetID() string {
	return z.ID
}

func getBlobberAllocationPartitions(blobberID string, balances state.StateContextI) (*partitions.Partitions, error) {
	return partitions.CreateIfNotExists(balances, getBlobberAllocationsKey(blobberID), blobberAllocationPartitionSize)
}
