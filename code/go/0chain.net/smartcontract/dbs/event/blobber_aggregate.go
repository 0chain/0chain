package event

import (
	"fmt"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BlobberAggregate struct {
	gorm.Model

	BlobberID string `json:"blobber_id" gorm:"index:idx_blobber_aggregate,unique"`
	Round     int64  `json:"round" gorm:"index:idx_blobber_aggregate,unique"`

	WritePrice   currency.Coin `json:"write_price"`
	Capacity     int64         `json:"capacity"`  // total blobber capacity
	Allocated    int64         `json:"allocated"` // allocated capacity
	SavedData    int64         `json:"saved_data"`
	ReadData     int64         `json:"read_data"`
	OffersTotal  currency.Coin `json:"offers_total"`
	UnstakeTotal currency.Coin `json:"unstake_total"`
	TotalStake   currency.Coin `json:"total_stake"`

	TotalServiceCharge  currency.Coin `json:"total_service_charge"`
	ChallengesPassed    uint64        `json:"challenges_passed"`
	ChallengesCompleted uint64        `json:"challenges_completed"`
	OpenChallenges      uint64        `json:"open_challenges"`
	InactiveRounds      int64         `json:"InactiveRounds"`
	RankMetric          float64       `json:"rank_metric" gorm:"index:idx_ba_rankmetric"`
}

func (edb *EventDb) ReplicateBlobberAggregate(round int64, offset, limit int) ([]BlobberAggregate, error) {
	var snapshots []BlobberAggregate

	queryBuilder := edb.Store.Get().
		Model(&BlobberAggregate{}).Where("round > ?", round).Offset(offset).Limit(limit)

	queryBuilder.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "blobber_id"},
		Desc:   false,
	})

	result := queryBuilder.Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	//we return only snapshots for one round, if different pages are returned we will cut snapshots from the next round
	res := snapshots
	if len(snapshots) > 0 {
		r := snapshots[0].Round
		for i, s := range snapshots {
			if r != s.Round {
				res = snapshots[:i]
			}
		}
	}
	return res, nil
}

func (edb *EventDb) updateBlobberAggregate(round, period int64, gs *globalSnapshot) {
	_, oldBlobbers, err := edb.getBlobberSnapshots(round, period)
	if err != nil {
		logging.Logger.Error("getting blobber snapshots", zap.Error(err))
		return
	}

	var currentBlobbers []Blobber
	result := edb.Store.Get().
		Raw(fmt.Sprintf("SELECT * FROM blobbers WHERE MOD(creation_round, %d) = ?", period), round%period).
		Scan(&currentBlobbers)
	if result.Error != nil {
		logging.Logger.Error("getting current blobbers", zap.Error(result.Error))
		return
	}

	if round <= period && len(currentBlobbers) > 0 {
		if err := edb.addBlobberSnapshot(currentBlobbers); err != nil {
			logging.Logger.Error("saving blobbers snapshots", zap.Error(err))
		}
	}

	var aggregates []BlobberAggregate
	for _, current := range currentBlobbers {
		old, found := oldBlobbers[current.BlobberID]
		if !found {
			continue
		}
		aggregate := BlobberAggregate{
			Round:     round,
			BlobberID: current.BlobberID,
		}
		aggregate.WritePrice = (old.WritePrice + current.WritePrice) / 2
		aggregate.Capacity = (old.Capacity + current.Capacity) / 2
		aggregate.Allocated = (old.Allocated + current.Allocated) / 2
		aggregate.SavedData = (old.SavedData + current.SavedData) / 2
		aggregate.ReadData = (old.ReadData + current.ReadData) / 2
		aggregate.TotalStake = (old.TotalStake + current.TotalStake) / 2
		aggregate.OffersTotal = (old.OffersTotal + current.OffersTotal) / 2
		aggregate.UnstakeTotal = (old.UnstakeTotal + current.UnstakeTotal) / 2
		aggregate.OpenChallenges = (old.OpenChallenges + current.OpenChallenges) / 2
		aggregate.RankMetric = current.RankMetric

		aggregate.ChallengesPassed = current.ChallengesPassed
		aggregate.ChallengesCompleted = current.ChallengesCompleted
		aggregate.InactiveRounds = current.InactiveRounds
		aggregate.TotalServiceCharge = current.TotalServiceCharge
		aggregates = append(aggregates, aggregate)

		gs.totalWritePricePeriod += aggregate.WritePrice

		// update global snapshot object
		ts, err := aggregate.TotalStake.Int64()
		if err != nil {
			logging.Logger.Error("converting coin to int64", zap.Error(err))
		}
		gs.TotalStaked = ts
		gs.SuccessfulChallenges += int64(aggregate.ChallengesPassed)
		gs.TotalChallenges += int64(aggregate.ChallengesCompleted)
		gs.AllocatedStorage += aggregate.Allocated
		gs.MaxCapacityStorage += aggregate.Capacity
		gs.UsedStorage += aggregate.SavedData

		const GB = currency.Coin(1024 * 1024 * 1024)
		ss, err := (aggregate.TotalStake * (GB / aggregate.WritePrice)).Int64()
		if err != nil {
			logging.Logger.Error("converting coin to int64", zap.Error(err))
		}
		gs.StakedStorage += ss

		gs.blobberCount++
	}
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

	// update global snapshot object
	if gs.blobberCount == 0 {
		return
	}
	twp, err := gs.totalWritePricePeriod.Int64()
	if err != nil {
		logging.Logger.Error("converting write price to coin", zap.Error(err))
		return
	}
	gs.AverageWritePrice = int64(twp / int64(gs.blobberCount))
}
