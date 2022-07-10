package event

// swagger:model Snapshot
type Snapshot struct {
	Round            int64 `gorm:"primaryKey;autoIncrement:false" json:"round"`
	TotalMint        int64 `json:"total_mint"`
	StorageCost      int64 //486 AVG show how much we moved to the challenge pool maybe we should subtract the returned to r/w pools
	StorageCostN     int64
	ActiveAllocated  int64 //496 SUM total amount of new allocation storage in a period (number of allocations active)
	AverageRWPrice   int64 //494 AVG it's the price from the terms and triggered with their updates
	AverageRWPriceN  int64
	TotalStaked      int64 //485 SUM All providers all pools
	QualityScore     int64 //493 AVG percentage of challenges failed by a particular blobber
	QualityScoreN    int64
	ZCNSupply        int64 //488 SUM total ZCN in circulation over a period of time (mints). (Mints - burns) summarized for every round
	AllocatedStorage int64 //490 SUM New allocation calculate the size (new + previous + update -sub fin+cancel or reduceed)
	AvailableStorage int64 //491 SUM available (in the terms)
	StakedStorage    int64 //491 SUM Allocated (allocations)
	UsedStorage      int64 //491 SUM Used - write markers (triggers challenge pool / the price).(bytes written used capacity)
	TotalValueLocked int64 //487 SUM Total value locked = Total staked ZCN * Price per ZCN (across all pools)
	ClientLocks      int64 //487 SUM How many clients locked in (write/read + challenge)  pools
	Capitalization   int64 //489 SUM Token price * minted
	DataUtilization  int64 //492 SUM amount saved across all allocations
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
					Round:            event.Round,
					TotalMint:        last.TotalMint,
					StorageCost:      0,
					StorageCostN:     0,
					ActiveAllocated:  last.ActiveAllocated,
					AverageRWPrice:   0,
					AverageRWPriceN:  0,
					TotalStaked:      last.TotalStaked,
					QualityScore:     0,
					QualityScoreN:    0,
					ZCNSupply:        last.ZCNSupply,
					AllocatedStorage: last.AllocatedStorage,
					AvailableStorage: last.AvailableStorage,
					StakedStorage:    last.StakedStorage,
					UsedStorage:      last.UsedStorage,
					TotalValueLocked: last.TotalValueLocked,
					ClientLocks:      last.ClientLocks,
					Capitalization:   last.Capitalization,
					DataUtilization:  last.DataUtilization,
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
			current.ClientLocks -= d.Amount
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
