package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

// getAllocation by ID
func (sc *StorageSmartContract) getAllocation(allocID string,
	balances chainstate.StateContextI) (alloc *StorageAllocation, err error) {

	alloc = new(StorageAllocation)
	alloc.ID = allocID
	err = balances.GetTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return nil, err
	}

	if len(alloc.BlobberAllocs) == 0 {
		alloc.BlobberAllocs = make([]*BlobberAllocation, len(alloc.Blobbers))
		alloc.BlobberAllocsMap = make(map[string]*BlobberAllocation, len(alloc.Blobbers))
		for i, b := range alloc.Blobbers {
			ba := newBlobberAllocation(b.BlobberID)
			alloc.BlobberAllocs[i] = ba
			alloc.BlobberAllocsMap[b.BlobberID] = ba
		}
	}

	return
}

func (sc *StorageSmartContract) addAllocation(alloc *StorageAllocation,
	balances chainstate.StateContextI) error {
	_, err := balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return common.NewErrorf("add_allocation_failed",
			"saving new allocation: %v", err)
	}

	err = alloc.emitAdd(balances)
	if err != nil {
		return common.NewErrorf("add_allocation_failed",
			"saving new allocation in db: %v", err)
	}

	// TODO: return simple message rather than whole allocation info
	//buff := alloc.Encode()
	//return string(buff), nil
	//return "add_allocation", nil
	return nil
}

type newAllocationRequest struct {
	Name                 string           `json:"name"`
	DataShards           int              `json:"data_shards"`
	ParityShards         int              `json:"parity_shards"`
	Size                 int64            `json:"size"`
	Expiration           common.Timestamp `json:"expiration_date"`
	Owner                string           `json:"owner_id"`
	OwnerPublicKey       string           `json:"owner_public_key"`
	Blobbers             []string         `json:"blobbers"`
	ReadPriceRange       PriceRange       `json:"read_price_range"`
	WritePriceRange      PriceRange       `json:"write_price_range"`
	ThirdPartyExtendable bool             `json:"third_party_extendable"`
	FileOptionsChanged   bool             `json:"file_options_changed"`
	FileOptions          uint16           `json:"file_options"`
}

// storageAllocation from the request
func (nar *newAllocationRequest) storageAllocation() (sa *StorageAllocation) {
	sa = new(StorageAllocation)
	sa.DataShards = nar.DataShards
	sa.ParityShards = nar.ParityShards
	sa.Size = nar.Size
	sa.Expiration = nar.Expiration
	sa.Owner = nar.Owner
	sa.OwnerPublicKey = nar.OwnerPublicKey
	sa.ReadPriceRange = nar.ReadPriceRange
	sa.WritePriceRange = nar.WritePriceRange
	sa.ThirdPartyExtendable = nar.ThirdPartyExtendable
	sa.FileOptions = nar.FileOptions

	return
}

func (nar *newAllocationRequest) validate(now time.Time, conf *Config) error {
	if nar.DataShards <= 0 {
		return errors.New("invalid number of data shards")
	}

	if len(nar.Blobbers) < (nar.DataShards + nar.ParityShards) {
		return errors.New("blobbers provided are not enough to honour the allocation")
	}

	if !nar.ReadPriceRange.isValid() {
		return errors.New("invalid read_price range")
	}

	if !nar.WritePriceRange.isValid() {
		return errors.New("invalid write_price range")
	}

	if nar.Size < conf.MinAllocSize {
		return errors.New("insufficient allocation size")
	}

	dur := common.ToTime(nar.Expiration).Sub(now)
	if dur < conf.TimeUnit {
		return errors.New("insufficient allocation duration")
	}
	return nil
}

func (nar *newAllocationRequest) decode(b []byte) error {
	return json.Unmarshal(b, nar)
}

func (nar *newAllocationRequest) encode() ([]byte, error) {
	return json.Marshal(nar)
}

// convert time.Duration to common.Timestamp truncating to seconds
func toSeconds(dur time.Duration) common.Timestamp {
	return common.Timestamp(dur / time.Second)
}

// size in gigabytes
func sizeInGB(size int64) float64 {
	return float64(size) / GB
}

// exclude blobbers with not enough token in stake pool to fit the size
func (sc *StorageSmartContract) filterBlobbersByFreeSpace(now common.Timestamp,
	size int64, balances chainstate.CommonStateContextI) (filter filterBlobberFunc) {

	return filterBlobberFunc(func(b *StorageNode) (kick bool, err error) {
		var sp *stakePool
		sp, err = sc.getStakePool(spenum.Blobber, b.ID, balances)
		switch err {
		case nil:
		case util.ErrValueNotPresent:
			return true, nil // kick off
		default:
			return false, err
		}

		if b.Terms.WritePrice == 0 {
			return false, nil // keep, ok or already filtered by bid
		}
		staked, err := sp.stake()
		if err != nil {
			logging.Logger.Error("filter blobber for stake, cannot total stake",
				zap.String("blobber id", b.ID))
			return true, nil
		}
		// clean capacity (without delegate pools want to 'unstake')
		free, err := unallocatedCapacity(b.Terms.WritePrice, staked, sp.TotalOffers)
		if err != nil {
			logging.Logger.Warn("could not get unallocated capacity when filtering blobbers by free space",
				zap.String("blobber id", b.ID),
				zap.Error(err))
			return true, nil // kick off
		}
		return free < size, nil // kick off if it hasn't enough free space
	})
}

// newAllocationRequest creates new allocation
func (sc *StorageSmartContract) newAllocationRequest(
	t *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
	timings map[string]time.Duration,
) (string, error) {
	var conf *Config
	var err error
	if conf, err = sc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("allocation_creation_failed",
			"can't get config: %v", err)
	}

	var request newAllocationRequest
	if err = request.decode(input); err != nil {
		logging.Logger.Error("new_allocation_request_failed: error decoding input",
			zap.String("txn", t.Hash),
			zap.Error(err))
		return "", common.NewErrorf("allocation_creation_failed",
			"malformed request: %v", err)
	}

	sa, err := sc.newAllocationRequestInternal(t, &request, conf, 0, balances, timings)
	if err != nil {
		return "", err
	}

	return string(sa.Encode()), nil
}

