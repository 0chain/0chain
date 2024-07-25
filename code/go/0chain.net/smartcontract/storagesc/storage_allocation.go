package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/dbs/event"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/0chain/common/core/currency"
	"go.uber.org/zap"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/util/entitywrapper"
	"github.com/0chain/common/core/logging"
)

//msgp:ignore StorageAllocation AllocationChallenges storageAllocationBase
//go:generate msgp -io=false -tests=false -unexported -v

func init() {
	entitywrapper.RegisterWrapper(&StorageAllocation{},
		map[string]entitywrapper.EntityI{
			entitywrapper.DefaultOriginVersion: &storageAllocationV1{},
			"v2":                               &storageAllocationV2{},
		})
}

type StorageAllocation struct {
	entitywrapper.Wrapper
}

func (sa *StorageAllocation) TypeName() string {
	return "storage_allocation"
}

func (sa *StorageAllocation) UnmarshalMsg(data []byte) ([]byte, error) {
	d := &StorageAllocationDecode{}
	o, err := d.UnmarshalMsgType(data, sa.TypeName())
	if err != nil {
		return nil, err
	}

	*sa = StorageAllocation(*d)

	sa.mustUpdateBase(func(base *storageAllocationBase) error {
		base.BlobberAllocsMap = make(map[string]*BlobberAllocation)
		for _, blobberAllocation := range base.BlobberAllocs {
			base.BlobberAllocsMap[blobberAllocation.BlobberID] = blobberAllocation
		}
		return nil
	})

	return o, nil
}

func (sa *StorageAllocation) UnmarshalJSON(data []byte) error {
	return sa.UnmarshalJSONType(data, sa.TypeName())
}

func (sa *StorageAllocation) Msgsize() (s int) {
	return sa.Entity().Msgsize()
}

func (sa *StorageAllocation) mustBase() *storageAllocationBase {
	a, ok := sa.Base().(*storageAllocationBase)
	if !ok {
		logging.Logger.Panic("invalid storage allocation base type")
	}
	return a
}

func (sa *StorageAllocation) mustUpdateBase(f func(*storageAllocationBase) error) error {
	return sa.UpdateBase(func(eb entitywrapper.EntityBaseI) error {
		b, ok := eb.(*storageAllocationBase)
		if !ok {
			logging.Logger.Panic("invalid storage allocation base type")
		}

		err := f(b)
		if err != nil {
			return err
		}
		return nil
	})
}

// implement provider.AbstractProvider interface
func (sa *StorageAllocation) Id() string {
	return sa.mustBase().ID
}

func (sa *StorageAllocation) Encode() []byte {
	buff, _ := json.Marshal(sa)
	return buff
}

func (sa *StorageAllocation) Decode(input []byte) error {
	err := json.Unmarshal(input, sa)
	if err != nil {
		return err
	}

	sa.mustUpdateBase(func(base *storageAllocationBase) error {
		base.BlobberAllocsMap = make(map[string]*BlobberAllocation)
		for _, blobberAllocation := range base.BlobberAllocs {
			base.BlobberAllocsMap[blobberAllocation.BlobberID] = blobberAllocation
		}
		return nil
	})
	return nil
}

