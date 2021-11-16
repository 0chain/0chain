package event

import (
	"encoding/json"
	"fmt"

	"0chain.net/core/common"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	CreditCards []CreditCard
}

type CreditCard struct {
	gorm.Model
	Number string
	UserID uint
}

type BlobberChallenge struct {
	gorm.Model
	BlobberID  string      //`gorm:"primary_key"`
	Challenges []Challenge //`gorm:"foreignKey:BlobberID;references:BlobberID"`
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
	var count int64
	result := edb.Store.Get().
		Model(&BlobberChallenge{}).
		Where("blobber_id", blobberId).
		Count(&count)

	if count == 0 {
		bc := BlobberChallenge{
			BlobberID: blobberId,
		}
		err := bc.add(edb)
		if err != nil {
			return err
		}
	}
	result = edb.Store.Get().
		Model(&BlobberChallenge{}).
		Find(&BlobberChallengeId{}).
		Where("blobber_id", blobberId).
		First(&bci)
	if result.RowsAffected == 0 {
		return fmt.Errorf("cannot create blobber challenge %v, db error %v",
			blobberId, result.Error)
	}

	return nil
}

type Challenge struct {
	gorm.Model
	BlobberChallengeID uint
	BlobberID          string           `json:"blobber_id"`
	Created            common.Timestamp `json:"created"`
	ChallengeID        string           `json:"challenge_id"`
	PrevID             string           `json:"prev_id"`
	//Validators         []ValidationNode `json:"validators" gorm:"foreignKey:ChallengeID;references:ChallengeID"`
	Validators     []ValidationNode `json:"validators"`
	RandomNumber   int64            `json:"seed"`
	AllocationID   string           `json:"allocation_id"`
	AllocationRoot string           `json:"allocation_root"`
	//Response           Response         `json:"challenge_response,omitempty" gorm:"foreignKey:ChallengeID;references:ChallengeID"`
	Response Response `json:"challenge_response,omitempty"`
	//LatestCompletedChallenge bool             `json:"-"`
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

	result := edb.Store.Get().Create(ch)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

type Response struct {
	gorm.Model
	ChallengeId uint
	ResponseID  string `json:"response_id"`
	//ValidationTickets []ValidationTicket `json:"validation_tickets" gorm:"foreignKey:ResponseID;references:ResponseID"`
	ValidationTickets []ValidationTicket `json:"validation_tickets"`
}

type ValidationNode struct {
	gorm.Model
	ChallengeId uint
	ValidatorID string `json:"id" gorm:"primary_key"`
	BaseURL     string `json:"url"`
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

func (edb *EventDb) migrateChallengeTable() error {
	var err error
	//err := edb.Store.Get().AutoMigrate(&ValidationTicket{})
	if err != nil {
		return err
	}
	//err = edb.Store.Get().AutoMigrate(&Response{})
	err = edb.Store.Get().Migrator().DropTable(&CreditCard{})
	if err != nil {
		return err
	}
	err = edb.Store.Get().Migrator().DropTable(&User{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().AutoMigrate(&User{}, &CreditCard{})
	if err != nil {
		return err
	}
	err = edb.Store.Get().Migrator().CreateConstraint(&User{}, "CreditCards")
	if err != nil {
		return err
	}
	err = edb.Store.Get().Migrator().CreateConstraint(&User{}, "fk_users_credit_cards")
	if err != nil {
		return err
	}
	err = edb.Store.Get().AutoMigrate(&User{}, &CreditCard{})
	if err != nil {
		return err
	}

	//	err = edb.Store.Get().AutoMigrate(&ValidationNode{})

	//err = edb.Store.Get().AutoMigrate(&Challenge{}, &BlobberChallenge{})
	if err != nil {
		return err
	}

	err = edb.Store.Get().Migrator().CreateConstraint(&BlobberChallenge{}, "Challenges")
	if err != nil {
		return err
	}
	err = edb.Store.Get().Migrator().
		//CreateConstraint(&BlobberChallenge{}, "fk_blobber_challenge_challenges;ref_blobber_id")
		CreateConstraint(&BlobberChallenge{}, "fk_blobber_challenges_challenges")
	if err != nil {
		return err
	}

	err = edb.Store.Get().AutoMigrate(&Challenge{}, &BlobberChallenge{})
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
	err = edb.Store.Get().Migrator().DropTable(&Response{})
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
