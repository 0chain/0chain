package event

import (
	"fmt"
	"time"

	"0chain.net/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"

	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/dbs"
)

//max_capacity - maybe change it max capacity in blobber config and everywhere else to be less confusing.
//staked - staked capacity by delegates
//unstaked - opportunity for delegates to stake until max capacity
//allocated - clients have locked up storage by purchasing allocations
//unallocated - this is equal to (staked - allocated) and allows clients to purchase allocations with free space blobbers.
//used - this is the actual usage or data that is in the server.
//staked + unstaked = max_capacity
//allocated + unallocated = staked

// swagger:model Snapshot
type Snapshot struct {
	Round                int64 `gorm:"primaryKey;autoIncrement:false" json:"round"`
	TotalMint            int64 `json:"total_mint"`
	StorageCost          int64 //486 AVG show how much we moved to the challenge pool maybe we should subtract the returned to r/w pools
	ActiveAllocatedDelta int64 //496 SUM total amount of new allocation storage in a period (number of allocations active)
	AverageRWPrice       int64 //494 AVG it's the price from the terms and triggered with their updates //???
	TotalStaked          int64 //485 SUM All providers all pools
	SuccessfulChallenges int64 //493 SUM percentage of challenges failed by a particular blobber
	FailedChallenges     int64 //493 SUM percentage of challenges failed by a particular blobber
	ZCNSupply            int64 //488 SUM total ZCN in circulation over a period of time (mints). (Mints - burns) summarized for every round
	AllocatedStorage     int64 //490 SUM clients have locked up storage by purchasing allocations (new + previous + update -sub fin+cancel or reduceed)
	MaxCapacityStorage   int64 //491 SUM all storage from blobber settings
	StakedStorage        int64 //491 SUM staked capacity by delegates
	UsedStorage          int64 //491 SUM this is the actual usage or data that is in the server - write markers (triggers challenge pool / the price).(bytes written used capacity)
	TotalValueLocked     int64 //487 SUM Total value locked = Total staked ZCN * Price per ZCN (across all pools)
	ClientLocks          int64 //487 SUM How many clients locked in (write/read + challenge)  pools
	Capitalization       int64 //489 SUM Token price * minted
	DataUtilization      int64 //492 SUM amount saved across all allocations
}

type FieldType int

const (
	Allocated = iota
	MaxCapacity
	Staked
	Used
)

type AllocationValueChanged struct {
	FieldType    FieldType
	AllocationId string
	Delta        int64
}
type AllocationBlobberValueChanged struct {
	FieldType    FieldType
	AllocationId string
	BlobberId    string
	Delta        int64
}

func (edb *EventDb) GetRoundsMintTotal(from, to time.Time, dataPoints uint16) ([]int64, error) {
	var totals []int64

	query := graphDataPointsGeneratorQuery(from.UnixNano(), to.UnixNano(), "sum(total_mint)", dataPoints)
	return totals, edb.Store.Get().Raw(query).Scan(&totals).Error
}

func (edb *EventDb) GetDataStorageCosts(from, to time.Time, dataPoints uint16) ([]float64, error) {
	var res []float64
	//486 AVG show how much we moved to the challenge pool maybe we should subtract the returned to r/w pools
	query := graphDataPointsGeneratorQuery(from.UnixNano(), to.UnixNano(), "avg(storage_cost)", dataPoints)
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}

func (edb *EventDb) GetDailyAllocations(from, to time.Time, dataPoints uint16) ([]int64, error) {
	var res []int64
	//496 SUM total amount of new allocation storage in a period (number of allocations active)
	query := graphDataPointsGeneratorQuery(from.UnixNano(), to.UnixNano(), "sum(active_allocated_delta)", dataPoints)
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}

func (edb *EventDb) GetDataReadWritePrice(from, to time.Time, dataPoints uint16) ([]float64, error) {
	var res []float64
	//494 AVG it's the price from the terms and triggered with their updates
	query := graphDataPointsGeneratorQuery(from.UnixNano(), to.UnixNano(), "avg(average_rw_price)", dataPoints)
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}

func (edb *EventDb) GetTotalStaked(from, to time.Time, dataPoints uint16) ([]int64, error) {
	var res []int64
	//485 SUM All providers all pools
	query := graphDataPointsGeneratorQuery(from.UnixNano(), to.UnixNano(), "sum(total_staked)", dataPoints)
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}

func (edb *EventDb) GetNetworkQualityScores(from, to time.Time, dataPoints uint16) ([]int64, error) {
	var res []int64
	//493 SUM percentage of challenges failed by a particular blobber
	query := graphDataPointsGeneratorQuery(
		from.UnixNano(),
		to.UnixNano(),
		"(((sum(successful_challenges)/(sum(failed_challenges) + sum(successful_challenges))) * 100)::INT)",
		dataPoints,
	)
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}

func (edb *EventDb) GetZCNSupply(from, to time.Time, dataPoints uint16) ([]int64, error) {
	var res []int64
	//488 SUM total ZCN in circulation over a period of time (mints). (Mints - burns) summarized for every round
	query := graphDataPointsGeneratorQuery(from.UnixNano(), to.UnixNano(), "sum(zcn_supply)", dataPoints)
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}

func (edb *EventDb) GetAllocatedStorage(from, to time.Time, dataPoints uint16) ([]int64, error) {
	var res []int64
	//490 SUM New allocation calculate the size (new + previous + update -sub fin+cancel or reduceed)
	query := graphDataPointsGeneratorQuery(from.UnixNano(), to.UnixNano(), "sum(allocated_storage)", dataPoints)
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}

