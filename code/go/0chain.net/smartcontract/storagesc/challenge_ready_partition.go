package storagesc

import (
	"fmt"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
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

	logging.Logger.Debug("jayash challengeReadyParts", zap.Any("challengeReadyParts", challengeReadyParts))

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