// StorageAllocation request and entity.
// swagger:model StorageAllocation
type storageAllocationV1 struct {
	// ID is unique allocation ID that is equal to hash of transaction with
	// which the allocation has created.
	ID string `json:"id"`
	// Tx keeps hash with which the allocation has created or updated. todo do we need this field?
	Tx string `json:"tx"`

	DataShards        int                     `json:"data_shards"`
	ParityShards      int                     `json:"parity_shards"`
	Size              int64                   `json:"size"`
	Expiration        common.Timestamp        `json:"expiration_date"`
	Owner             string                  `json:"owner_id"`
	OwnerPublicKey    string                  `json:"owner_public_key"`
	Stats             *StorageAllocationStats `json:"stats"`
	DiverseBlobbers   bool                    `json:"diverse_blobbers"`
	PreferredBlobbers []string                `json:"preferred_blobbers"`
	// Blobbers not to be used anywhere except /allocation and /allocations table
	// if Blobbers are getting used in any smart-contract, we should avoid.
	BlobberAllocs    []*BlobberAllocation          `json:"blobber_details"`
	BlobberAllocsMap map[string]*BlobberAllocation `json:"-" msg:"-"`

	// Flag to determine if anyone can extend this allocation
	ThirdPartyExtendable bool `json:"third_party_extendable"`

	// FileOptions to define file restrictions on an allocation for third-parties
	// default 00000000 for all crud operations suggesting only owner has the below listed abilities.
	// enabling option/s allows any third party to perform certain ops
	// 00000001 - 1  - upload
	// 00000010 - 2  - delete
	// 00000100 - 4  - update
	// 00001000 - 8  - move
	// 00010000 - 16 - copy
	// 00100000 - 32 - rename
	FileOptions uint16 `json:"file_options"`

	WritePool currency.Coin `json:"write_pool"`

	// Requested ranges.
	ReadPriceRange  PriceRange `json:"read_price_range"`
	WritePriceRange PriceRange `json:"write_price_range"`

	// StartTime is time when the allocation has been created. We will
	// use it to check blobber's MaxOfferTime extending the allocation.
	StartTime common.Timestamp `json:"start_time"`
	// Finalized is true where allocation has been finalized.
	Finalized bool `json:"finalized,omitempty"`
	// Canceled set to true where allocation finalized by cancel_allocation
	// transaction.
	Canceled bool `json:"canceled,omitempty"`

	// MovedToChallenge is number of tokens moved to challenge pool.
	MovedToChallenge currency.Coin `json:"moved_to_challenge,omitempty"`
	// MovedBack is number of tokens moved from challenge pool to
	// related write pool (the Back) if a data has deleted.
	MovedBack currency.Coin `json:"moved_back,omitempty"`
	// MovedToValidators is total number of tokens moved to validators
	// of the allocation.
	MovedToValidators currency.Coin `json:"moved_to_validators,omitempty"`

	// TimeUnit configured in Storage SC when the allocation created. It can't
	// be changed for this allocation anymore. Even using expire allocation.
	TimeUnit time.Duration `json:"time_unit"`
}

func (sa1 *storageAllocationV1) GetVersion() string {
	return entitywrapper.DefaultOriginVersion
}

func (sa1 *storageAllocationV1) InitVersion() {
	// do nothing cause it's original version of storage allocation
}

func (sa1 *storageAllocationV1) GetBase() entitywrapper.EntityBaseI {
	sa := storageAllocationBase(*sa1)
	return &sa
}

func (sa1 *storageAllocationV1) MigrateFrom(e entitywrapper.EntityI) error {
	// nothing to migrate as this is original version of the storage allocation
	return nil
}

// use storageAllocationV1 as the base
type storageAllocationBase storageAllocationV1

func (sab *storageAllocationBase) CommitChangesTo(e entitywrapper.EntityI) {
	switch v := e.(type) {
	case *storageAllocationV1:
		*v = storageAllocationV1(*sab)
	case *storageAllocationV2:
		v.ApplyBaseChanges(*sab)
	}
}

