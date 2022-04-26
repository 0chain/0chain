package storagesc

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/dbs"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

func StorageChallengeToChallengeTable(ch *StorageChallengeInfo) *event.Challenge {
	var validators []string
	for _, v := range ch.Validators {
		validators = append(validators, v.ID)
	}
	return &event.Challenge{
		ChallengeID:    ch.ID,
		CreatedAt:      ch.Created,
		AllocationID:   ch.AllocationID,
		BlobberID:      ch.BlobberID,
		PrevID:         ch.PrevID,
		ValidatorsID:   validators,
		Seed:           ch.RandomNumber,
		AllocationRoot: ch.AllocationRoot,
		Responded:      ch.Responded,
	}
}

func emitAddOrOverwriteChallenge(ch *StorageChallengeInfo, balances cstate.StateContextI) error {
	data, err := json.Marshal(StorageChallengeToChallengeTable(ch))
	if err != nil {
		return fmt.Errorf("marshalling challenge: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteChallenge, ch.ID, string(data))
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
