package event

import "gorm.io/gorm"

type BlobberSnapshot struct {
	gorm.Model
	Round     int64  `json:"round" gorm:"index:idx_blobber_snapshot"`
	BlobberID string `json:"blobber_id" gorm:"index:idx_blobber_snapshot"`
}

func (edb *EventDb) updateBlobberSnapshot(e events) error {
	return nil
}
