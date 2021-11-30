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
	BlobberUrl         string           `json:"url"`
	Created            common.Timestamp `json:"created"`
	ChallengeID        string           `json:"id" gorm:"uniqueIndex"`
	PrevID             string           `json:"prev_id"`
	Validators         []ValidationNode `json:"validators"`
	RandomNumber       int64            `json:"seed"`
	AllocationID       string           `json:"allocation_id"`
	AllocationRoot     string           `json:"allocation_root"`
}

func (challenge *Challenge) AddOrUpdate(edb *EventDb) error {
	exists, err := challenge.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return challenge.update(edb)
	}

	return challenge.Add(edb)
}

func (ch *Challenge) update(edb *EventDb) error {
	result := edb.Store.Get().
		Model(&Challenge{}).
		Where(&Challenge{ChallengeID: ch.ChallengeID}).
		Updates(map[string]interface{}{
			"blobber_url":     ch.BlobberUrl,
			"created":         ch.Created,
			"prev_id":         ch.PrevID,
			"validators":      ch.Validators,
			"random_number":   ch.RandomNumber,
			"allocation_id":   ch.AllocationID,
			"allocation_root": ch.AllocationRoot,
		})
	return result.Error
}

func (ch *Challenge) exists(edb *EventDb) (bool, error) {
	var count int64
	result := edb.Get().
		Model(&Challenge{}).
		Where(&Challenge{ChallengeID: ch.ChallengeID}).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("error searching for challlenge %v, error %v",
			ch.ChallengeID, result.Error)
	}
	return count > 0, nil
}

func (ch *Challenge) Add(edb *EventDb) error {
	bci := BlobberChallengeId{
		BlobberID: ch.BlobberID,
		Url:       ch.BlobberUrl,
	}
	if err := bci.getOrCreate(edb); err != nil {
		return err
	}
	ch.BlobberChallengeID = bci.ID
	return edb.Store.Get().Create(&ch).Error
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
