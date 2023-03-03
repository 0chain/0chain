package event

import (
	"fmt"
	"time"

	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

// swagger:model WriteMarker
type WriteMarker struct {
	model.UpdatableModel
	ClientID      string `json:"client_id"`
	BlobberID     string `json:"blobber_id"`
	AllocationID  string `json:"allocation_id" gorm:"index:idx_walloc_block,priority:1;index:idx_walloc_file,priority:2"` //used in alloc_write_marker_count, alloc_written_size
	TransactionID string `json:"transaction_id" gorm:"uniqueIndex"`

	AllocationRoot         string `json:"allocation_root"`
	PreviousAllocationRoot string `json:"previous_allocation_root"`
	FileMetaRoot           string `json:"file_meta_root"`
	Size                   int64  `json:"size"`
	Timestamp              int64  `json:"timestamp"`
	Signature              string `json:"signature"`
	BlockNumber            int64  `json:"block_number" gorm:"index:idx_wblocknum,priority:1;index:idx_walloc_block,priority:2"` //used in alloc_written_size

	MovedTokens currency.Coin `json:"-" gorm:"-"`

	//ref
	User       User       `gorm:"foreignKey:ClientID;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Allocation Allocation `gorm:"references:AllocationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
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

func (edb *EventDb) GetWriteMarkerCount(allocationID string) (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&WriteMarker{}).Where("allocation_id = ?", allocationID).Count(&total).Error
}

func (edb *EventDb) GetWriteMarkers(limit common.Pagination) ([]WriteMarker, error) {
	var wm []WriteMarker
	return wm, edb.
		Get().
		Model(&WriteMarker{}).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
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

func (edb *EventDb) addWriteMarkers(wms []WriteMarker) error {
	ts := time.Now()
	defer func() {
		du := time.Since(ts)
		if du.Milliseconds() > 50 {
			logging.Logger.Debug("event db - add write markers slow",
				zap.Duration("duration", du),
				zap.Int("num", len(wms)))
		}
	}()
	return edb.Store.Get().Create(&wms).Error
}

func mergeAddWriteMarkerEvents() *eventsMergerImpl[WriteMarker] {
	return newEventsMerger[WriteMarker](TagAddWriteMarker)
}

func (edb *EventDb) GetWriteMakerFromFilter(filter, value string) (WriteMarker, error) {
	var wm WriteMarker
	result := edb.Store.Get().
		Model(&WriteMarker{}).
		Where(filter+" = ?", value).
		First(&wm)
	return wm, result.Error
}

func (edb *EventDb) GetWriteMakersFromFilter(filter, value string, limit common.Pagination) ([]WriteMarker, error) {
	var wms []WriteMarker
	result := edb.Store.Get().
		Model(&WriteMarker{}).
		Where(filter+" = ?", value).
		Offset(limit.Offset).Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   limit.IsDescending,
		}).Scan(&wms)
	return wms, result.Error
}

func (edb *EventDb) GetWriteMarkersByFilters(filters WriteMarker, selectString string, limit common.Pagination) ([]interface{}, error) {
	var wm []interface{}

	edbRef := edb.Store.Get()
	if len(selectString) > 0 {
		edbRef = edbRef.Select(selectString)
	}

	res := edbRef.
		Joins("User").
		Joins("Allocation").
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
