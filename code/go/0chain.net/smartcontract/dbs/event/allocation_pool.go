package event

import "gorm.io/gorm"

type AllocationPool struct {
	gorm.Model
	AllocationID  string `gorm:"uniqueIndex"`
	TransactionId string
	UserID        string
	Balance       int64
	Blobbers      []BlobberPool `gorm:"foreignKey:AllocationPoolID;references:AllocationID"`
	IsWritePool   bool
}

func (edb *EventDb) addAllocationPool(allocationPool AllocationPool) error {
	return edb.Get().Create(&allocationPool).Error
}
