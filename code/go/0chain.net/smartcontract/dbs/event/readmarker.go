package event

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"0chain.net/core/common"
)

type ReadMarker struct {
	gorm.Model
	ClientID      string  `json:"client_id"`
	BlobberID     string  `json:"blobber_id"`
	AllocationID  string  `json:"allocation_id"`
	TransactionID string  `json:"transaction_id"`
	OwnerID       string  `json:"owner_id"`
	Timestamp     int64   `json:"timestamp"`
	ReadSize      int64   `json:"read_size"`
	ReadSizeInGB  float64 `json:"read_size_in_gb"`
	Signature     string  `json:"signature"`
	PayerID       string  `json:"payer_id"`
	AuthTicket    string  `json:"auth_ticket"`
	BlockNumber   int64   `json:"block_number"`
}

func (edb *EventDb) GetReadMarkersFromQueryPaginated(query ReadMarker, offset, limit int, isDescending bool) ([]ReadMarker, error) {
	queryBuilder := edb.Store.Get().
		Model(&ReadMarker{}).
		Where(query)
	if offset > 0 {
		queryBuilder = queryBuilder.Offset(offset)
	}
	if limit > 0 {
		queryBuilder = queryBuilder.Limit(limit)
	}
	queryBuilder.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   isDescending,
	})
	var rms []ReadMarker
	return rms, queryBuilder.Scan(&rms).Error
}

func (edb EventDb) CountReadMarkersFromQuery(query *ReadMarker) (count int64, err error) {

	if query == nil {
		err = common.NewError("count_read_markers", "empty query")
		return
	}

	result := edb.Get().
		Model(&ReadMarker{}).
		Where(query).
		Count(&count)

	if result.Error != nil {
		err = errors.New("error searching for read marker")
		return
	}

	err = nil
	return
}

func (edb *EventDb) GetReadDataSizeForAllocation(allocationID string, blockNumber int) (int64, error) {
	var total int64
	return total, edb.Get().Model(&ReadMarker{}).Select("sum(read_size) as total").Where("allocation_id = ? AND block_number > ?", allocationID, blockNumber).Scan(&total).Error
}

func (edb *EventDb) overwriteReadMarker(rm ReadMarker) error {
	result := edb.Store.Get().
		Model(&ReadMarker{}).
		Where(&ReadMarker{TransactionID: rm.TransactionID}).
		Updates(&rm)
	return result.Error
}

func (edb *EventDb) addOrOverwriteReadMarker(rm ReadMarker) error {
	exists, err := rm.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteReadMarker(rm)
	}

	result := edb.Store.Get().Create(&rm)
	return result.Error
}

func (rm *ReadMarker) exists(edb *EventDb) (bool, error) {
	var readMarker ReadMarker
	result := edb.Get().
		Model(&ReadMarker{}).
		Where(&ReadMarker{TransactionID: rm.TransactionID}).
		Take(&readMarker)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for read marker txn: %v, error %v",
			rm.TransactionID, result.Error)
	}
	return true, nil
}
