package event

import (
	"0chain.net/chaincore/currency"
	"gorm.io/gorm"
)

type BlobberSnapshot struct {
	gorm.Model
	Round     int64  `json:"round" gorm:"index:idx_blobber_snapshot"`
	BlobberID string `json:"blobber_id" gorm:"index:idx_blobber_snapshot"`

	Stake         currency.Coin `json:"stake"`
	ReservedSpace int64         `json:"reserved-space"`
	BlockRewards  currency.Coin `json:"blockRewards"`
	Updated       int64         `json:"written_to"`
	Downloaded    int64         `json:"downloaded"`
}

func (edb *EventDb) updateBlobberSnapshot(e events) error {
	if len(e) == 0 {
		return nil
	}
	last, err := edb.getBlobberSnapshot(e[0].Round)
	current := BlobberSnapshot{}

	for i, event := range e {

	}

	if err := edb.addBlobberSnapshot(current); err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) getBlobberSnapshot(round int64) (BlobberSnapshot, error) {
	snapshot := BlobberSnapshot{}
	res := edb.Store.Get().Model(BlobberSnapshot{}).Where(BlobberSnapshot{Round: round}).First(&snapshot)
	return snapshot, res.Error
}

func (edb *EventDb) addBlobberSnapshot(bs BlobberSnapshot) error {
	res := edb.Store.Get().Create(&bs)
	return res.Error
}
