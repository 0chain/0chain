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
	//	round := e[0].Round
	snapshots := make(map[string]BlobberSnapshot)

	//	for i, event := range e {

	//}

	if err := edb.addBlobberSnapshot(snapshots); err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) getBlobberSnapshot(blobberId string, round int64) (BlobberSnapshot, error) {
	snapshot := BlobberSnapshot{}
	res := edb.Store.Get().Model(BlobberSnapshot{}).Where(BlobberSnapshot{
		BlobberID: blobberId,
		Round:     round},
	).First(&snapshot)
	return snapshot, res.Error
}

func (edb *EventDb) addBlobberSnapshot(bs map[string]BlobberSnapshot) error {
	res := edb.Store.Get().Create(&bs)
	return res.Error
}
