package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"0chain.net/core/common"

	"0chain.net/smartcontract/dbs"

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

func challengeTableToStorageChallengeInfo(ch *event.Challenge, balances cstate.StateContextI) (*StorageChallengeResponse, error) {
	vIDs := strings.Split(ch.ValidatorsID, ",")
	if len(vIDs) == 0 {
		return nil, errors.New("no validators in challenge")
	}
	validators, err := getValidators(vIDs, balances.GetEventDB())
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

func emitAddChallenge(ch *StorageChallengeResponse, balances cstate.StateContextI) error {
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
	balances cstate.StateContextI) ([]*StorageChallengeResponse, error) {

	var chs []*StorageChallengeResponse
	challenges, err := balances.GetEventDB().GetOpenChallengesForBlobber(blobberID,
		common.Timestamp(time.Now().Unix()), cct)
	if err != nil {
		return nil, err
	}

	for _, ch := range challenges {
		challInfo, err := challengeTableToStorageChallengeInfo(ch, balances)
		if err != nil {
			return nil, err
		}
		chs = append(chs, challInfo)
	}
	return chs, nil
}

func getChallengeForBlobber(blobberID, challengeID string,
	balances cstate.StateContextI) (*StorageChallengeResponse, error) {

	challenge, err := balances.GetEventDB().GetChallengeForBlobber(blobberID, challengeID)
	if err != nil {
		return nil, err
	}

	challInfo, err := challengeTableToStorageChallengeInfo(challenge, balances)
	if err != nil {
		return nil, err
	}
	return challInfo, nil
}
