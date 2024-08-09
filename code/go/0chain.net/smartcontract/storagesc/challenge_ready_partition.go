package storagesc

import (
	"fmt"
	"math"

	"0chain.net/chaincore/chain/state"
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

	partWeights, err = blobberWeightsPartitions(balances, p)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get blobber weights partitions: %v", err)
	}

	// check if need to migrate from challenge ready blobber partitions,
	// this should only be done once when apollo round hits
	if partWeights.needSync {
		logging.Logger.Debug("add_challenge - apollo hit - sync blobber weights!!")
		err = partWeights.sync(balances, p)
		if err != nil {
			logging.Logger.Error("add_challenge - apollo hit - sync blobber weights failed", zap.Error(err))
			return nil, nil, err
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

	var exist bool
	exist, err = parts.Exist(state, blobberID)
	if err != nil {
		return fmt.Errorf("could not check if blobber exists: %v", err)
	}

	if exist {
		// update
		err = partsWeight.update(state, *crb)
		if err != nil {
			return fmt.Errorf("could not update blobber weight: %v", err)
		}
	}

	// add new item
	err = partsWeight.add(state, *crb)
	if err != nil {
		return fmt.Errorf("could not add blobber to challenge ready partition: %v", err)
	}

	return nil
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
	return partsWeight.update(state, *crb)
}

func partitionsChallengeReadyBlobbersRemove(state state.StateContextI, blobberID string) error {
	_, partsWeight, err := partitionsChallengeReadyBlobbers(state)
	if err != nil {
		return err
	}
	return partsWeight.remove(state, blobberID)
}

func init() {
	regInitPartsFunc(func(state state.StateContextI) error {
		_, err := partitions.CreateIfNotExists(state, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
		return err
	})
}
