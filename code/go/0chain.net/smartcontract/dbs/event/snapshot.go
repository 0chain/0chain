package event

import (
	"0chain.net/chaincore/state"
	"github.com/0chain/common/core/logging"
	"gorm.io/gorm/clause"

	"go.uber.org/zap"

	"github.com/0chain/common/core/currency"
)

//max_capacity - maybe change it max capacity in blobber config and everywhere else to be less confusing.
//staked - staked capacity by delegates
//unstaked - opportunity for delegates to stake until max capacity
//allocated - clients have locked up storage by purchasing allocations
//unallocated - this is equal to (staked - allocated) and allows clients to purchase allocations with free space blobbers.
//used - this is the actual usage or data that is in the server.
//staked + unstaked = max_capacity
//allocated + unallocated = staked

type Snapshot struct {
	Round int64 `gorm:"primaryKey;autoIncrement:false" json:"round"`

	TotalMint            int64 `json:"total_mint"`
	TotalChallengePools  int64 `json:"total_challenge_pools"`  //486 AVG show how much we moved to the challenge pool maybe we should subtract the returned to r/w pools
	ActiveAllocatedDelta int64 `json:"active_allocated_delta"` //496 SUM total amount of new allocation storage in a period (number of allocations active)
	ZCNSupply            int64 `json:"zcn_supply"`             //488 SUM total ZCN in circulation over a period of time (mints). (Mints - burns) summarized for every round
	TotalValueLocked     int64 `json:"total_value_locked"`     //487 SUM Total value locked = Total staked ZCN * Price per ZCN (across all pools)
	ClientLocks          int64 `json:"client_locks"`           //487 SUM How many clients locked in (write/read + challenge)  pools

	// updated from blobber snapshot aggregate table
	AverageWritePrice    int64 `json:"average_write_price"`   //*494 AVG it's the price from the terms and triggered with their updates //???
	TotalStaked          int64 `json:"total_staked"`          //*485 SUM All providers all pools
	SuccessfulChallenges int64 `json:"successful_challenges"` //*493 SUM percentage of challenges failed by a particular blobber
	TotalChallenges      int64 `json:"total_challenges"`      //*493 SUM percentage of challenges failed by a particular blobber
	AllocatedStorage     int64 `json:"allocated_storage"`     //*490 SUM clients have locked up storage by purchasing allocations (new + previous + update -sub fin+cancel or reduceed)
	MaxCapacityStorage   int64 `json:"max_capacity_storage"`  //*491 SUM all storage from blobber settings
	StakedStorage        int64 `json:"staked_storage"`        //*491 SUM staked capacity by delegates
	UsedStorage          int64 `json:"used_storage"`          //*491 SUM this is the actual usage or data that is in the server - write markers (triggers challenge pool / the price).(bytes written used capacity)
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

func (edb *EventDb) ReplicateSnapshots(offset int, limit int) ([]Snapshot, error) {
	var snapshots []Snapshot

	queryBuilder := edb.Store.Get().
		Model(&Snapshot{}).Offset(offset).Limit(limit)

	queryBuilder.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "round"},
		Desc:   false,
	})

	result := queryBuilder.Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	return snapshots, nil
}

type globalSnapshot struct {
	Snapshot
	totalWritePricePeriod currency.Coin
	blobberCount          int
}

func (edb *EventDb) addSnapshot(s Snapshot) error {
	return edb.Store.Get().Create(&s).Error
}

func (edb *EventDb) GetGlobal() (Snapshot, error) {
	s := Snapshot{}
	res := edb.Store.Get().Model(Snapshot{}).Order("round desc").First(&s)
	return s, res.Error
}

func (gs *globalSnapshot) update(e []Event) {
	for _, event := range e {
		logging.Logger.Debug("update snapshot",
			zap.String("tag", event.Tag.String()),
			zap.Int64("block_number", event.BlockNumber))
		switch event.Tag {
		case TagToChallengePool:
			cp, ok := fromEvent[ChallengePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TotalChallengePools += cp.Amount
		case TagFromChallengePool:
			cp, ok := fromEvent[ChallengePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TotalChallengePools -= cp.Amount
		case TagAddMint:
			m, ok := fromEvent[state.Mint](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TotalMint += int64(m.Amount)
			gs.ZCNSupply += int64(m.Amount)
			logging.Logger.Info("snapshot update TagAddMint",
				zap.Any("total mint and zcn mint", gs))
		case TagBurn:
			m, ok := fromEvent[state.Burn](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.ZCNSupply -= int64(m.Amount)
			logging.Logger.Info("snapshot update TagBurn",
				zap.Any("zcn burn", gs))
		case TagLockStakePool:
			d, ok := fromEvent[DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TotalValueLocked += d.Amount
			logging.Logger.Debug("update lock stake pool", zap.Int64("round", event.BlockNumber), zap.Int64("amount", d.Amount),
				zap.Int64("total_amount", gs.TotalValueLocked))
		case TagUnlockStakePool:
			d, ok := fromEvent[DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TotalValueLocked -= d.Amount
		case TagLockWritePool:
			d, ok := fromEvent[WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.ClientLocks += d.Amount
			gs.TotalValueLocked += d.Amount
		case TagUnlockWritePool:
			d, ok := fromEvent[WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.ClientLocks -= d.Amount
			gs.TotalValueLocked -= d.Amount
		case TagLockReadPool:
			d, ok := fromEvent[ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.ClientLocks += d.Amount
			gs.TotalValueLocked += d.Amount
		case TagUnlockReadPool:
			d, ok := fromEvent[ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.ClientLocks -= d.Amount
			gs.TotalValueLocked -= d.Amount
		}
	}
}
