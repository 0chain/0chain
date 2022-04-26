package storagesc

import (
	state "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
)

//go:generate msgp -io=false -tests=false -v

//------------------------------------------------------------------------------

type BlobberChallengeAllocationNode struct {
	ID string `json:"id"`
}

func (z *BlobberChallengeAllocationNode) GetID() string {
	return z.ID
}

func getBlobbersChallengeAllocationList(blobberID string, balances state.StateContextI) (*partitions.Partitions, error) {
	return partitions.CreateIfNotExists(balances, getBlobberChallengeAllocationKey(blobberID), blobberChallengeAllocationPartitionSize)
}
