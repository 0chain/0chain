package storagesc

import (
	"fmt"

	"0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	common2 "0chain.net/smartcontract/partitions"
	partitions_v_1 "0chain.net/smartcontract/partitions_v_1"
	partitions_v_2 "0chain.net/smartcontract/partitions_v_2"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

const allChallengeReadyBlobbersPartitionSize = 50

//go:generate msgp -io=false -tests=false -unexported=true -v
func partitionsChallengeReadyBlobbers(balances state.StateContextI) (p common2.Partitions, w *blobberWeightPartitionsWrap, e error) {
	actError := cstate.WithActivation(balances, "apollo", func() error {
		p, w, e = partitionsChallengeReadyBlobbers_v_1(balances)
		return nil
	}, func() error {
		p, w, e = partitionsChallengeReadyBlobbers_v_2(balances)
		return nil
	})

	if actError != nil {
		return nil, nil, actError
	}

	return
}

// This is a partition that will only record the blobbers ids that are ready to be challenged.
// Only after blobbers have received writemarkers/readmarkers will it be added to the partitions.
func partitionsChallengeReadyBlobbers_v_1(balances state.StateContextI) (common2.Partitions, *blobberWeightPartitionsWrap, error) {
	var (
		p           *partitions_v_1.Partitions
		partWeights *blobberWeightPartitionsWrap
		err         error
	)

	p, err = partitions_v_1.CreateIfNotExists(balances, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
	if err != nil {
		return nil, nil, err
	}

	return p, partWeights, nil
}

// This is a partition that will only record the blobbers ids that are ready to be challenged.
// Only after blobbers have received writemarkers/readmarkers will it be added to the partitions.
func partitionsChallengeReadyBlobbers_v_2(balances state.StateContextI) (common2.Partitions, *blobberWeightPartitionsWrap, error) {
	var (
		p           *partitions_v_2.Partitions
		partWeights *blobberWeightPartitionsWrap
		err         error
	)

	p, err = partitions_v_2.CreateIfNotExists(balances, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
	if err != nil {
		return nil, nil, err
	}

	partWeights, e := blobberWeightsPartitions(balances, p)
	if e != nil {
		e = fmt.Errorf("could not get blobber weights partitions: %v", e)
		return nil, nil, e
	}

	// check if need to migrate from challenge ready blobber partitions,
	// this should only be done once when apollo round hits
	if partWeights.needSync {
		logging.Logger.Debug("add_challenge - apollo hit - sync blobber weights!!")
		e = partWeights.sync(balances, p)
		if e != nil {
			logging.Logger.Error("add_challenge - apollo hit - sync blobber weights failed", zap.Error(e))
		}
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

func PartitionsChallengeReadyBlobberAddOrUpdate(balance state.StateContextI, blobberID string, stake currency.Coin, usedCapacity uint64) error {
	return cstate.WithActivation(balance, "apollo", func() error {
		return PartitionsChallengeReadyBlobberAddOrUpdate_v_1(balance, blobberID, stake, usedCapacity)
	}, func() error {
		return PartitionsChallengeReadyBlobberAddOrUpdate_v_2(balance, blobberID, stake, usedCapacity)
	})
}

func PartitionsChallengeReadyBlobberAddOrUpdate_v_1(state state.StateContextI, blobberID string, stake currency.Coin, usedCapacity uint64) error {
	parts, _, err := partitionsChallengeReadyBlobbers_v_1(state)
	if err != nil {
		return fmt.Errorf("could not get challenge ready partitions, %v", err)
	}

	crb := &ChallengeReadyBlobber{BlobberID: blobberID, UsedCapacity: usedCapacity, Stake: stake}

	e := parts.Add(state, crb)
	if err != nil {
		if !common2.ErrItemExist(e) {
			return e
		}

		// item exists, update
		e = parts.UpdateItem(state, crb)
		if e != nil {
			return e
		}
	}

	e = parts.Save(state)
	if e != nil {
		e = fmt.Errorf("could not add or update challenge ready partitions: %v", e)
	}
	return e
}
func PartitionsChallengeReadyBlobberAddOrUpdate_v_2(state state.StateContextI, blobberID string, stake currency.Coin, usedCapacity uint64) error {
	parts, partsWeight, err := partitionsChallengeReadyBlobbers_v_2(state)
	if err != nil {
		return fmt.Errorf("could not get challenge ready partitions, %v", err)
	}

	crb := &ChallengeReadyBlobber{BlobberID: blobberID, UsedCapacity: usedCapacity, Stake: stake}

	exist, e := parts.Exist(state, blobberID)
	if e != nil {
		e = fmt.Errorf("could not check if blobber exists: %v", e)
		return e
	}

	if exist {
		// update
		e = partsWeight.update(state, *crb)
		if e != nil {
			e = fmt.Errorf("could not update blobber weight: %v", e)
		}
		return e
	}

	// add new item
	e = partsWeight.add(state, *crb)
	if e != nil {
		e = fmt.Errorf("could not add blobber to challenge ready partition: %v", e)
	}
	return e
}

func PartitionsChallengeReadyBlobberUpdate(balance state.StateContextI, blobberID string, stake currency.Coin, usedCapacity uint64) error {
	return cstate.WithActivation(balance, "apollo", func() error {
		return PartitionsChallengeReadyBlobberUpdate_v_1(balance, blobberID, stake, usedCapacity)
	}, func() error {
		return PartitionsChallengeReadyBlobberUpdate_v_2(balance, blobberID, stake, usedCapacity)
	})
}

func PartitionsChallengeReadyBlobberUpdate_v_1(state state.StateContextI, blobberID string, stake currency.Coin, usedCapacity uint64) error {
	parts, _, err := partitionsChallengeReadyBlobbers_v_1(state)
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
	e := parts.UpdateItem(state, crb)
	if e != nil {
		return e
	}

	return parts.Save(state)

}
func PartitionsChallengeReadyBlobberUpdate_v_2(state state.StateContextI, blobberID string, stake currency.Coin, usedCapacity uint64) error {
	parts, partsWeight, err := partitionsChallengeReadyBlobbers_v_2(state)
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
	return partsWeight.update(state, *crb)
}
func partitionsChallengeReadyBlobbersRemove(state state.StateContextI, blobberID string) error {
	return cstate.WithActivation(state, "apollo", func() error {
		return partitionsChallengeReadyBlobbersRemove_v_1(state, blobberID)
	}, func() error {
		return partitionsChallengeReadyBlobbersRemove_v_2(state, blobberID)
	})
}

func partitionsChallengeReadyBlobbersRemove_v_1(state state.StateContextI, blobberID string) error {
	challengeReadyParts, _, err := partitionsChallengeReadyBlobbers_v_1(state)
	if err != nil {
		return err
	}
	err = challengeReadyParts.Remove(state, blobberID)
	if err != nil {
		return err
	}

	return challengeReadyParts.Save(state)
}
func partitionsChallengeReadyBlobbersRemove_v_2(state state.StateContextI, blobberID string) error {
	_, partsWeight, err := partitionsChallengeReadyBlobbers_v_2(state)
	if err != nil {
		return err
	}
	return partsWeight.remove(state, blobberID)
}

func init() {
	regInitPartsFunc(func(state state.StateContextI) error {
		return cstate.WithActivation(state, "apollo", func() error {
			_, err := partitions_v_1.CreateIfNotExists(state, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
			if err != nil {
				return err
			}
			return nil
		}, func() error {
			_, err := partitions_v_2.CreateIfNotExists(state, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
			if err != nil {
				return err
			}
			return nil
		})
	})
}
