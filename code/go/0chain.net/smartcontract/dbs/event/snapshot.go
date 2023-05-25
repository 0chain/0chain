package event

import (
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs"
	"github.com/0chain/common/core/logging"

	"go.uber.org/zap"
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
	ClientLocks          int64 `json:"client_locks"`           //487 SUM How many clients locked in (write/read + challenge)  pools
	TotalReadPoolLocked	 int64 `json:"total_read_pool_locked"` //487 SUM How many tokens are locked in all read pools
	MinedTotal           int64 `json:"mined_total"`            // SUM total mined for all providers, never decrease
	// updated from blobber snapshot aggregate table
	TotalStaked          int64 `json:"total_staked"`                     //*485 SUM All providers all pools
	StorageTokenStake	 int64 `json:"storage_token_stake"`              //*485 SUM of all stake amount for storage blobbers
	TotalRewards         int64 `json:"total_rewards"`                    //SUM total of all rewards
	SuccessfulChallenges int64 `json:"successful_challenges"`            //*493 SUM percentage of challenges failed by a particular blobber
	TotalChallenges      int64 `json:"total_challenges"`                 //*493 SUM percentage of challenges failed by a particular blobber
	AllocatedStorage     int64 `json:"allocated_storage"`                //*490 SUM clients have locked up storage by purchasing allocations (new + previous + update -sub fin+cancel or reduceed)
	MaxCapacityStorage   int64 `json:"max_capacity_storage"`             //*491 SUM all storage from blobber settings
	StakedStorage        int64 `json:"staked_storage"`                   //*491 SUM staked capacity by delegates
	UsedStorage          int64 `json:"used_storage"`                     //*491 SUM this is the actual usage or data that is in the server - write markers (triggers challenge pool / the price).(bytes written used capacity)
	TransactionsCount    int64 `json:"transactions_count"`               // Total number of transactions in a block
	UniqueAddresses      int64 `json:"unique_addresses"`                 // Total unique address
	BlockCount           int64 `json:"block_count"`                      // Total number of blocks currently
	TotalTxnFee        int64 `json:"avg_txn_fee"`                        // Total fees of all transactions
	CreatedAt            int64 `gorm:"autoCreateTime" json:"created_at"` // Snapshot creation date
	BlobberCount		 int64 `json:"blobber_count"`                    // Total number of blobbers
	MinerCount			 int64 `json:"miner_count"`                      // Total number of miners
	SharderCount		 int64 `json:"sharder_count"`                    // Total number of sharders
	ValidatorCount		 int64 `json:"validator_count"`                  // Total number of validators
	AuthorizerCount		 int64 `json:"authorizer_count"`                  // Total number of authorizers
	MinerTotalRewards	 int64 `json:"miner_total_rewards"`              // Total rewards of miners
	SharderTotalRewards	 int64 `json:"sharder_total_rewards"`            // Total rewards of sharders
	BlobberTotalRewards  int64 `json:"blobber_total_rewards"`            // Total rewards of blobbers
}

