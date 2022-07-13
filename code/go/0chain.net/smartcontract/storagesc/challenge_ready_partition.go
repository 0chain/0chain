package storagesc

import (
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
)

const allChallengeReadyBlobbersPartitionSize = 50

//go:generate msgp -io=false -tests=false -unexported=true -v

// This is a partition that will only record the blobbers ids that are ready to be challenged.
// Only after blobbers have received writemarkers/readmarkers will it be added to the partitions.
func partitionsChallengeReadyBlobbers(balances state.StateContextI) (*partitions.Partitions, error) {
	return partitions.GetPartitions(balances, ALL_CHALLENGE_READY_BLOBBERS_KEY)
}

// ChallengeReadyBlobber represents a node that is ready to be challenged,
// it will be saved in challenge ready blobbers partitions.
type ChallengeReadyBlobber struct {
	BlobberID string `json:"blobber_id"`
	Weight    uint64 `json:"weight"`
}

func (bc *ChallengeReadyBlobber) GetID() string {
	return bc.BlobberID
}

func partitionsChallengeReadyBlobbersAdd(state state.StateContextI,
	blobberID string, weight uint64) (*partitions.PartitionLocation, error) {
	challengeReadyParts, err := partitionsChallengeReadyBlobbers(state)
	if err != nil {
		return nil, fmt.Errorf("could not get challenge ready partitions, %v", err)
	}

	partIdx, err := challengeReadyParts.AddItem(state, &ChallengeReadyBlobber{
		BlobberID: blobberID,
		Weight:    weight,
	})
	if err != nil {
		return nil, fmt.Errorf("could not add blobber to challenge ready partition, %v", err)
	}

	if err := challengeReadyParts.Save(state); err != nil {
		return nil, fmt.Errorf("could not update challenge ready partitions: %v", err)
	}

	return &partitions.PartitionLocation{Location: partIdx}, nil
}

func partitionsChallengeReadyBlobbersRemove(state state.StateContextI,
	pl *partitions.PartitionLocation, blobberID string) error {
	challengeReadyParts, err := partitionsChallengeReadyBlobbers(state)
	if err != nil {
		return fmt.Errorf("could not get blobber challenge ready partitions: %v", err)
	}

	err = challengeReadyParts.RemoveItem(state, pl.Location, blobberID)
	if err != nil {
		return fmt.Errorf("could not remove blobber from challenge partitions: %v", err)
	}

	return challengeReadyParts.Save(state)
}

func init() {
	regInitPartsFunc(func(state state.StateContextI) error {
		_, err := partitions.CreateIfNotExists(
			state, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
		return err
	})
}
