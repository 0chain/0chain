package event

import (
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
}

func (edb *EventDb) AddChallenge(challenge Challenge) error {
	bci := BlobberChallengeId{}
	if err := bci.getOrCreate(edb, challenge.BlobberID); err != nil {
		return err
	}
	challenge.BlobberChallengeID = bci.ID
	return edb.Store.Get().Create(challenge).Error
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
