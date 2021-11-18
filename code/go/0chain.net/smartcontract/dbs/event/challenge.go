package event

import (
	"encoding/json"
	"fmt"

	"0chain.net/core/common"
	"gorm.io/gorm"
)

type Challenge struct {
	gorm.Model
	BlobberChallengeID uint
	BlobberID          string           `json:"blobber_id"`
	Created            common.Timestamp `json:"created"`
	ChallengeID        string           `json:"id" gorm:"uniqueIndex"`
	PrevID             string           `json:"prev_id"`
	Validators         []ValidationNode `json:"validators"`
	RandomNumber       int64            `json:"seed"`
	AllocationID       string           `json:"allocation_id"`
	AllocationRoot     string           `json:"allocation_root"`
	//Response           Response         `json:"challenge_response,omitempty"`
}

/*
type Response struct {
	gorm.Model
	ChallengeId       uint
	ResponseID        string             `json:"response_id"`
	ValidationTickets []ValidationTicket `json:"validation_tickets"`
}


type ValidationTicket struct {
	gorm.Model
	ResponseId   uint
	ValidatorID  string           `json:"validator_id"`
	ValidatorKey string           `json:"validator_key"`
	Result       bool             `json:"success"`
	Message      string           `json:"message"`
	MessageCode  string           `json:"message_code"`
	Timestamp    common.Timestamp `json:"timestamp"`
	Signature    string           `json:"signature"`
}
*/

func (ch *Challenge) add(edb *EventDb, data []byte) error {
	err := json.Unmarshal(data, ch)
	if err != nil {
		return err
	}

	bci := BlobberChallengeId{}
	if err := bci.getOrCreate(edb, ch.BlobberID); err != nil {
		return err
	}
	ch.BlobberChallengeID = bci.ID
	result := edb.Store.Get().Create(ch)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (edb *EventDb) getChallenges(blobberId string) ([]Challenge, error) {
	var challenges []Challenge
	result := edb.Store.Get().
		Model(&Challenge{}).
		Where("blobber_id", blobberId).
		Find(&challenges)
	if result.Error != nil {
		return nil, result.Error
	}
	for i, ch := range challenges {
		validators, err := edb.getValidationNoes(ch.ID)
		if err != nil {
			return nil, fmt.Errorf("challenge %v: %v", ch.ChallengeID, err)
		}
		challenges[i].Validators = validators
	}
	return challenges, nil
}

func (edb *EventDb) GetChallenge(challengeId string) (*Challenge, error) {
	var ch Challenge
	result := edb.Store.Get().
		Model(&Challenge{}).
		Find(&Challenge{}).
		Where("challenge_id", challengeId).
		First(&ch)
	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving challenge, id %v, error %v",
			challengeId, result.Error)
	}

	validators, err := edb.getValidationNoes(ch.ID)
	if err != nil {
		return nil, fmt.Errorf("challenge %v: %v", ch.ChallengeID, err)
	}
	ch.Validators = validators
	return &ch, nil
}

func (edb *EventDb) removeChallenge(challengeId string) error {
	result := edb.Store.Get().Delete(&Challenge{}, "challenge_id", challengeId)
	return result.Error
}
