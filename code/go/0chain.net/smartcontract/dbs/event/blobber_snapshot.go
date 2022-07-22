package event

import (
	"0chain.net/chaincore/currency"
	"0chain.net/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BlobberSnapshot struct {
	gorm.Model
	Round     int64  `json:"round" gorm:"index:idx_blobber_snapshot"`
	BlobberID string `json:"blobber_id" gorm:"index:idx_blobber_snapshot"`

	Stake currency.Coin `json:"stake"`
	// todo implement the following fields
	//ReservedSpace int64         `json:"reserved-space"`
	//BlockRewards  currency.Coin `json:"blockRewards"`
	//Updated       int64         `json:"written_to"`
	//Downloaded    int64         `json:"downloaded"`
}

func (edb *EventDb) getRow(id string, round int64, snapshots map[string]*BlobberSnapshot) (*BlobberSnapshot, error) {
	if row, ok := snapshots[id]; ok {
		return row, nil
	}
	if round <= 1 {
		return &BlobberSnapshot{}, nil
	}
	return edb.getBlobberSnapshot(id, round-1)
}

func (edb *EventDb) updateBlobberSnapshot(e events) error {
	if len(e) == 0 {
		return nil
	}
	round := e[0].Round
	round = round
	snapshots := make(map[string]*BlobberSnapshot)

	for _, event := range e {
		switch EventTag(event.Tag) {
		case TagLockStakePool:
			d, ok := fromEvent[DelegatePoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			row, err := edb.getRow(d.ProviderId, round, snapshots)
			if err != nil {
				logging.Logger.Error("get snapshot row", zap.Error(err))
			}
			row.Stake, err = currency.AddCoin(row.Stake, d.Amount)
			if err != nil {
				logging.Logger.Error("increment stake", zap.Error(err))
			}
		case TagUnlockStakePool:
			d, ok := fromEvent[DelegatePoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			row, err := edb.getRow(d.ProviderId, round, snapshots)
			if err != nil {
				logging.Logger.Error("get snapshot row", zap.Error(err))
			}
			row.Stake, err = currency.MinusCoin(row.Stake, d.Amount)
			if err != nil {
				logging.Logger.Error("decrement stake", zap.Error(err))
			}
		}

	}

	edb.addBlobberSnapshot(snapshots)
	return nil
}

func (edb *EventDb) getBlobberSnapshot(blobberId string, round int64) (*BlobberSnapshot, error) {
	snapshot := BlobberSnapshot{}
	res := edb.Store.Get().Model(BlobberSnapshot{}).Where(BlobberSnapshot{
		BlobberID: blobberId,
		Round:     round},
	).First(&snapshot)
	return &snapshot, res.Error
}

func (edb *EventDb) addBlobberSnapshot(bs map[string]*BlobberSnapshot) {
	for _, row := range bs {
		res := edb.Store.Get().Create(&row)
		if res.Error != nil {
			logging.Logger.Error("adding row to blobber snapshot", zap.Error(res.Error))
		}
	}
}
