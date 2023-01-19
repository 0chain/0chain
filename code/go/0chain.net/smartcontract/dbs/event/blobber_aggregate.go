package event

import (
	"math"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

type BlobberAggregate struct {
	model.ImmutableModel
	BlobberID           string        `json:"blobber_id" gorm:"index:idx_blobber_aggregate,priority:2,unique"`
	Round               int64         `json:"round" gorm:"index:idx_blobber_aggregate,priority:1,unique"`
	BucketID            int64         `json:"bucket_id"`
	WritePrice          currency.Coin `json:"write_price"`
	Capacity            int64         `json:"capacity"`  // total blobber capacity
	Allocated           int64         `json:"allocated"` // allocated capacity
	SavedData           int64         `json:"saved_data"`
	ReadData            int64         `json:"read_data"`
	OffersTotal         currency.Coin `json:"offers_total"`
	UnstakeTotal        currency.Coin `json:"unstake_total"`
	TotalStake          currency.Coin `json:"total_stake"`
	TotalServiceCharge  currency.Coin `json:"total_service_charge"`
	TotalRewards        currency.Coin `json:"total_rewards"`
	ChallengesPassed    uint64        `json:"challenges_passed"`
	ChallengesCompleted uint64        `json:"challenges_completed"`
	OpenChallenges      uint64        `json:"open_challenges"`
	InactiveRounds      int64         `json:"InactiveRounds"`
	RankMetric          float64       `json:"rank_metric" gorm:"index:idx_ba_rankmetric"`
	Downtime            uint64        `json:"downtime"`
}

func (edb *EventDb) ReplicateBlobberAggregate(p common.Pagination) ([]BlobberAggregate, error) {
	var snapshots []BlobberAggregate

	queryBuilder := edb.Store.Get().
		Model(&BlobberAggregate{}).Offset(p.Offset).Limit(p.Limit)
	queryBuilder.Clauses(clause.OrderBy{
		Columns: []clause.OrderByColumn{{
			Column: clause.Column{Name: "round"},
		}, {
			Column: clause.Column{Name: "blobber_id"},
		}},
	})

	result := queryBuilder.Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	return snapshots, nil
}
func (edb *EventDb) updateBlobberAggregate(round, pageAmount int64, gs *globalSnapshot) {
	currentBucket := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM blobbers where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	var count int64
	r := edb.Store.Get().Raw("SELECT count(*) FROM temp_ids").Scan(&count)
	if r.Error != nil {
		logging.Logger.Error("getting ids count", zap.Error(r.Error))
		return
	}
	if count == 0 {
		return
	}
	pageCount := count / edb.PageLimit()

	for i := int64(0); i <= pageCount; i++ {
		edb.calculateBlobberAggregate(gs, round, edb.PageLimit(), i*edb.PageLimit())
	}

}

// paginate divides `count` of items in exactly `round` pages and returns
// the size of the page, current page number and amount of subpages if needed
// for example, we have round=101, pageAmount=2, count=11, then
// size will be 6, current page 1, and subpage count 1
func paginate(round, pageAmount, count, pageLimit int64) (int64, int64, int) {
	size := int64(math.Ceil(float64(count) / float64(pageAmount)))
	currentPageNumber := round % pageAmount

	subpageCount := 1
	if size > pageLimit {
		subpageCount = int(math.Ceil(float64(size) / float64(pageLimit)))
	}
	return size, currentPageNumber, subpageCount
}

func (edb *EventDb) calculateBlobberAggregate(gs *globalSnapshot, round, limit, offset int64) {

	var ids []string
	r := edb.Store.Get().
		Raw("select id from temp_ids ORDER BY ID limit ? offset ?", limit, offset).Scan(&ids)
	if r.Error != nil {
		logging.Logger.Error("getting ids", zap.Error(r.Error))
		return
	}
	logging.Logger.Debug("getting blobber aggregate ids", zap.Int("num", len(ids)))

	var currentBlobbers []Blobber
	result := edb.Store.Get().
		Raw("SELECT * FROM blobbers WHERE id in (select id from temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&currentBlobbers)
	if result.Error != nil {
		logging.Logger.Error("getting current blobbers", zap.Error(result.Error))
		return
	}
	logging.Logger.Debug("blobber_snapshot", zap.Int("total_current_blobbers", len(currentBlobbers)))

	if round <= edb.AggregatePeriod() && len(currentBlobbers) > 0 {
		if err := edb.addBlobberSnapshot(currentBlobbers); err != nil {
			logging.Logger.Error("saving blobbers snapshots", zap.Error(err))
		}
	}

	oldBlobbers, err := edb.getBlobberSnapshots(limit, offset)
	if err != nil {
		logging.Logger.Error("getting blobber snapshots", zap.Error(err))
		return
	}
	logging.Logger.Debug("blobber_snapshot", zap.Int("total_old_blobbers", len(oldBlobbers)))

	var aggregates []BlobberAggregate
	for _, current := range currentBlobbers {
		old, found := oldBlobbers[current.ID]
		if !found {
			continue
		}
		aggregate := BlobberAggregate{
			Round:     round,
			BlobberID: current.ID,
			BucketID:  current.BucketId,
		}
		aggregate.WritePrice = (old.WritePrice + current.WritePrice) / 2
		aggregate.Capacity = (old.Capacity + current.Capacity) / 2
		aggregate.Allocated = (old.Allocated + current.Allocated) / 2
		aggregate.SavedData = (old.SavedData + current.SavedData) / 2
		aggregate.ReadData = (old.ReadData + current.ReadData) / 2
		aggregate.TotalStake = (old.TotalStake + current.TotalStake) / 2
		aggregate.TotalRewards = (old.TotalRewards + current.Rewards.TotalRewards) / 2
		aggregate.OffersTotal = (old.OffersTotal + current.OffersTotal) / 2
		aggregate.UnstakeTotal = (old.UnstakeTotal + current.UnstakeTotal) / 2
		aggregate.OpenChallenges = (old.OpenChallenges + current.OpenChallenges) / 2
		aggregate.Downtime = current.Downtime
		aggregate.RankMetric = current.RankMetric

		aggregate.ChallengesPassed = current.ChallengesPassed
		aggregate.ChallengesCompleted = current.ChallengesCompleted
		//aggregate.TotalServiceCharge = current.TotalServiceCharge
		aggregates = append(aggregates, aggregate)

		gs.totalWritePricePeriod += aggregate.WritePrice

		gs.SuccessfulChallenges += int64(aggregate.ChallengesPassed - old.ChallengesPassed)
		gs.TotalChallenges += int64(aggregate.ChallengesCompleted - old.ChallengesCompleted)
		gs.AllocatedStorage += aggregate.Allocated - old.Allocated
		gs.MaxCapacityStorage += aggregate.Capacity - old.Capacity
		gs.UsedStorage += aggregate.SavedData - old.SavedData

		const GB = currency.Coin(1024 * 1024 * 1024)
		if aggregate.WritePrice == 0 {
			gs.StakedStorage = gs.MaxCapacityStorage
		} else {
			ss, err := ((aggregate.TotalStake - old.TotalStake) * (GB / aggregate.WritePrice)).Int64()
			if err != nil {
				logging.Logger.Error("converting coin to int64", zap.Error(err))
			}
			gs.StakedStorage += ss
		}

		gs.blobberCount++ //todo figure out why we increment blobberCount on every update
	}
	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			logging.Logger.Error("saving aggregates", zap.Error(result.Error))
		}
	}
	logging.Logger.Debug("blobber_snapshot", zap.Int("aggregates", len(aggregates)))

	if len(currentBlobbers) > 0 {
		if err := edb.addBlobberSnapshot(currentBlobbers); err != nil {
			logging.Logger.Error("saving blobbers snapshots", zap.Error(err))
		}
	}

	logging.Logger.Debug("blobber_snapshot", zap.Int("current_blobebrs", len(currentBlobbers)))

	// update global snapshot object
	if gs.blobberCount != 0 {

		twp, err := gs.totalWritePricePeriod.Int64()
		if err != nil {
			logging.Logger.Error("converting write price to coin", zap.Error(err))
			return
		}
		gs.AverageWritePrice = int64(twp / int64(gs.blobberCount))
	}
}