// newAllocationRequest creates new allocation
// func (sc *StorageSmartContract) newAllocationRequestInternal(
func (sc *StorageSmartContract) newAllocationRequestInternal(
	txn *transaction.Transaction,
	request *newAllocationRequest,
	conf *Config,
	mintNewTokens currency.Coin,
	balances chainstate.StateContextI,
	timings map[string]time.Duration,
) (sa *StorageAllocation, err error) {
	m := Timings{timings: timings, start: common.ToTime(txn.CreationDate)}
	if err := request.validate(common.ToTime(txn.CreationDate), conf); err != nil {
		return nil, common.NewErrorf("allocation_creation_failed", "invalid request: "+err.Error())
	}

	if request.Owner == "" {
		request.Owner = txn.ClientID
		request.OwnerPublicKey = txn.PublicKey
	}
	m.tick("decode and validate")

	var (
		blobbers []*StorageNode
		bil      BlobberOfferStakeList
	)

	cr := concurrentReader{}
	cr.add(func() error {
		var err error
		blobbers, err = getBlobbersByIDs(request.Blobbers, balances)
		if err != nil {
			return common.NewErrorf("allocation_creation_failed", "get blobbers failed: %v", err)
		}

		if len(blobbers) < (request.DataShards + request.ParityShards) {
			logging.Logger.Error("new_allocation_request_failed: blobbers fetched are less than requested blobbers",
				zap.String("txn", txn.Hash),
				zap.Int("fetched blobbers", len(blobbers)),
				zap.Int("data shards", request.DataShards),
				zap.Int("parity_shards", request.ParityShards))
			return common.NewErrorf("allocation_creation_failed",
				"Not enough provided blobbers found in mpt")
		}

		return nil
	})

	cr.add(func() error {
		var err error
		bil, err = getBlobbersInfoList(balances)
		return err
	})

	if mintNewTokens == 0 {
		// check lock token balance
		cr.add(func() error {
			return stakepool.CheckClientBalance(txn.ClientID, txn.Value, balances)
		})
	}

	if err := cr.do(); err != nil {
		return nil, err
	}

	if request.Owner == "" {
		request.Owner = txn.ClientID
		request.OwnerPublicKey = txn.PublicKey
	}

	logging.Logger.Debug("new_allocation_request", zap.String("t_hash", txn.Hash), zap.Strings("blobbers", request.Blobbers), zap.Any("amount", txn.Value))
	sa, blobberNodes, err := setupNewAllocation(*request, blobbers, bil, m, txn.CreationDate, conf, txn.Hash)
	if err != nil {
		return nil, err
	}

	m.tick("setup new alloc")

	for _, b := range blobberNodes {
		offer := currency.Coin(sizeInGB(sa.bSize()) * float64(b.Terms.WritePrice))
		if err := bil[b.Index].addOffer(offer); err != nil {
			logging.Logger.Error("new_allocation_request_failed: error adding offer to blobber",
				zap.String("txn", txn.Hash),
				zap.String("blobber", b.ID),
				zap.Error(err))
			return nil, fmt.Errorf("ading offer: %v", err)
		}

		emitUpdateBlobberAllocatedSavedHealth(b.ID, b.LastHealthCheck, bil[b.Index].Allocated, bil[b.Index].SavedData, balances)
	}

	if err := bil.Save(balances); err != nil {
		return nil, err
	}

	//if _, err := partitionsAllocBlobbersAdd(balances, sa.ID); err != nil {
	//	return nil, fmt.Errorf("could not bind allocation to blobbers: %v", err)
	//}

	var options []WithOption
	if mintNewTokens > 0 {
		options = []WithOption{WithTokenMint(mintNewTokens)}
	}
	// create write pool and lock tokens
	if err := sa.addToWritePool(txn, balances, options...); err != nil {
		logging.Logger.Error("new_allocation_request_failed: error adding to allocation write pool",
			zap.String("txn", txn.Hash),
			zap.Error(err))
		return nil, common.NewError("allocation_creation_failed", err.Error())
	}

	cost, err := sa.cost(blobbers)
	if err != nil {
		return nil, err
	}
	if sa.WritePool < cost {
		return nil, common.NewError("allocation_creation_failed",
			fmt.Sprintf("not enough tokens to cover the allocation cost"+" (%d < %d)", sa.WritePool, cost))
	}

	cancelCosts, err := cancellationCharge(conf.CancellationCharge, cost)
	if err != nil {
		return nil, common.NewErrorf("allocation_creation_failed", "cancellation charge: %v", err)
	}
	sa.CancelCost = cancelCosts

	if err := sa.checkFunding(); err != nil {
		return nil, common.NewError("allocation_creation_failed", err.Error())
	}
	m.tick("create_write_pool")
	if err := sc.addAllocation(sa, balances); err != nil {
		logging.Logger.Error("new_allocation_request_failed: error adding allocation",
			zap.String("txn", txn.Hash),
			zap.Error(err))
		return nil, common.NewErrorf("allocation_creation_failed", "%v", err)
	}
	m.tick("add_allocation")

	// emit event to eventDB
	emitChallengePoolEvent(sa, balances)
	emitAddOrOverwriteAllocationBlobberTerms(sa, balances, txn)

	return sa, nil
}

func setupNewAllocation(
	request newAllocationRequest,
	blobbers []*StorageNode,
	bil BlobberOfferStakeList,
	m Timings,
	now common.Timestamp,
	conf *Config,
	allocId string,
) (*StorageAllocation, []*StorageNode, error) {
	if len(request.Blobbers) < (request.DataShards + request.ParityShards) {
		logging.Logger.Error("new_allocation_request_failed: input blobbers less than requirement",
			zap.Int("request blobbers", len(request.Blobbers)),
			zap.Int("data shards", request.DataShards),
			zap.Int("parity_shards", request.ParityShards))
		return nil, nil, common.NewErrorf("allocation_creation_failed",
			"Blobbers provided are not enough to honour the allocation")
	}

	//if more than limit blobbers sent, just cut them
	if len(request.Blobbers) > conf.MaxBlobbersPerAllocation {
		logging.Logger.Error("new_allocation_request_failed: request blobbers more than max_blobbers_per_allocation",
			zap.Int("requested blobbers", len(request.Blobbers)),
			zap.Int("max blobbers per allocation", conf.MaxBlobbersPerAllocation))
		logging.Logger.Info("Too many blobbers selected, max available", zap.Int("max_blobber_size", conf.MaxBlobbersPerAllocation))
		request.Blobbers = request.Blobbers[:conf.MaxBlobbersPerAllocation]
	}

	logging.Logger.Debug("new_allocation_request", zap.Strings("blobbers", request.Blobbers))
	sa := request.storageAllocation() // (set fields, including expiration)
	sa.TimeUnit = conf.TimeUnit
	sa.ID = allocId
	sa.Tx = allocId

	blobberNodes, bSize, err := validateBlobbers(common.ToTime(now), sa, blobbers, bil, conf)
	if err != nil {
		logging.Logger.Error("new_allocation_request_failed: error validating blobbers",
			zap.Error(err))
		return nil, nil, common.NewErrorf("allocation_creation_failed", "%v", err)
	}
	m.tick("validate_blobbers")

	for _, b := range blobberNodes {
		rdtu, err := sa.restDurationInTimeUnits(now, sa.TimeUnit)
		if err != nil {
			return nil, nil, fmt.Errorf("could not get rest duration in time unit: %v", err)
		}

		mld, err := b.Terms.minLockDemand(sizeInGB(sa.bSize()), rdtu)
		if err != nil {
			return nil, nil, fmt.Errorf("could not get min lock demand: %v", err)
		}

		sa.Blobbers = append(sa.Blobbers, &AllocBlobber{
			BlobberID:     b.ID,
			Terms:         b.Terms,
			MinLockDemand: mld,
		})
		bil[b.Index].Allocated += bSize
	}
	m.tick("add_offer")

	if request.FileOptionsChanged {
		sa.FileOptions = request.FileOptions
	} else {
		sa.FileOptions = 63
	}

	sa.StartTime = now
	return sa, blobberNodes, nil
}

type Timings struct {
	timings map[string]time.Duration
	start   time.Time
}

func (t *Timings) tick(name string) {
	if t.timings == nil {
		return
	}
	t.timings[name] = time.Since(t.start)
	t.start = time.Now()
}

func validateBlobbers(
	creationDate time.Time,
	sa *StorageAllocation,
	blobbers []*StorageNode,
	bil BlobberOfferStakeList,
	conf *Config,
) ([]*StorageNode, int64, error) {
	sa.TimeUnit = conf.TimeUnit // keep the initial time unit

	// number of blobbers required
	size := sa.DataShards + sa.ParityShards
	// size of allocation for a blobber
	bSize := sa.bSize()
	list, errs := sa.validateEachBlobber(blobbers, bil, common.Timestamp(creationDate.Unix()), conf)
	if len(list) < size {
		return nil, 0, errors.New("Not enough blobbers to honor the allocation: " + strings.Join(errs, ", "))
	}

	sa.BlobberAllocs = make([]*BlobberAllocation, 0)
	sa.Stats = &StorageAllocationStats{}

	return list[:size], bSize, nil
}

