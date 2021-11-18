package event

import (
	"fmt"

	"gorm.io/gorm"
)

type BlobberChallenge struct {
	gorm.Model
	BlobberID  string      `json:"blobber_id" gorm:"uniqueIndex"`
	Challenges []Challenge `json:"challenges"`
}

type BlobberChallengeId struct {
	ID        uint
	BlobberID string
}

func (bc *BlobberChallenge) add(edb *EventDb) error {
	result := edb.Store.Get().Create(bc)
	return result.Error
}

func (bci *BlobberChallengeId) getOrCreate(edb *EventDb, blobberId string) error {
	var count int64
	result := edb.Store.Get().
		Model(&BlobberChallenge{}).
		Where("blobber_id", blobberId).
		Count(&count)
	if result.Error != nil {
		return fmt.Errorf("error counting blobber challenge, blobber %v, error %v",
			blobberId, result.Error)
	}

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

func (edb *EventDb) GetBlobberChallenges(blobberId string) (*BlobberChallenge, error) {
	var bc BlobberChallenge
	result := edb.Store.Get().
		Model(&BlobberChallenge{}).
		Find(&BlobberChallengeId{}).
		Where("blobber_id", blobberId).
		First(&bc)
	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving blobber challenge, blobber id %v, error %v",
			blobberId, result.Error)
	}

	challenges, err := edb.getChallenges(blobberId)
	if err != nil {
		return nil, err
	}
	bc.Challenges = challenges

	return &bc, nil
}

func (edb *EventDb) createChallengeTable() error {
	return edb.Store.Get().Migrator().CreateTable(
		&BlobberChallenge{},
		&Challenge{},
		//&Response{},
		&ValidationNode{},
		//&ValidationTicket{},
	)
}

func (edb *EventDb) dropChallengeTable() error {
	var err error
	//err := edb.Store.Get().Migrator().DropTable(&ValidationTicket{})
	if err != nil {
		return err
	}
	//err = edb.Store.Get().Migrator().DropTable(&Response{})
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
