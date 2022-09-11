package event

import (
	"errors"

	common2 "0chain.net/smartcontract/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"0chain.net/core/common"
)

// swagger:model ReadMarker
type ReadMarker struct {
	gorm.Model
	ClientID      string  `json:"client_id"`
	BlobberID     string  `json:"blobber_id"`
	AllocationID  string  `json:"allocation_id" gorm:"index:idx_ralloc_block,priority:1;index:idx_rauth_alloc,priority:2"` //used in alloc_read_size, used in readmarkers
	TransactionID string  `json:"transaction_id"`
	OwnerID       string  `json:"owner_id"`
	Timestamp     int64   `json:"timestamp"`
	ReadCounter   int64   `json:"read_counter"`
	ReadSize      float64 `json:"read_size"`
	Signature     string  `json:"signature"`
	PayerID       string  `json:"payer_id"`
	AuthTicket    string  `json:"auth_ticket" gorm:"index:idx_rauth_alloc,priority:1"`   //used in readmarkers
	BlockNumber   int64   `json:"block_number" gorm:"index:idx_ralloc_block,priority:2"` //used in alloc_read_size
	//ref
	User       User       `gorm:"foreignKey:ClientID;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Owner      User       `gorm:"foreignKey:OwnerID;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Allocation Allocation `gorm:"references:AllocationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (edb *EventDb) GetDataReadFromAllocationForLastNBlocks(blockNumber int64, allocationID string) (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&ReadMarker{}).
		Select("sum(read_size)").
		Where(&ReadMarker{AllocationID: allocationID, BlockNumber: blockNumber}).
		Find(&total).Error
}

func (edb *EventDb) GetReadMarkersFromQueryPaginated(query ReadMarker, limit common2.Pagination) ([]ReadMarker, error) {
	queryBuilder := edb.Store.Get().
		Model(&ReadMarker{}).
		Where(query).Offset(limit.Offset).Limit(limit.Limit)

	queryBuilder.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   limit.IsDescending,
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

func (edb *EventDb) overwriteReadMarker(rm ReadMarker) error {
	result := edb.Store.Get().
		Model(&ReadMarker{}).
		Where(&ReadMarker{TransactionID: rm.TransactionID}).
		Updates(&rm)
	return result.Error
}

func (edb *EventDb) addOrOverwriteReadMarker(rms []ReadMarker) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "transaction_id"}},
		UpdateAll: true,
	}).Create(&rms).Error
}
