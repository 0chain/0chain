package event

import (
	"fmt"

	"0chain.net/chaincore/currency"

	"gorm.io/gorm"
)

// swagger:model BlobberSnapshot
type BlobberSnapshot struct {
	gorm.Model
	BlobberID           string        `json:"id" gorm:"index"`
	WritePrice          currency.Coin `json:"write_price"`
	Capacity            int64         `json:"capacity"`  // total blobber capacity
	Allocated           int64         `json:"allocated"` // allocated capacity
	Used                int64         `json:"used"`      // total of files saved on blobber
	SavedData           int64         `json:"saved_data"`
	OffersTotal         currency.Coin `json:"offers_total"`
	UnstakeTotal        currency.Coin `json:"unstake_total"`
	TotalServiceCharge  currency.Coin `json:"total_service_charge"`
	TotalStake          currency.Coin `json:"total_stake"`
	ChallengesPassed    uint64        `json:"challenges_passed"`
	ChallengesCompleted uint64        `json:"challenges_completed"`
	InactiveRounds      int64         `json:"inactive_rounds"`
}

func (edb *EventDb) getBlobberSnapshots(round, period int64) ([]string, map[string]BlobberSnapshot, error) {
	var snapshots []BlobberSnapshot
	result := edb.Store.Get().
		Raw(fmt.Sprintf("SELECT * FROM BlobberSnapshot WHERE MOD(creation_date, %d) = ?", period), round%period).
		Scan(&snapshots)
	if result.Error != nil {
		return nil, nil, result.Error
	}

	var mapSnapshots = make(map[string]BlobberSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		mapSnapshots[snapshot.BlobberID] = snapshot
	}

	var ids []string
	for _, snapshot := range snapshots {
		ids = append(ids, snapshot.BlobberID)
	}
	result = edb.Store.Get().Where("blobber_id IN ?", ids).Delete(&BlobberSnapshot{})
	return ids, mapSnapshots, result.Error
}

func (edb *EventDb) addBlobberSnapshot(blobbers []Blobber) error {
	var snapshots []BlobberSnapshot
	for _, blobber := range blobbers {
		snapshots = append(snapshots, BlobberSnapshot{
			BlobberID:           blobber.BlobberID,
			WritePrice:          blobber.WritePrice,
			Capacity:            blobber.Capacity,
			Allocated:           blobber.Allocated,
			Used:                blobber.Used,
			SavedData:           blobber.SavedData,
			OffersTotal:         blobber.OffersTotal,
			UnstakeTotal:        blobber.UnstakeTotal,
			TotalServiceCharge:  blobber.TotalServiceCharge,
			TotalStake:          blobber.TotalStake,
			ChallengesPassed:    blobber.ChallengesPassed,
			ChallengesCompleted: blobber.ChallengesCompleted,
			InactiveRounds:      blobber.InactiveRounds,
		})
	}
	return edb.Store.Get().Create(&snapshots).Error
}
