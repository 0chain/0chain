package storagesc

import (
	"fmt"
	"math"

	"0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

var allChallengeReadyBlobbersPartitionSize = 50

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

	afterHardFork1 := func() (e error) {
		partWeights, e = blobberWeightsPartitions(balances, p)
		if e != nil {
			e = fmt.Errorf("could not get blobber weights partitions: %v", e)
			return
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
		return
	}

	actError := cstate.WithActivation(balances, "apollo", func() (e error) { return }, afterHardFork1)
	if actError != nil {
		return nil, nil, actError
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

func (bc *ChallengeReadyBlobber) GetWeightV1() uint64 {
	return uint64((float64(bc.Stake) * float64(bc.UsedCapacity)) / 1e10)
}

// GetWeightV2
// weight = 20*stake + 10k*log(used + 1)
// stake in ZCN
// used in MB
// weight is capped with 10KK
func (bc *ChallengeReadyBlobber) GetWeightV2() uint64 {
	stake, err := bc.Stake.ToZCN()
	if err != nil {
		return 0
	}
	used := float64(bc.UsedCapacity) / 1e6

	weightFloat := 20*stake + 10000*math.Log2(used+2)
	weight := uint64(10000000)

	if weightFloat < float64(weight) {
		weight = uint64(weightFloat)
	}

	return weight
}

func PartitionsChallengeReadyBlobberAddOrUpdate(state state.StateContextI, blobberID string, stake currency.Coin, usedCapacity uint64) error {
	parts, partsWeight, err := partitionsChallengeReadyBlobbers(state)
	if err != nil {
		return fmt.Errorf("could not get challenge ready partitions, %v", err)
	}

	crb := &ChallengeReadyBlobber{BlobberID: blobberID, UsedCapacity: usedCapacity, Stake: stake}

	beforeHardFork1 := func() (e error) {
		e = parts.Add(state, crb)
		if e != nil {
			if !partitions.ErrItemExist(e) {
				return
			}

			// item exists, update
			e = parts.UpdateItem(state, crb)
			if e != nil {
				return
			}
		}

		e = parts.Save(state)
		if e != nil {
			e = fmt.Errorf("could not add or update challenge ready partitions: %v", e)
		}
		return
	}

	afterHardFork1 := func() (e error) {
		var exist bool
		exist, e = parts.Exist(state, blobberID)
		if e != nil {
			e = fmt.Errorf("could not check if blobber exists: %v", e)
			return
		}

		if exist {
			// update
			e = partsWeight.update(state, *crb)
			if e != nil {
				e = fmt.Errorf("could not update blobber weight: %v", e)
			}
			return
		}

		// add new item
		e = partsWeight.add(state, *crb)
		if e != nil {
			e = fmt.Errorf("could not add blobber to challenge ready partition: %v", e)
		}
		return
	}

	return cstate.WithActivation(state, "apollo", beforeHardFork1, afterHardFork1)
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
	beforeHardFork1 := func() (e error) {
		e = parts.UpdateItem(state, crb)
		if e != nil {
			return
		}

		return parts.Save(state)
	}

	afterHardFork1 := func() (e error) {
		return partsWeight.update(state, *crb)
	}

	return cstate.WithActivation(state, "apollo", beforeHardFork1, afterHardFork1)
}

func partitionsChallengeReadyBlobbersRemove(state state.StateContextI, blobberID string) error {
	challengeReadyParts, partsWeight, err := partitionsChallengeReadyBlobbers(state)
	if err != nil {
		return err
	}

	beforeHardFork1 := func() error {
		err = challengeReadyParts.Remove(state, blobberID)
		if err != nil {
			return err
		}

		return challengeReadyParts.Save(state)
	}

	afterHardFork1 := func() error {
		return partsWeight.remove(state, blobberID)
	}

	return cstate.WithActivation(state, "apollo", beforeHardFork1, afterHardFork1)
}

func init() {
	regInitPartsFunc(func(state state.StateContextI) error {
		_, err := partitions.CreateIfNotExists(state, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
		return err
	})
}
