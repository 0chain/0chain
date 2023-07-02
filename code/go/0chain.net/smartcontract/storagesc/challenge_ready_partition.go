package storagesc

import (
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
)

const (
	allChallengeReadyBlobbersPartitionSize = 5
	challengeReadyAllocsPartitionSize      = 5
)

var CHALLENGE_READY_ALLOCS_KEY = encryption.Hash("challenge_ready_allocs")

//go:generate msgp -io=false -tests=false -unexported=true -v

// This is a partition that will only record the blobbers ids that are ready to be challenged.
// Only after blobbers have received writemarkers/readmarkers will it be added to the partitions.
func partitionsChallengeReadyBlobbers(balances state.StateContextI) (*partitions.Partitions, error) {
	return partitions.CreateIfNotExists(balances, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
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

func partitionsChallengeReadyBlobberAddOrUpdate(state state.StateContextI, blobberID string, weight uint64) error {
	parts, err := partitionsChallengeReadyBlobbers(state)
	if err != nil {
		return fmt.Errorf("could not get challenge ready partitions, %v", err)
	}

	crb := &ChallengeReadyBlobber{BlobberID: blobberID, Weight: weight}
	if err := parts.Add(state, crb); err != nil {
		if !partitions.ErrItemExist(err) {
			return err
		}

		// item exists, update
		if err := parts.UpdateItem(state, crb); err != nil {
			return err
		}
	}

	if err := parts.Save(state); err != nil {
		return fmt.Errorf("could not add or update challenge ready partitions: %v", err)
	}

	return nil
}

func partitionsChallengeReadyBlobbersRemove(state state.StateContextI, blobberID string) error {
	challengeReadyParts, err := partitionsChallengeReadyBlobbers(state)
	if err != nil {
		return err
	}

	err = challengeReadyParts.Remove(state, blobberID)
	if err != nil {
		return err
	}

	return challengeReadyParts.Save(state)
}

type ChallengeReadyAllocNode struct {
	AllocID string `msg:"a"`
}

func (c *ChallengeReadyAllocNode) GetID() string {
	return c.AllocID
}

func partitionsChallengeReadyAllocs(state state.StateContextI) (*partitions.Partitions, error) {
	return partitions.CreateIfNotExists(state, CHALLENGE_READY_ALLOCS_KEY, challengeReadyAllocsPartitionSize)
}

func partitionsChallengeReadyAllocsAdd(state state.StateContextI, allocID string) error {
	part, err := partitionsChallengeReadyAllocs(state)
	if err != nil {
		return err
	}

	if err := part.Add(state, &ChallengeReadyAllocNode{
		AllocID: allocID,
	}); err != nil {
		if partitions.ErrItemExist(err) {
			return nil
		} else {
			return err
		}
	}

	if err := part.Save(state); err != nil {
		return fmt.Errorf("could not save challenge ready alloc partition: %v", err)
	}

	return nil
}

func partitionsChallengeReadyAllocsRemove(state state.StateContextI, allocID string) error {
	part, err := partitionsChallengeReadyAllocs(state)
	if err != nil {
		return err
	}

	if err := part.Remove(state, allocID); err != nil {
		return fmt.Errorf("remove challenge ready alloc failed: %v", err)
	}

	return nil
}

func init() {
	regInitPartsFunc(
		func(state state.StateContextI) error {
			_, err := partitions.CreateIfNotExists(state, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
			return err
		},
		func(state state.StateContextI) error {
			_, err := partitions.CreateIfNotExists(state, CHALLENGE_READY_ALLOCS_KEY, challengeReadyAllocsPartitionSize)
			return err
		})
}
