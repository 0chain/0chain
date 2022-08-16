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

//query := graphDataPointsGeneratorQueryByBlobber(
//	from, to, aggregate, dataPoints, table, id,
//)
//var res []float64
//return res, edb.Store.Get().Raw(query).Scan(&res).Error
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
	from, to int64, dataPoints int64, row, table, id string,
) ([]int64, error) {
	return edb.GetTotalId(from, to, dataPoints, row, table, id)
}

func (edb *EventDb) GetTotalId(start, end, roundsPerPoint int64, row, table, blobberId string) ([]int64, error) {
	if roundsPerPoint < edb.Config().AggregatePeriod {
		return nil, fmt.Errorf("too many points %v for aggregate period %v",
			roundsPerPoint, edb.Config().AggregatePeriod)
	}
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s
		WHERE ( round BETWEEN %v AND %v ) 
				AND ( Mod(round, %v) < %v )
		        AND ( blobber_id = '%v' )
		ORDER BY round ASC	`,
		row, table, start, end, roundsPerPoint, edb.dbConfig.AggregatePeriod-1, blobberId)

	var deltas []int64
	res := edb.Store.Get().Raw(query).Scan(&deltas)
	return deltas, res.Error
}

func (edb *EventDb) GetDifferenceId(start, end int64, roundsPerPoint int64, row, table, blobberId string) ([]int64, error) {
	if roundsPerPoint < edb.Config().AggregatePeriod {
		return nil, fmt.Errorf("too many points %v for aggregate period %v",
			roundsPerPoint, edb.Config().AggregatePeriod)
	}
	query := fmt.Sprintf(`
		SELECT %s - LAG(%s,1, CAST(0 AS Bigint)) OVER(ORDER BY round ASC) 
		FROM %s
		WHERE ( round BETWEEN %v AND %v ) 
				AND ( Mod(round, %v) < %v )
		        AND ( blobber_id = '%v' )
		ORDER BY round ASC	`,
		row, row, table, start, end, roundsPerPoint, edb.dbConfig.AggregatePeriod-1, blobberId)

	var deltas []int64
	res := edb.Store.Get().Raw(query).Scan(&deltas)
	return deltas, res.Error
}
