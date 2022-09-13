package event

import (
	"fmt"
	"time"

	"0chain.net/chaincore/currency"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// swagger:model WriteMarker
type WriteMarker struct {
	gorm.Model
	ClientID      string `json:"client_id"`
	BlobberID     string `json:"blobber_id"`
	AllocationID  string `json:"allocation_id" gorm:"index:idx_walloc_block,priority:1;index:idx_walloc_file,priority:2"` //used in alloc_write_marker_count, alloc_written_size
	TransactionID string `json:"transaction_id"`

	AllocationRoot         string `json:"allocation_root"`
	PreviousAllocationRoot string `json:"previous_allocation_root"`
	Size                   int64  `json:"size"`
	Timestamp              int64  `json:"timestamp"`
	Signature              string `json:"signature"`
	BlockNumber            int64  `json:"block_number" gorm:"index:idx_wblocknum,priority:1;index:idx_walloc_block,priority:2"` //used in alloc_written_size

	// file info
	LookupHash  string `json:"lookup_hash" gorm:"index:idx_wlookup,priority:1"`
	Name        string `json:"name" gorm:"index:idx_wname,priority:1;idx_walloc_file,priority:1"`
	ContentHash string `json:"content_hash" gorm:"index:idx_wcontent,priority:1"`

	MovedTokens currency.Coin `json:"-" gorm:"-"`

	//ref
	User       User       `gorm:"foreignKey:ClientID;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Allocation Allocation `gorm:"references:AllocationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (w *WriteMarker) AfterCreate(tx *gorm.DB) error {
	// update blobber alloc stat
	vs := map[string]interface{}{
		"used":       gorm.Expr("blobbers.used + excluded.used"),
		"saved_data": gorm.Expr("blobbers.saved_data + excluded.saved_data"),
	}
	if err := tx.Model(&Blobber{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "blobber_id"}},
		DoUpdates: clause.Assignments(vs),
	}).Create(&Blobber{BlobberID: w.BlobberID, Used: w.Size, SavedData: w.Size}).Error; err != nil {
		return err
	}

	// update allocation stat
	vs = map[string]interface{}{
		"used_size":  gorm.Expr("allocations.used_size + excluded.used_size"),
		"num_writes": gorm.Expr("allocations.num_writes + 1"),
	}

	alloc := Allocation{
		AllocationID: w.AllocationID,
		UsedSize:     w.Size,
	}

	if w.Size > 0 {
		alloc.MovedToChallenge = w.MovedTokens
		vs["moved_to_challenge"] = gorm.Expr("allocations.moved_to_challenge + excluded.moved_to_challenge")
		vs["write_pool"] = gorm.Expr("allocations.write_pool + excluded.moved_to_challenge")
	} else if w.Size < 0 {
		alloc.MovedBack = w.MovedTokens
		vs["moved_back"] = gorm.Expr("allocations.moved_back + excluded.moved_back")
		vs["write_pool"] = gorm.Expr("allocations.write_pool - excluded.moved_back")
	}

	return tx.Model(&Allocation{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "allocation_id"}},
		DoUpdates: clause.Assignments(vs),
	}).Create(&alloc).Error
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

func (edb *EventDb) GetAllocationWrittenSizeInBlocks(startBlockNum, endBlockNum int64) (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&WriteMarker{}).
		Select("COALESCE(SUM(size),0)").
		Where("block_number > ? AND block_number < ?", startBlockNum, endBlockNum).
		Find(&total).Error
}

func (edb *EventDb) GetWriteMarkerCount(allocationID string) (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&WriteMarker{}).Where("allocation_id = ?", allocationID).Count(&total).Error
}

func (edb *EventDb) GetWriteMarkers(limit common.Pagination) ([]WriteMarker, error) {
	var wm []WriteMarker
	return wm, edb.Get().Model(&WriteMarker{}).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   limit.IsDescending,
	}).Scan(&wm).Error
}

func (edb *EventDb) GetWriteMarkersForAllocationID(allocationID string, limit common.Pagination) ([]WriteMarker, error) {
	var wms []WriteMarker
	result := edb.Store.Get().
		Model(&WriteMarker{}).
		Where(&WriteMarker{AllocationID: allocationID}).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   limit.IsDescending,
	}).Scan(&wms)
	return wms, result.Error
}

func (edb *EventDb) GetWriteMarkersForAllocationFile(allocationID string, filename string, limit common.Pagination) ([]WriteMarker, error) {
	var wms []WriteMarker
	result := edb.Store.Get().
		Model(&WriteMarker{}).
		Where(&WriteMarker{AllocationID: allocationID, Name: filename}).Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   limit.IsDescending,
	}).Scan(&wms)
	return wms, result.Error
}

func (edb *EventDb) addWriteMarkers(wms []WriteMarker) error {
	ts := time.Now()
	defer func() {
		du := time.Since(ts)
		if du.Milliseconds() > 50 {
			logging.Logger.Debug("event db - add write markers slow",
				zap.Any("duration", du),
				zap.Int("num", len(wms)))
		}
	}()
	return edb.Store.Get().Create(&wms).Error
}

func (edb *EventDb) GetWriteMarkersByFilters(filters WriteMarker, selectString string, limit common.Pagination) ([]interface{}, error) {
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
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "block_number"},
			Desc:   limit.IsDescending,
		}).
		Scan(&wm)

	return wm, res.Error
}
