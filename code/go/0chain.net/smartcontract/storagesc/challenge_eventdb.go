package storagesc

import (
	"encoding/json"
	"fmt"

	"0chain.net/core/common"

	"0chain.net/smartcontract/dbs"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

func storageChallengeToChallengeTable(ch *StorageChallengeInfo) *event.Challenge {
	var validators []string
	for _, v := range ch.Validators {
		validators = append(validators, v.ID)
	}
	return &event.Challenge{
		ChallengeID:    ch.ID,
		CreatedAt:      ch.Created,
		AllocationID:   ch.AllocationID,
		BlobberID:      ch.BlobberID,
		ValidatorsID:   validators,
		Seed:           ch.RandomNumber,
		AllocationRoot: ch.AllocationRoot,
		Responded:      ch.Responded,
	}
}

func challengeTableToStorageChallengeInfo(ch *event.Challenge) *StorageChallengeInfo {
	return &StorageChallengeInfo{
		ID:             ch.ChallengeID,
		Created:        ch.CreatedAt,
		RandomNumber:   ch.Seed,
		AllocationID:   ch.AllocationID,
		AllocationRoot: ch.AllocationRoot,
		BlobberID:      ch.BlobberID,
		Responded:      ch.Responded,
	}
}

func emitAddChallenge(ch *StorageChallengeInfo, balances cstate.StateContextI) error {
	data, err := json.Marshal(storageChallengeToChallengeTable(ch))
	if err != nil {
		return fmt.Errorf("marshalling challenge: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagAddChallenge, ch.ID, string(data))
	return nil
}

func emitUpdateChallengeResponse(chID string, responded bool, balances cstate.StateContextI) error {
	data, err := json.Marshal(&dbs.DbUpdates{
		Id: chID,
		Updates: map[string]interface{}{
			"responded": responded,
		},
	})
	if err != nil {
		return fmt.Errorf("marshalling update: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateChallenge, chID, string(data))
	return nil
}

func getOpenChallengesForBlobber(blobberID string, cct common.Timestamp,
	balances cstate.StateContextI) ([]*StorageChallengeInfo, error) {

	var chs []*StorageChallengeInfo
	challenges, err := balances.GetEventDB().GetOpenChallengesForBlobber(blobberID,
		balances.GetTransaction().CreationDate, cct)
	if err != nil {
		return nil, err
	}

	for _, ch := range challenges {
		chs = append(chs, challengeTableToStorageChallengeInfo(ch))
	}
	return chs, nil
}

func getChallengeForBlobber(blobberID, challengeID string,
	balances cstate.StateContextI) (*StorageChallengeInfo, error) {

	challenge, err := balances.GetEventDB().GetChallengeForBlobber(blobberID, challengeID)
	if err != nil {
		return nil, err
	}

	return challengeTableToStorageChallengeInfo(challenge), nil
}