// StorageAllocation request and entity.
// swagger:model StorageAllocation
type storageAllocationV2 struct {
	// ID is unique allocation ID that is equal to hash of transaction with
	// which the allocation has created.
	ID string `json:"id"`
	// Tx keeps hash with which the allocation has created or updated. todo do we need this field?
	Tx string `json:"tx"`

	DataShards        int                     `json:"data_shards"`
	ParityShards      int                     `json:"parity_shards"`
	Size              int64                   `json:"size"`
	Expiration        common.Timestamp        `json:"expiration_date"`
	Owner             string                  `json:"owner_id"`
	OwnerPublicKey    string                  `json:"owner_public_key"`
	Stats             *StorageAllocationStats `json:"stats"`
	DiverseBlobbers   bool                    `json:"diverse_blobbers"`
	PreferredBlobbers []string                `json:"preferred_blobbers"`
	// Blobbers not to be used anywhere except /allocation and /allocations table
	// if Blobbers are getting used in any smart-contract, we should avoid.
	BlobberAllocs    []*BlobberAllocation          `json:"blobber_details"`
	BlobberAllocsMap map[string]*BlobberAllocation `json:"-" msg:"-"`

	// Flag to determine if anyone can extend this allocation
	ThirdPartyExtendable bool `json:"third_party_extendable"`

	// FileOptions to define file restrictions on an allocation for third-parties
	// default 00000000 for all crud operations suggesting only owner has the below listed abilities.
	// enabling option/s allows any third party to perform certain ops
	// 00000001 - 1  - upload
	// 00000010 - 2  - delete
	// 00000100 - 4  - update
	// 00001000 - 8  - move
	// 00010000 - 16 - copy
	// 00100000 - 32 - rename
	FileOptions uint16 `json:"file_options"`

	WritePool currency.Coin `json:"write_pool"`

	// Requested ranges.
	ReadPriceRange  PriceRange `json:"read_price_range"`
	WritePriceRange PriceRange `json:"write_price_range"`

	// StartTime is time when the allocation has been created. We will
	// use it to check blobber's MaxOfferTime extending the allocation.
	StartTime common.Timestamp `json:"start_time"`
	// Finalized is true where allocation has been finalized.
	Finalized bool `json:"finalized,omitempty"`
	// Canceled set to true where allocation finalized by cancel_allocation
	// transaction.
	Canceled bool `json:"canceled,omitempty"`

	// MovedToChallenge is number of tokens moved to challenge pool.
	MovedToChallenge currency.Coin `json:"moved_to_challenge,omitempty"`
	// MovedBack is number of tokens moved from challenge pool to
	// related write pool (the Back) if a data has deleted.
	MovedBack currency.Coin `json:"moved_back,omitempty"`
	// MovedToValidators is total number of tokens moved to validators
	// of the allocation.
	MovedToValidators currency.Coin `json:"moved_to_validators,omitempty"`

	// TimeUnit configured in Storage SC when the allocation created. It can't
	// be changed for this allocation anymore. Even using expire allocation.
	TimeUnit time.Duration `json:"time_unit"`

	Version      string `json:"version" msg:"version"`
	IsEnterprise *bool  `json:"is_enterprise"`
}

const storageAllocationV2Version = "v2"

func (sa2 *storageAllocationV2) GetVersion() string {
	return storageAllocationV2Version
}

func (sa2 *storageAllocationV2) InitVersion() {
	sa2.Version = storageAllocationV2Version
}

func (sa2 *storageAllocationV2) GetBase() entitywrapper.EntityBaseI {
	return &storageAllocationBase{
		ID:                   sa2.ID,
		Tx:                   sa2.Tx,
		DataShards:           sa2.DataShards,
		ParityShards:         sa2.ParityShards,
		Size:                 sa2.Size,
		Expiration:           sa2.Expiration,
		Owner:                sa2.Owner,
		OwnerPublicKey:       sa2.OwnerPublicKey,
		Stats:                sa2.Stats,
		DiverseBlobbers:      sa2.DiverseBlobbers,
		PreferredBlobbers:    sa2.PreferredBlobbers,
		BlobberAllocs:        sa2.BlobberAllocs,
		BlobberAllocsMap:     sa2.BlobberAllocsMap,
		ThirdPartyExtendable: sa2.ThirdPartyExtendable,
		FileOptions:          sa2.FileOptions,
		WritePool:            sa2.WritePool,
		ReadPriceRange:       sa2.ReadPriceRange,
		WritePriceRange:      sa2.WritePriceRange,
		StartTime:            sa2.StartTime,
		Finalized:            sa2.Finalized,
		Canceled:             sa2.Canceled,
		MovedToChallenge:     sa2.MovedToChallenge,
		MovedBack:            sa2.MovedBack,
		MovedToValidators:    sa2.MovedToValidators,
		TimeUnit:             sa2.TimeUnit,
	}
}

func (sa2 *storageAllocationV2) MigrateFrom(e entitywrapper.EntityI) error {
	v1, ok := e.(*storageAllocationV1)
	if !ok {
		return errors.New("struct migrate fail, wrong storageAllocation type")
	}
	sa2.ApplyBaseChanges(storageAllocationBase(*v1))
	sa2.Version = "v2"
	return nil
}

