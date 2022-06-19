package event

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// swagger:model WriteMarker
type WriteMarker struct {
	gorm.Model

	// foreign keys
	// todo: as user(ID), allocation(ID) and transaction(ID) tables are created, enable it
	ClientID      string `json:"client_id"`
	BlobberID     string `json:"blobber_id"`
	AllocationID  string `json:"allocation_id" gorm:"index:idx_walloc_block,priority:1;index:idx_walloc_file,priority:2"` //used in alloc_write_marker_count, alloc_written_size
	TransactionID string `json:"transaction_id"`

	AllocationRoot         string `json:"allocation_root"`
	PreviousAllocationRoot string `json:"previous_allocation_root"`
	Size                   int64  `json:"size"`
	Timestamp              int64  `json:"timestamp"`
	Signature              string `json:"signature"`
	BlockNumber            int64  `json:"block_number" gorm:"index:idx_walloc_block,priority:2"` //used in alloc_written_size

	// file info
	LookupHash  string `json:"lookup_hash" gorm:"index:idx_wlookup,priority:1"`
	Name        string `json:"name" gorm:"index:idx_wname,priority:1;idx_walloc_file,priority:1"`
	ContentHash string `json:"content_hash" gorm:"index:idx_wcontent,priority:1"`
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
	return total, edb.Store.Get().Model(&WriteMarker{}).
		Select("sum(size)").
		Where(&WriteMarker{AllocationID: allocationID, BlockNumber: blockNumber}).
		Find(&total).Error
}

func (edb *EventDb) GetWriteMarkerCount(allocationID string) (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&WriteMarker{}).Where("allocation_id = ?", allocationID).Count(&total).Error
}

func (edb *EventDb) GetWriteMarkers(limit Pagination) ([]WriteMarker, error) {
	var wm []WriteMarker
	return wm, edb.Get().Model(&WriteMarker{}).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   limit.IsDescending,
	}).Scan(&wm).Error
}

func (edb *EventDb) GetWriteMarkersForAllocationID(allocationID string, limit Pagination) ([]WriteMarker, error) {
	var wms []WriteMarker
	result := edb.Store.Get().
		Model(&WriteMarker{}).
		Where(&WriteMarker{AllocationID: allocationID}).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   limit.IsDescending,
	}).Scan(&wms)
	return wms, result.Error
}

func (edb *EventDb) GetWriteMarkersForAllocationFile(allocationID string, filename string, limit Pagination) ([]WriteMarker, error) {
	var wms []WriteMarker
	result := edb.Store.Get().
		Model(&WriteMarker{}).
		Where(&WriteMarker{AllocationID: allocationID, Name: filename}).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   limit.IsDescending,
	}).Scan(&wms)
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

func (edb *EventDb) GetWriteMarkersByFilters(filters WriteMarker, selectString string, limit Pagination) ([]interface{}, error) {
	var wm []interface{}

	edbRef := edb.Store.Get()
	if len(selectString) > 0 {
		edbRef = edbRef.Select(selectString)
	}

	res := edbRef.
		Model(WriteMarker{}).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Where(filters).
		Scan(&wm)

	return wm, res.Error
}
