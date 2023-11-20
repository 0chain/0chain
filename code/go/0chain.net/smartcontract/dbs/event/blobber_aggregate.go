package event

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
)

type BlobberAggregate struct {
	model.ImmutableModel
	BlobberID           string        		`json:"blobber_id" gorm:"index:idx_blobber_aggregate,priority:2,unique"`
	Round               int64         		`json:"round" gorm:"index:idx_blobber_aggregate,priority:1,unique"`
	LastHealthCheck     common.Timestamp	`json:"last_health_check"`
	WritePrice          currency.Coin 		`json:"write_price"`
	Capacity            int64         		`json:"capacity"`  // total blobber capacity
	Allocated           int64         		`json:"allocated"` // allocated capacity
	SavedData           int64         		`json:"saved_data"`
	ReadData            int64         		`json:"read_data"`
	OffersTotal         currency.Coin 		`json:"offers_total"`
	TotalStake          currency.Coin 		`json:"total_stake"`
	TotalServiceCharge  currency.Coin 		`json:"total_service_charge"`
	TotalRewards        currency.Coin 		`json:"total_rewards"`
	TotalBlockRewards   currency.Coin 		`json:"total_block_rewards"`
	TotalStorageIncome  currency.Coin 		`json:"total_storage_income"`
	TotalReadIncome     currency.Coin 		`json:"total_read_income"`
	TotalSlashedStake   currency.Coin 		`json:"total_slashed_stake"`
	ChallengesPassed    uint64        		`json:"challenges_passed"`
	ChallengesCompleted uint64        		`json:"challenges_completed"`
	OpenChallenges      uint64        		`json:"open_challenges"`
	InactiveRounds      int64         		`json:"InactiveRounds"`
	RankMetric          float64       		`json:"rank_metric"`
	Downtime            uint64        		`json:"downtime"`
	
}

func (edb *EventDb) CreateBlobberAggregates(blobbers []*Blobber, round int64) error {
	var aggregates []BlobberAggregate
	for _, blobber := range blobbers {
		aggregate := BlobberAggregate{
			Round:     round,
			BlobberID: blobber.ID,
			LastHealthCheck: blobber.LastHealthCheck,
		}
		aggregate.WritePrice = blobber.WritePrice
		aggregate.Capacity = blobber.Capacity
		aggregate.Allocated = blobber.Allocated
		aggregate.SavedData = blobber.SavedData
		aggregate.ReadData = blobber.ReadData
		aggregate.TotalStake = blobber.TotalStake
		aggregate.TotalRewards = blobber.Rewards.TotalRewards
		aggregate.OffersTotal = blobber.OffersTotal
		aggregate.OpenChallenges = blobber.OpenChallenges
		aggregate.TotalBlockRewards = blobber.TotalBlockRewards
		aggregate.TotalStorageIncome = blobber.TotalStorageIncome
		aggregate.TotalReadIncome = blobber.TotalReadIncome
		aggregate.TotalSlashedStake = blobber.TotalSlashedStake
		aggregate.Downtime = blobber.Downtime
		aggregate.ChallengesPassed = blobber.ChallengesPassed
		aggregate.ChallengesCompleted = blobber.ChallengesCompleted
		if blobber.ChallengesCompleted == 0 {
			aggregate.RankMetric = 0
		} else {
			aggregate.RankMetric = float64(blobber.ChallengesPassed) / float64(blobber.ChallengesCompleted)
		}
		aggregates = append(aggregates, aggregate)
	}
	return edb.Store.Get().Create(&aggregates).Error
}