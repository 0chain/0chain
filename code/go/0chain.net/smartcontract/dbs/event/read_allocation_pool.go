package event

import "gorm.io/gorm"

type ReadAllocationPool struct {
	gorm.Model
	AllocationId  string
	TransactionId string
	UserID        string
	Balance       int64
	Blobbers      []BlobberPool `gorm:"foreignKey:AllocationPoolID;references:ID"`
}

type BlobberPool struct {
	gorm.Model
	AllocationPoolID uint
	Balance          int64
}

func (edb *EventDb) addReadAllocationPool(readAllocationPool ReadAllocationPool) error {
	return edb.Get().Create(&readAllocationPool).Error
}
