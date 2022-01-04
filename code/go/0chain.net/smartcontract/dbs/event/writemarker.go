package event

import "gorm.io/gorm"

type WriteMarker struct {
	gorm.Model

	// foreign keys
	// todo: as user(ID), allocation(ID) and transaction(ID) tables are created, enable it
	ClientID      string
	BlobberID     string
	AllocationID  string
	TransactionID string

	AllocationRoot         string
	PreviousAllocationRoot string
	Size                   int64
	Timestamp              int64
	Signature              string
	BlockNumber            int64
}
