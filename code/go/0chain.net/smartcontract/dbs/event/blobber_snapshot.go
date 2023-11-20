package event

import (
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm/clause"
)

// swagger:model BlobberSnapshot
type BlobberSnapshot struct {
	BlobberID           string        `json:"id" gorm:"uniquIndex"`
	Round 			 	int64         `json:"round"`
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

func (bs *BlobberSnapshot) GetID() string {
	return bs.BlobberID
}

func (bs *BlobberSnapshot) GetRound() int64 {
	return bs.CreationRound
}

func (bs *BlobberSnapshot) SetID(id string) {
	bs.BlobberID = id
}

func (bs *BlobberSnapshot) SetRound(round int64) {
	bs.CreationRound = round
}

func (bs *BlobberSnapshot) IsOffline() bool {
	return bs.IsKilled || bs.IsShutdown
}

func (edb *EventDb) addBlobberSnapshot(blobbers []*Blobber, round int64) error {
	var snapshots []*BlobberSnapshot
	for _, blobber := range blobbers {
		snapshots = append(snapshots, createBlobberSnapshotFromBlobber(blobber, round))
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "blobber_id"}},
		UpdateAll: true,
	}).Create(&snapshots).Error
}

func createBlobberSnapshotFromBlobber(b *Blobber, round int64) *BlobberSnapshot {
	return &BlobberSnapshot{
		BlobberID:          b.ID,
		Round: 				round,
		WritePrice:         b.WritePrice,
		Capacity:           b.Capacity,
		Allocated:          b.Allocated,
		SavedData:          b.SavedData,
		ReadData:           b.ReadData,
		OffersTotal:        b.OffersTotal,
		TotalRewards:       b.Rewards.TotalRewards,
		TotalBlockRewards:  b.TotalBlockRewards,
		TotalStorageIncome: b.TotalStorageIncome,
		TotalReadIncome:    b.TotalReadIncome,
		TotalSlashedStake:  b.TotalSlashedStake,
		TotalStake:          b.TotalStake,
		ChallengesPassed:    b.ChallengesPassed,
		ChallengesCompleted: b.ChallengesCompleted,
		OpenChallenges:      b.OpenChallenges,
		CreationRound:       b.CreationRound,
		RankMetric:          b.RankMetric,
		IsKilled:            b.IsKilled,
		IsShutdown:          b.IsShutdown,
	}
}