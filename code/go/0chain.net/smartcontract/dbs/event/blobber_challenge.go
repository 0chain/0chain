package event

import (
	"fmt"

	"gorm.io/gorm"
)

type BlobberChallenge struct {
	gorm.Model
	BlobberID  string      `json:"blobber_id" gorm:"uniqueIndex"`
	Url        string      `json:"url"`
	Challenges []Challenge `json:"challenges"`
}

type BlobberChallengeId struct {
	ID        uint
	BlobberID string
	Url       string
}

func (bc *BlobberChallenge) add(edb *EventDb) error {
	result := edb.Store.Get().Create(bc)
	return result.Error
}

func (_ *BlobberChallenge) exists(edb *EventDb, blobberId string) (bool, error) {
	var count int64
	result := edb.Store.Get().
		Model(&BlobberChallenge{}).
		Where("blobber_id", blobberId).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("error counting blobber challenge, blobber %v, error %v",
			blobberId, result.Error)
	}
	return count > 0, nil
}

func (bci *BlobberChallengeId) getOrCreate(edb *EventDb) error {
	exists, err := (&BlobberChallenge{}).exists(edb, bci.BlobberID)
	if err != nil {
		return err
	}

	if !exists {
		bc := BlobberChallenge{
			BlobberID: bci.BlobberID,
			Url:       bci.Url,
		}
		err := bc.add(edb)
		if err != nil {
			return err
		}
	}
	result := edb.Store.Get().
		Model(&BlobberChallenge{}).
		Find(&BlobberChallengeId{}).
		Where(&BlobberChallenge{BlobberID: bci.BlobberID}).
		First(&bci)
	if result.RowsAffected == 0 {
		return fmt.Errorf("cannot create blobber challenge, %v", result.Error)
	}

	return nil
}

func (edb *EventDb) GetBlobberChallenges(blobberId string) (*BlobberChallenge, error) {
	exists, err := (&BlobberChallenge{}).exists(edb, blobberId)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

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
