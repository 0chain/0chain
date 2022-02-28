package event

import (
	"github.com/guregu/null"
	"gorm.io/gorm"
)

type ReadAllocationPool struct {
	gorm.Model
	AllocationID  string `gorm:"uniqueIndex"`
	TransactionId string
	UserID        string
	Balance       int64
	Blobbers      []BlobberPool `gorm:"foreignKey:ReadAllocationPoolID;references:AllocationID"`
	ZcnBalance    int64
	ZcnID         string
	ExpireAt      int64
}

type ReadAllocationPoolFilter struct {
	gorm.Model
	AllocationID  null.String
	TransactionId null.String
	UserID        null.String
	Balance       null.Int
	ExpireAt      null.Int
}

func (edb *EventDb) addReadAllocationPool(readAllocationPool ReadAllocationPool) error {
	return edb.Get().Model(&ReadAllocationPool{}).Create(&readAllocationPool).Error
}

func (edb *EventDb) GetReadAllocationPoolWithFilterAndPagination(filter ReadAllocationPoolFilter, offset, limit int) ([]ReadAllocationPool, error) {
	query := edb.Get().Model(&ReadAllocationPool{}).Where(&filter)
	if offset != -1 {
		query = query.Offset(offset)
	}
	if limit != -1 {
		query = query.Limit(limit)
	}
	var allocationPools []ReadAllocationPool
	return allocationPools, query.Scan(&allocationPools).Error
}
