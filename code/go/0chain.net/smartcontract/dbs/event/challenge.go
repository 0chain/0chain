package event

import (
	"encoding/json"
	"fmt"

	"0chain.net/core/common"
	"gorm.io/gorm"
)

type BlobberChallenge struct {
	gorm.Model
	BlobberID  string      `gorm:"primary_key"`
	Challenges []Challenge `gorm:"foreignKey:BlobberID;references:BlobberID"`
	//ChallengeIds             []string              `json:"challenge_ids" gorm:"-"`
	//ChallengeMap             map[string]*Challenge `json:"-" gorm:"-"`
	//LatestCompletedChallenge *Challenge            `json:"latest_completed_challenge" gorm:"-"`
}

func (bc *BlobberChallenge) add(edb *EventDb) error {
	result := edb.Store.Get().Create(bc)
	return result.Error
}

type BlobberChallengeId struct {
	ID        int
	BlobberID string
}

func (bci *BlobberChallengeId) getOrCreate(edb *EventDb, blobberId string) error {
	result := edb.Store.Get().
		Model(&BlobberChallenge{}).
		Find(&BlobberChallengeId{}).
		Where("blobber_id", blobberId).
		First(&bci)

	if result.RowsAffected == 0 {
		bc := BlobberChallenge{
			BlobberID: blobberId,
		}
		err := bc.add(edb)
		if err != nil {
			return err
		}
		result := edb.Store.Get().
			Model(&BlobberChallenge{}).
			Find(&BlobberChallengeId{}).
			Where("blobber_id", blobberId).
			First(&bci)
		if result.RowsAffected == 0 {
			return fmt.Errorf("cannot create blobber challenge %v, db error %v",
				blobberId, result.Error)
		}
	}
	return nil
}

type Challenge struct {
	gorm.Model
	BlobberID                string             `json:"blobber_id"`
	Created                  common.Timestamp   `json:"created"`
	ChallengeID              string             `json:"challenge_id" gorm:"primary_key"`
	PrevID                   string             `json:"prev_id"`
	Validators               []ValidationNode   `json:"validators" gorm:"ForeignKey:storage_challenge_id"`
	RandomNumber             int64              `json:"seed"`
	AllocationID             string             `json:"allocation_id"`
	AllocationRoot           string             `json:"allocation_root"`
	Response                 *ChallengeResponse `json:"challenge_response,omitempty" gorm:"ForeignKey:storage_challenge_id"`
	LatestCompletedChallenge bool               `json:"-"`
}

func (ch *Challenge) add(edb *EventDb, data []byte) error {
	err := json.Unmarshal(data, ch)
	if err != nil {
		return err
	}

	bc := BlobberChallengeId{}
	if err := bc.getOrCreate(edb, ch.BlobberID); err != nil {
		return err
	}

	edb.Store.Get().Create(ch)

	return nil
}

type ChallengeResponse struct {
	gorm.Model
	ChallengeID       string             `json:"storage_challenge_id" gorm:"storage_challenge_id"`
	ResponseID        string             `json:"challenge_id"`
	ValidationTickets []ValidationTicket `json:"validation_tickets" gorm:"ForeignKey:challenge_response_id"`
}

type ValidationNode struct {
	gorm.Model
	ChallengeID string `json:"storage_challenge_id" gorm:"storage_challenge_id"`
	ValidatorID string `json:"id"`
	BaseURL     string `json:"url"`
}

type ValidationTicket struct {
	gorm.Model
	ChallengeResponseId string           `json:"challenge_response_id" gorm:"challenge_response_id"`
	ChallengeID         string           `json:"challenge_id"`
	BlobberID           string           `json:"blobber_id"`
	ValidatorID         string           `json:"validator_id"`
	ValidatorKey        string           `json:"validator_key"`
	Result              bool             `json:"success"`
	Message             string           `json:"message"`
	MessageCode         string           `json:"message_code"`
	Timestamp           common.Timestamp `json:"timestamp"`
	Signature           string           `json:"signature"`
}

func (edb *EventDb) migrateChallengeTable() error {
	err := edb.Store.Get().AutoMigrate(&ValidationTicket{})
	if err != nil {
		return err
	}
	err = edb.Store.Get().AutoMigrate(&ChallengeResponse{})
	if err != nil {
		return err
	}
	err = edb.Store.Get().AutoMigrate(&ValidationNode{})
	if err != nil {
		return err
	}
	err = edb.Store.Get().AutoMigrate(&Challenge{})
	if err != nil {
		return err
	}
	err = edb.Store.Get().AutoMigrate(&BlobberChallenge{})
	if err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) dropChallengeTable() error {
	err := edb.Store.Get().Migrator().DropTable(&ValidationTicket{})
	if err != nil {
		return err
	}
	err = edb.Store.Get().Migrator().DropTable(&ChallengeResponse{})
	if err != nil {
		return err
	}
	err = edb.Store.Get().Migrator().DropTable(&ValidationNode{})
	if err != nil {
		return err
	}
	err = edb.Store.Get().Migrator().DropTable(&Challenge{})
	if err != nil {
		return err
	}
	err = edb.Store.Get().Migrator().DropTable(&BlobberChallenge{})
	if err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) removeChallenge(challengeId string) error {
	result := edb.Store.Get().Delete(&Challenge{}, "challenge_id", challengeId)
	return result.Error
}

/*








































 */
