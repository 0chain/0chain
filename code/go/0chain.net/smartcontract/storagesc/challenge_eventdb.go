package storagesc

import (
	"errors"
	"strings"
	"time"

	"0chain.net/core/common"
	common2 "0chain.net/smartcontract/common"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

func storageChallengeToChallengeTable(ch *StorageChallengeResponse) *event.Challenge {
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
	}, nil
}

func emitAddChallenge(ch *StorageChallengeResponse, balances cstate.StateContextI) {

	balances.EmitEvent(event.TypeStats, event.TagAddChallenge, ch.ID, storageChallengeToChallengeTable(ch))
	return
}

func emitUpdateChallengeResponse(chID string, responded bool, balances cstate.StateContextI) {
	balances.EmitEvent(event.TypeStats, event.TagUpdateChallenge, chID, event.Challenge{
		ChallengeID: chID,
		Responded:   responded,
	})
}

func emitUpdateBlobberChallengeStats(blobberId string, passed bool, balances cstate.StateContextI) {
	b := event.Blobber{
		BlobberID:           blobberId,
		ChallengesCompleted: 1, // incremental value,
	}

	if passed {
		b.ChallengesPassed = 1 // incremental value
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberChallenge, blobberId, b)
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

func getChallengeForBlobber(blobberID, challengeID string,
	edb *event.EventDb) (*StorageChallengeResponse, error) {

	challenge, err := edb.GetChallengeForBlobber(blobberID, challengeID)
	if err != nil {
		return nil, err
	}

	challInfo, err := challengeTableToStorageChallengeInfo(challenge, edb)
	if err != nil {
		return nil, err
	}
	return challInfo, nil
}
