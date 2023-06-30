package storagesc

import (
	"errors"
	"log"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

//go:generate msgp -io=false -tests=false -v

// AllocationChallengeStats represents the challenge stats of allocation
// First one is allocation stats, remaining as the blobbers' stats, they
// are in the same order of blobbers in each allocation
type AllocationChallengeStats struct {
	Stats []*StorageAllocationStats // should be updated when allocation remove or add blobbers
}

func (acs *AllocationChallengeStats) Init(blobberNum int) {
	acs.Stats = make([]*StorageAllocationStats, blobberNum+1)
	for i := 0; i < blobberNum+1; i++ {
		acs.Stats[i] = &StorageAllocationStats{}
	}
}

func (acs *AllocationChallengeStats) GetAllocStats() *StorageAllocationStats {
	if len(acs.Stats) == 0 {
		return nil
	}

	return acs.Stats[0]
}

func (acs *AllocationChallengeStats) GetBlobberStatsByIndex(idx int) (*StorageAllocationStats, error) {
	if len(acs.Stats) == 0 {
		return nil, errors.New("no stats in allocation")
	}

	if idx+1 >= len(acs.Stats) {
		log.Panic("invalid blobber index")
		return nil, errors.New("invalid blobber index")
	}

	return acs.Stats[idx+1], nil
}

func (acs *AllocationChallengeStats) GetBlobbersStats() []*StorageAllocationStats {
	if len(acs.Stats) == 0 {
		return nil
	}

	return acs.Stats[1:]
}

//func (acs *AllocationChallenges)

func (acs *AllocationChallengeStats) AddAllocOpenChallenge(bIdx int) error {
	if len(acs.Stats) == 0 {
		return errors.New("no stats in allocation")
	}

	if bIdx+1 >= len(acs.Stats) {
		log.Panic("invalid blobber index:", bIdx, len(acs.Stats))
		return errors.New("invalid blobber index")
	}

	acs.Stats[0].OpenChallenges++
	acs.Stats[0].TotalChallenges++

	acs.Stats[bIdx+1].OpenChallenges++
	acs.Stats[bIdx+1].TotalChallenges++
	return nil
}

func (acs *AllocationChallengeStats) Save(balances state.StateContextI, allocID string) error {
	_, err := balances.InsertTrieNode(allocationChallengeStatsKey(allocID), acs)
	return err
}

func (acs *AllocationChallengeStats) FailChallenges(bIdx int8) {
	acs.Stats[0].OpenChallenges--
	acs.Stats[0].FailedChallenges++
	acs.Stats[bIdx+1].OpenChallenges--
	acs.Stats[bIdx+1].FailedChallenges++
}

func allocationChallengeStatsKey(id string) string {
	return encryption.Hash("alloc_challenge_stats:" + id)
}

func getAllocationChallengeStats(balances state.StateContextI, allocID string) (*AllocationChallengeStats, error) {
	var acs AllocationChallengeStats
	if err := balances.GetTrieNode(allocationChallengeStatsKey(allocID), &acs); err != nil {
		return nil, err
	}

	return &acs, nil
}

func allocChallengeStatsCreateIfNotExist(
	balances state.StateContextI,
	allocID string,
	blobberNum int) (*AllocationChallengeStats, error) {
	acs, err := getAllocationChallengeStats(balances, allocID)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		//log.Println("init stats:", blobberNum)

		// does not exist, create and init acs
		acs = &AllocationChallengeStats{}
		acs.Init(blobberNum)
	}

	return acs, nil
}
