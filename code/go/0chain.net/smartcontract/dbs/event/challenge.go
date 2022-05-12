package event

import (
	"fmt"

	"0chain.net/smartcontract/dbs"

	"0chain.net/core/common"
	"gorm.io/gorm"
)

type Challenge struct {
	gorm.Model
	ChallengeID    string           `json:"challenge_id" gorm:"index:challenge_id,unique"`
	CreatedAt      common.Timestamp `json:"created_at" gorm:"index:idx_open_challenge,priority:1"`
	AllocationID   string           `json:"allocation_id"`
	BlobberID      string           `json:"blobber_id" gorm:"index:idx_open_challenge,priority:2"`
	ValidatorsID   string           `json:"validators_id"`
	Seed           int64            `json:"seed"`
	AllocationRoot string           `json:"allocation_root"`
	Responded      bool             `json:"responded" gorm:"index:idx_open_challenge,priority:3"`
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
		Where("created_at > ? AND blobber_id = ? AND responded = ?",
			expiry, blobberID, false).Order("created_at asc").Find(&chs)
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
	result := edb.Store.Get().Create(&ch)

	return result.Error
}

func (edb *EventDb) updateChallenge(updates dbs.DbUpdates) error {
	result := edb.Store.Get().
		Model(&Challenge{}).
		Where(&Challenge{ChallengeID: updates.Id}).
		Updates(updates.Updates)
	return result.Error
}
