package event

import (
	"errors"

	"github.com/guregu/null"
	"gorm.io/gorm"
)

type WriteAllocationPool struct {
	gorm.Model
	PoolID       string `gorm:"uniqueIndex"`
	AllocationID string
	UserID       string
	Balance      int64
	Blobbers     []BlobberPool `gorm:"foreignKey:WriteAllocationPoolID;references:AllocationID"`
	StateBalance int64
	ExpireAt     int64
}

type WriteAllocationPoolFilter struct {
	gorm.Model
	AllocationID  null.String
	TransactionId null.String
	UserID        null.String
	Balance       null.Int
	ExpireAt      null.Int
}

func (edb *EventDb) addOrOverwriteWriteAllocationPool(writeAllocationPool WriteAllocationPool) error {
	if !edb.isWritePoolExists(writeAllocationPool.PoolID) {
		writeAllocationPool.StateBalance = writeAllocationPool.Balance
		return edb.Get().Model(&WriteAllocationPool{}).Create(&writeAllocationPool).Error
	}
	return edb.Get().Model(&WriteAllocationPool{}).Where(&WriteAllocationPool{AllocationID: writeAllocationPool.AllocationID}).Updates(&writeAllocationPool).Error
}

func (edb *EventDb) isWritePoolExists(poolID string) bool {
	err := edb.Get().Model(&WriteAllocationPool{}).Where(&WriteAllocationPool{PoolID: poolID}).Take(&WriteAllocationPool{}).Error
	if errors.Is(gorm.ErrRecordNotFound, err) {
		return false
	}
	return true
}

func (edb *EventDb) GetWriteAllocationPoolWithFilterAndPagination(filter WriteAllocationPoolFilter, offset, limit int) ([]WriteAllocationPool, error) {
	query := edb.Get().Model(&WriteAllocationPool{}).Where(&filter)
	if offset != -1 {
		query = query.Offset(offset)
	}
	if limit != -1 {
		query = query.Limit(limit)
	}
	var allocationPools []WriteAllocationPool
	return allocationPools, query.Scan(&allocationPools).Error
}
