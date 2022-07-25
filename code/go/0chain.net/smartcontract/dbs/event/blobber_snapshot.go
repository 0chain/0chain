package event

import (
	"0chain.net/chaincore/currency"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// swagger:model BlobberSnapshot
type BlobberSnapshot struct {
	gorm.Model
	Round     int64  `json:"round" gorm:"index:idx_blobber_snapshot"`
	BlobberID string `json:"blobber_id" gorm:"index:idx_blobber_snapshot"`

	WritePrice         currency.Coin `json:"write_price"`
	Capacity           int64         `json:"capacity"`  // total blobber capacity
	Allocated          int64         `json:"allocated"` // allocated capacity
	SavedData          int64         `json:"saved_data"`
	OffersTotal        currency.Coin `json:"offers_total"`
	UnstakeTotal       currency.Coin `json:"unstake_total"`
	TotalServiceCharge currency.Coin `json:"total_service_charge"`
	TotalStake         currency.Coin `json:"total_stake"`
}

func (edb *EventDb) updateBlobberSnapshot(e events) {
	if len(e) == 0 {
		return
	}
	thisRound := e[0].BlockNumber
	blobberIds := make(map[string]struct{})

	for _, event := range e {
		switch EventTag(event.Tag) {
		case TagUpdateBlobber:
			updates, ok := fromEvent[dbs.DbUpdates](event.Data)
			if !ok {
				logging.Logger.Error("blobber snapshot", zap.Error(ErrInvalidEventData))
				continue
			}
			if _, found := blobberIds[updates.Id]; !found {
				blobberIds[updates.Id] = struct{}{}
			}
		case TagStakePoolReward: // maybe Distribute reward
			spu, ok := fromEvent[dbs.StakePoolReward](event.Data)
			if !ok {
				logging.Logger.Error("blobber snapshot", zap.Error(ErrInvalidEventData))
				continue
			}
			if spu.ProviderType == int(spenum.Blobber) {
				if _, found := blobberIds[spu.ProviderId]; !found {
					blobberIds[spu.ProviderId] = struct{}{}
				}
			}
		case TagAddOrOverwriteBlobber: // ok
			blobber, ok := fromEvent[Blobber](event.Data)
			if !ok {
				logging.Logger.Error("blobber snapshot", zap.Error(ErrInvalidEventData))
				continue
			}
			if _, found := blobberIds[blobber.BlobberID]; !found {
				blobberIds[blobber.BlobberID] = struct{}{}
			}
		}
	}

	if len(blobberIds) == 0 {
		return
	}

	var blobbers []Blobber
	var blobberIdsSlice []string
	for key := range blobberIds {
		blobberIdsSlice = append(blobberIdsSlice, key)
	}

	result := edb.Store.Get().
		Model(&Blobber{}).
		Where("blobber_id IN ?", blobberIdsSlice).
		Find(&blobbers)
	if result.Error != nil {
		logging.Logger.Error("getting blobber list for blobber snapshot",
			zap.Strings("blobberIds", blobberIdsSlice),
			zap.Error(result.Error))
		return
	}

	var snapshots []BlobberSnapshot
	for _, blobber := range blobbers {
		snapshots = append(snapshots, BlobberSnapshot{
			Round:              thisRound,
			BlobberID:          blobber.BlobberID,
			WritePrice:         blobber.WritePrice,
			Capacity:           blobber.Capacity,
			Allocated:          blobber.Allocated,
			OffersTotal:        blobber.OffersTotal,
			UnstakeTotal:       blobber.UnstakeTotal,
			TotalServiceCharge: blobber.TotalServiceCharge,
			TotalStake:         blobber.TotalStake,
		})
	}
	edb.Store.Get().Create(&snapshots)
}

func (edb *EventDb) GetBlobberSnapshot(blobberId string, round int64) (BlobberSnapshot, error) {
	snapshot := BlobberSnapshot{}
	res := edb.Store.Get().
		Model(BlobberSnapshot{}).
		Where("blobber_id = ? and round <= ?", blobberId, round).
		Order("round desc").
		First(&snapshot)
	return snapshot, res.Error
}
