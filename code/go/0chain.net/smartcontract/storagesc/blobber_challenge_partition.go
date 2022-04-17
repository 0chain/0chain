package storagesc

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
)

const allBlobbersChallengePartitionSize = 50

//go:generate msgp -io=false -tests=false -unexported=true -v

func getBlobbersChallengeList(balances state.StateContextI) (*partitions.Partitions, error) {
	return partitions.GetPartitions(balances, ALL_BLOBBERS_CHALLENGE_KEY)
}

type BlobberChallengeNode struct {
	BlobberID string `json:"blobber_id"`
}

func (bc *BlobberChallengeNode) GetID() string {
	return bc.BlobberID
}

func init() {
	regInitPartsFunc(func(state state.StateContextI) error {
		_, err := partitions.CreateIfNotExists(state, ALL_BLOBBERS_CHALLENGE_KEY, allBlobbersChallengePartitionSize)
		return err
	})
}
