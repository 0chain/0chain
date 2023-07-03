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
			_, err := partitions.CreateIfNotExists(state, CHALLENGE_READY_ALLOCS_KEY, challengeReadyAllocsPartitionSize)
			return err
		})
}