type updateAllocationRequest struct {
	ID                      string           `json:"id"`               // allocation id
	Name                    string           `json:"name"`             // allocation name
	OwnerID                 string           `json:"owner_id"`         // Owner of the allocation
	OwnerPublicKey          string           `json:"owner_public_key"` // Owner Public Key of the allocation
	Size                    int64            `json:"size"`             // difference
	Expiration              common.Timestamp `json:"expiration_date"`  // difference
	UpdateTerms             bool             `json:"update_terms"`
	AddBlobberId            string           `json:"add_blobber_id"`
	RemoveBlobberId         string           `json:"remove_blobber_id"`
	SetThirdPartyExtendable bool             `json:"set_third_party_extendable"`
	FileOptionsChanged      bool             `json:"file_options_changed"`
	FileOptions             uint16           `json:"file_options"`
}

func (uar *updateAllocationRequest) decode(b []byte) error {
	return json.Unmarshal(b, uar)
}

// validate request
func (uar *updateAllocationRequest) validate(
	conf *Config,
	alloc *StorageAllocation,
) error {
	if uar.Size == 0 &&
		uar.Expiration == 0 &&
		len(uar.AddBlobberId) == 0 &&
		len(uar.Name) == 0 &&
		(!uar.SetThirdPartyExtendable || (uar.SetThirdPartyExtendable && alloc.ThirdPartyExtendable)) &&
		(!uar.FileOptionsChanged || uar.FileOptions == alloc.FileOptions) &&
		(alloc.Owner == uar.OwnerID) {
		return errors.New("update allocation changes nothing")
	} else {
		if ns := alloc.Size + uar.Size; ns < conf.MinAllocSize {
			return fmt.Errorf("new allocation size is too small: %d < %d",
				ns, conf.MinAllocSize)
		}
	}

	// Allocation expiry date shouldn't be reduced
	if uar.Expiration < 0 {
		return errors.New("duration of an allocation cannot be reduced")
	}

	if len(alloc.BlobberAllocs) == 0 {
		return errors.New("invalid allocation for updating: no blobbers")
	}

	if len(uar.AddBlobberId) > 0 {
		if _, found := alloc.BlobberAllocsMap[uar.AddBlobberId]; found {
			return fmt.Errorf("cannot add blobber %s, already in allocation", uar.AddBlobberId)
		}
	} else {
		if len(uar.RemoveBlobberId) > 0 {
			return errors.New("cannot remove a blobber without adding one")
		}
	}

	if len(uar.RemoveBlobberId) > 0 {
		if _, found := alloc.BlobberAllocsMap[uar.RemoveBlobberId]; !found {
			return fmt.Errorf("cannot remove blobber %s, not in allocation", uar.RemoveBlobberId)
		}
	}

	if uar.FileOptions > 63 {
		return fmt.Errorf("FileOptions %d incorrect", uar.FileOptions)
	}

	return nil
}

// calculate size difference for every blobber of the allocations
func (uar *updateAllocationRequest) getBlobbersSizeDiff(
	alloc *StorageAllocation) (diff int64) {
	return int64(math.Ceil(float64(uar.Size) / float64(alloc.DataShards)))
}

// new size of blobbers' allocation
func (uar *updateAllocationRequest) getNewBlobbersSize(
	alloc *StorageAllocation) (newSize int64) {

	return alloc.BSize + uar.getBlobbersSizeDiff(alloc)
}

// get blobbers by IDs concurrently, return error if any of them could not be acquired.
func getBlobbersByIDs(ids []string, balances chainstate.CommonStateContextI) ([]*StorageNode, error) {
	return chainstate.GetItemsByIDs(ids,
		func(id string, balances chainstate.CommonStateContextI) (*StorageNode, error) {
			return getBlobber(id, balances)
		},
		balances)
}

//func getStakePoolsByIDs(providerType spenum.Provider, ids []string, balances chainstate.CommonStateContextI) ([]*stakepool.StakePool, error) {
//	return chainstate.GetItemsByIDs(ids,
//		func(id string, balances chainstate.CommonStateContextI) (*stakepool.StakePool, error) {
//			sp, err := getStakePool(providerType, id, balances)
//			if err != nil {
//				return nil, err
//			}
//			return sp.StakePool, nil
//		},
//		balances)
//}

//func getBlobbersBriefByIDs(ids []string, balances chainstate.CommonStateContextI) ([]*StorageNode, error) {
//	return chainstate.GetItemsByIDs(ids,
//		func(id string, balances chainstate.CommonStateContextI) (*StorageNode, error) {
//			return getBlobberBrief(id, balances)
//		},
//		balances)
//}

//func getStakePoolsBriefByIDs(ids []string, providerType spenum.Provider, balances chainstate.CommonStateContextI) (map[string]*stakePoolBrief, error) {
//	type stakePoolPID struct {
//		pid  string
//		pool *stakePoolBrief
//	}
//
//	stakePools, err := chainstate.GetItemsByIDs(ids,
//		func(id string, balances chainstate.CommonStateContextI) (*stakePoolPID, error) {
//			sp, err := getStakePoolBrief(providerType, id, balances)
//			if err != nil {
//				return nil, err
//			}
//
//			return &stakePoolPID{
//				pid:  id,
//				pool: sp,
//			}, nil
//		},
//		balances)
//	if err != nil {
//		return nil, err
//	}
//
//	stakePoolMap := make(map[string]*stakePoolBrief, len(ids))
//	for _, sp := range stakePools {
//		stakePoolMap[sp.pid] = sp.pool
//	}
//
//	return stakePoolMap, nil
//}

//func getStakePoolsByIDs(ids []string, providerType spenum.Provider, balances chainstate.CommonStateContextI) (map[string]*stakePool, error) {
//	type stakePoolPID struct {
//		pid  string
//		pool *stakePool
//	}
//
//	stakePools, err := chainstate.GetItemsByIDs(ids,
//		func(id string, balances chainstate.CommonStateContextI) (*stakePoolPID, error) {
//			sp, err := getStakePool(providerType, id, balances)
//			if err != nil {
//				return nil, err
//			}
//
//			return &stakePoolPID{
//				pid:  id,
//				pool: sp,
//			}, nil
//		},
//		balances)
//	if err != nil {
//		return nil, err
//	}
//
//	stakePoolMap := make(map[string]*stakePool, len(ids))
//	for _, sp := range stakePools {
//		stakePoolMap[sp.pid] = sp.pool
//	}
//
//	return stakePoolMap, nil
//}

// getAllocationBlobbers loads blobbers of an allocation from store
func (sc *StorageSmartContract) getAllocationBlobbers(alloc *StorageAllocation,
	balances chainstate.StateContextI) (blobbers []*StorageNode, err error) {
	ids := make([]string, 0, len(alloc.Blobbers))
	for _, ba := range alloc.Blobbers {
		ids = append(ids, ba.BlobberID)
	}

	return chainstate.GetItemsByIDs(ids,
		func(id string, balances chainstate.CommonStateContextI) (*StorageNode, error) {
			return sc.getBlobber(id, balances)
		},
		balances)
}

// closeAllocation making it expired; the allocation will be alive the
// challenge_completion_time and be closed then
func (sc *StorageSmartContract) closeAllocation(
	t *transaction.Transaction,
	alloc *StorageAllocation,
	blobbers []*StorageNode,
	bil BlobberOfferStakeList,
	maxChallengeCompletionTime time.Duration,
	balances chainstate.StateContextI) (
	resp string, err error) {

	if alloc.Expiration-t.CreationDate <
		toSeconds(maxChallengeCompletionTime) {
		return "", common.NewError("allocation_closing_failed",
			"doesn't need to close allocation is about to expire")
	}

	// mark as expired, but it will be alive at least chellenge_competion_time
	alloc.Expiration = t.CreationDate

	for i := range alloc.BlobberAllocs {
		b := blobbers[i]
		if err := bil[b.Index].reduceOffer(getOffer(alloc.BSize, alloc.bTerms(i))); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"error removing offer: "+err.Error())
		}
	}

	// Save allocation
	if err := bil.Save(balances); err != nil {
		return "", common.NewError("allocation_closing_failed", err.Error())
	}

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("allocation_closing_failed",
			"can't Save allocation: "+err.Error())
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, alloc.ID, alloc.buildDbUpdates())

	return string(alloc.Encode()), nil // closing, TODO: return close status rather than whole allocation
}