// ApplyDiff applies diff values of global snapshot fields to the current snapshot according to each field's update formula.
func (s *Snapshot) ApplyDiff(diff *Snapshot) {
	s.TotalMint += diff.TotalMint
	s.TotalChallengePools += diff.TotalChallengePools
	s.ActiveAllocatedDelta += diff.ActiveAllocatedDelta
	s.ZCNSupply += diff.ZCNSupply
	s.ClientLocks += diff.ClientLocks
	s.TotalReadPoolLocked += diff.TotalReadPoolLocked
	s.MinedTotal += diff.MinedTotal
	s.TotalStaked += diff.TotalStaked
	s.TotalRewards += diff.TotalRewards
	s.MinerTotalRewards += diff.MinerTotalRewards
	s.SharderTotalRewards += diff.SharderTotalRewards
	s.BlobberTotalRewards += diff.BlobberTotalRewards
	s.StorageTokenStake += diff.StorageTokenStake
	s.SuccessfulChallenges += diff.SuccessfulChallenges
	s.TotalChallenges += diff.TotalChallenges
	s.AllocatedStorage += diff.AllocatedStorage
	s.MaxCapacityStorage += diff.MaxCapacityStorage
	s.StakedStorage +=  diff.StakedStorage
	s.UsedStorage += diff.UsedStorage
	s.TransactionsCount += diff.TransactionsCount
	s.UniqueAddresses += diff.UniqueAddresses
	s.BlockCount += diff.BlockCount
	s.TotalTxnFee += diff.TotalTxnFee
	s.BlobberCount += diff.BlobberCount
	s.MinerCount += diff.MinerCount
	s.SharderCount += diff.SharderCount
	s.ValidatorCount += diff.ValidatorCount
	s.AuthorizerCount += diff.AuthorizerCount

	if s.StakedStorage > s.MaxCapacityStorage {
		s.StakedStorage = s.MaxCapacityStorage
	}
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

func (edb *EventDb) ReplicateSnapshots(round int64, limit int) ([]Snapshot, error) {
	var snapshots []Snapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM snapshots WHERE round > ? ORDER BY round LIMIT ?", round, limit).Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}
	return snapshots, nil
}

func (edb *EventDb) addSnapshot(s Snapshot) error {
	return edb.Store.Get().Create(&s).Error
}

func (edb *EventDb) GetGlobal() (Snapshot, error) {
	s := Snapshot{}
	res := edb.Store.Get().Model(Snapshot{}).Order("round desc").First(&s)
	return s, res.Error
}

func (gs *Snapshot) update(e []Event) {
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
				zap.Int64("total_mint", gs.TotalMint), zap.Int64("zcn_supply", gs.ZCNSupply))
		case TagBurn:
			m, ok := fromEvent[state.Burn](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.ZCNSupply -= int64(m.Amount)
			logging.Logger.Info("snapshot update TagBurn",
				zap.Int64("zcn_supply", gs.ZCNSupply))
		case TagLockWritePool:
			ds, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			for _, d := range *ds {
				gs.ClientLocks += d.Amount
			}
		case TagUnlockWritePool:
			ds, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			for _, d := range *ds {
				gs.ClientLocks -= d.Amount
			}
		case TagLockReadPool:
			ds, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			for _, d := range *ds {
				gs.ClientLocks += d.Amount
				gs.TotalReadPoolLocked += d.Amount
			}
		case TagUnlockReadPool:
			ds, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			for _, d := range *ds {
				gs.ClientLocks -= d.Amount
				gs.TotalReadPoolLocked -= d.Amount
			}
		case TagStakePoolReward:
			spus, ok := fromEvent[[]dbs.StakePoolReward](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			for _, spu := range *spus {
				for _, r := range spu.DelegateRewards {
					dr, err := r.Int64()
					if err != nil {
						logging.Logger.Error("snapshot",
							zap.Any("event", event.Data), zap.Error(err))
						continue
					}
					gs.MinedTotal += dr
				}
			}
		case TagFinalizeBlock:
			gs.BlockCount += 1
		case TagUniqueAddress:
			gs.UniqueAddresses += 1
		case TagAddTransactions:
			txns, ok := fromEvent[[]Transaction](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue
			}
			gs.TransactionsCount += int64(len(*txns))
			totalFee := 0
			for _, txn := range *txns {
				totalFee += int(txn.Fee)
			}
			gs.TotalTxnFee += int64(totalFee)
		}

	}
}


type ProcessingEntity struct {
	Entity interface{}
	Processed bool
}

// MakeProcessingMap wraps map entries into a struct with "Processed" boolean flag
func MakeProcessingMap[T any](mapIn map[string]T) (map[string]ProcessingEntity) {
	mpOut := make(map[string]ProcessingEntity)
	for k, v := range mapIn {
		mpOut[k] = ProcessingEntity{Entity: v, Processed: false}
	}
	return mpOut
}
