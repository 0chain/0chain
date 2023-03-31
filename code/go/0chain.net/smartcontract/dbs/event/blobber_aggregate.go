package event

import (
	"math"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
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

func (edb *EventDb) updateBlobberAggregate(round, pageAmount int64, gs *Snapshot) {
	currentBucket := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM blobbers where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	exec = edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS old_temp_ids "+
		"ON COMMIT DROP AS SELECT blobber_id as id FROM blobber_snapshots where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating old temp table", zap.Error(exec.Error))
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

	logging.Logger.Debug("blobber aggregate/snapshot started", zap.Int64("round", round), zap.Int64("bucket_id", currentBucket), zap.Int64("page_limit", edb.PageLimit()))
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

func (edb *EventDb) calculateBlobberAggregate(gs *Snapshot, round, limit, offset int64) {
	const GB = float64(1024 * 1024 * 1024)
	var ids []string
	r := edb.Store.Get().
		Raw("select id from temp_ids ORDER BY ID limit ? offset ?", limit, offset).Scan(&ids)
	if r.Error != nil {
		logging.Logger.Error("getting ids", zap.Error(r.Error))
		return
	}

	var currentBlobbers []Blobber
	result := edb.Store.Get().Model(&Blobber{}).
		Where("blobbers.id in (select id from temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Joins("Rewards").
		Find(&currentBlobbers)

	if result.Error != nil {
		logging.Logger.Error("getting current blobbers", zap.Error(result.Error))
		return
	}

	oldBlobbers, err := edb.getBlobberSnapshots(limit, offset)
	if err != nil {
		logging.Logger.Error("getting blobber snapshots", zap.Error(err))
		return
	}
	
	var (
		oldBlobbersProcessingMap = MakeProcessingMap(oldBlobbers) 
		aggregates []BlobberAggregate
		gsDiff	   Snapshot
		old BlobberSnapshot
		ok bool
	)

	for _, current := range currentBlobbers {
		processingEntity, found := oldBlobbersProcessingMap[current.ID]
		if !found {
			old = BlobberSnapshot{ /* zero values */ }
			gsDiff.BlobberCount += 1
		} else {
			processingEntity.Processed = true
			old, ok = processingEntity.Entity.(BlobberSnapshot)
			if !ok {
				logging.Logger.Error("error converting processable entity to blobber snapshot")
				continue
			}
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
		aggregates = append(aggregates, aggregate)

		gsDiff.SuccessfulChallenges += int64(current.ChallengesPassed - old.ChallengesPassed)
		gsDiff.TotalChallenges += int64(current.ChallengesCompleted - old.ChallengesCompleted)
		gsDiff.TotalStaked += int64(current.TotalStake - old.TotalStake)
		gsDiff.StorageTokenStake += int64(current.TotalStake - old.TotalStake)
		gsDiff.AllocatedStorage += current.Allocated - old.Allocated
		gsDiff.MaxCapacityStorage += current.Capacity - old.Capacity
		gsDiff.UsedStorage += current.SavedData - old.SavedData
		gsDiff.TotalRewards += int64(current.Rewards.TotalRewards - old.TotalRewards)
		gsDiff.BlobberTotalRewards += int64(current.Rewards.TotalRewards - old.TotalRewards)

		// Change in staked storage (staked_storage = total_stake / write_price)
		oldSS := old.Capacity
		if old.WritePrice > 0 {
			oldSS = int64((float64(old.TotalStake) / float64(old.WritePrice)) * GB)
		}
		newSS := current.Capacity
		if current.WritePrice > 0 {
			newSS = int64((float64(current.TotalStake) / float64(current.WritePrice)) * GB)
		}
		gsDiff.StakedStorage += (newSS - oldSS)

		oldBlobbersProcessingMap[current.ID] = processingEntity
	}
	
	// Decrease global snapshot and blobber_snapshots based on deleted blobbers
	var snapshotIdsToDelete []string
	for _, processingEntity := range oldBlobbersProcessingMap {
		if processingEntity.Entity == nil || processingEntity.Processed {
			continue
		}
		old, ok = processingEntity.Entity.(BlobberSnapshot)
		if !ok {
			logging.Logger.Error("error converting processable entity to blobber snapshot")
			continue
		}
		snapshotIdsToDelete = append(snapshotIdsToDelete, old.BlobberID)
		gsDiff.SuccessfulChallenges += int64(-old.ChallengesPassed)
		gsDiff.TotalChallenges += int64(-old.ChallengesCompleted)
		gsDiff.AllocatedStorage += -old.Allocated
		gsDiff.MaxCapacityStorage += -old.Capacity
		gsDiff.UsedStorage += -old.SavedData
		gsDiff.TotalRewards += int64(-old.TotalRewards)
		gsDiff.TotalStaked += int64(-old.TotalStake)
		gsDiff.StorageTokenStake += int64(-old.TotalStake)
		gsDiff.BlobberCount -= 1

		if old.WritePrice > 0 {
			ss := int64((float64(old.TotalStake) / float64(old.WritePrice)) * GB)
			gsDiff.StakedStorage += -ss
		} else {
			gsDiff.StakedStorage += -old.Capacity
		}
	}

	if len(snapshotIdsToDelete) > 0 {
		if result := edb.Store.Get().Where("blobber_id IN (?)", snapshotIdsToDelete).Delete(&BlobberSnapshot{}); result.Error != nil {
			logging.Logger.Error("deleting blobber snapshots", zap.Error(result.Error))
		}
	}
	gs.ApplyDiff(&gsDiff)

	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			logging.Logger.Error("saving aggregates", zap.Error(result.Error))
		}
	}

	if len(currentBlobbers) > 0 {
		if err := edb.addBlobberSnapshot(currentBlobbers); err != nil {
			logging.Logger.Error("saving blobbers snapshots", zap.Error(err))
		}
	}

	logging.Logger.Debug("blobber aggregate/snapshots finished successfully",
		zap.Int("current_blobbers", len(currentBlobbers)),
		zap.Int("old_blobbers", len(oldBlobbers)),
		zap.Int("aggregates", len(aggregates)),
		zap.Int("deleted_snapshots", len(snapshotIdsToDelete)),
		zap.Any("global_snapshot_after", gs),
	)
}