func (sa *StorageAllocation) saveUpdatedAllocation(
	blobbers []*StorageNode,
	bil BlobberOfferStakeList,
	balances chainstate.StateContextI,
) (err error) {
	for _, b := range blobbers {
		emitUpdateBlobberAllocatedSavedHealth(b.ID, b.LastHealthCheck, bil[b.Index].Allocated, bil[b.Index].SavedData, balances)
	}
	// Save allocation
	_, err = balances.InsertTrieNode(sa.GetKey(ADDRESS), sa)
	if err != nil {
		return
	}

	if err := bil.Save(balances); err != nil {
		return fmt.Errorf("saving blobbers info list failed: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, sa.ID, sa.buildDbUpdates())
	return
}

func (sa *StorageAllocation) saveUpdatedStakes(balances chainstate.StateContextI) (err error) {
	// Save allocation
	_, err = balances.InsertTrieNode(sa.GetKey(ADDRESS), sa)
	if err != nil {
		return
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationStakes, sa.ID, sa.buildStakeUpdateEvent())
	return
}

// allocation period used to calculate weighted average prices
type allocPeriod struct {
	read   currency.Coin    // read price
	write  currency.Coin    // write price
	period common.Timestamp // period (duration)
	size   int64            // size for period
}

func (ap *allocPeriod) weight() float64 {
	return float64(ap.period) * float64(ap.size)
}

// returns weighted average read and write prices
func (ap *allocPeriod) join(np *allocPeriod) (avgRead, avgWrite currency.Coin, err error) {
	var (
		apw, npw = ap.weight(), np.weight() // weights
		ws       = apw + npw                // weights sum
		rp, wp   float64                    // read sum, write sum (weighted)
	)

	apReadF, err := ap.read.Float64()
	if err != nil {
		return 0, 0, err
	}

	apWriteF, err := ap.write.Float64()
	if err != nil {
		return 0, 0, err
	}

	npReadF, err := np.read.Float64()
	if err != nil {
		return 0, 0, err
	}

	npWriteF, err := np.write.Float64()
	if err != nil {
		return 0, 0, err
	}

	rp = (apReadF * apw) + (npReadF * npw)
	wp = (apWriteF * apw) + (npWriteF * npw)

	avgRead, err = currency.Float64ToCoin(rp / ws)
	if err != nil {
		return 0, 0, err
	}
	avgWrite, err = currency.Float64ToCoin(wp / ws)
	if err != nil {
		return 0, 0, err
	}
	return
}

func weightedAverage(prev, next *Terms, tx, pexp, expDiff common.Timestamp,
	psize, sizeDiff int64) (avg Terms, err error) {

	// allocation periods
	var left, added allocPeriod
	left.read, left.write = prev.ReadPrice, prev.WritePrice   // } prices
	added.read, added.write = next.ReadPrice, next.WritePrice // }
	left.size, added.size = psize, psize+sizeDiff             // sizes
	left.period, added.period = pexp-tx, pexp+expDiff-tx      // periods
	// join
	avg.ReadPrice, avg.WritePrice, err = left.join(&added)
	if err != nil {
		return
	}

	// just copy from next
	avg.MinLockDemand = next.MinLockDemand
	return
}

// The adjustChallengePool moves more or moves some tokens back from or to
// challenge pool during allocation extending or reducing.
func (sc *StorageSmartContract) adjustChallengePool(
	alloc *StorageAllocation,
	odr, ndr common.Timestamp,
	oterms []Terms,
	timeUnit time.Duration,
	balances chainstate.StateContextI,
) error {
	changes, err := alloc.challengePoolChanges(odr, ndr, timeUnit, oterms)
	if err != nil {
		return fmt.Errorf("adjust_challenge_pool: %v", err)
	}

	var changed bool
	sum := currency.Coin(0)
	for _, ch := range changes {
		_, err = ch.Int64()
		if err != nil {
			return err
		}
		switch {
		case ch > 0:
			err = alloc.moveToChallengePool(ch)
			sum += ch
			changed = true
		default:
			// no changes for the blobber
		}
		if err != nil {
			return fmt.Errorf("adjust_challenge_pool: %v", err)
		}
	}

	if changed {
		emitChallengePoolEvent(alloc, balances)
		i := int64(0)
		i, err = sum.Int64()
		if err != nil {
			return err
		}
		balances.EmitEvent(event.TypeStats, event.TagToChallengePool, alloc.ID, event.ChallengePoolLock{
			Client:       alloc.Owner,
			AllocationId: alloc.ID,
			Amount:       i,
		})
	}

	return nil
}

// extendAllocation extends size or/and expiration (one of them can be reduced);
// here we use new terms of blobbers
func (sc *StorageSmartContract) extendAllocation(
	txn *transaction.Transaction,
	conf *Config,
	alloc *StorageAllocation,
	blobbers []*StorageNode,
	bil BlobberOfferStakeList,
	req *updateAllocationRequest,
	balances chainstate.StateContextI,
) (err error) {
	var (
		diff   = req.getBlobbersSizeDiff(alloc) // size difference
		size   = req.getNewBlobbersSize(alloc)  // blobber size
		gbSize = sizeInGB(size)                 // blobber size in GB

		// keep original terms to adjust challenge pool value
		originalTerms = make([]Terms, 0, len(alloc.BlobberAllocs))
		// original allocation duration remains
		originalRemainingDuration = alloc.Expiration - txn.CreationDate
	)

	// adjust the expiration if changed, boundaries has already checked
	var prevExpiration = alloc.Expiration
	alloc.Expiration += req.Expiration // new expiration
	alloc.Size += req.Size             // new size

	// 1. update terms
	for i, details := range alloc.BlobberAllocs {
		originalTerms = append(originalTerms, alloc.bTerms(i)) // keep original terms will be changed
		oldOffer := getOffer(alloc.BSize, alloc.bTerms(i))
		var b = blobbers[i]
		if b.ID != details.BlobberID {
			return common.NewErrorf("allocation_extending_failed",
				"blobber %s and %s don't match", b.ID, details.BlobberID)
		}

		if b.Capacity == 0 {
			return common.NewErrorf("allocation_extending_failed",
				"blobber %s no longer provides its service", b.ID)
		}
		bi := bil[b.Index]
		if req.Size > 0 {
			if b.Capacity-bi.Allocated-diff < 0 {
				return common.NewErrorf("allocation_extending_failed",
					"blobber %s doesn't have enough free space", b.ID)
			}
		}

		bi.Allocated += diff // new capacity used

		// update terms using weighted average
		alloc.Blobbers[i].Terms, err = weightedAverage(&alloc.Blobbers[i].Terms, &b.Terms,
			txn.CreationDate, prevExpiration, alloc.Expiration, alloc.BSize,
			diff)
		if err != nil {
			return err
		}

		// since, new terms is weighted average based on previous terms and
		// past allocation time and new terms and new allocation time; then
		// we can easily recalculate new min_lock_demand value from allocation
		// start to its new end using the new weighted average terms; but, we
		// can't reduce the min_lock_demand_value; that's all;

		// new blobber's min lock demand (alloc.Expiration is already updated
		// and we can use restDurationInTimeUnits method here)
		rdtu, err := alloc.restDurationInTimeUnits(alloc.StartTime, conf.TimeUnit)
		if err != nil {
			return common.NewError("allocation_extending_failed", err.Error())
		}

		bt := alloc.bTerms(i)
		nbmld, err := bt.minLockDemand(gbSize, rdtu)
		if err != nil {
			return err
		}

		// min_lock_demand can be increased only
		if nbmld > alloc.bMinLockDemand(i) {
			alloc.Blobbers[i].MinLockDemand = nbmld
		}

		newOffer := getOffer(size, alloc.bTerms(i))
		if newOffer != oldOffer {
			if newOffer > oldOffer {
				coin, err := currency.MinusCoin(newOffer, oldOffer)
				if err != nil {
					return err
				}
				if err := bil[b.Index].addOffer(coin); err != nil {
					return fmt.Errorf("adding offer: %v", err)
				}
			} else {
				coin, err := currency.MinusCoin(oldOffer, newOffer)
				if err != nil {
					return err
				}
				if err := bil[b.Index].reduceOffer(coin); err != nil {
					return fmt.Errorf("reduce offer: %v", err)
				}
			}
		}
	}
	alloc.BSize = size // update to new size

	// lock tokens if this transaction provides them
	if txn.Value > 0 {
		if err = alloc.addToWritePool(txn, balances); err != nil {
			return common.NewErrorf("allocation_extending_failed", "%v", err)
		}
	}

	// add more tokens to related challenge pool, or move some tokens back
	var remainingDuration = alloc.Expiration - txn.CreationDate
	err = sc.adjustChallengePool(alloc, originalRemainingDuration, remainingDuration, originalTerms, conf.TimeUnit, balances)
	if err != nil {
		return common.NewErrorf("allocation_extending_failed", "%v", err)
	}
	return nil
}

// reduceAllocation reduces size or/and expiration (no one can be increased);
// here we use the same terms of related blobbers
func (sc *StorageSmartContract) reduceAllocation(
	txn *transaction.Transaction,
	conf *Config,
	alloc *StorageAllocation,
	blobbers []*StorageNode,
	bil BlobberOfferStakeList,
	req *updateAllocationRequest,
	balances chainstate.StateContextI,
) (err error) {
	var (
		diff    = req.getBlobbersSizeDiff(alloc) // newSize difference
		newSize = req.getNewBlobbersSize(alloc)  // blobber newSize

		// original allocation duration remains
		originalRemainingDuration = alloc.Expiration - txn.CreationDate
	)

	// adjust the expiration if changed, boundaries has already checked
	alloc.Expiration += req.Expiration
	alloc.Size += req.Size

	// 1. update terms
	for i := range alloc.BlobberAllocs {
		var b = blobbers[i]
		bi := bil[b.Index]
		oldOffer := getOffer(alloc.BSize, alloc.bTerms(i))
		bi.Allocated += diff // new capacity used

		// update stake pool
		newOffer := getOffer(newSize, alloc.bTerms(i))
		if newOffer != oldOffer {
			if newOffer < oldOffer {
				if err := bil[b.Index].reduceOffer(oldOffer - newOffer); err != nil {
					return fmt.Errorf("removing offer: %v", err)
				}
			} else {
				// if we are adding a blobber then we will want to add a new offer for that blobber
				if err := bil[b.Index].addOffer(newOffer - oldOffer); err != nil {
					return fmt.Errorf("adding offer: %v", err)
				}
			}

			emitUpdateBlobberAllocatedSavedHealth(b.ID, b.LastHealthCheck, bi.Allocated, bi.SavedData, balances)
		}
	}
	alloc.BSize = newSize // update to new bSize

	// lock tokens if this transaction provides them
	if txn.Value > 0 {
		if err = alloc.addToWritePool(txn, balances); err != nil {
			return common.NewErrorf("allocation_reducing_failed", "%v", err)
		}
	}

	// new allocation duration remains
	var remainingDuration = alloc.Expiration - txn.CreationDate
	err = sc.adjustChallengePool(alloc, originalRemainingDuration, remainingDuration, nil, conf.TimeUnit,
		balances)
	if err != nil {
		return common.NewErrorf("allocation_reducing_failed", "%v", err)
	}
	return nil

}

// update allocation allows to change allocation size or expiration;
// if expiration reduced or unchanged, then existing terms of blobbers used,
// otherwise new terms used; also, it locks additional tokens if size is
// extended and it checks blobbers for required stake;
func (sc *StorageSmartContract) updateAllocationRequest(
	txn *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (resp string, err error) {
	var conf *Config
	if conf, err = sc.getConfig(balances, false); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get SC configurations: "+err.Error())
	}
	var req updateAllocationRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"invalid request: "+err.Error())
	}

	alloc, bil, conf, err := sc.preloadUpdateAllocation(req.ID, balances)
	if err != nil {
		return "", err
	}

	return sc.updateAllocationRequestInternal(txn, req, alloc, bil, conf, balances)
}

