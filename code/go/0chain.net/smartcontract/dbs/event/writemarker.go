package event

import (
	"fmt"
	"gorm.io/gorm"
)

type WriteMarker struct {
	gorm.Model

	// foreign keys
	// todo: as user(ID), allocation(ID) and transaction(ID) tables are created, enable it
	ClientID      string
	BlobberID     string
	AllocationID  string
	TransactionID string

	AllocationRoot         string
	PreviousAllocationRoot string
	Size                   int64
	Timestamp              int64
	Signature              string
	BlockNumber            int64
}

func (edb *EventDb) overwriteWriteMarker(wm WriteMarker) error {
	result := edb.Store.Get().
		Model(&WriteMarker{}).
		Where(&WriteMarker{TransactionID: wm.TransactionID}).
		Updates(&wm)
	return result.Error
}

func (edb *EventDb) addOrOverwriteWriteMarker(wm WriteMarker) error {
	exists, err := wm.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteWriteMarker(wm)
	}

	result := edb.Store.Get().Create(&wm)
	return result.Error
}

func (wm *WriteMarker) exists(edb *EventDb) (bool, error) {
	var count int64
	result := edb.Get().
		Model(&WriteMarker{}).
		Where(&WriteMarker{TransactionID: wm.TransactionID}).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("error searching for write marker txn: %v, error %v",
			wm.TransactionID, result.Error)
	}
	return count > 0, nil
}
