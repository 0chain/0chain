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

	OffersTotal        currency.Coin `json:"offers_total"`
	UnstakeTotal       currency.Coin `json:"unstake_total"`
	TotalServiceCharge currency.Coin `json:"total_service_charge"`
	TotalStake         currency.Coin `json:"total_stake"`

	ChallengesPassed    uint64 `json:"challenges_passed"`
	ChallengesCompleted uint64 `json:"challenges_completed"`
	InactiveRounds      int64  `json:"inactive_rounds"`
}

func (edb *EventDb) updateBlobberAggregate(round, period int64) {
	ids, oldBlobbers, err := edb.getBlobberSnapshots(round, period)
	if err != nil {
		logging.Logger.Error("getting blobber snapshots", zap.Error(err))
		return
	}

	currentBlobbers, err := edb.GetBlobbersFromIDs(ids)
	if err != nil {
		logging.Logger.Error("getting blobbers", zap.Error(err))
		return
	}

	var aggregates []BlobberAggregate
	for _, current := range currentBlobbers {
		old := oldBlobbers[current.BlobberID]
		aggregate := BlobberAggregate{
			Round:     round,
			BlobberID: current.BlobberID,
		}
		aggregate.WritePrice = (old.WritePrice + current.WritePrice) / 2
		aggregate.Capacity = (old.Capacity + current.Capacity) / 2
		aggregate.Allocated = (old.Allocated + current.Allocated) / 2
		aggregate.SavedData = (old.SavedData + current.SavedData) / 2

		aggregate.ChallengesPassed = current.ChallengesPassed - old.ChallengesPassed
		aggregate.ChallengesCompleted = current.ChallengesCompleted - old.ChallengesPassed
		aggregate.InactiveRounds = current.InactiveRounds - old.InactiveRounds
		aggregates = append(aggregates, aggregate)
	}

	if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
		logging.Logger.Error("saving aggregates", zap.Error(result.Error))
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

func (edb *EventDb) GetAggregateData(from, to int64, dataPoints uint16, aggregate, table string) ([]float64, error) {
	query := graphDataPointsGeneratorQuery2(
		from, to, aggregate, dataPoints, "blobber_aggregates",
	)
	var res []float64
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}
func (edb *EventDb) GetDifference(start, end int64, roundsPerPoint int64, row, table, bloberId string) ([]int64, error) {
	if roundsPerPoint < edb.Config().BlobberAggregatePeriod {
		//common.Respond(w, r, nil, common.NewErrInternal("too many points for aggregate period"))
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

func graphDataPointsGeneratorQuery2(from, to int64, aggQuery string, dataPoints uint16, table string) string {
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
		LEFT JOIN %s s ON s.round BETWEEN r.r_min AND r.r_max
		GROUP BY r.r_min
		ORDER BY r.r_min;
	`, dataPoints, from, to, aggQuery, table)

	return query
}