func (sc *StorageSmartContract) preloadUpdateAllocation(allocID string, balances chainstate.StateContextI) (
	*StorageAllocation, BlobberOfferStakeList, *Config, error) {
	var (
		alloc *StorageAllocation
		bil   BlobberOfferStakeList
		conf  *Config
	)
	cr := concurrentReader{}
	cr.add(func() error {
		var err error
		alloc, err = sc.getAllocation(allocID, balances)
		if err != nil {
			return common.NewErrorf("allocation_updating_failed",
				"can't get existing allocation: %v", err)
		}

		return nil
	})
	cr.add(func() error {
		var err error
		bil, err = getBlobbersInfoList(balances)
		if err != nil {
			return common.NewErrorf("allocation_updating_failed",
				"could not get blobbers info list: %v", err)
		}

		return nil
	})
	cr.add(func() error {
		var err error
		conf, err = sc.getConfig(balances, false)
		if err != nil {
			return common.NewError("allocation_updating_failed", "can't get SC configurations: "+err.Error())
		}
		return nil
	})

	if err := cr.do(); err != nil {
		return nil, nil, nil, err
	}

	return alloc, bil, conf, nil
}

func (sc *StorageSmartContract) updateAllocationRequestInternal(
	t *transaction.Transaction,
	request updateAllocationRequest,
	alloc *StorageAllocation,
	bil BlobberOfferStakeList,
	conf *Config,
	balances chainstate.StateContextI,
) (resp string, err error) {
	if t.ClientID == "" {
		return "", common.NewError("allocation_updating_failed",
			"missing client_id in transaction")
	}

	if request.OwnerID == "" {
		request.OwnerID = t.ClientID
	}

	if t.ClientID != alloc.Owner {
		if !alloc.ThirdPartyExtendable || (request.Size <= 0 && request.Expiration <= 0) {
			return "", common.NewError("allocation_updating_failed",
				"only owner can update the allocation")
		}
	}

	if err = request.validate(conf, alloc); err != nil {
		return "", common.NewError("allocation_updating_failed", err.Error())
	}

	// can't update expired allocation
	if alloc.Expiration < t.CreationDate {
		return "", common.NewError("allocation_updating_failed",
			"can't update expired allocation")
	}

	// update allocation transaction hash
	alloc.Tx = t.Hash

	var blobbers []*StorageNode
	cr := concurrentReader{}
	cr.add(func() error {
		var err error
		blobbers, err = sc.getAllocationBlobbers(alloc, balances)
		if err != nil {
			return common.NewError("allocation_updating_failed",
				err.Error())
		}
		return nil
	})

	var addBlobber *StorageNode
	if len(request.AddBlobberId) > 0 {
		cr.add(func() error {
			var err error
			addBlobber, err = sc.getBlobber(request.AddBlobberId, balances)
			if err != nil {
				return common.NewError("allocation_updating_failed",
					err.Error())
			}
			return nil
		})
	}

	// check lock token balance
	if t.Value > 0 {
		cr.add(func() error {
			return stakepool.CheckClientBalance(t.ClientID, t.Value, balances)
		})
	}

	if err := cr.do(); err != nil {
		return "", err
	}

	// If the txn client_id is not the owner of the allocation, should just be able to extend the allocation if permissible
	// This way, even if an atttacker of an innocent user incorrectly tries to modify any other part of the allocation, it will not have any effect
	if t.ClientID != alloc.Owner /* Third-party actions */ {
		if request.Size < 0 || request.Expiration < 0 {
			return "", common.NewError("allocation_updating_failed", "third party can only extend the allocation")
		}

		err = sc.extendAllocation(t, conf, alloc, blobbers, bil, &request, balances)
		if err != nil {
			return "", err
		}
	} else /* Owner Actions */ {

		// update allocation transaction hash
		alloc.Tx = t.Hash

		// adjust expiration
		var newExpiration = alloc.Expiration + request.Expiration
		// close allocation now

		if newExpiration <= t.CreationDate {
			return sc.closeAllocation(t, alloc, blobbers, bil, conf.MaxChallengeCompletionTime, balances) // update alloc tx, expir
		}

		// an allocation can't be shorter than configured in SC
		// (prevent allocation shortening for entire period)
		if request.Expiration > 0 {
			if newExpiration-t.CreationDate < toSeconds(conf.TimeUnit) {
				return "", common.NewError("allocation_updating_failed",
					"allocation duration becomes too short")
			}
		}

		var newSize = request.Size + alloc.Size
		if newSize < conf.MinAllocSize || newSize < alloc.UsedSize {
			return "", common.NewError("allocation_updating_failed",
				"allocation size becomes too small")
		}

		if len(request.AddBlobberId) > 0 {
			blobbers, err = alloc.changeBlobbers(
				conf,
				blobbers,
				bil,
				request.AddBlobberId,
				request.RemoveBlobberId,
				addBlobber,
				sc, t.CreationDate, balances)
			if err != nil {
				return "", common.NewError("allocation_updating_failed", err.Error())
			}
		}

		if len(blobbers) != len(alloc.BlobberAllocs) {
			return "", common.NewError("allocation_updating_failed",
				"error allocation blobber size mismatch")
		}

		if request.UpdateTerms {
			for i := range alloc.BlobberAllocs {
				if alloc.bTerms(i).WritePrice >= blobbers[i].Terms.WritePrice {
					alloc.Blobbers[i].Terms.WritePrice = blobbers[i].Terms.WritePrice
				}
				if alloc.bTerms(i).ReadPrice >= blobbers[i].Terms.ReadPrice {
					alloc.Blobbers[i].Terms.ReadPrice = blobbers[i].Terms.ReadPrice
				}
				alloc.Blobbers[i].Terms.MinLockDemand = blobbers[i].Terms.MinLockDemand
			}
		}

		// if size or expiration increased, then we use new terms
		// otherwise, we use the same terms
		if request.Size > 0 || request.Expiration > 0 {
			err = sc.extendAllocation(t, conf, alloc, blobbers, bil, &request, balances)
		} else if request.Size < 0 || request.Expiration < 0 {
			err = sc.reduceAllocation(t, conf, alloc, blobbers, bil, &request, balances)
		} else if len(request.AddBlobberId) > 0 {
			err = sc.extendAllocation(t, conf, alloc, blobbers, bil, &request, balances)
		}
		if err != nil {
			return "", err
		}

		if err := alloc.checkFunding(); err != nil {
			return "", common.NewError("allocation_updating_failed", err.Error())
		}

		if request.SetThirdPartyExtendable {
			alloc.ThirdPartyExtendable = true
		}

		if request.FileOptionsChanged {
			alloc.FileOptions = request.FileOptions
		}

		if len(request.RemoveBlobberId) > 0 {
			balances.EmitEvent(event.TypeStats, event.TagDeleteAllocationBlobberTerm, t.Hash, []event.AllocationBlobberTerm{
				{
					AllocationID: alloc.ID,
					BlobberID:    request.RemoveBlobberId,
				},
			})
		}

		if request.OwnerID != alloc.Owner {
			alloc.Owner = request.OwnerID
			if request.OwnerPublicKey == "" {
				return "", common.NewError("allocation_updating_failed", "owner public key is required when updating owner id")
			}
			alloc.OwnerPublicKey = request.OwnerPublicKey
		}
	}

	err = alloc.saveUpdatedAllocation(blobbers, bil, balances)
	if err != nil {
		return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

	emitAddOrOverwriteAllocationBlobberTerms(alloc, balances, t)

	return string(alloc.Encode()), nil // TODO: return update allocation status rather than the whole allocation
}

func getPreferredBlobbers(preferredBlobbers []string, allBlobbers []*StorageNode) (selectedBlobbers []*StorageNode, err error) {
	blobberMap := make(map[string]*StorageNode)
	for _, storageNode := range allBlobbers {
		blobberMap[storageNode.BaseURL] = storageNode
	}
	for _, blobberURL := range preferredBlobbers {
		selectedBlobber, ok := blobberMap[blobberURL]
		if !ok {
			err = errors.New("invalid preferred blobber URL")
			return
		}
		selectedBlobbers = append(selectedBlobbers, selectedBlobber)
	}
	return
}

func randomizeNodes(in []*StorageNode, out []*StorageNode, n int, seed int64) []*StorageNode {
	nOut := minInt(len(in), n)
	nOut = maxInt(1, nOut)
	randGen := rand.New(rand.NewSource(seed))
	for {
		i := randGen.Intn(len(in))
		if !checkExists(in[i], out) {
			out = append(out, in[i])
		}
		if len(out) >= nOut {
			break
		}
	}
	return out
}

func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func checkExists(c *StorageNode, sl []*StorageNode) bool {
	for _, s := range sl {
		if s.ID == c.ID {
			return true
		}
	}
	return false
}

func (sc *StorageSmartContract) finalizedPassRates(alloc *StorageAllocation) ([]float64, error) {
	if alloc.Stats == nil {
		alloc.Stats = &StorageAllocationStats{}
	}
	var failed, succesful int64 = 0, 0
	var passRates = make([]float64, 0, len(alloc.BlobberAllocs))
	for _, ba := range alloc.BlobberAllocs {
		if ba.Stats == nil {
			ba.Stats = new(StorageAllocationStats)
			passRates = append(passRates, 1.0)
			continue
		}
		ba.Stats.FailedChallenges += ba.Stats.OpenChallenges
		ba.Stats.OpenChallenges = 0

		baTotal := ba.Stats.FailedChallenges + ba.Stats.SuccessChallenges
		if baTotal == 0 {
			passRates = append(passRates, 1.0)
			continue
		}

		if ba.Stats.TotalChallenges == 0 {
			logging.Logger.Warn("empty total challenges on finalizedPassRates",
				zap.Int64("OpenChallenges", ba.Stats.OpenChallenges),
				zap.Int64("FailedChallenges", ba.Stats.FailedChallenges),
				zap.Int64("SuccessChallenges", ba.Stats.SuccessChallenges))
			return nil, errors.New("empty total challenges")
		}

		passRates = append(passRates, float64(ba.Stats.SuccessChallenges)/float64(ba.Stats.TotalChallenges))
		succesful += ba.Stats.SuccessChallenges
		failed += ba.Stats.FailedChallenges
	}
	alloc.Stats.SuccessChallenges = succesful
	alloc.Stats.FailedChallenges = failed
	alloc.Stats.OpenChallenges = 0
	return passRates, nil
}

// a blobber can not send a challenge response, thus we have to check out
// challenge requests and their expiration
func (sc *StorageSmartContract) canceledPassRates(
	alloc *StorageAllocation,
	now common.Timestamp,
	maxChallengeCompletionTime time.Duration,
	balances chainstate.StateContextI,
) (
	passRates []float64, err error) {

	if alloc.Stats == nil {
		alloc.Stats = &StorageAllocationStats{}
	}
	passRates = make([]float64, 0, len(alloc.BlobberAllocs))

	allocChallenges, err := sc.getAllocationChallenges(alloc.ID, balances)
	switch err {
	case util.ErrValueNotPresent:
	case nil:
		for _, oc := range allocChallenges.OpenChallenges {
			ba, ok := alloc.BlobberAllocsMap[oc.BlobberID]
			if !ok {
				continue
			}

			if ba.Stats == nil {
				ba.Stats = new(StorageAllocationStats) // make sure
			}

			var expire = oc.CreatedAt + toSeconds(maxChallengeCompletionTime)
			if expire < now {
				ba.Stats.FailedChallenges++
				alloc.Stats.FailedChallenges++
			} else {
				ba.Stats.SuccessChallenges++
				alloc.Stats.SuccessChallenges++
			}
			ba.Stats.OpenChallenges--
			alloc.Stats.OpenChallenges--
		}

	default:
		return nil, fmt.Errorf("getting allocation challenge: %v", err)
	}

	for _, ba := range alloc.BlobberAllocs {
		if ba.Stats.OpenChallenges > 0 {
			logging.Logger.Warn("not all challenges canceled", zap.Int64("remaining", ba.Stats.OpenChallenges))

			ba.Stats.FailedChallenges += ba.Stats.OpenChallenges
			alloc.Stats.FailedChallenges += ba.Stats.OpenChallenges

			ba.Stats.OpenChallenges = 0
		}

		if ba.Stats.TotalChallenges == 0 {
			passRates = append(passRates, 1.0)
			continue
		}
		// success rate for the blobber allocation
		//fmt.Println("pass rate i", i, "successful", d.Stats.SuccessChallenges, "failed", d.Stats.FailedChallenges)
		passRates = append(passRates, float64(ba.Stats.SuccessChallenges)/float64(ba.Stats.TotalChallenges))
	}

	alloc.Stats.OpenChallenges = 0
	return passRates, nil
}

// If blobbers doesn't provide their services, then user can use this
// cancel_allocation transaction to close allocation and unlock all tokens
// of write pool back to himself. The  cancel_allocation doesn't pay min_lock
// demand to blobbers.
func (sc *StorageSmartContract) cancelAllocationRequest(
	t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (resp string, err error) {
	var req lockRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("alloc_cancel_failed", err.Error())
	}
	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(req.AllocationID, balances)

	if err != nil {
		return "", common.NewError("alloc_cancel_failed", err.Error())
	}

	if alloc.Owner != t.ClientID {
		return "", common.NewError("alloc_cancel_failed",
			"only owner can cancel an allocation")
	}

	if alloc.Expiration < t.CreationDate {
		return "", common.NewError("alloc_cancel_failed",
			"trying to cancel expired allocation")
	}

	conf, err := getConfig(balances)
	if err != nil {
		return "", common.NewError("can't get config", err.Error())
	}
	var passRates []float64
	passRates, err = sc.canceledPassRates(alloc, t.CreationDate, conf.MaxChallengeCompletionTime, balances)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed",
			"calculating rest challenges success/fail rates: "+err.Error())
	}

	bil, err := getBlobbersInfoList(balances)
	if err != nil {
		return "", common.NewErrorf("alloc_cancel_failed", "could not get blobbers info list: %v", err)
	}
	bids := make([]string, 0, len(alloc.Blobbers))
	for _, b := range alloc.Blobbers {
		bids = append(bids, b.BlobberID)
	}

	blobbers, err := getBlobbersByIDs(bids, balances)
	if err != nil {
		return "", common.NewErrorf("alloc_cancel_failed", "could not get blobbers: %v", err)
	}

	//sps, err := getStakePoolsByIDs(spenum.Blobber, bids, balances)
	//if err != nil {
	//	return "", common.NewErrorf("alloc_cancel_failed", "could not get stake pools: %v", err)
	//}

	// can cancel
	// new values
	alloc.Expiration = t.CreationDate

	//sps := make([]*stakePool, 0, len(alloc.BlobberAllocs))
	for i := range alloc.BlobberAllocs {
		b := blobbers[i]
		//var sp *stakePool
		//if sp, err = sc.getStakePool(spenum.Blobber, d.BlobberID, balances); err != nil {
		//	return "", common.NewError("fini_alloc_failed",
		//		"can't get stake pool of "+d.BlobberID+": "+err.Error())
		//}
		if err := bil[b.Index].reduceOffer(getOffer(alloc.BSize, alloc.bTerms(i))); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"error removing offer: "+err.Error())
		}
		//sps = append(sps, sp)
	}

	err = sc.finishAllocation(t, alloc, blobbers, bil, passRates, balances)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed", err.Error())
	}

	if err := bil.Save(balances); err != nil {
		return "", common.NewErrorf("alloc_cancel_failed", "could not save blobbers info list: %v", err)
	}

	alloc.Finalized, alloc.Canceled = true, true
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed",
			"saving allocation: "+err.Error())
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, alloc.ID, alloc.buildDbUpdates())

	emitDeleteAllocationBlobberTerms(alloc, balances, t)

	return "canceled", nil
}

