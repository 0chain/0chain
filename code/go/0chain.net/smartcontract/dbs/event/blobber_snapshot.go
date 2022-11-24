package event

import (
	"github.com/0chain/common/core/currency"
)

// swagger:model BlobberSnapshot
type BlobberSnapshot struct {
	BlobberID           string        `json:"id" gorm:"index"`
	WritePrice          currency.Coin `json:"write_price"`
	Capacity            int64         `json:"capacity"`  // total blobber capacity
	Allocated           int64         `json:"allocated"` // allocated capacity
	SavedData           int64         `json:"saved_data"`
	ReadData            int64         `json:"read_data"`
	OffersTotal         currency.Coin `json:"offers_total"`
	UnstakeTotal        currency.Coin `json:"unstake_total"`
	TotalServiceCharge  currency.Coin `json:"total_service_charge"`
	TotalStake          currency.Coin `json:"total_stake"`
	ChallengesPassed    uint64        `json:"challenges_passed"`
	ChallengesCompleted uint64        `json:"challenges_completed"`
	OpenChallenges      uint64        `json:"open_challenges"`
	InactiveRounds      int64         `json:"inactive_rounds"`
	CreationRound       int64         `json:"creation_round" gorm:"index"`
	RankMetric          float64       `json:"rank_metric"`
}

func (edb *EventDb) getBlobberSnapshots(limit, offset int64) (map[string]BlobberSnapshot, error) {
	var snapshots []BlobberSnapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM blobber_snapshots WHERE blobber_id in (select id from temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	var mapSnapshots = make(map[string]BlobberSnapshot, len(snapshots))

	for _, snapshot := range snapshots {
		mapSnapshots[snapshot.BlobberID] = snapshot
	}

	result = edb.Store.Get().Where("blobber_id IN (select id from temp_ids ORDER BY ID limit ? offset ?)", limit, offset).Delete(&BlobberSnapshot{})

	return mapSnapshots, result.Error
}

func (edb *EventDb) addBlobberSnapshot(blobbers []Blobber) error {
	var snapshots []BlobberSnapshot
	for _, blobber := range blobbers {
		snapshots = append(snapshots, BlobberSnapshot{
			BlobberID:    blobber.ID,
			WritePrice:   blobber.WritePrice,
			Capacity:     blobber.Capacity,
			Allocated:    blobber.Allocated,
			SavedData:    blobber.SavedData,
			ReadData:     blobber.ReadData,
			OffersTotal:  blobber.OffersTotal,
			UnstakeTotal: blobber.UnstakeTotal,
			//TotalServiceCharge:  blobber.TotalServiceCharge,
			TotalStake:          blobber.TotalStake,
			ChallengesPassed:    blobber.ChallengesPassed,
			ChallengesCompleted: blobber.ChallengesCompleted,
			OpenChallenges:      blobber.OpenChallenges,
			InactiveRounds:      blobber.InactiveRounds,
			CreationRound:       blobber.CreationRound,
			RankMetric:          blobber.RankMetric,
		})
	}

	return edb.Store.Get().Create(&snapshots).Error
}
