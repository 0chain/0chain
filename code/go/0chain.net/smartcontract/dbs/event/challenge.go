package event

import (
	"errors"
	"fmt"

	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
)

type Challenge struct {
	gorm.Model
	ChallengeID    string           `json:"challenge_id" gorm:"index:challenge_id"`
	CreatedAt      common.Timestamp `json:"created_at" gorm:"created_at"`
	AllocationID   string           `json:"allocation_id" gorm:"allocation_id"`
	BlobberID      string           `json:"blobber_id" gorm:"blobber_id"`
	PrevID         string           `json:"prev_id" gorm:"prev_id"`
	ValidatorsID   []string         `json:"validators_id" gorm:"validators_id"`
	Seed           int64            `json:"seed" gorm:"seed"`
	AllocationRoot string           `json:"allocation_root" gorm:"allocation_root"`
	Responded      bool             `json:"responded" gorm:"responded"`
}

func (ch *Challenge) exists(edb *EventDb) (bool, error) {
	err := edb.Get().
		Model(&Challenge{}).
		Where(&Challenge{ChallengeID: ch.ChallengeID}).
		Take(ch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check challenge existence %v, error %v", ch, err)
	}
	return true, nil
}

func (edb *EventDb) GetChallenge(challengeID string) (Challenge, error) {
	var ch Challenge

	result := edb.Store.Get().Model(&Challenge{}).Where(&Challenge{ChallengeID: challengeID}).First(&ch)
	if result.Error != nil {
		return ch, fmt.Errorf("error retriving Challenge node with ID %v; error: %v", challengeID, result.Error)
	}

	return ch, nil
}

func (edb *EventDb) overwriteChallenge(ch *Challenge) error {

	result := edb.Store.Get().Model(&Challenge{}).Where(&Challenge{ChallengeID: ch.ChallengeID}).Updates(&ch)
	return result.Error
}

func (edb *EventDb) addOrOverwriteChallenge(ch *Challenge) error {
	exists, err := ch.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteChallenge(ch)
	}

	result := edb.Store.Get().Create(&ch)

	return result.Error
}

func (edb *EventDb) updateChallenge(updates dbs.DbUpdates) error {
	var challenge = Challenge{ChallengeID: updates.Id}
	exists, err := challenge.exists(edb)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("challenge %v not in database cannot update",
			challenge.ChallengeID)
	}

	result := edb.Store.Get().
		Model(&Challenge{}).
		Where(&Challenge{ChallengeID: challenge.ChallengeID}).
		Updates(updates.Updates)
	return result.Error
}