//
// finalize an allocation (after expire + challenge completion time)
//

// 1. challenge pool                  -> blobbers or write pool
// 2. write pool min_lock_demand left -> blobbers
// 3. remove offer from blobber (stake pool)
// 4. update blobbers used and in all blobbers list too
// 5. write pool                      -> client
func (sc *StorageSmartContract) finalizeAllocation(
	t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (resp string, err error) {

	var req lockRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(req.AllocationID, balances)
	if err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	// should be owner or one of blobbers of the allocation
	if !alloc.IsValidFinalizer(t.ClientID) {
		return "", common.NewError("fini_alloc_failed",
			"not allowed, unknown finalization initiator")
	}

	// should not be finalized
	if alloc.Finalized {
		return "", common.NewError("fini_alloc_failed",
			"allocation already finalized")
	}

	conf, err := getConfig(balances)
	if err != nil {
		return "", common.NewError("can't get config", err.Error())
	}

	// should be expired
	if alloc.Until(conf.MaxChallengeCompletionTime) > t.CreationDate {
		return "", common.NewError("fini_alloc_failed",
			"allocation is not expired yet, or waiting a challenge completion")
	}

	var passRates []float64
	passRates, err = sc.finalizedPassRates(alloc)
	if err != nil {
		return "", common.NewError("fini_alloc_failed",
			"calculating rest challenges success/fail rates: "+err.Error())
	}

	bids := make([]string, 0, len(alloc.Blobbers))
	for _, b := range alloc.Blobbers {
		bids = append(bids, b.BlobberID)
	}

	blobbers, err := getBlobbersByIDs(bids, balances)
	if err != nil {
		return "", common.NewErrorf("alloc_cancel_failed", "could not get blobbers: %v", err)
	}

	//sps, err := getStakePoolsByIDs(spenum.Blobber, bids, balances)
	//if err != nil {
	//	return "", common.NewErrorf("alloc_cancel_failed", "could not get stake pools: %v", err)
	//}

	bil, err := getBlobbersInfoList(balances)
	if err != nil {
		return "", common.NewErrorf("alloc_cancel_failed", "could not get blobbers info list: %v", err)
	}

	err = sc.finishAllocation(t, alloc, blobbers, bil, passRates, balances)
	if err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	if err := bil.Save(balances); err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	alloc.Finalized = true
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed",
			"saving allocation: "+err.Error())
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, alloc.ID, alloc.buildDbUpdates())

	emitUpdateAllocationBlobberTerms(alloc, balances, t)

	return "finalized", nil
}

