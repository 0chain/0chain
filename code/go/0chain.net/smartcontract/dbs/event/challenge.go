package event

import (
	"errors"
	"fmt"

	"0chain.net/smartcontract/dbs"

	"0chain.net/core/common"
	"gorm.io/gorm"
)

type Challenge struct {
	gorm.Model
	ChallengeID    string           `json:"challenge_id" gorm:"index:challenge_id"`
	CreatedAt      common.Timestamp `json:"created_at" gorm:"created_at"`
	AllocationID   string           `json:"allocation_id" gorm:"allocation_id"`
	BlobberID      string           `json:"blobber_id" gorm:"blobber_id"`
	ValidatorsID   string           `json:"validators_id" gorm:"validators_id"`
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

func (edb *EventDb) GetChallenge(challengeID string) (*Challenge, error) {
	var ch Challenge

	result := edb.Store.Get().Model(&Challenge{}).Where(&Challenge{ChallengeID: challengeID}).First(&ch)
	if result.Error != nil {
		return nil, fmt.Errorf("error retriving Challenge node with ID %v; error: %v", challengeID, result.Error)
	}

	return &ch, nil
}

func (edb *EventDb) GetOpenChallengesForBlobber(blobberID string, now, cct common.Timestamp) ([]*Challenge, error) {
	var chs []*Challenge
	expiry := now - cct

	result := edb.Store.Get().Model(&Challenge{}).
		Where("blobber_id = ? AND responded = ? AND created_at >= ?",
			blobberID, false, expiry).Find(&chs)
	if result.Error != nil {
		return nil, fmt.Errorf("error retriving open Challenges with blobberid %v; error: %v",
			blobberID, result.Error)
	}

	return chs, nil
}

func (edb *EventDb) GetChallengeForBlobber(blobberID, challengeID string) (*Challenge, error) {
	var ch *Challenge

	result := edb.Store.Get().Model(&Challenge{}).
		Where("challenge_id = ? AND blobber_id = ?", challengeID, blobberID).First(&ch)
	if result.Error != nil {
		return nil, fmt.Errorf("error retriving Challenge with blobberid %v challengeid: %v; error: %v",
			blobberID, challengeID, result.Error)
	}

	return ch, nil
}

func (edb *EventDb) addChallenge(ch *Challenge) error {
	exists, err := ch.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("challenge already exists in db. cannot add")
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
