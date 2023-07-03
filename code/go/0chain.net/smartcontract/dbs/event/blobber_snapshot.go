package event

import (
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// swagger:model BlobberSnapshot
type BlobberSnapshot struct {
	BlobberID           string        `json:"id" gorm:"index"`
	BucketId            int64         `json:"bucket_id"`
	WritePrice          currency.Coin `json:"write_price"`
	Capacity            int64         `json:"capacity"`  // total blobber capacity
	Allocated           int64         `json:"allocated"` // allocated capacity
	SavedData           int64         `json:"saved_data"`
	ReadData            int64         `json:"read_data"`
	OffersTotal         currency.Coin `json:"offers_total"`
	TotalServiceCharge  currency.Coin `json:"total_service_charge"`
	TotalRewards        currency.Coin `json:"total_rewards"`
	TotalStake          currency.Coin `json:"total_stake"`
	TotalBlockRewards   currency.Coin `json:"total_block_rewards"`
	TotalStorageIncome  currency.Coin `json:"total_storage_income"`
	TotalReadIncome     currency.Coin `json:"total_read_income"`
	TotalSlashedStake   currency.Coin `json:"total_slashed_stake"`
	ChallengesPassed    uint64        `json:"challenges_passed"`
	ChallengesCompleted uint64        `json:"challenges_completed"`
	OpenChallenges      uint64        `json:"open_challenges"`
	CreationRound       int64         `json:"creation_round"`
	RankMetric          float64       `json:"rank_metric"`
	IsKilled            bool          `json:"is_killed"`
	IsShutdown          bool          `json:"is_shutdown"`
}

func (bs *BlobberSnapshot) IsOffline() bool {
	return bs.IsKilled || bs.IsShutdown
}

func (edb *EventDb) getBlobberSnapshots(limit, offset int64) (map[string]BlobberSnapshot, error) {
	var snapshots []BlobberSnapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM blobber_snapshots WHERE blobber_id in (select id from old_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	var mapSnapshots = make(map[string]BlobberSnapshot, len(snapshots))
	logging.Logger.Debug("get_blobber_snapshot", zap.Int("snapshots selected", len(snapshots)))
	logging.Logger.Debug("get_blobber_snapshot", zap.Int64("snapshots rows selected", result.RowsAffected))

	for _, snapshot := range snapshots {
		mapSnapshots[snapshot.BlobberID] = snapshot
	}

	result = edb.Store.Get().Where("blobber_id IN (select id from temp_ids ORDER BY ID limit ? offset ?)", limit, offset).Delete(&BlobberSnapshot{})
	logging.Logger.Debug("get_blobber_snapshot", zap.Int64("deleted rows", result.RowsAffected))
	return mapSnapshots, result.Error
}

func (edb *EventDb) addBlobberSnapshot(blobbers []Blobber) error {
	var snapshots []BlobberSnapshot
	for _, blobber := range blobbers {
		snapshots = append(snapshots, BlobberSnapshot{
			BlobberID:          blobber.ID,
			BucketId:           blobber.BucketId,
			WritePrice:         blobber.WritePrice,
			Capacity:           blobber.Capacity,
			Allocated:          blobber.Allocated,
			SavedData:          blobber.SavedData,
			ReadData:           blobber.ReadData,
			OffersTotal:        blobber.OffersTotal,
			TotalRewards:       blobber.Rewards.TotalRewards,
			TotalBlockRewards:  blobber.TotalBlockRewards,
			TotalStorageIncome: blobber.TotalStorageIncome,
			TotalReadIncome:    blobber.TotalReadIncome,
			TotalSlashedStake:  blobber.TotalSlashedStake,
			//TotalServiceCharge:  blobber.TotalServiceCharge,
			TotalStake:          blobber.TotalStake,
			ChallengesPassed:    blobber.ChallengesPassed,
			ChallengesCompleted: blobber.ChallengesCompleted,
			OpenChallenges:      blobber.OpenChallenges,
			CreationRound:       blobber.CreationRound,
			RankMetric:          blobber.RankMetric,
			IsKilled:            blobber.IsKilled,
			IsShutdown:          blobber.IsShutdown,
		})
	}

	return edb.Store.Get().Create(&snapshots).Error
}