func (sc *StorageSmartContract) finishAllocation(
	t *transaction.Transaction,
	alloc *StorageAllocation,
	blobbers []*StorageNode,
	bil BlobberOfferStakeList,
	passRates []float64,
	//sps []*stakepool.StakePool,
	balances chainstate.StateContextI,
) (err error) {
	//before := make([]currency.Coin, len(blobbers))
	deductionFromWritePool := currency.Coin(0)

	challenges, err := sc.getAllocationChallenges(alloc.ID, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return fmt.Errorf("could not get allocation challenges: %v", err)
		}
	}

	// we can use the i for the blobbers list above because of algorithm
	// of the getAllocationBlobbers method; also, we can use the i in the
	// passRates list above because of algorithm of the adjustChallenges
	for i, d := range alloc.BlobberAllocs {
		// min lock demand rest
		mld := alloc.bMinLockDemand(i)
		if mld > d.Spent {
			delta, err := currency.MinusCoin(mld, d.Spent)
			if err != nil {
				return err
			}
			if alloc.WritePool < delta {
				return fmt.Errorf("paying min_lock for blobber %v"+
					"ammount was short by %v", d.BlobberID, delta)
			}
			alloc.WritePool, err = currency.MinusCoin(alloc.WritePool, delta)
			if err != nil {
				return err
			}
			deductionFromWritePool, err = currency.AddCoin(deductionFromWritePool, delta)
			if err != nil {
				return err
			}
			//before[i] = bil[blobbers[i].Index].TotalStake
			//if err != nil {
			//	return err
			//}

			//err = sps[i].DistributeRewards(delta, d.BlobberID, spenum.Blobber, spenum.MinLockDemandReward, balances)
			//if err != nil {
			//	return fmt.Errorf("distribute rewards failed, paying min_lock %v for blobber "+
			//		"%v from write pool %v, minlock demand %v spent %v error %v",
			//		delta, d.BlobberID, alloc.WritePool, mld, d.Spent, err.Error())
			//}
			bReward, err := currency.AddCoin(bil[blobbers[i].Index].Rewards, delta)
			if err != nil {
				return fmt.Errorf("pay min lock demand failed: %v", err.Error())
			}
			bil[blobbers[i].Index].Rewards = bReward
			d.Spent, err = currency.AddCoin(d.Spent, delta)
			if err != nil {
				return err
			}
		}
	}

	var passPayments currency.Coin
	for i, d := range alloc.BlobberAllocs {
		if alloc.UsedSize > 0 && alloc.ChallengePool > 0 && passRates[i] > 0 && d.Stats != nil {
			ratio := float64(d.Stats.UsedSize) / float64(alloc.UsedSize)
			cpBalance, err := alloc.ChallengePool.Float64()
			if err != nil {
				return err
			}

			reward, err := currency.Float64ToCoin(cpBalance * ratio * passRates[i])
			if err != nil {
				return err
			}

			//err = sps[i].DistributeRewards(reward, d.BlobberID, spenum.Blobber, spenum.ChallengePassReward, balances)
			//if err != nil {
			//	return fmt.Errorf("failed to distribute rewards blobber: %s, err: %v", d.BlobberID, err)
			//}
			bReward, err := currency.AddCoin(bil[blobbers[i].Index].Rewards, reward)
			if err != nil {
				return fmt.Errorf("pass payments failed: %v", err)
			}
			bil[blobbers[i].Index].Rewards = bReward

			d.Spent, err = currency.AddCoin(d.Spent, reward)
			if err != nil {
				return fmt.Errorf("blobber alloc spent: %v", err)
			}
			passPayments, err = currency.AddCoin(passPayments, reward)
			if err != nil {
				return fmt.Errorf("pass payments: %v", err)
			}
		}
	}

	prevBal := alloc.ChallengePool
	alloc.ChallengePool, err = currency.MinusCoin(alloc.ChallengePool, passPayments)
	if err != nil {
		return err
	}

	if alloc.ChallengePool > 0 {
		alloc.MovedBack, err = currency.AddCoin(alloc.MovedBack, alloc.ChallengePool)
		if err != nil {
			return err
		}

		err = alloc.moveFromChallengePool(alloc.ChallengePool)
		if err != nil {
			return fmt.Errorf("failed to move challenge pool back to write pool: %v", err)
		}
	}

	cancelCost := alloc.CancelCost
	if alloc.WritePool < alloc.CancelCost {
		cancelCost = alloc.WritePool
		logging.Logger.Error("insufficient funds, %v, for cancellation charge, %v. distributing the remaining write pool.")
	}

	alloc.WritePool, err = currency.MinusCoin(alloc.WritePool, cancelCost)
	if err != nil {
		return fmt.Errorf("failed to deduct cancellation charges from write pool: %v", err)
	}
	// This event just decreases the cancelation charge from the write pool's reflection in global snapshot's total client locked tokens
	deductionFromWritePool, err = currency.AddCoin(deductionFromWritePool, cancelCost)
	if err != nil {
		return fmt.Errorf("failed to add cancellation charge to deduction from write pool: %v", err)
	}
	amt, err := deductionFromWritePool.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert deduction from write pool to int64: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUnlockWritePool, alloc.ID, event.WritePoolLock{
		Client:       t.ClientID,
		AllocationId: alloc.ID,
		Amount:       amt,
	})

	reward, _, err := currency.DistributeCoin(cancelCost, int64(len(alloc.BlobberAllocs)))
	if err != nil {
		return fmt.Errorf("failed to distribute cancellation charge: %v", err)
	}

	for i, ba := range alloc.BlobberAllocs {
		dReward, err := currency.AddCoin(bil[blobbers[i].Index].Rewards, reward)
		if err != nil {
			return fmt.Errorf("cancel charge reward failed: %v", err)
		}
		b := blobbers[i]
		bil[b.Index].Rewards = dReward
		//err = sps[i].DistributeRewards(reward, ba.BlobberID, spenum.Blobber, spenum.CancellationChargeReward, balances)
		//if err != nil {
		//	return fmt.Errorf("failed to distribute rewards, blobber: %s, err: %v", ba.BlobberID, err)
		//}

		//if err = sps[i].Save(spenum.Blobber, ba.BlobberID, balances); err != nil {
		//	return fmt.Errorf("failed to save stake pool: %s, err: %v", ba.BlobberID, err)
		//}

		// TODO: update when locking new stake or collect rewards
		//staked, err := sps[i].Stake()
		//if err != nil {
		//	return err
		//}
		//bil[i].TotalStake = staked

		//tag, data := event.NewUpdateBlobberTotalStakeEvent(ba.BlobberID, staked)
		//balances.EmitEvent(event.TypeStats, tag, ba.BlobberID, data)

		bil[b.Index].Allocated -= alloc.BSize
		bil[b.Index].SavedData -= ba.Stats.UsedSize
		allocated := bil[b.Index].Allocated

		// get blobber allocations partitions
		blobberAllocParts, err := partitionsBlobberAllocations(ba.BlobberID, balances)
		if err != nil {
			return common.NewErrorf("fini_alloc_failed",
				"error getting blobber_challenge_allocation list: %v", err)
		}
		if err := partitionsBlobberAllocationsRemove(balances, ba.BlobberID, ba.AllocationID, blobberAllocParts); err != nil {
			return err
		}
		if err := blobberAllocParts.Save(balances); err != nil {
			return common.NewErrorf("fini_alloc_failed",
				"error saving blobber allocation partitions: %v", err)
		}

		// Update saved data on events_db
		emitUpdateBlobberAllocatedSavedHealth(b.ID, b.LastHealthCheck, allocated, bil[b.Index].SavedData, balances)
	}

	emitChallengePoolEvent(alloc, balances)
	pbi, err := prevBal.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert balance: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagFromChallengePool, alloc.ID, event.ChallengePoolLock{
		Client:       alloc.Owner,
		AllocationId: alloc.ID,
		Amount:       pbi,
	})

	if challenges != nil {
		for _, challenge := range challenges.OpenChallenges {
			ba, ok := alloc.BlobberAllocsMap[challenge.BlobberID]

			if ok {
				emitUpdateChallenge(&StorageChallenge{
					ID:           challenge.ID,
					AllocationID: alloc.ID,
					BlobberID:    challenge.BlobberID,
				}, true, balances, alloc.Stats, ba.Stats)
			}
		}
	}

	alloc.Finalized = true
	return nil
}

func emitUpdateAllocationStatEvent(w *WriteMarker, movedTokens currency.Coin, balances chainstate.StateContextI) {
	alloc := event.Allocation{
		AllocationID: w.AllocationID,
		UsedSize:     w.Size,
	}

	if w.Size > 0 {
		alloc.MovedToChallenge = movedTokens
	} else if w.Size < 0 {
		alloc.MovedBack = movedTokens
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationStat, alloc.AllocationID, &alloc)
}