func (edb *EventDb) GetCloudGrowthData(from, to time.Time, dataPoints uint16) ([]int64, error) {
	var res []int64
	//491 SUM available (in the terms)
	query := graphDataPointsGeneratorQuery(from.Unix(), to.Unix(), "sum(allocated_storage) - sum(used_storage)", dataPoints)
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}

func (edb *EventDb) GetTotalLocked(from, to time.Time, dataPoints uint16) ([]int64, error) {
	var res []int64
	//487 SUM Total value locked = Total staked ZCN * Price per ZCN (across all pools)
	query := graphDataPointsGeneratorQuery(from.UnixNano(), to.UnixNano(), "sum(total_value_locked)", dataPoints)
	return res, edb.Store.Get().Raw(query).Scan(&res).Error

}

func (edb *EventDb) GetDataCap(from, to time.Time, dataPoints uint16) ([]float64, error) {
	var res []float64
	//489 SUM Token price * minted
	query := graphDataPointsGeneratorQuery(from.UnixNano(), to.UnixNano(), "avg(capitalization)", dataPoints)
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}

func (edb *EventDb) GetDataUtilization(from, to time.Time, dataPoints uint16) ([]int64, error) {
	var res []int64
	//492 SUM amount saved across all allocations
	query := graphDataPointsGeneratorQuery(from.UnixNano(), to.UnixNano(), "sum(data_utilization)", dataPoints)
	return res, edb.Store.Get().Raw(query).Scan(&res).Error
}

func (edb *EventDb) updateSnapshot(e events) {
	if len(e) == 0 {
		return
	}
	thisRound := e[0].BlockNumber
	var current Snapshot
	var err error
	if thisRound > 1 {
		current, err = edb.getSnapshot(thisRound)
		if err != nil {
			current, err = edb.getSnapshot(thisRound - 1)
			if err != nil {
				logging.Logger.Error("getting last snapshot", zap.Int64("last round", thisRound-1), zap.Error(err))
			}
			current.StorageCost = 0
			current.ActiveAllocatedDelta = 0
			current.AverageRWPrice = 0
			current.SuccessfulChallenges = 0
			current.FailedChallenges = 0
		}
	}
	current.Round = thisRound

	for _, event := range e {
		switch EventTag(event.Tag) {
		case TagAddMint:
			u, ok := fromEvent[User](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			change, err := u.Change.Int64()
			if err != nil {
				logging.Logger.Error("snapshot", zap.Error(err))
				continue
			}
			current.TotalMint += change
			current.ZCNSupply += change
		case TagBurn:
			b, ok := fromEvent[currency.Coin](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			i2, err := b.Int64()
			if err != nil {
				logging.Logger.Error("snapshot", zap.Error(err))
				continue
			}
			current.ZCNSupply -= i2
		case TagLockStakePool:
			d, ok := fromEvent[DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			current.TotalStaked += d.Amount
			current.TotalValueLocked += d.Amount
		case TagUnlockStakePool:
			d, ok := fromEvent[DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			current.TotalStaked -= d.Amount
			current.TotalValueLocked -= d.Amount
		case TagLockWritePool:
			d, ok := fromEvent[WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			current.ClientLocks += d.Amount
			current.TotalValueLocked += d.Amount
		case TagUnlockWritePool:
			d, ok := fromEvent[WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			current.ClientLocks -= d.Amount
			current.TotalValueLocked -= d.Amount
		case TagLockReadPool:
			d, ok := fromEvent[ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			current.ClientLocks += d.Amount
			current.TotalValueLocked += d.Amount
		case TagUnlockReadPool:
			d, ok := fromEvent[ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			current.ClientLocks -= d.Amount
			current.TotalValueLocked -= d.Amount
		case TagToChallengePool:
			d, ok := fromEvent[ChallengePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			current.StorageCost += d.Amount
		case TagUpdateChallenge:
			updates, ok := fromEvent[dbs.DbUpdates](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			var is interface{}
			is, ok = updates.Updates["responded"]
			if ok {
				b := is.(bool)
				if b {
					current.SuccessfulChallenges++
				} else {
					current.FailedChallenges++
				}
			}
		case TagAllocValueChange:
			updates, ok := fromEvent[AllocationValueChanged](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			switch updates.FieldType {
			case Allocated:
				current.ActiveAllocatedDelta += updates.Delta
				current.AllocatedStorage += updates.Delta
			}
		case TagAllocBlobberValueChange:
			updates, ok := fromEvent[AllocationBlobberValueChanged](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			switch updates.FieldType {
			case MaxCapacity:
				current.MaxCapacityStorage += updates.Delta
			case Staked:
				current.StakedStorage += updates.Delta
			}
		case TagAddWriteMarker:
			updates, ok := fromEvent[WriteMarker](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			current.UsedStorage += updates.Size
			current.DataUtilization = current.AllocatedStorage / current.UsedStorage
		}
	}

	if err := edb.addOrOverwriteSnapshot(current); err != nil {
		logging.Logger.Error("snapshot", zap.Error(err))
	}
}

func (edb *EventDb) getSnapshot(round int64) (Snapshot, error) {
	s := Snapshot{}
	res := edb.Store.Get().Model(Snapshot{}).Where(Snapshot{Round: round}).First(&s)
	return s, res.Error
}

func (edb *EventDb) addOrOverwriteSnapshot(s Snapshot) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&s).Error
}

func graphDataPointsGeneratorQuery(from, to int64, aggQuery string, dataPoints uint16) string {
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
		LEFT JOIN snapshots s ON s.round BETWEEN r.r_min AND r.r_max
		GROUP BY r.r_min
		ORDER BY r.r_min;
	`, dataPoints, from, to, aggQuery)

	return query
}
