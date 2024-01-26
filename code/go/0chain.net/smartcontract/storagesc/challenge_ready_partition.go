package storagesc

import (
	"fmt"

	"0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
)

const allChallengeReadyBlobbersPartitionSize = 50

//go:generate msgp -io=false -tests=false -unexported=true -v

// This is a partition that will only record the blobbers ids that are ready to be challenged.
// Only after blobbers have received writemarkers/readmarkers will it be added to the partitions.
func partitionsChallengeReadyBlobbers(balances state.StateContextI) (*partitions.Partitions, *blobberWeightPartitionsWrap, error) {
	var (
		p           *partitions.Partitions
		partWeights *blobberWeightPartitionsWrap
		err         error
	)

	p, err = partitions.CreateIfNotExists(balances, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
	if err != nil {
		return nil, nil, err
	}

	afterHardFork1 := func() {
		partWeights, err = blobberWeightsPartitions(balances, p)
		if err != nil {
			err = fmt.Errorf("could not get blobber weights partitions: %v", err)
			return
		}

		// check if need to migrate from challenge ready blobber partitions,
		// this should only be done once when hard_fork_1 round hits
		if partWeights.needMigrate {
			logging.Logger.Debug("add_challenge - hard_fork_1 hit - sync blobber weights!!")
			partWeights.migrate(balances, p)
		}
	}

	cstate.WithActivation(balances, "hard_fork_1", func() {}, afterHardFork1)
	if err != nil {
		return nil, nil, err
	}

	return p, partWeights, nil
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
	parts, partsWeight, err := partitionsChallengeReadyBlobbers(state)
	if err != nil {
		return fmt.Errorf("could not get challenge ready partitions, %v", err)
	}

	crb := &ChallengeReadyBlobber{BlobberID: blobberID, UsedCapacity: usedCapacity, Stake: stake}

	beforeHardFork1 := func() {
		err = parts.Add(state, crb)
		if err != nil {
			if !partitions.ErrItemExist(err) {
				return
			}

			// item exists, update
			err = parts.UpdateItem(state, crb)
			if err != nil {
				return
			}
		}

		err := parts.Save(state)
		if err != nil {
			err = fmt.Errorf("could not add or update challenge ready partitions: %v", err)
		}
	}

	afterHardFork1 := func() {
		var exist bool
		exist, err = parts.Exist(state, blobberID)
		if err != nil {
			err = fmt.Errorf("could not check if blobber exists: %v", err)
			return
		}

		if exist {
			// update
			err = partsWeight.updateWeight(state, *crb)
			if err != nil {
				err = fmt.Errorf("could not update blobber weight: %v", err)
			}
			return
		}

		// add new item
		err = partsWeight.add(state, *crb)
		if err != nil {
			err = fmt.Errorf("could not add blobber to challenge ready partition: %v", err)
		}
		return
	}

	cstate.WithActivation(state, "hard_fork_1", beforeHardFork1, afterHardFork1)

	return err
}

func PartitionsChallengeReadyBlobberUpdate(state state.StateContextI, blobberID string, stake currency.Coin, usedCapacity uint64) error {
	parts, partsWeight, err := partitionsChallengeReadyBlobbers(state)
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
	beforeHardFork1 := func() {
		err = parts.UpdateItem(state, crb)
		if err != nil {
			return
		}

		err = parts.Save(state)
	}

	afterHardFork1 := func() {
		err = partsWeight.updateWeight(state, *crb)
	}

	cstate.WithActivation(state, "hard_fork_1", beforeHardFork1, afterHardFork1)

	return err
}

func partitionsChallengeReadyBlobbersRemove(state state.StateContextI, blobberID string) error {
	challengeReadyParts, partsWeight, err := partitionsChallengeReadyBlobbers(state)
	if err != nil {
		return err
	}

	beforeHardFork1 := func() {
		err = challengeReadyParts.Remove(state, blobberID)
		if err != nil {
			return
		}

		err = challengeReadyParts.Save(state)
	}

	afterHardFork1 := func() {
		err = partsWeight.remove(state, blobberID)
	}

	cstate.WithActivation(state, "hard_fork_1", beforeHardFork1, afterHardFork1)
	return err
}

func init() {
	regInitPartsFunc(func(state state.StateContextI) error {
		_, err := partitions.CreateIfNotExists(state, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
		return err
	})
}
