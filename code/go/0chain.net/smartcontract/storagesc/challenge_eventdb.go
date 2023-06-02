package storagesc

import (
	"errors"
	"strings"
	"time"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/logging"
)

func storageChallengeToChallengeTable(ch *StorageChallengeResponse, expiredN int) *event.Challenge { //nolint
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
		},
		Seed:           ch.Seed,
		AllocationRoot: ch.AllocationRoot,
		Validators:     validators,
		Timestamp:      ch.Timestamp,
	}, nil
}

func emitAddChallenge(ch *StorageChallengeResponse, expiredCountMap map[string]int, expiredN int, balances cstate.StateContextI, allocStats, blobberStats *StorageAllocationStats) {
	balances.EmitEvent(event.TypeStats, event.TagAddChallenge, ch.ID, storageChallengeToChallengeTable(ch, expiredN))
	balances.EmitEvent(event.TypeStats, event.TagAddChallengeToAllocation, ch.AllocationID, event.Allocation{
		AllocationID:         ch.AllocationID,
		OpenChallenges:       allocStats.OpenChallenges,
		TotalChallenges:      allocStats.TotalChallenges,
		SuccessfulChallenges: allocStats.SuccessChallenges,
		FailedChallenges:     allocStats.FailedChallenges,
	})

	// Update open challenges count of challenge blobber
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberOpenChallenges, ch.BlobberID, event.Blobber{
		Provider:            event.Provider{ID: ch.BlobberID},
		OpenChallenges:      uint64(blobberStats.OpenChallenges),
		ChallengesCompleted: uint64(blobberStats.TotalChallenges),
		ChallengesPassed:    uint64(blobberStats.SuccessChallenges),
	})

	logging.Logger.Debug("emitted add_challenge")
}

func emitUpdateChallenge(sc *StorageChallenge, passed bool, balances cstate.StateContextI, allocStats, blobberStats *StorageAllocationStats) {
	clg := event.Challenge{
		ChallengeID:    sc.ID,
		AllocationID:   sc.AllocationID,
		BlobberID:      sc.BlobberID,
		RoundResponded: balances.GetBlock().Round,
		Passed:         passed,
	}
	if passed {
		clg.Responded = int64(1) // Passed challenge
	} else {
		clg.Responded = int64(2) // Failed challenge
	}

	a := event.Allocation{
		AllocationID:             sc.AllocationID,
		OpenChallenges:           allocStats.OpenChallenges,
		TotalChallenges:          allocStats.TotalChallenges,
		FailedChallenges:         allocStats.FailedChallenges,
		SuccessfulChallenges:     allocStats.SuccessChallenges,
		LatestClosedChallengeTxn: sc.ID,
	}

	b := event.Blobber{
		Provider:            event.Provider{ID: sc.BlobberID},
		ChallengesCompleted: uint64(blobberStats.TotalChallenges),

		ChallengesPassed: uint64(blobberStats.SuccessChallenges),
		OpenChallenges:   uint64(blobberStats.OpenChallenges),
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateChallenge, sc.ID, clg)
	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationChallenge, sc.AllocationID, a)
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberChallenge, sc.BlobberID, b)
}

func getOpenChallengesForBlobber(blobberID string, from, cct common.Timestamp, limit common2.Pagination, edb *event.EventDb) ([]*StorageChallengeResponse, error) {
	var chs []*StorageChallengeResponse
	challenges, err := edb.GetOpenChallengesForBlobber(blobberID, from,
		common.Timestamp(time.Now().Unix()), cct, limit)
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
