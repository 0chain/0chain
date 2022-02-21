package event

import "gorm.io/gorm"

type BlobberPool struct {
	gorm.Model
	AllocationPoolID string
	BlobberID        string
	Balance          int64
}
