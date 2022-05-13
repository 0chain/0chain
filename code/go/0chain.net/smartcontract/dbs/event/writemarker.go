package event

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	// file info
	LookupHash  string `json:"lookup_hash"`
	Name        string `json:"name"`
	ContentHash string `json:"content_hash"`
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

func (edb *EventDb) GetAllocationWrittenSizeInLastNBlocks(blockNumber int64, allocationID string) (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&WriteMarker{}).Select("sum(size)").Where("block_number > ?", blockNumber).Where("allocation_id = ?", allocationID).Find(&total).Error
}

func (edb *EventDb) GetWriteMarkerCount(allocationID string) (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&WriteMarker{}).Where("allocation_id = ?", allocationID).Count(&total).Error
}

func (edb *EventDb) GetWriteMarkers(offset, limit int, isDescending bool) ([]WriteMarker, error) {
	var wm []WriteMarker
	return wm, edb.Get().Model(&WriteMarker{}).Offset(offset).Limit(limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   isDescending,
	}).Scan(&wm).Error
}

func (edb *EventDb) GetWriteMarkersForAllocationID(allocationID string) ([]WriteMarker, error) {
	var wms []WriteMarker
	result := edb.Store.Get().
		Model(&WriteMarker{}).
		Where(&WriteMarker{AllocationID: allocationID}).
		Find(&wms)
	return wms, result.Error
}

func (edb *EventDb) GetWriteMarkersForAllocationFile(allocationID string, filename string) ([]WriteMarker, error) {
	var wms []WriteMarker
	result := edb.Store.Get().
		Model(&WriteMarker{}).
		Where(&WriteMarker{AllocationID: allocationID, Name: filename}).
		Find(&wms)
	return wms, result.Error
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

	var writeMarker WriteMarker

	result := edb.Get().
		Model(&WriteMarker{}).
		Where(&WriteMarker{TransactionID: wm.TransactionID}).
		Take(&writeMarker)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for write marker txn: %v, error %v",
			wm.TransactionID, result.Error)
	}
	return true, nil
}