func (sa2 *storageAllocationV2) ApplyBaseChanges(sab storageAllocationBase) {
	sa2.ID = sab.ID
	sa2.Tx = sab.Tx
	sa2.DataShards = sab.DataShards
	sa2.ParityShards = sab.ParityShards
	sa2.Size = sab.Size
	sa2.Expiration = sab.Expiration
	sa2.Owner = sab.Owner
	sa2.OwnerPublicKey = sab.OwnerPublicKey
	sa2.Stats = sab.Stats
	sa2.DiverseBlobbers = sab.DiverseBlobbers
	sa2.PreferredBlobbers = sab.PreferredBlobbers
	sa2.BlobberAllocs = sab.BlobberAllocs
	sa2.BlobberAllocsMap = sab.BlobberAllocsMap
	sa2.ThirdPartyExtendable = sab.ThirdPartyExtendable
	sa2.FileOptions = sab.FileOptions
	sa2.WritePool = sab.WritePool
	sa2.ReadPriceRange = sab.ReadPriceRange
	sa2.WritePriceRange = sab.WritePriceRange
	sa2.StartTime = sab.StartTime
	sa2.Finalized = sab.Finalized
	sa2.Canceled = sab.Canceled
	sa2.MovedToChallenge = sab.MovedToChallenge
	sa2.MovedBack = sab.MovedBack
	sa2.MovedToValidators = sab.MovedToValidators
	sa2.TimeUnit = sab.TimeUnit
}

func (sab *storageAllocationBase) checkFunding() error {
	allocCost, err := sab.cost()
	if err != nil {
		return fmt.Errorf("failed to get allocation cost: %v", err)
	}

	if sab.WritePool < allocCost {
		return fmt.Errorf("not enough tokens to honor the allocation cost %v < %v",
			sab.WritePool, allocCost)
	}

	return nil
}

func (sab *storageAllocationBase) addToWritePool(
	txn *transaction.Transaction,
	balances cstate.StateContextI,
	transfer *Transfer,
) error {
	value, err := transfer.transfer(balances)
	if err != nil {
		return err
	}
	if value == 0 {
		return nil
	}
	if writePool, err := currency.AddCoin(sab.WritePool, value); err != nil {
		return err
	} else {
		sab.WritePool = writePool
	}

	i, err := txn.Value.Int64()
	if err != nil {
		return err
	}
	balances.EmitEvent(event.TypeStats, event.TagLockWritePool, sab.ID, event.WritePoolLock{
		Client:       txn.ClientID,
		AllocationId: sab.ID,
		Amount:       i,
		IsMint:       transfer.isMint,
	})
	return nil
}

func (sab *storageAllocationBase) cost() (currency.Coin, error) {
	var cost currency.Coin
	for _, ba := range sab.BlobberAllocs {
		c, err := currency.MultFloat64(ba.Terms.WritePrice, sizeInGB(ba.Size))
		if err != nil {
			return 0, err
		}
		cost, err = currency.AddCoin(cost, c)
		if err != nil {
			return 0, err
		}
	}
	return cost, nil
}

func (sab *storageAllocationBase) costForRDTU(now common.Timestamp) (currency.Coin, error) {
	rdtu, err := sab.restDurationInTimeUnits(now, sab.TimeUnit)
	if err != nil {
		return 0, fmt.Errorf("failed to get rest duration in time units: %v", err)

	}

	var cost currency.Coin
	for _, ba := range sab.BlobberAllocs {
		c, err := currency.MultFloat64(ba.Terms.WritePrice, sizeInGB(ba.Size))
		if err != nil {
			return 0, err
		}

		c, err = currency.MultFloat64(c, rdtu)
		if err != nil {
			return 0, err
		}

		cost, err = currency.AddCoin(cost, c)
		if err != nil {
			return 0, err
		}
	}
	return cost, nil
}

func (sab *storageAllocationBase) payCostForRdtuForEnterpriseAllocation(t *transaction.Transaction, balances cstate.StateContextI) (currency.Coin, error) {
	rdtu, err := sab.restDurationInTimeUnits(t.CreationDate, sab.TimeUnit)
	if err != nil {
		return 0, fmt.Errorf("failed to get rest duration in time units: %v", err)

	}

	var cost currency.Coin
	for _, ba := range sab.BlobberAllocs {
		c, err := currency.MultFloat64(ba.Terms.WritePrice, sizeInGB(ba.Size))
		if err != nil {
			return 0, err
		}

		c, err = currency.MultFloat64(c, rdtu)
		if err != nil {
			return 0, err
		}

		transfer := NewTokenTransfer(c, t.ClientID, ba.BlobberID, false)
		_, err = transfer.transfer(balances)
		if err != nil {
			return 0, err
		}

		sab.WritePool, err = currency.MinusCoin(sab.WritePool, c)
		if err != nil {
			return 0, err
		}

		cost, err = currency.AddCoin(cost, c)
		if err != nil {
			return 0, err
		}

	}
	return cost, nil
}

