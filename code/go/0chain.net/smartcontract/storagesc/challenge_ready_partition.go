package storagesc

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
	"fmt"
	"github.com/0chain/common/core/currency"
)

const allChallengeReadyBlobbersPartitionSize = 50

//go:generate msgp -io=false -tests=false -unexported=true -v

// This is a partition that will only record the blobbers ids that are ready to be challenged.
// Only after blobbers have received writemarkers/readmarkers will it be added to the partitions.
func partitionsChallengeReadyBlobbers(balances state.StateContextI) (*partitions.Partitions, error) {
	return partitions.CreateIfNotExists(balances, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
}

// ChallengeReadyBlobber represents a node that is ready to be challenged,
// it will be saved in challenge ready blobbers partitions.
type ChallengeReadyBlobber struct {
	BlobberID    string        `json:"blobber_id"`
	Stake        currency.Coin `json:"stake"`
	UsedCapacity uint64        `json:"usedCapacity"`
}

func (bc *ChallengeReadyBlobber) GetID() string {
	return bc.BlobberID
}

func (bc *ChallengeReadyBlobber) GetWeight() uint64 {
	return uint64((float64(bc.Stake) * float64(bc.UsedCapacity)) / 1e10)
}

func PartitionsChallengeReadyBlobberAddOrUpdate(state state.StateContextI, blobberID string, stake currency.Coin, usedCapacity uint64) error {
	parts, err := partitionsChallengeReadyBlobbers(state)
	if err != nil {
		return fmt.Errorf("could not get challenge ready partitions, %v", err)
	}

	crb := &ChallengeReadyBlobber{BlobberID: blobberID, UsedCapacity: usedCapacity, Stake: stake}
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

func PartitionsChallengeReadyBlobberUpdate(state state.StateContextI, blobberID string, stake currency.Coin, usedCapacity uint64) error {
	parts, err := partitionsChallengeReadyBlobbers(state)
	if err != nil {
		return fmt.Errorf("could not get challenge ready partitions, %v", err)
	}

	blobberExist, err := parts.Exist(state, blobberID)
	if err != nil {
		return err
	}

	if !blobberExist {
		return nil
	}

	crb := &ChallengeReadyBlobber{BlobberID: blobberID, UsedCapacity: usedCapacity, Stake: stake}
	if err := parts.UpdateItem(state, crb); err != nil {
		return err
	}

	if err := parts.Save(state); err != nil {
		return err
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

func init() {
	regInitPartsFunc(func(state state.StateContextI) error {
		_, err := partitions.CreateIfNotExists(state, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
		return err
	})
}
