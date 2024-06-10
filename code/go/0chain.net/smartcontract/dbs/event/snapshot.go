package event

import (
	"fmt"
	"reflect"

	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
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

type FieldType int

type ProviderIDMap map[string]dbs.ProviderID

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

type IProviderSnapshot interface {
	GetID() string
	GetRound() int64
	SetID(id string)
	SetRound(round int64)
}

// swagger:model Snapshot
type Snapshot struct {
	Round int64 `gorm:"primaryKey;autoIncrement:false" json:"round"`

	TotalMint            int64 `json:"total_mint"`
	TotalChallengePools  int64 `json:"total_challenge_pools"`  //486 AVG show how much we moved to the challenge pool maybe we should subtract the returned to r/w pools
	ActiveAllocatedDelta int64 `json:"active_allocated_delta"` //496 SUM total amount of new allocation storage in a period (number of allocations active)
	ZCNSupply            int64 `json:"zcn_supply"`             //488 SUM total ZCN in circulation over a period of time (mints). (Mints - burns) summarized for every round
	ClientLocks          int64 `json:"client_locks"`           //487 SUM How many clients locked in (write/read + challenge)  pools
	TotalReadPoolLocked  int64 `json:"total_read_pool_locked"` //487 SUM How many tokens are locked in all read pools
	// updated from blobber snapshot aggregate table
	TotalStaked          int64 `json:"total_staked"`                     //*485 SUM All providers all pools
	StorageTokenStake    int64 `json:"storage_token_stake"`              //*485 SUM of all stake amount for storage blobbers
	TotalRewards         int64 `json:"total_rewards"`                    //SUM total of all rewards
	SuccessfulChallenges int64 `json:"successful_challenges"`            //*493 SUM percentage of challenges failed by a particular blobber
	TotalChallenges      int64 `json:"total_challenges"`                 //*493 SUM percentage of challenges failed by a particular blobber
	AllocatedStorage     int64 `json:"allocated_storage"`                //*490 SUM clients have locked up storage by purchasing allocations (new + previous + update -sub fin+cancel or reduceed)
	TotalAllocations	 int64 `json:"total_allocations"`                //*490 SUM total number of allocations
	MaxCapacityStorage   int64 `json:"max_capacity_storage"`             //*491 SUM all storage from blobber settings
	StakedStorage        int64 `json:"staked_storage"`                   //*491 SUM staked capacity by delegates
	UsedStorage          int64 `json:"used_storage"`                     //*491 SUM this is the actual usage or data that is in the server - write markers (triggers challenge pool / the price).(bytes written used capacity)
	TransactionsCount    int64 `json:"transactions_count"`               // Total number of transactions in a block
	UniqueAddresses      int64 `json:"unique_addresses"`                 // Total unique address
	BlockCount           int64 `json:"block_count"`                      // Total number of blocks currently
	TotalTxnFee          int64 `json:"avg_txn_fee"`                      // Total fees of all transactions
	CreatedAt            int64 `gorm:"autoCreateTime" json:"created_at"` // Snapshot creation date
	BlobberCount         int64 `json:"blobber_count"`                    // Total number of blobbers
	MinerCount           int64 `json:"miner_count"`                      // Total number of miners
	SharderCount         int64 `json:"sharder_count"`                    // Total number of sharders
	ValidatorCount       int64 `json:"validator_count"`                  // Total number of validators
	AuthorizerCount      int64 `json:"authorizer_count"`                 // Total number of authorizers
	MinerTotalRewards    int64 `json:"miner_total_rewards"`              // Total rewards of miners
	SharderTotalRewards  int64 `json:"sharder_total_rewards"`            // Total rewards of sharders
	BlobberTotalRewards  int64 `json:"blobber_total_rewards"`            // Total rewards of blobbers
}

const (
	Allocated = iota
	MaxCapacity
	Staked
	Used
)

const GB = float64(1024 * 1024 * 1024)

// ApplyDiff applies diff values of global snapshot fields to the current snapshot according to each field's update formula.
//
// Parameters:
//   - diff: diff values of global snapshot fields
//   - gs: current global snapshot.
func ApplyProvidersDiff[P IProvider, S IProviderSnapshot](edb *EventDb, gs *Snapshot, providers []P) error {
	var (
		snaphots     []S
		snapshotsMap = make(map[string]S)
		snapIds      = make([]string, 0, len(providers))
		pModel       P
		sModel       S
		pReflectType = reflect.TypeOf(pModel).Elem()
		sReflectType = reflect.TypeOf(sModel).Elem()
		ptypeName    = ProviderTextMapping[pReflectType]
	)
	for _, provider := range providers {
		snapIds = append(snapIds, provider.GetID())
	}

	err := edb.Store.Get().Where(fmt.Sprintf("%v_id IN (?)", ptypeName), snapIds).Find(&snaphots).Error
	if err != nil {
		return common.NewError("apply_providers_diff", "error getting providers snapshots from db")
	}
	for _, snapshot := range snaphots {
		snapshotsMap[snapshot.GetID()] = snapshot
	}

	// Active providers
	isNew := false
	if len(providers) > 0 {
		for _, provider := range providers {
			snap, ok := snapshotsMap[provider.GetID()]
			if !ok {
				snap = reflect.New(sReflectType).Interface().(S)
				snap.SetID(provider.GetID())
				isNew = true
			}

			err = gs.ApplySingleProviderDiff(spenum.ToProviderType(ptypeName))(provider, snap, isNew)
			if err != nil {
				logging.Logger.Error("error applying provider diff to global snapshot",
					zap.String("provider_id", provider.GetID()), zap.String("provider_type", ptypeName), zap.Error(err))
				return common.NewError("apply_providers_diff", fmt.Sprintf("error applying provider %v:%v diff to global snapshot", ptypeName, provider.GetID()))
			}
			isNew = false
		}
	}

	return nil
}

func (s *Snapshot) ApplyDiff(diff *Snapshot) {
	s.TotalMint += diff.TotalMint
	s.TotalChallengePools += diff.TotalChallengePools
	s.ActiveAllocatedDelta += diff.ActiveAllocatedDelta
	s.ZCNSupply += diff.ZCNSupply
	s.ClientLocks += diff.ClientLocks
	s.TotalReadPoolLocked += diff.TotalReadPoolLocked
	s.TotalStaked += diff.TotalStaked
	s.TotalRewards += diff.TotalRewards
	s.MinerTotalRewards += diff.MinerTotalRewards
	s.SharderTotalRewards += diff.SharderTotalRewards
	s.BlobberTotalRewards += diff.BlobberTotalRewards
	s.StorageTokenStake += diff.StorageTokenStake
	s.SuccessfulChallenges += diff.SuccessfulChallenges
	s.TotalChallenges += diff.TotalChallenges
	s.AllocatedStorage += diff.AllocatedStorage
	s.TotalAllocations += diff.TotalAllocations
	s.MaxCapacityStorage += diff.MaxCapacityStorage
	s.StakedStorage += diff.StakedStorage
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

// Facade for provider-specific diff appliers.
func (s *Snapshot) ApplySingleProviderDiff(ptype spenum.Provider) func(provider IProvider, snapshot IProviderSnapshot, isNew bool) error {
	switch ptype {
	case spenum.Blobber:
		return s.ApplyDiffBlobber
	case spenum.Miner:
		return s.ApplyDiffMiner
	case spenum.Sharder:
		return s.ApplyDiffSharder
	case spenum.Validator:
		return s.ApplyDiffValidator
	case spenum.Authorizer:
		return s.ApplyDiffAuthorizer
	default:
		return nil
	}
}

func (s *Snapshot) ApplyDiffBlobber(provider IProvider, snapshot IProviderSnapshot, isNew bool) error {
	current, ok := provider.(*Blobber)
	if !ok {
		return common.NewError("apply_provider_diff", "invalid blobber data")
	}
	previous, ok := snapshot.(*BlobberSnapshot)
	if !ok {
		return common.NewError("invalid_blobber_aggregate", "invalid blobber snapshot data")
	}
	if current.IsOffline() && !previous.IsOffline() {
		return s.ApplyDiffOfflineBlobber(snapshot)
	}
	if isNew {
		s.BlobberCount++
	}
	s.SuccessfulChallenges += int64(current.ChallengesPassed - previous.ChallengesPassed)
	s.TotalChallenges += int64(current.ChallengesCompleted - previous.ChallengesCompleted)
	s.TotalStaked += int64(current.TotalStake - previous.TotalStake)
	s.StorageTokenStake += int64(current.TotalStake - previous.TotalStake)
	s.AllocatedStorage += current.Allocated - previous.Allocated
	s.MaxCapacityStorage += current.Capacity - previous.Capacity
	s.UsedStorage += current.SavedData - previous.SavedData
	s.TotalRewards += int64(current.Rewards.TotalRewards - previous.TotalRewards)
	s.BlobberTotalRewards += int64(current.Rewards.TotalRewards - previous.TotalRewards)

	// Change in staked storage (staked_storage = total_stake / write_price)
	previousSS := previous.Capacity
	if previous.WritePrice > 0 {
		previousSS = int64((float64(previous.TotalStake) / float64(previous.WritePrice)) * GB)
	}
	newSS := current.Capacity
	if current.WritePrice > 0 {
		newSS = int64((float64(current.TotalStake) / float64(current.WritePrice)) * GB)
	}
	s.StakedStorage += (newSS - previousSS)
	return nil
}

func (s *Snapshot) ApplyDiffMiner(provider IProvider, snapshot IProviderSnapshot, isNew bool) error {
	current, ok := provider.(*Miner)
	if !ok {
		return common.NewError("apply_provider_diff", "invalid miner data")
	}
	previous, ok := snapshot.(*MinerSnapshot)
	if !ok {
		return common.NewError("apply_provider_diff", "invalid miner snapshot data")
	}
	if current.IsOffline() && !previous.IsOffline() {
		return s.ApplyDiffOfflineMiner(snapshot)
	}
	if isNew {
		s.MinerCount++
	}
	s.TotalRewards += int64(current.Rewards.TotalRewards - previous.TotalRewards)
	s.MinerTotalRewards += int64(current.Rewards.TotalRewards - previous.TotalRewards)
	s.TotalStaked += int64(current.TotalStake - previous.TotalStake)
	return nil
}

func (s *Snapshot) ApplyDiffSharder(provider IProvider, snapshot IProviderSnapshot, isNew bool) error {
	current, ok := provider.(*Sharder)
	if !ok {
		return common.NewError("apply_provider_diff", "invalid sharder data")
	}
	previous, ok := snapshot.(*SharderSnapshot)
	if !ok {
		return common.NewError("apply_provider_diff", "invalid sharder snapshot data")
	}
	if current.IsOffline() && !previous.IsOffline() {
		return s.ApplyDiffOfflineSharder(snapshot)
	}
	if isNew {
		s.SharderCount++
	}
	s.TotalRewards += int64(current.Rewards.TotalRewards - previous.TotalRewards)
	s.SharderTotalRewards += int64(current.Rewards.TotalRewards - previous.TotalRewards)
	s.TotalStaked += int64(current.TotalStake - previous.TotalStake)
	return nil
}

func (s *Snapshot) ApplyDiffValidator(provider IProvider, snapshot IProviderSnapshot, isNew bool) error {
	current, ok := provider.(*Validator)
	if !ok {
		return common.NewError("apply_provider_diff", "invalid validator data")
	}
	previous, ok := snapshot.(*ValidatorSnapshot)
	if !ok {
		return common.NewError("apply_provider_diff", "invalid validator snapshot data")
	}
	if current.IsOffline() && !previous.IsOffline() {
		return s.ApplyDiffOfflineValidator(snapshot)
	}
	if isNew {
		s.ValidatorCount++
	}
	s.TotalRewards += int64(current.Rewards.TotalRewards - previous.TotalRewards)
	s.TotalStaked += int64(current.TotalStake - previous.TotalStake)
	return nil
}

func (s *Snapshot) ApplyDiffAuthorizer(provider IProvider, snapshot IProviderSnapshot, isNew bool) error {
	current, ok := provider.(*Authorizer)
	if !ok {
		return common.NewError("apply_provider_diff", "invalid authorizer data")
	}
	previous, ok := snapshot.(*AuthorizerSnapshot)
	if !ok {
		return common.NewError("apply_provider_diff", "invalid authorizer snapshot data")
	}
	if current.IsOffline() && !previous.IsOffline() {
		return s.ApplyDiffOfflineAuthorizer(snapshot)
	}
	if isNew {
		s.AuthorizerCount++
	}
	s.TotalRewards += int64(current.Rewards.TotalRewards - previous.TotalRewards)
	s.TotalStaked += int64(current.TotalStake - previous.TotalStake)
	s.TotalMint += int64(current.TotalMint - previous.TotalMint)
	return nil
}

func (s *Snapshot) ApplyDiffOfflineBlobber(snapshot IProviderSnapshot) error {
	previous, ok := snapshot.(*BlobberSnapshot)
	if !ok {
		return common.NewError("invalid_blobber_aggregate", "invalid blobber snapshot data")
	}
	s.SuccessfulChallenges += int64(-previous.ChallengesPassed)
	s.TotalChallenges += int64(-previous.ChallengesCompleted)
	s.AllocatedStorage += -previous.Allocated
	s.MaxCapacityStorage += -previous.Capacity
	s.UsedStorage += -previous.SavedData
	s.TotalRewards += int64(-previous.TotalRewards)
	s.TotalStaked += int64(-previous.TotalStake)
	s.StorageTokenStake += int64(-previous.TotalStake)
	s.BlobberTotalRewards += int64(-previous.TotalRewards)
	s.BlobberCount -= 1

	if previous.WritePrice > 0 {
		ss := int64((float64(previous.TotalStake) / float64(previous.WritePrice)) * GB)
		s.StakedStorage += -ss
	} else {
		s.StakedStorage += -previous.Capacity
	}

	return nil
}

func (s *Snapshot) ApplyDiffOfflineMiner(snapshot IProviderSnapshot) error {
	previous, ok := snapshot.(*MinerSnapshot)
	if !ok {
		return common.NewError("invalid_miner_aggregate", "invalid miner snapshot data")
	}
	s.TotalRewards += int64(-previous.TotalRewards)
	s.TotalStaked += int64(-previous.TotalStake)
	s.MinerTotalRewards += int64(-previous.TotalRewards)
	s.MinerCount -= 1
	return nil
}

func (s *Snapshot) ApplyDiffOfflineSharder(snapshot IProviderSnapshot) error {
	previous, ok := snapshot.(*SharderSnapshot)
	if !ok {
		return common.NewError("invalid_sharder_aggregate", "invalid sharder snapshot data")
	}
	s.TotalRewards += int64(-previous.TotalRewards)
	s.TotalStaked += int64(-previous.TotalStake)
	s.SharderTotalRewards += int64(-previous.TotalRewards)
	s.SharderCount -= 1
	return nil
}

func (s *Snapshot) ApplyDiffOfflineValidator(snapshot IProviderSnapshot) error {
	previous, ok := snapshot.(*ValidatorSnapshot)
	if !ok {
		return common.NewError("invalid_validator_aggregate", "invalid validator snapshot data")
	}
	s.TotalRewards += int64(-previous.TotalRewards)
	s.TotalStaked += int64(-previous.TotalStake)
	s.ValidatorCount -= 1
	return nil
}

func (s *Snapshot) ApplyDiffOfflineAuthorizer(snapshot IProviderSnapshot) error {
	previous, ok := snapshot.(*AuthorizerSnapshot)
	if !ok {
		return common.NewError("invalid_authorizer_aggregate", "invalid authorizer snapshot data")
	}
	s.TotalRewards += int64(-previous.TotalRewards)
	s.TotalStaked += int64(-previous.TotalStake)
	s.TotalMint += int64(-previous.TotalMint)
	s.AuthorizerCount -= 1
	return nil
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

// UpdateSnapshot updates the global snapshot based on the events
//
// Parameters:
// gs: global globale snapshot
// e: events to apply to the snapshot
func (edb *EventDb) UpdateSnapshotFromEvents(gs *Snapshot, e []Event) error {
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
				return common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			gs.TotalChallengePools += cp.Amount
		case TagFromChallengePool:
			cp, ok := fromEvent[ChallengePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			gs.TotalChallengePools -= cp.Amount
		case TagLockWritePool:
			ds, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, d := range *ds {
				gs.ClientLocks += d.Amount
				if d.IsMint {
					gs.TotalMint += d.Amount
					gs.ZCNSupply += d.Amount
				}
			}
		case TagAddAllocation:
			gs.TotalAllocations += 1
		case TagUnlockWritePool:
			ds, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, d := range *ds {
				gs.ClientLocks -= d.Amount
			}
		case TagLockReadPool:
			ds, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, d := range *ds {
				gs.ClientLocks += d.Amount
				gs.TotalReadPoolLocked += d.Amount
				if d.IsMint {
					gs.TotalMint += d.Amount
					gs.ZCNSupply += d.Amount
				}
			}
		case TagUnlockReadPool:
			ds, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
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
				return common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, spu := range *spus {
				if spu.RewardType == spenum.BlockRewardMiner ||
					spu.RewardType == spenum.BlockRewardSharder ||
					spu.RewardType == spenum.BlockRewardBlobber {

					gs.TotalMint += int64(spu.TotalReward())
					gs.ZCNSupply += int64(spu.TotalReward())
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
				return common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			gs.TransactionsCount += int64(len(*txns))
			totalFee := 0
			for _, txn := range *txns {
				totalFee += int(txn.Fee)
			}
			gs.TotalTxnFee += int64(totalFee)
		}
	}
	return nil
}

// UpdateSnapshotFromProviders updates the snapshot from the given providers.
// It applies the diff between the current snapshot and the providers.
// It returns an error if the providers are not valid.
//
// Parameters:
// - gs: the current snapshot
// - providers: the providers to apply
func (edb *EventDb) UpdateSnapshotFromProviders(gs *Snapshot, providers ProvidersMap) error {

	blobbers := make([]*Blobber, 0, len(providers[spenum.Blobber]))
	for _, p := range providers[spenum.Blobber] {
		b, ok := p.(*Blobber)
		if !ok {
			return common.NewError("update_snapshot", fmt.Sprintf("invalid blobber provider: %v", p))
		}
		blobbers = append(blobbers, b)
	}

	err := ApplyProvidersDiff[*Blobber, *BlobberSnapshot](edb, gs, blobbers)
	if err != nil {
		return common.NewError("update_snapshot", fmt.Sprintf("error applying blobber snapshot: %v", err))
	}

	miners := make([]*Miner, 0, len(providers[spenum.Miner]))
	for _, p := range providers[spenum.Miner] {
		b, ok := p.(*Miner)
		if !ok {
			return common.NewError("update_snapshot", fmt.Sprintf("invalid miner provider: %v", p))
		}
		miners = append(miners, b)
	}

	err = ApplyProvidersDiff[*Miner, *MinerSnapshot](edb, gs, miners)
	if err != nil {
		return common.NewError("update_snapshot", fmt.Sprintf("error applying blobber snapshot: %v", err))
	}

	sharders := make([]*Sharder, 0, len(providers[spenum.Sharder]))
	for _, p := range providers[spenum.Sharder] {
		b, ok := p.(*Sharder)
		if !ok {
			return common.NewError("update_snapshot", fmt.Sprintf("invalid sharder provider: %v", p))
		}
		sharders = append(sharders, b)
	}

	err = ApplyProvidersDiff[*Sharder, *SharderSnapshot](edb, gs, sharders)
	if err != nil {
		return common.NewError("update_snapshot", fmt.Sprintf("error applying sharder snapshot: %v", err))
	}

	authorizers := make([]*Authorizer, 0, len(providers[spenum.Authorizer]))
	for _, p := range providers[spenum.Authorizer] {
		b, ok := p.(*Authorizer)
		if !ok {
			return common.NewError("update_snapshot", fmt.Sprintf("invalid authorizer provider: %v", p))
		}
		authorizers = append(authorizers, b)
	}

	err = ApplyProvidersDiff[*Authorizer, *AuthorizerSnapshot](edb, gs, authorizers)
	if err != nil {
		return common.NewError("update_snapshot", fmt.Sprintf("error applying authorizer snapshot: %v", err))
	}

	validators := make([]*Validator, 0, len(providers[spenum.Validator]))
	for _, p := range providers[spenum.Validator] {
		b, ok := p.(*Validator)
		if !ok {
			return common.NewError("update_snapshot", fmt.Sprintf("invalid validator provider: %v", p))
		}
		validators = append(validators, b)
	}

	err = ApplyProvidersDiff[*Validator, *ValidatorSnapshot](edb, gs, validators)
	if err != nil {
		return common.NewError("update_snapshot", fmt.Sprintf("error applying validator snapshot: %v", err))
	}

	return nil
}

type ProcessingEntity struct {
	Entity    interface{}
	Processed bool
}

// MakeProcessingMap wraps map entries into a struct with "Processed" boolean flag
func MakeProcessingMap[T any](mapIn map[string]T) map[string]ProcessingEntity {
	mpOut := make(map[string]ProcessingEntity)
	for k, v := range mapIn {
		mpOut[k] = ProcessingEntity{Entity: v, Processed: false}
	}
	return mpOut
}
