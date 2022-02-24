package event

import (
	"github.com/guregu/null"
	"gorm.io/gorm"
)

type AllocationPool struct {
	gorm.Model
	AllocationID  string `gorm:"uniqueIndex"`
	TransactionId string
	UserID        string
	Balance       int64
	Blobbers      []BlobberPool `gorm:"foreignKey:AllocationPoolID;references:AllocationID"`
	IsWritePool   bool
	ZcnBalance    int64
	ZcnID         string
	ExpireAt      int64
}

func (edb *EventDb) addAllocationPool(allocationPool AllocationPool) error {
	return edb.Get().Create(&allocationPool).Error
}

type AllocationPoolFilter struct {
	gorm.Model
	AllocationID  null.String
	TransactionId null.String
	UserID        null.String
	Balance       null.Int
	ExpireAt      null.Int
	IsWritePool   null.Bool
}

func (edb *EventDb) GetAllocationPoolWithFilterAndPagination(filter AllocationPoolFilter, offset, limit int) ([]AllocationPool, error) {
	query := edb.Get().Model(&AllocationPool{}).Where(&filter)
	if offset != -1 {
		query = query.Offset(offset)
	}
	if limit != -1 {
		query = query.Limit(limit)
	}
	var allocationPools []AllocationPool
	return allocationPools, query.Scan(&allocationPools).Error
}
