package event

import (
	"fmt"
	"gorm.io/gorm"
)

type WriteMarker struct {
	gorm.Model

	// foreign keys
	// todo: as user(ID), allocation(ID) and transaction(ID) tables are created, enable it
	ClientID      string `json:"client_id"`
	BlobberID     string `json:"blobber_id"`
	AllocationID  string `json:"allocation_id"`
	TransactionID string `json:"transaction_id"`

	AllocationRoot         string `json:"allocation_root"`
	PreviousAllocationRoot string `json:"previous_allocation_root"`
	Size                   int64  `json:"size"`
	Timestamp              int64  `json:"timestamp"`
	Signature              string `json:"signature"`
	BlockNumber            int64  `json:"block_number"`
}

func (edb *EventDb) GetWriteMarker(txnID string) (*WriteMarker, error) {
	var wm WriteMarker

	result := edb.Store.Get().
		Model(&WriteMarker{}).
		Where(&WriteMarker{TransactionID: txnID}).
		First(&wm)
	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving write marker (txn)%v, error %v",
			txnID, result.Error)
	}

	return &wm, nil
}

func (edb *EventDb) GetWriteMarkersForAllocationID(allocationID string) (*[]WriteMarker, error) {
	var wms []WriteMarker
	result := edb.Store.Get().
		Model(&WriteMarker{}).
		Where(&WriteMarker{AllocationID: allocationID}).
		Find(&wms)
	return &wms, result.Error
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
