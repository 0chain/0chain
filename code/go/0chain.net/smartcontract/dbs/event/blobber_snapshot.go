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

	Capacity           int64         `json:"capacity"`  // total blobber capacity
	Allocated          int64         `json:"allocated"` // allocated capacity
	Used               int64         `json:"used"`      // total of files saved on blobber "`
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
		case TagAddWriteMarker:
			wm, ok := fromEvent[WriteMarker](event.Data)
			if !ok {
				logging.Logger.Error("", zap.Error(ErrInvalidEventData))
				continue
			}
			if _, found := blobberIds[wm.BlobberID]; !found {
				blobberIds[wm.BlobberID] = struct{}{}
			}
		case TagUpdateBlobber:
			updates, ok := fromEvent[dbs.DbUpdates](event.Data)
			if !ok {
				logging.Logger.Error("", zap.Error(ErrInvalidEventData))
				continue
			}
			if _, found := blobberIds[updates.Id]; !found {
				blobberIds[updates.Id] = struct{}{}
			}
		case TagStakePoolReward:
			spu, ok := fromEvent[dbs.StakePoolReward](event.Data)
			if !ok {
				logging.Logger.Error("", zap.Error(ErrInvalidEventData))
				continue
			}
			if spu.ProviderType == int(spenum.Blobber) {
				if _, found := blobberIds[spu.ProviderId]; !found {
					blobberIds[spu.ProviderId] = struct{}{}
				}
			}
		case TagAddOrOverwriteBlobber:
			blobber, ok := fromEvent[Blobber](event.Data)
			if !ok {
				logging.Logger.Error("", zap.Error(ErrInvalidEventData))
				continue
			}
			if _, found := blobberIds[blobber.BlobberID]; !found {
				blobberIds[blobber.BlobberID] = struct{}{}
			}
		}
	}

	for blobberId := range blobberIds {
		blobber, err := edb.GetBlobber(blobberId)
		if err != nil {
			logging.Logger.Error("getting blobber "+blobberId, zap.Error(err))
			continue
		}
		row := BlobberSnapshot{
			Round:              thisRound,
			BlobberID:          blobberId,
			Capacity:           blobber.Capacity,
			Allocated:          blobber.Allocated,
			Used:               blobber.Used,
			SavedData:          blobber.SavedData,
			OffersTotal:        blobber.OffersTotal,
			UnstakeTotal:       blobber.UnstakeTotal,
			TotalServiceCharge: blobber.TotalServiceCharge,
			TotalStake:         blobber.TotalStake,
		}

		res := edb.Store.Get().Create(&row)
		if res.Error != nil {
			logging.Logger.Error("adding row to blobber snapshot", zap.Error(res.Error))
			continue
		}
	}
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
