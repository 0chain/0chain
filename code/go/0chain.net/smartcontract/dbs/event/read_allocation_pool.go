package event

import (
	"errors"

	"github.com/guregu/null"
	"gorm.io/gorm"
)

type ReadAllocationPool struct {
	gorm.Model
	PoolID       string `gorm:"uniqueIndex"`
	AllocationID string
	UserID       string
	StateBalance int64
	Balance      int64
	Blobbers     []BlobberPool `gorm:"foreignKey:ReadAllocationPoolID;references:AllocationID"`
	ExpireAt     int64
}

type ReadAllocationPoolFilter struct {
	gorm.Model
	AllocationID  null.String
	TransactionId null.String
	UserID        null.String
	Balance       null.Int
	ExpireAt      null.Int
}

func (edb *EventDb) addOrOverwriteReadAllocationPool(readAllocationPool ReadAllocationPool) error {
	if !edb.isReadPoolExists(readAllocationPool.PoolID) {
		readAllocationPool.StateBalance = readAllocationPool.Balance
		return edb.Get().Model(&ReadAllocationPool{}).Create(&readAllocationPool).Error
	}
	return edb.Get().Model(&ReadAllocationPool{}).Where(&ReadAllocationPool{AllocationID: readAllocationPool.AllocationID}).Updates(&readAllocationPool).Error
}

func (edb *EventDb) isReadPoolExists(poolID string) bool {
	err := edb.Get().Model(&ReadAllocationPool{}).Where(&ReadAllocationPool{PoolID: poolID}).Take(&ReadAllocationPool{}).Error
	if errors.Is(gorm.ErrRecordNotFound, err) {
		return false
	}
	return true
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