func (sab *storageAllocationBase) payCostForRdtuForReplaceEnterpriseBlobber(t *transaction.Transaction, blobberID string, balances cstate.StateContextI) (currency.Coin, error) {
	rdtu, err := sab.restDurationInTimeUnits(t.CreationDate, sab.TimeUnit)
	if err != nil {
		return 0, fmt.Errorf("failed to get rest duration in time units: %v", err)

	}

	var cost currency.Coin
	for _, ba := range sab.BlobberAllocs {
		if ba.BlobberID == blobberID {
			c, err := currency.MultFloat64(ba.Terms.WritePrice, sizeInGB(ba.Size))
			if err != nil {
				return 0, err
			}

			c, err = currency.MultFloat64(c, rdtu)
			if err != nil {
				return 0, err
			}

			transfer := NewTokenTransfer(c, t.ClientID, ba.BlobberID, false)
			_, err = transfer.transfer(balances)
			if err != nil {
				return 0, err
			}

			cost, err = currency.AddCoin(cost, c)
			if err != nil {
				return 0, err
			}
		}
	}
	return cost, nil
}

// The restDurationInTimeUnits return rest duration of the allocation in time
// units as a float64 value.
func (sab *storageAllocationBase) restDurationInTimeUnits(now common.Timestamp, timeUnit time.Duration) (float64, error) {
	if sab.Expiration < now {
		logging.Logger.Error("rest duration time overflow, timestamp is beyond alloc expiration",
			zap.Int64("now", int64(now)),
			zap.Int64("alloc expiration", int64(sab.Expiration)))
		return 0, errors.New("rest duration time overflow, timestamp is beyond alloc expiration")
	}
	logging.Logger.Info("rest_duration", zap.Int64("expiration", int64(sab.Expiration)), zap.Int64("now", int64(now)), zap.Float64("timeUnit", float64(timeUnit)), zap.Int64("rest", int64(sab.Expiration-now)))
	return sab.durationInTimeUnits(sab.Expiration-now, timeUnit)
}

// The durationInTimeUnits returns given duration (represented as
// common.Timestamp) as duration in time units (float point value) for
// this allocation (time units for the moment of the allocation creation).
func (sab *storageAllocationBase) durationInTimeUnits(dur common.Timestamp, timeUnit time.Duration) (float64, error) {
	if dur < 0 {
		return 0, errors.New("negative duration")
	}
	return float64(dur.Duration()) / float64(timeUnit), nil
}

func (sa *StorageAllocation) GetKey(globalKey string) datastore.Key {
	return globalKey + sa.mustBase().ID
}

func GetAllocKey(globalKey, allocID string) datastore.Key {
	return globalKey + allocID
}

func (sab *storageAllocationBase) buildEventBlobberTerms() []event.AllocationBlobberTerm {
	bTerms := make([]event.AllocationBlobberTerm, 0, len(sab.BlobberAllocs))
	for i, b := range sab.BlobberAllocs {
		bTerms = append(bTerms, event.AllocationBlobberTerm{
			AllocationIdHash: sab.ID,
			BlobberID:        b.BlobberID,
			ReadPrice:        int64(b.Terms.ReadPrice),
			WritePrice:       int64(b.Terms.WritePrice),
			AllocBlobberIdx:  int64(i),
		})
	}

	return bTerms
}

