package event

import (
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

func (edb *EventDb) GetRoundsMintTotal(from, to int64) ([]int64, error) {
	var totals []int64

	//WITH ranges AS (
	//    SELECT (ten*10)::text||'-'||(ten*10+9)::text AS range,
	//           ten*10 AS r_min, ten*10+9 AS r_max
	//      FROM generate_series(0,90) AS t(ten))
	//SELECT r.range, count(s.*), sum(total_mint)
	//  FROM ranges r
	//  LEFT JOIN snapshots s ON s.round BETWEEN r.r_min AND r.r_max
	// GROUP BY r.range
	// ORDER BY r.range;

	return totals, nil

}

func (edb *EventDb) updateSnapshot(e events) error {
	current := Snapshot{}
	for i, event := range e {
		if i == 0 { //first event on this round
			previousRound := event.Round - 1
			if previousRound > -1 {
				last, err := edb.getSnapshot(int64(i))
				if err != nil {
					return err
				}
				current = Snapshot{
					Round:                event.Round,
					TotalMint:            last.TotalMint,
					StorageCost:          0,
					ActiveAllocatedDelta: 0,
					AverageRWPrice:       0,
					TotalStaked:          last.TotalStaked,
					SuccessfulChallenges: 0,
					FailedChallenges:     0,
					ZCNSupply:            last.ZCNSupply,
					AllocatedStorage:     last.AllocatedStorage,
					MaxCapacityStorage:   last.MaxCapacityStorage,
					UsedStorage:          last.UsedStorage,
					StakedStorage:        last.StakedStorage,
					TotalValueLocked:     last.TotalValueLocked,
					ClientLocks:          last.ClientLocks,
					Capitalization:       last.Capitalization,
					DataUtilization:      last.DataUtilization,
				}
			}
		}

		//	TagSendTransfer
		//	TagReceiveTransfer
		//	TagLockStakePool
		//	TagUnlockStakePool
		//	TagLockWritePool
		//	TagUnlockWritePool
		//	TagLockReadPool
		//	TagUnlockReadPool
		//	TagToChallengePool
		//	TagFromChallengePool
		//	TagAddMint
		switch EventTag(event.Tag) {
		case TagAddMint:
			u, ok := fromEvent[User](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			change, err := u.Change.Int64()
			if err != nil {
				return err
			}
			current.TotalMint += change
			current.ZCNSupply += change
		case TagBurn:
			b, ok := fromEvent[currency.Coin](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			i2, err := b.Int64()
			if err != nil {
				return ErrInvalidEventData
			}
			current.ZCNSupply -= i2
		case TagLockStakePool:
			d, ok := fromEvent[DelegatePoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			current.TotalStaked += d.Amount
		case TagUnlockStakePool:
			d, ok := fromEvent[DelegatePoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			current.TotalStaked -= d.Amount
		case TagLockWritePool:
			d, ok := fromEvent[WritePoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			current.ClientLocks += d.Amount
		case TagUnlockWritePool:
			d, ok := fromEvent[WritePoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			current.ClientLocks -= d.Amount
		case TagLockReadPool:
			d, ok := fromEvent[ReadPoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			current.ClientLocks += d.Amount
		case TagUnlockReadPool:
			d, ok := fromEvent[ReadPoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			current.ClientLocks -= d.Amount
		case TagToChallengePool:
			d, ok := fromEvent[ChallengePoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			current.StorageCost += d.Amount
		case TagUpdateChallenge:
			updates, ok := fromEvent[dbs.DbUpdates](event.Data)
			is, ok := updates.Updates["responded"]
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
				return ErrInvalidEventData
			}
			switch updates.FieldType {
			case Allocated:
				current.ActiveAllocatedDelta += updates.Delta
				current.AllocatedStorage += updates.Delta
			}
		case TagAllocBlobberValueChange:
			updates, ok := fromEvent[AllocationBlobberValueChanged](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			switch updates.FieldType {
			case MaxCapacity:
				current.MaxCapacityStorage += updates.Delta
			case Staked:
				current.StakedStorage += updates.Delta
			}

		}

	}
	if err := edb.addSnapshot(current); err != nil {
		return err
	}

	return nil
}

func (edb *EventDb) getSnapshot(round int64) (Snapshot, error) {
	s := Snapshot{}
	res := edb.Store.Get().Model(Snapshot{}).Where(Snapshot{Round: round}).First(&s)
	return s, res.Error
}

func (edb *EventDb) addSnapshot(s Snapshot) error {
	res := edb.Store.Get().Create(&s)
	return res.Error
}
