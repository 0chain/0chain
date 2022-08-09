package event

import (
	"fmt"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"0chain.net/chaincore/currency"
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
	Round int64 `gorm:"primaryKey;autoIncrement:false" json:"round"`

	TotalMint            int64 `json:"total_mint"`
	TotalChllengePools   int64 //486 AVG show how much we moved to the challenge pool maybe we should subtract the returned to r/w pools
	ActiveAllocatedDelta int64 //496 SUM total amount of new allocation storage in a period (number of allocations active)
	ZCNSupply            int64 //488 SUM total ZCN in circulation over a period of time (mints). (Mints - burns) summarized for every round
	TotalValueLocked     int64 //487 SUM Total value locked = Total staked ZCN * Price per ZCN (across all pools)
	ClientLocks          int64 //487 SUM How many clients locked in (write/read + challenge)  pools

	// updated from blobber snapshot aggregate table
	AverageWritePrice    int64 //*494 AVG it's the price from the terms and triggered with their updates //???
	TotalStaked          int64 //*485 SUM All providers all pools
	SuccessfulChallenges int64 //*493 SUM percentage of challenges failed by a particular blobber
	TotalChallenges      int64 //*493 SUM percentage of challenges failed by a particular blobber
	AllocatedStorage     int64 //*490 SUM clients have locked up storage by purchasing allocations (new + previous + update -sub fin+cancel or reduceed)
	MaxCapacityStorage   int64 //*491 SUM all storage from blobber settings
	StakedStorage        int64 //*491 SUM staked capacity by delegates
	UsedStorage          int64 //*491 SUM this is the actual usage or data that is in the server - write markers (triggers challenge pool / the price).(bytes written used capacity)
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

func (edb *EventDb) GetDifference(start, end int64, roundsPerPoint int64, row, table string) ([]int64, error) {
	query := fmt.Sprintf(`
		SELECT %s - LAG(%s,1, CAST(0 AS Bigint)) OVER(ORDER BY round ASC) 
		FROM %s
		WHERE ( round BETWEEN %v AND %v ) 
				AND ( Mod(round, %v) < %v )
		ORDER BY round ASC	`,
		row, row, table, start, end, roundsPerPoint, edb.dbConfig.AggregatePeriod-1)

	var deltas []int64
	res := edb.Store.Get().Raw(query).Scan(&deltas)
	return deltas, res.Error
}

func (edb *EventDb) GetAverage(start, end int64, roundsPerPoint int64, row, table string) ([]int64, error) {
	query := fmt.Sprintf(`
		SELECT ( %s + LAG(%s,1, CAST(0 AS Bigint)) OVER(ORDER BY round ASC) )/2
		FROM %s
		WHERE ( round BETWEEN %v AND %v ) 
				AND ( Mod(round, %v) < %v )
		ORDER BY round ASC	`,
		row, row, table, start, end, roundsPerPoint, edb.dbConfig.AggregatePeriod-1)

	var deltas []int64
	res := edb.Store.Get().Raw(query).Scan(&deltas)
	return deltas, res.Error
}

func (edb *EventDb) GetTotal(start, end int64, roundsPerPoint int64, row, table string) ([]int64, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s
		WHERE ( round BETWEEN %v AND %v ) 
				AND ( Mod(round, %v) < %v )
		ORDER BY round ASC	`,
		row, table, start, end, roundsPerPoint, edb.dbConfig.AggregatePeriod-1)

	var deltas []int64
	res := edb.Store.Get().Raw(query).Scan(&deltas)
	return deltas, res.Error
}

type globalSnapshot struct {
	Snapshot
	totalWritePricePeriod currency.Coin
	blobberCount          int
}

func newGlobalSnapshot() *globalSnapshot {
	return &globalSnapshot{}
}

func (edb *EventDb) addSnapshot(s Snapshot) error {
	return edb.Store.Get().Create(&s).Error
}

func (edb *EventDb) getSnapshot(round int64) (Snapshot, error) {
	s := Snapshot{}
	res := edb.Store.Get().Model(Snapshot{}).Where(Snapshot{Round: round}).First(&s)
	return s, res.Error
}

func (edb *EventDb) GetGlobal() (Snapshot, error) {
	s := Snapshot{}
	res := edb.Store.Get().Model(Snapshot{}).Order("round desc").First(&s)
	return s, res.Error
}

func (gs *globalSnapshot) update(e []Event) {
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
			gs.TotalMint += change
			gs.ZCNSupply += change
			logging.Logger.Info("piers snapshot update",
				zap.Any("total mint and zcn mint", gs))
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
			gs.ZCNSupply -= i2
			logging.Logger.Info("piers snapshot update",
				zap.Any("zcn burn", gs))
		case TagLockStakePool:
			d, ok := fromEvent[DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TotalValueLocked += d.Amount
			logging.Logger.Info("piers snapshot update",
				zap.Any("lock stake pool", gs))
		case TagUnlockStakePool:
			d, ok := fromEvent[DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TotalValueLocked -= d.Amount
			logging.Logger.Info("piers snapshot update",
				zap.Any("unlock stake pool", gs))
		case TagLockWritePool:
			d, ok := fromEvent[WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.ClientLocks += d.Amount
			gs.TotalValueLocked += d.Amount
			logging.Logger.Info("piers snapshot update",
				zap.Any("lock write pool", gs))
		case TagUnlockWritePool:
			d, ok := fromEvent[WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.ClientLocks -= d.Amount
			gs.TotalValueLocked -= d.Amount
			logging.Logger.Info("piers snapshot update",
				zap.Any("unlock write pool", gs))
		case TagLockReadPool:
			d, ok := fromEvent[ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.ClientLocks += d.Amount
			gs.TotalValueLocked += d.Amount
			logging.Logger.Info("piers snapshot update",
				zap.Any("lock read pool", gs))
		case TagUnlockReadPool:
			d, ok := fromEvent[ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.ClientLocks -= d.Amount
			gs.TotalValueLocked -= d.Amount
			logging.Logger.Info("piers snapshot update",
				zap.Any("unlock write pool", gs))
		}
	}
}