func (sa *StorageAllocation) buildDbUpdates(balances cstate.StateContextI) event.Allocation {
	sab := sa.mustBase()
	eAlloc := event.Allocation{
		AllocationID:         sab.ID,
		TransactionID:        sab.Tx,
		DataShards:           sab.DataShards,
		ParityShards:         sab.ParityShards,
		Size:                 sab.Size,
		Expiration:           int64(sab.Expiration),
		Owner:                sab.Owner,
		OwnerPublicKey:       sab.OwnerPublicKey,
		ReadPriceMin:         sab.ReadPriceRange.Min,
		ReadPriceMax:         sab.ReadPriceRange.Max,
		WritePriceMin:        sab.WritePriceRange.Min,
		WritePriceMax:        sab.WritePriceRange.Max,
		StartTime:            int64(sab.StartTime),
		Finalized:            sab.Finalized,
		Cancelled:            sab.Canceled,
		UsedSize:             sab.Stats.UsedSize,
		MovedToChallenge:     sab.MovedToChallenge,
		MovedBack:            sab.MovedBack,
		MovedToValidators:    sab.MovedToValidators,
		TimeUnit:             int64(sab.TimeUnit),
		WritePool:            sab.WritePool,
		ThirdPartyExtendable: sab.ThirdPartyExtendable,
		FileOptions:          sab.FileOptions,
	}

	_ = cstate.WithActivation(balances, "electra", func() error {
		return nil
	}, func() error {
		if v2, ok := sa.Entity().(*storageAllocationV2); ok && v2.IsEnterprise != nil {
			eAlloc.IsEnterprise = *v2.IsEnterprise
		}
		return nil
	})

	if sab.Stats != nil {
		eAlloc.NumWrites = sab.Stats.NumWrites
		eAlloc.NumReads = sab.Stats.NumReads
		eAlloc.TotalChallenges = sab.Stats.TotalChallenges
		eAlloc.OpenChallenges = sab.Stats.OpenChallenges
		eAlloc.SuccessfulChallenges = sab.Stats.SuccessChallenges
		eAlloc.FailedChallenges = sab.Stats.FailedChallenges
		eAlloc.LatestClosedChallengeTxn = sab.Stats.LastestClosedChallengeTxn
	}

	return eAlloc
}

func (sab *storageAllocationBase) buildStakeUpdateEvent() event.Allocation {
	return event.Allocation{
		AllocationID:      sab.ID,
		WritePool:         sab.WritePool,
		MovedToChallenge:  sab.MovedToChallenge,
		MovedBack:         sab.MovedBack,
		MovedToValidators: sab.MovedToValidators,
	}
}

func (sa *StorageAllocation) saveUpdatedAllocation(
	blobbers []*StorageNode,
	balances cstate.StateContextI,
) (err error) {
	for _, b := range blobbers {
		if _, err = balances.InsertTrieNode(b.GetKey(), b); err != nil {
			return
		}
		emitUpdateBlobberAllocatedSavedHealth(b, balances)
	}
	// Save allocation
	_, err = balances.InsertTrieNode(sa.GetKey(ADDRESS), sa)
	if err != nil {
		return
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, sa.mustBase().ID, sa.buildDbUpdates(balances))
	return
}

func (sa *StorageAllocation) saveUpdatedStakes(balances cstate.StateContextI) (err error) {
	// Save allocation
	_, err = balances.InsertTrieNode(sa.GetKey(ADDRESS), sa)
	if err != nil {
		return
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationStakes, sa.mustBase().ID, sa.mustBase().buildStakeUpdateEvent())
	return
}

func (sab *storageAllocationBase) deepCopy(input *storageAllocationBase) {
	input.ID = sab.ID
	input.Tx = sab.Tx
	input.DataShards = sab.DataShards
	input.ParityShards = sab.ParityShards
	input.Size = sab.Size
	input.Expiration = sab.Expiration
	input.Owner = sab.Owner
	input.OwnerPublicKey = sab.OwnerPublicKey
	input.Stats = sab.Stats
	input.DiverseBlobbers = sab.DiverseBlobbers
	input.PreferredBlobbers = sab.PreferredBlobbers
	input.BlobberAllocs = sab.BlobberAllocs
	input.BlobberAllocsMap = sab.BlobberAllocsMap
	input.ThirdPartyExtendable = sab.ThirdPartyExtendable
	input.FileOptions = sab.FileOptions
	input.WritePool = sab.WritePool
	input.ReadPriceRange = sab.ReadPriceRange
	input.WritePriceRange = sab.WritePriceRange
	input.StartTime = sab.StartTime
	input.Finalized = sab.Finalized
	input.Canceled = sab.Canceled
	input.MovedToChallenge = sab.MovedToChallenge
	input.MovedBack = sab.MovedBack
	input.MovedToValidators = sab.MovedToValidators
	input.TimeUnit = sab.TimeUnit
}
