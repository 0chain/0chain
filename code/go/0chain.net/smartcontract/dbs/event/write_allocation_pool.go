package event

import "gorm.io/gorm"

type WriteAllocationPool struct {
	gorm.Model
	AllocationId  string
	TransactionId string
	UserID        string
	Balance       int64
	Blobbers      []BlobberPool `gorm:"foreignKey:AllocationPoolID;references:ID"`
}

func (edb *EventDb) addWriteAllocationPool(readAllocationPool WriteAllocationPool) error {
	return edb.Get().Create(&readAllocationPool).Error
}
