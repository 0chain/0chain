package event

import (
	"fmt"

	"0chain.net/chaincore/currency"
	"0chain.net/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BlobberAggregate struct {
	gorm.Model

	BlobberID string `json:"blobber_id" gorm:"index:idx_blobber_aggregate,unique"`
	Round     int64  `json:"round" gorm:"index:idx_blobber_aggregate,unique"`

	WritePrice currency.Coin `json:"write_price"`
	Capacity   int64         `json:"capacity"`  // total blobber capacity
	Allocated  int64         `json:"allocated"` // allocated capacity
	SavedData  int64         `json:"saved_data"`

	OffersTotal  currency.Coin `json:"offers_total"`
	UnstakeTotal currency.Coin `json:"unstake_total"`
	TotalStake   currency.Coin `json:"total_stake"`

	TotalServiceCharge  currency.Coin `json:"total_service_charge"`
	ChallengesPassed    uint64        `json:"challenges_passed"`
	ChallengesCompleted uint64        `json:"challenges_completed"`
	InactiveRounds      int64         `json:"InactiveRounds"`
}

func (edb *EventDb) updateBlobberAggregate(round, period int64) {
	logging.Logger.Info("piers updateBlobberAggregate",
		zap.Int64("round", round), zap.Int64("period", period),
	)
	_, oldBlobbers, err := edb.getBlobberSnapshots(round, period)
	if err != nil {
		logging.Logger.Error("piers getting blobber snapshots", zap.Error(err))
		return
	}

	var currentBlobbers []Blobber
	result := edb.Store.Get().
		Raw(fmt.Sprintf("SELECT * FROM blobbers WHERE MOD(creation_round, %d) = ?", period), round%period).
		Scan(&currentBlobbers)
	if result.Error != nil {
		logging.Logger.Error("piers error getting current blobbers", zap.Error(result.Error))
		return
	}
	logging.Logger.Info("piers updateBlobberAggregate",
		zap.Any("currentBlobbers", currentBlobbers))

	if round <= period && len(currentBlobbers) > 0 {
		if err := edb.addBlobberSnapshot(currentBlobbers); err != nil {
			logging.Logger.Error("error saving blobbers snapshots", zap.Error(err))
		}
	}

	var aggregates []BlobberAggregate
	for _, current := range currentBlobbers {
		logging.Logger.Info("piers updateBlobberAggregate for loop", zap.String("id", current.BlobberID))
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
		aggregate.TotalStake = (old.TotalStake + current.TotalStake) / 2
		aggregate.OffersTotal = (old.OffersTotal + current.OffersTotal) / 2
		aggregate.UnstakeTotal = (old.UnstakeTotal + current.UnstakeTotal) / 2

		aggregate.ChallengesPassed = current.ChallengesPassed
		aggregate.ChallengesCompleted = current.ChallengesCompleted
		aggregate.InactiveRounds = current.InactiveRounds
		aggregate.TotalServiceCharge = current.TotalServiceCharge
		aggregates = append(aggregates, aggregate)
	}
	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			logging.Logger.Error("piers saving aggregates", zap.Error(result.Error))
		}
	}
	logging.Logger.Info("piers updateBlobberAggregate",
		zap.Any("new aggregates", aggregates),
	)

	if len(currentBlobbers) > 0 {
		if err := edb.addBlobberSnapshot(currentBlobbers); err != nil {
			logging.Logger.Error("piers error saving blobbers snapshots", zap.Error(err))
		}
	}
}

func (edb *EventDb) GetBlobberAggregate(id string, round int64) (BlobberAggregate, error) {
	var aggregate BlobberAggregate
	res := edb.Store.Get().
		Model(BlobberAggregate{}).
		Where("blobber_id = ? and round <= ?", id, round).
		Order("round desc").
		First(&aggregate)
	return aggregate, res.Error
}

func (edb *EventDb) GetAggregateData(
	from, to int64, dataPoints uint16, aggregate, table, id string,
) ([]float64, error) {
	query := graphDataPointsGeneratorQueryByBlobber(
		from, to, aggregate, dataPoints, table, id,
	)
	var res []float64
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}
func (edb *EventDb) GetDifference(start, end int64, roundsPerPoint int64, row, table, bloberId string) ([]int64, error) {
	if roundsPerPoint < edb.Config().BlobberAggregatePeriod {
		return nil, fmt.Errorf("too many points %v for aggregate period %v",
			roundsPerPoint, edb.Config().BlobberAggregatePeriod)
	}
	query := fmt.Sprintf(`
		SELECT %s - LAG(%s,1, CAST(0 AS Bigint)) OVER(ORDER BY round ASC) 
		FROM %s
		WHERE ( round BETWEEN %v AND %v ) 
				AND ( Mod(round, %v) < %v )
		        AND ( blobber_id = '%v' )
		ORDER BY round ASC	`,
		row, row, table, start, end, roundsPerPoint, edb.dbConfig.BlobberAggregatePeriod-1, bloberId)

	var deltas []int64
	res := edb.Store.Get().Raw(query).Scan(&deltas)
	return deltas, res.Error

}

func graphDataPointsGeneratorQueryByBlobber(
	from, to int64, aggQuery string, dataPoints uint16, table, id string,
) string {
	query := fmt.Sprintf(`
		WITH
		block_info as (
			select b.from as from, b.to as to, ceil((b.to::FLOAT - b.from::FLOAT)/ %d)::INTEGER as step from
				(select min(round) as from, max(round) as to from blocks where creation_date between %d and %d) as b
		),
		ranges AS (
			SELECT t AS r_min, t+(select step from block_info)-1 AS r_max
			FROM generate_series((select "from" from block_info), (select "to" from block_info), (select step from block_info)) as t
		)
		SELECT coalesce(%s, 0) as val
		FROM ranges r
		LEFT JOIN %s s ON (s.round BETWEEN r.r_min AND r.r_max) AND (blobber_id = '%s')
		GROUP BY r.r_min
		ORDER BY r.r_min;
	`, dataPoints, from, to, aggQuery, table, id)

	return query
}
