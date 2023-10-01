package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/event"
	"errors"
	"github.com/0chain/common/core/logging"
	"strings"
)

func storageChallengeToChallengeTable(ch *StorageChallengeResponse, expiredN int) *event.Challenge { // nolint
	var validators = make([]string, 0, len(ch.Validators))
	for _, v := range ch.Validators {
		validators = append(validators, v.ID)
	}
	validatorsStr := strings.Join(validators, ",")
	return &event.Challenge{
		ChallengeID:    ch.ID,
		CreatedAt:      ch.Created,
		AllocationID:   ch.AllocationID,
		BlobberID:      ch.BlobberID,
		ValidatorsID:   validatorsStr,
		Seed:           ch.Seed,
		AllocationRoot: ch.AllocationRoot,
		Responded:      ch.Responded,
		ExpiredN:       expiredN,
		Timestamp:      ch.Timestamp,
		RoundCreatedAt: ch.RoundCreatedAt,
	}
}

func challengeTableToStorageChallengeInfo(ch *event.Challenge, edb *event.EventDb) (*StorageChallengeResponse, error) {
	vIDs := strings.Split(ch.ValidatorsID, ",")
	if len(vIDs) == 0 {
		return nil, errors.New("no validators in challenge")
	}
	validators, err := getValidators(vIDs, edb)
	if err != nil {
		return nil, err
	}
	return &StorageChallengeResponse{
		StorageChallenge: &StorageChallenge{
			Created:         ch.CreatedAt,
			ID:              ch.ChallengeID,
			TotalValidators: 0,
			AllocationID:    ch.AllocationID,
			BlobberID:       ch.BlobberID,
			Responded:       ch.Responded,
			RoundCreatedAt:  ch.RoundCreatedAt,
		},
		Seed:           ch.Seed,
		AllocationRoot: ch.AllocationRoot,
		Validators:     validators,
		Timestamp:      ch.Timestamp,
	}, nil
}

func emitAddChallenge(
	ch *StorageChallengeResponse,
	expiredN int,
	balances cstate.StateContextI,
	allocStats *StorageAllocationStats,
) error {
	balances.EmitEvent(event.TypeStats, event.TagAddChallenge, ch.ID, storageChallengeToChallengeTable(ch, expiredN))
	balances.EmitEvent(event.TypeStats, event.TagAddChallengeToAllocation, ch.AllocationID, event.Allocation{
		AllocationID:         ch.AllocationID,
		OpenChallenges:       allocStats.OpenChallenges,
		TotalChallenges:      allocStats.TotalChallenges,
		SuccessfulChallenges: allocStats.SuccessChallenges,
		FailedChallenges:     allocStats.FailedChallenges,
	})

	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberChallenge, ch.BlobberID, event.ChallengeStatsDeltas{
		Id:             ch.BlobberID,
		PassedDelta:    0,
		CompletedDelta: 1,
		OpenDelta:      1,
	})

	logging.Logger.Debug("emitted add_challenge")
	return nil
}

func emitUpdateChallenge(
	sc *StorageChallenge,
	passed bool,
	responded BlobberChallengeResponded,
	balances cstate.StateContextI,
	allocStats *StorageAllocationStats,
) error {
	clg := event.Challenge{
		ChallengeID:    sc.ID,
		AllocationID:   sc.AllocationID,
		BlobberID:      sc.BlobberID,
		RoundResponded: balances.GetBlock().Round,
		Passed:         passed,
		Responded:      int64(responded),
	}

	a := event.Allocation{
		AllocationID:             sc.AllocationID,
		OpenChallenges:           allocStats.OpenChallenges,
		TotalChallenges:          allocStats.TotalChallenges,
		FailedChallenges:         allocStats.FailedChallenges,
		SuccessfulChallenges:     allocStats.SuccessChallenges,
		LatestClosedChallengeTxn: sc.ID,
	}

	blobberOpenChallenges := int64(0)
	if responded == ChallengeNotResponded {
		blobberOpenChallenges = 1
	} else {
		blobberOpenChallenges = -1
	}

	blobberPassedChallenges := int64(0)
	if passed {
		blobberPassedChallenges = 1
	}

	b := event.ChallengeStatsDeltas{
		Id:             sc.BlobberID,
		OpenDelta:      blobberOpenChallenges,
		CompletedDelta: 0,
		PassedDelta:    blobberPassedChallenges,
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateChallenge, sc.ID, clg)
	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationChallenge, sc.AllocationID, a)
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberChallenge, sc.BlobberID, b)
	return nil
}

func emitUpdateAllocationAndBlobberStatsOnAllocFinalization(alloc *StorageAllocation, blobbersSettledChallengesCount []int64, balances cstate.StateContextI) {
	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationChallenge, alloc.ID, event.Allocation{
		AllocationID:         alloc.ID,
		OpenChallenges:       alloc.Stats.OpenChallenges,
		TotalChallenges:      alloc.Stats.TotalChallenges,
		SuccessfulChallenges: alloc.Stats.SuccessChallenges,
		FailedChallenges:     alloc.Stats.FailedChallenges,
	})

	for idx, ba := range alloc.BlobberAllocs {
		balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberChallenge, ba.BlobberID, event.ChallengeStatsDeltas{
			Id:             ba.BlobberID,
			OpenDelta:      0,
			CompletedDelta: 0,
			PassedDelta:    blobbersSettledChallengesCount[idx],
		})
	}
}

func emitUpdateAllocationAndBlobberStatsOnBlobberRemoval(alloc *StorageAllocation, blobberID string, blobbersSettledChallengesCount int64, balances cstate.StateContextI) {
	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationChallenge, alloc.ID, event.Allocation{
		AllocationID:         alloc.ID,
		OpenChallenges:       alloc.Stats.OpenChallenges,
		TotalChallenges:      alloc.Stats.TotalChallenges,
		SuccessfulChallenges: alloc.Stats.SuccessChallenges,
		FailedChallenges:     alloc.Stats.FailedChallenges,
	})

	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberChallenge, blobberID, event.ChallengeStatsDeltas{
		Id:             blobberID,
		OpenDelta:      0,
		CompletedDelta: 0,
		PassedDelta:    blobbersSettledChallengesCount,
	})

}

func getOpenChallengesForBlobber(blobberID string, from int64, limit common2.Pagination, edb *event.EventDb) ([]*StorageChallengeResponse, error) {

	var chs []*StorageChallengeResponse
	challenges, err := edb.GetOpenChallengesForBlobber(blobberID, from, limit)
	if err != nil {
		return nil, err
	}

	for _, ch := range challenges {
		challInfo, err := challengeTableToStorageChallengeInfo(ch, edb)
		if err != nil {
			return nil, err
		}
		chs = append(chs, challInfo)
	}

	return chs, nil
}

func getChallenge(challengeID string,
	edb *event.EventDb) (*StorageChallengeResponse, error) {

	challenge, err := edb.GetChallenge(challengeID)
	if err != nil {
		return nil, err
	}

	challInfo, err := challengeTableToStorageChallengeInfo(challenge, edb)
	if err != nil {
		return nil, err
	}
	return challInfo, nil
}
