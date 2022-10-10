package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
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

	return
}

func (sc *StorageSmartContract) addAllocation(alloc *StorageAllocation,
	balances chainstate.StateContextI) (string, error) {
	ta := &StorageAllocation{}
	err := balances.GetTrieNode(alloc.GetKey(sc.ID), ta)
	if err == nil {
		return "", common.NewErrorf("add_allocation_failed",
			"allocation id already used in trie: %v", alloc.GetKey(sc.ID))
	}
	if err != util.ErrValueNotPresent {
		return "", common.NewErrorf("add_allocation_failed",
			"unexpected error: %v", err)
	}

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("add_allocation_failed",
			"saving new allocation: %v", err)
	}

	err = alloc.emitAdd(balances)
	if err != nil {
		return "", common.NewErrorf("add_allocation_failed",
			"saving new allocation in db: %v", err)
	}

	buff := alloc.Encode()
	return string(buff), nil
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
	IsImmutable          bool             `json:"is_immutable"`
	ThirdPartyExtendable bool             `json:"third_party_extendable"`
	FileOptions          uint8            `json:"file_options"`
}

// storageAllocation from the request
func (nar *newAllocationRequest) storageAllocation() (sa *StorageAllocation) {
	sa = new(StorageAllocation)
	sa.Name = nar.Name
	sa.DataShards = nar.DataShards
	sa.ParityShards = nar.ParityShards
	sa.Size = nar.Size
	sa.Expiration = nar.Expiration
	sa.Owner = nar.Owner
	sa.OwnerPublicKey = nar.OwnerPublicKey
	sa.PreferredBlobbers = nar.Blobbers
	sa.ReadPriceRange = nar.ReadPriceRange
	sa.WritePriceRange = nar.WritePriceRange
	sa.IsImmutable = nar.IsImmutable
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
	if dur < conf.MinAllocDuration {
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

	resp, err := sc.newAllocationRequestInternal(t, input, conf, 0, balances, timings)
	if err != nil {
		return "", err
	}

	return resp, err
}

type blobberWithPool struct {
	*StorageNode
	Pool *stakePool
	idx  int
}

// newAllocationRequest creates new allocation
func (sc *StorageSmartContract) newAllocationRequestInternal(
	txn *transaction.Transaction,
	input []byte,
	conf *Config,
	mintNewTokens currency.Coin,
	balances chainstate.StateContextI,
	timings map[string]time.Duration,
) (resp string, err error) {
	m := Timings{timings: timings, start: common.ToTime(txn.CreationDate)}
	var request newAllocationRequest
	if err = request.decode(input); err != nil {
		logging.Logger.Error("new_allocation_request_failed: error decoding input",
			zap.String("txn", txn.Hash),
			zap.Error(err))
		return "", common.NewErrorf("allocation_creation_failed",
			"malformed request: %v", err)
	}
	if err := request.validate(common.ToTime(txn.CreationDate), conf); err != nil {
		return "", common.NewErrorf("allocation_creation_failed", "invalid request: "+err.Error())
	}

	if request.Owner == "" {
		request.Owner = txn.ClientID
		request.OwnerPublicKey = txn.PublicKey
	}

	blobbers, err := getBlobbersByIDs(request.Blobbers, balances)
	if err != nil {
		return "", common.NewErrorf("allocation_creation_failed", "get blobbers failed: %v", err)
	}

	if len(blobbers) < (request.DataShards + request.ParityShards) {
		logging.Logger.Error("new_allocation_request_failed: blobbers fetched are less than requested blobbers",
			zap.String("txn", txn.Hash),
			zap.Int("fetched blobbers", len(blobbers)),
			zap.Int("data shards", request.DataShards),
			zap.Int("parity_shards", request.ParityShards))
		return "", common.NewErrorf("allocation_creation_failed",
			"Not enough provided blobbers found in mpt")
	}

	if request.Owner == "" {
		request.Owner = txn.ClientID
		request.OwnerPublicKey = txn.PublicKey
	}

	logging.Logger.Debug("new_allocation_request", zap.String("t_hash", txn.Hash), zap.Strings("blobbers", request.Blobbers))
	var sa = request.storageAllocation() // (set fields, including expiration)
	spMap, err := getStakePoolsByIDs(request.Blobbers, spenum.Blobber, balances)
	if err != nil {
		return "", common.NewErrorf("allocation_creation_failed", "getting stake pools: %v", err)
	}
	if len(spMap) != len(blobbers) {
		return "", common.NewErrorf("allocation_creation_failed", "missing blobber's stake pool: %v", err)
	}
	var sns []*storageNodeResponse
	for i := 0; i < len(blobbers); i++ {
		stake, err := spMap[blobbers[i].ID].stake()
		if err != nil {
			return "", common.NewErrorf("allocation_creation_failed", "cannot total stake pool for blobber %s: %v", blobbers[i].ID, err)
		}
		sns = append(sns, &storageNodeResponse{
			StorageNode: blobbers[i],
			TotalOffers: spMap[blobbers[i].ID].TotalOffers,
			TotalStake:  stake,
		})
	}

	sa, blobberNodes, err := setupNewAllocation(request, sns, m, txn.CreationDate, conf, txn.Hash)
	if err != nil {
		return "", err
	}

	for _, b := range blobberNodes {
		_, err = balances.InsertTrieNode(b.GetKey(sc.ID), b)
		if err != nil {
			logging.Logger.Error("new_allocation_request_failed: error inserting blobber",
				zap.String("txn", txn.Hash),
				zap.String("blobber", b.ID),
				zap.Error(err))
			return "", fmt.Errorf("can't save blobber: %v", err)
		}

		if err := spMap[b.ID].addOffer(sa.BlobberAllocsMap[b.ID].Offer()); err != nil {
			logging.Logger.Error("new_allocation_request_failed: error adding offer to blobber",
				zap.String("txn", txn.Hash),
				zap.String("blobber", b.ID),
				zap.Error(err))
			return "", fmt.Errorf("ading offer: %v", err)
		}

		if err = spMap[b.ID].save(spenum.Blobber, b.ID, balances); err != nil {
			logging.Logger.Error("new_allocation_request_failed: error saving blobber pool",
				zap.String("txn", txn.Hash),
				zap.String("blobber", b.ID),
				zap.Error(err))
			return "", fmt.Errorf("can't save blobber's stake pool: %v", err)
		}
	}

	var options []WithOption
	if mintNewTokens > 0 {
		options = []WithOption{WithTokenMint(mintNewTokens)}
	}
	// create write pool and lock tokens
	if err := sa.addToWritePool(txn, balances, options...); err != nil {
		logging.Logger.Error("new_allocation_request_failed: error adding to allocation write pool",
			zap.String("txn", txn.Hash),
			zap.Error(err))
		return "", common.NewError("allocation_creation_failed", err.Error())
	}

	cost, err := sa.cost()
	if err != nil {
		return "", err
	}
	if sa.WritePool < cost {
		return "", common.NewError("allocation_creation_failed",
			fmt.Sprintf("not enough tokens to cover the allocatin cost"+" (%d < %d)", sa.WritePool, cost))
	}

	if err := sa.checkFunding(conf.CancellationCharge); err != nil {
		return "", common.NewError("allocation_creation_failed", err.Error())
	}
	m.tick("create_write_pool")

	if err = sc.createChallengePool(txn, sa, balances); err != nil {
		logging.Logger.Error("new_allocation_request_failed: error creating challenge pool",
			zap.String("txn", txn.Hash),
			zap.Error(err))
		return "", common.NewError("allocation_creation_failed", err.Error())
	}
	m.tick("create_challenge_pool")

	if resp, err = sc.addAllocation(sa, balances); err != nil {
		logging.Logger.Error("new_allocation_request_failed: error adding allocation",
			zap.String("txn", txn.Hash),
			zap.Error(err))
		return "", common.NewErrorf("allocation_creation_failed", "%v", err)
	}
	m.tick("add_allocation")

	// emit event to eventDB
	emitAddOrOverwriteAllocationBlobberTerms(sa, balances, txn)

	return resp, err
}

func setupNewAllocation(
	request newAllocationRequest,
	blobbers []*storageNodeResponse,
	m Timings,
	now common.Timestamp,
	conf *Config,
	allocId string,
) (*StorageAllocation, []*StorageNode, error) {
	var err error
	m.tick("decode")
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
	var sa = request.storageAllocation() // (set fields, including expiration)
	m.tick("fetch_pools")
	sa.TimeUnit = conf.TimeUnit
	sa.ID = allocId
	sa.Tx = allocId

	blobberNodes, bSize, err := validateBlobbers(common.ToTime(now), sa, blobbers, conf)
	if err != nil {
		logging.Logger.Error("new_allocation_request_failed: error validating blobbers",
			zap.Error(err))
		return nil, nil, common.NewErrorf("allocation_creation_failed", "%v", err)
	}
	bi := make([]string, 0, len(blobberNodes))
	for _, b := range blobberNodes {
		bi = append(bi, b.ID)
	}
	logging.Logger.Debug("new_allocation_request", zap.Int64("size", bSize), zap.Strings("blobbers", bi))
	m.tick("validate_blobbers")

	sa.BlobberAllocsMap = make(map[string]*BlobberAllocation, len(blobberNodes))
	for _, b := range blobberNodes {
		balloc, err := newBlobberAllocation(bSize, sa, b, now, conf.TimeUnit)
		if err != nil {
			return nil, nil, common.NewErrorf("allocation_creation_failed",
				"can't create blobber allocation: %v", err)
		}
		sa.BlobberAllocs = append(sa.BlobberAllocs, balloc)
		sa.BlobberAllocsMap[b.ID] = balloc
		b.Allocated += bSize
	}
	m.tick("add_offer")

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
}

func (sc *StorageSmartContract) selectBlobbers(
	creationDate time.Time,
	allBlobbersList StorageNodes,
	sa *StorageAllocation,
	randomSeed int64,
	balances chainstate.CommonStateContextI,
) ([]*StorageNode, int64, error) {
	var err error
	var conf *Config
	if conf, err = getConfig(balances); err != nil {
		return nil, 0, fmt.Errorf("can't get config: %v", err)
	}

	sa.TimeUnit = conf.TimeUnit // keep the initial time unit

	// number of blobbers required
	var size = sa.DataShards + sa.ParityShards
	// size of allocation for a blobber
	var bSize = sa.bSize()
	timestamp := common.Timestamp(creationDate.Unix())

	list, err := sa.filterBlobbers(allBlobbersList.Nodes.copy(), timestamp,
		bSize, filterHealthyBlobbers(timestamp),
		sc.filterBlobbersByFreeSpace(timestamp, bSize, balances))
	if err != nil {
		return nil, 0, fmt.Errorf("could not filter blobbers: %v", err)
	}

	if len(list) < size {
		return nil, 0, errors.New("not enough blobbers to honor the allocation")
	}

	sa.BlobberAllocs = make([]*BlobberAllocation, 0)
	sa.Stats = &StorageAllocationStats{}

	var blobberNodes []*StorageNode
	if len(sa.PreferredBlobbers) > 0 {
		blobberNodes, err = getPreferredBlobbers(sa.PreferredBlobbers, list)
		if err != nil {
			return nil, 0, common.NewError("allocation_creation_failed",
				err.Error())
		}
	}

	if len(blobberNodes) < size {
		blobberNodes = randomizeNodes(list, blobberNodes, size, randomSeed)
	}

	return blobberNodes[:size], bSize, nil
}

func validateBlobbers(
	creationDate time.Time,
	sa *StorageAllocation,
	blobbers []*storageNodeResponse,
	conf *Config,
) ([]*StorageNode, int64, error) {
	sa.TimeUnit = conf.TimeUnit // keep the initial time unit

	// number of blobbers required
	var size = sa.DataShards + sa.ParityShards
	// size of allocation for a blobber
	var bSize = sa.bSize()
	var list, errs = sa.validateEachBlobber(blobbers, common.Timestamp(creationDate.Unix()))

	if len(list) < size {
		return nil, 0, errors.New("Not enough blobbers to honor the allocation: " + strings.Join(errs, ", "))
	}

	sa.BlobberAllocs = make([]*BlobberAllocation, 0)
	sa.Stats = &StorageAllocationStats{}

	return list[:size], bSize, nil
}

type updateAllocationRequest struct {
	ID                   string           `json:"id"`              // allocation id
	Name                 string           `json:"name"`            // allocation name
	OwnerID              string           `json:"owner_id"`        // Owner of the allocation
	Size                 int64            `json:"size"`            // difference
	Expiration           common.Timestamp `json:"expiration_date"` // difference
	SetImmutable         bool             `json:"set_immutable"`
	UpdateTerms          bool             `json:"update_terms"`
	AddBlobberId         string           `json:"add_blobber_id"`
	RemoveBlobberId      string           `json:"remove_blobber_id"`
	ThirdPartyExtendable bool             `json:"third_party_extendable"`
	FileOptions          uint8            `json:"file_options"`
}

func (uar *updateAllocationRequest) decode(b []byte) error {
	return json.Unmarshal(b, uar)
}

// validate request
func (uar *updateAllocationRequest) validate(
	conf *Config,
	alloc *StorageAllocation,
) error {
	if uar.SetImmutable && alloc.IsImmutable {
		return errors.New("allocation is already immutable")
	}
	if uar.Size == 0 && uar.Expiration == 0 && len(uar.AddBlobberId) == 0 && len(uar.Name) == 0 {
		if !uar.SetImmutable {
			return errors.New("update allocation changes nothing")
		}
	} else {
		if ns := alloc.Size + uar.Size; ns < conf.MinAllocSize {
			return fmt.Errorf("new allocation size is too small: %d < %d",
				ns, conf.MinAllocSize)
		}
	}

	//if uar.Expiration < 0 {
	//	return errors.New("duration of an allocation cannot be reduced")
	//}

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

	return alloc.BlobberAllocs[0].Size + uar.getBlobbersSizeDiff(alloc)
}

// get blobbers by IDs concurrently, return error if any of them could not be acquired.
func getBlobbersByIDs(ids []string, balances chainstate.CommonStateContextI) ([]*StorageNode, error) {
	return chainstate.GetItemsByIDs(ids,
		func(id string, balances chainstate.CommonStateContextI) (*StorageNode, error) {
			return getBlobber(id, balances)
		},
		balances)
}

func getStakePoolsByIDs(ids []string, providerType spenum.Provider, balances chainstate.CommonStateContextI) (map[string]*stakePool, error) {
	type stakePoolPID struct {
		pid  string
		pool *stakePool
	}

	stakePools, err := chainstate.GetItemsByIDs(ids,
		func(id string, balances chainstate.CommonStateContextI) (*stakePoolPID, error) {
			sp, err := getStakePool(providerType, id, balances)
			if err != nil {
				return nil, err
			}

			return &stakePoolPID{
				pid:  id,
				pool: sp,
			}, nil
		},
		balances)
	if err != nil {
		return nil, err
	}

	stakePoolMap := make(map[string]*stakePool, len(ids))
	for _, sp := range stakePools {
		stakePoolMap[sp.pid] = sp.pool
	}

	return stakePoolMap, nil
}

// getAllocationBlobbers loads blobbers of an allocation from store
func (sc *StorageSmartContract) getAllocationBlobbers(alloc *StorageAllocation,
	balances chainstate.StateContextI) (blobbers []*StorageNode, err error) {
	ids := make([]string, 0, len(alloc.BlobberAllocs))
	for _, ba := range alloc.BlobberAllocs {
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
func (sc *StorageSmartContract) closeAllocation(t *transaction.Transaction,
	alloc *StorageAllocation, balances chainstate.StateContextI) (
	resp string, err error) {

	if alloc.Expiration-t.CreationDate <
		toSeconds(getMaxChallengeCompletionTime()) {
		return "", common.NewError("allocation_closing_failed",
			"doesn't need to close allocation is about to expire")
	}

	// mark as expired, but it will be alive at least chellenge_competion_time
	alloc.Expiration = t.CreationDate

	for _, ba := range alloc.BlobberAllocs {
		var sp *stakePool
		if sp, err = sc.getStakePool(spenum.Blobber, ba.BlobberID, balances); err != nil {
			return "", fmt.Errorf("can't get stake pool of %s: %v", ba.BlobberID,
				err)
		}
		if err := sp.reduceOffer(ba.Offer()); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"error removing offer: "+err.Error())
		}
		if err = sp.save(spenum.Blobber, ba.BlobberID, balances); err != nil {
			return "", fmt.Errorf("can't save stake pool of %s: %v", ba.BlobberID,
				err)
		}
	}

	// save allocation

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("allocation_closing_failed",
			"can't save allocation: "+err.Error())
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, alloc.ID, alloc.buildDbUpdates())

	return string(alloc.Encode()), nil // closing
}

func (sa *StorageAllocation) saveUpdatedAllocation(
	blobbers []*StorageNode,
	balances chainstate.StateContextI,
) (err error) {
	for _, b := range blobbers {
		if _, err = balances.InsertTrieNode(b.GetKey(ADDRESS), b); err != nil {
			return
		}
		if err := emitUpdateBlobber(b, balances); err != nil {
			return fmt.Errorf("emmiting blobber %v: %v", b, err)
		}
	}

	// save allocation
	_, err = balances.InsertTrieNode(sa.GetKey(ADDRESS), sa)
	if err != nil {
		return
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, sa.ID, sa.buildDbUpdates())
	return
}

func (sa *StorageAllocation) saveUpdatedStakes(balances chainstate.StateContextI) (err error) {
	// save allocation
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
	avg.MaxOfferDuration = next.MaxOfferDuration
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
) (err error) {

	var (
		changes = alloc.challengePoolChanges(odr, ndr, timeUnit, oterms)
		cp      *challengePool
	)

	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("adjust_challenge_pool: %v", err)
	}

	var changed bool

	for _, ch := range changes {
		_, err = ch.Int64()
		if err != nil {
			return err
		}
		switch {
		case ch > 0:
			err = alloc.moveToChallengePool(cp, ch)
			changed = true
		default:
			// no changes for the blobber
		}
		if err != nil {
			return fmt.Errorf("adjust_challenge_pool: %v", err)
		}
	}

	if changed {
		err = cp.save(sc.ID, alloc, balances)
	}

	return
}

// extendAllocation extends size or/and expiration (one of them can be reduced);
// here we use new terms of blobbers
func (sc *StorageSmartContract) extendAllocation(
	txn *transaction.Transaction,
	conf *Config,
	alloc *StorageAllocation,
	blobbers []*StorageNode,
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
		originalTerms = append(originalTerms, details.Terms) // keep original terms will be changed
		oldOffer := details.Offer()
		var b = blobbers[i]
		if b.ID != details.BlobberID {
			return common.NewErrorf("allocation_extending_failed",
				"blobber %s and %s don't match", b.ID, details.BlobberID)
		}

		if b.Capacity == 0 {
			return common.NewErrorf("allocation_extending_failed",
				"blobber %s no longer provides its service", b.ID)
		}
		if req.Size > 0 {
			if b.Capacity-b.Allocated-diff < 0 {
				return common.NewErrorf("allocation_extending_failed",
					"blobber %s doesn't have enough free space", b.ID)
			}
		}

		b.Allocated += diff // new capacity used

		// update terms using weighted average
		details.Terms, err = weightedAverage(&details.Terms, &b.Terms,
			txn.CreationDate, prevExpiration, alloc.Expiration, details.Size,
			diff)
		if err != nil {
			return err
		}

		details.Size = size // new size

		if req.Expiration > toSeconds(b.Terms.MaxOfferDuration) {
			return common.NewErrorf("allocation_extending_failed",
				"blobber %s doesn't allow so long offers", b.ID)
		}

		// since, new terms is weighted average based on previous terms and
		// past allocation time and new terms and new allocation time; then
		// we can easily recalculate new min_lock_demand value from allocation
		// start to its new end using the new weighted average terms; but, we
		// can't reduce the min_lock_demand_value; that's all;

		// new blobber's min lock demand (alloc.Expiration is already updated
		// and we can use restDurationInTimeUnits method here)
		nbmld, err := details.Terms.minLockDemand(gbSize,
			alloc.restDurationInTimeUnits(alloc.StartTime, conf.TimeUnit))
		if err != nil {
			return err
		}

		// min_lock_demand can be increased only
		if nbmld > details.MinLockDemand {
			details.MinLockDemand = nbmld
		}

		newOffer := details.Offer()
		if newOffer != oldOffer {
			var sp *stakePool
			if sp, err = sc.getStakePool(spenum.Blobber, details.BlobberID, balances); err != nil {
				return fmt.Errorf("can't get stake pool of %s: %v", details.BlobberID, err)
			}
			if newOffer > oldOffer {
				coin, err := currency.MinusCoin(newOffer, oldOffer)
				if err != nil {
					return err
				}
				if err := sp.addOffer(coin); err != nil {
					return fmt.Errorf("adding offer: %v", err)
				}
			} else {
				coin, err := currency.MinusCoin(oldOffer, newOffer)
				if err != nil {
					return err
				}
				if err := sp.reduceOffer(coin); err != nil {
					return fmt.Errorf("adding offer: %v", err)
				}
			}
			if err = sp.save(spenum.Blobber, details.BlobberID, balances); err != nil {
				return fmt.Errorf("can't save stake pool of %s: %v", details.BlobberID,
					err)
			}

		}
	}

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
	req *updateAllocationRequest,
	balances chainstate.StateContextI,
) (err error) {
	var (
		diff = req.getBlobbersSizeDiff(alloc) // size difference
		size = req.getNewBlobbersSize(alloc)  // blobber size

		// original allocation duration remains
		originalRemainingDuration = alloc.Expiration - txn.CreationDate
	)

	// adjust the expiration if changed, boundaries has already checked
	alloc.Expiration += req.Expiration
	alloc.Size += req.Size

	// 1. update terms
	for i, ba := range alloc.BlobberAllocs {
		var b = blobbers[i]
		oldOffer := ba.Offer()
		b.Allocated += diff // new capacity used
		ba.Size = size      // new size
		// update stake pool
		newOffer := ba.Offer()
		if newOffer != oldOffer {
			var sp *stakePool
			if sp, err = sc.getStakePool(spenum.Blobber, ba.BlobberID, balances); err != nil {
				return fmt.Errorf("can't get stake pool of %s: %v", ba.BlobberID,
					err)
			}
			if newOffer < oldOffer {
				if err := sp.reduceOffer(oldOffer - newOffer); err != nil {
					return fmt.Errorf("removing offer: %v", err)
				}
			} else {
				// if we are adding a blobber then we will want to add a new offer for that blobber
				if err := sp.addOffer(newOffer - oldOffer); err != nil {
					return fmt.Errorf("adding offer: %v", err)
				}
			}

			if err = sp.save(spenum.Blobber, ba.BlobberID, balances); err != nil {
				return fmt.Errorf("can't save stake pool of %s: %v", ba.BlobberID,
					err)
			}
			if err := emitUpdateBlobber(b, balances); err != nil {
				return fmt.Errorf("emitting blobber %s, error:%v", b.ID, err)
			}
		}
	}

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
	return sc.updateAllocationRequestInternal(txn, input, conf, balances)
}

func (sc *StorageSmartContract) updateAllocationRequestInternal(
	t *transaction.Transaction,
	input []byte,
	conf *Config,
	balances chainstate.StateContextI,
) (resp string, err error) {
	if t.ClientID == "" {
		return "", common.NewError("allocation_updating_failed",
			"missing client_id in transaction")
	}

	var request updateAllocationRequest
	if err = request.decode(input); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"invalid request: "+err.Error())
	}

	if request.OwnerID == "" {
		request.OwnerID = t.ClientID
	}

	var alloc *StorageAllocation
	if alloc, err = sc.getAllocation(request.ID, balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get existing allocation: "+err.Error())
	}

	if t.ClientID != alloc.Owner || request.OwnerID != alloc.Owner {
		return "", common.NewError("allocation_updating_failed",
			"only owner can update the allocation")
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
	if len(request.Name) > 0 {
		alloc.Name = request.Name
	}

	// adjust expiration
	var newExpiration = alloc.Expiration + request.Expiration
	// close allocation now

	if newExpiration <= t.CreationDate {
		return sc.closeAllocation(t, alloc, balances) // update alloc tx, expir
	}

	// an allocation can't be shorter than configured in SC
	// (prevent allocation shortening for entire period)
	if newExpiration-t.CreationDate < toSeconds(conf.MinAllocDuration) {
		return "", common.NewError("allocation_updating_failed",
			"allocation duration becomes too short")
	}

	var newSize = request.Size + alloc.Size
	if newSize < conf.MinAllocSize || newSize < alloc.UsedSize {
		return "", common.NewError("allocation_updating_failed",
			"allocation size becomes too small")
	}

	// get blobber of the allocation to update them
	var blobbers []*StorageNode
	if blobbers, err = sc.getAllocationBlobbers(alloc, balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			err.Error())
	}

	if len(request.AddBlobberId) > 0 {
		blobbers, err = alloc.changeBlobbers(
			conf, blobbers, request.AddBlobberId, request.RemoveBlobberId, sc, t.CreationDate, balances,
		)
		if err != nil {
			return "", common.NewError("allocation_updating_failed", err.Error())
		}
	}

	if len(blobbers) != len(alloc.BlobberAllocs) {
		return "", common.NewError("allocation_updating_failed",
			"error allocation blobber size mismatch")
	}

	if request.UpdateTerms {
		for i, bd := range alloc.BlobberAllocs {
			if bd.Terms.WritePrice >= blobbers[i].Terms.WritePrice {
				bd.Terms.WritePrice = blobbers[i].Terms.WritePrice
			}
			if bd.Terms.ReadPrice >= blobbers[i].Terms.ReadPrice {
				bd.Terms.ReadPrice = blobbers[i].Terms.ReadPrice
			}
			bd.Terms.MinLockDemand = blobbers[i].Terms.MinLockDemand
			bd.Terms.MaxOfferDuration = blobbers[i].Terms.MaxOfferDuration
		}
	}

	// if size or expiration increased, then we use new terms
	// otherwise, we use the same terms
	if request.Size > 0 || request.Expiration > 0 {
		err = sc.extendAllocation(t, conf, alloc, blobbers, &request, balances)
	} else if request.Size < 0 || request.Expiration < 0 {
		err = sc.reduceAllocation(t, conf, alloc, blobbers, &request, balances)
	} else if len(request.AddBlobberId) > 0 {
		err = sc.extendAllocation(t, conf, alloc, blobbers, &request, balances)
	}
	if err != nil {
		return "", err
	}

	if err := alloc.checkFunding(conf.CancellationCharge); err != nil {
		return "", common.NewError("allocation_updating_failed", err.Error())
	}

	if request.SetImmutable {
		alloc.IsImmutable = true
	}

	err = alloc.saveUpdatedAllocation(blobbers, balances)
	if err != nil {
		return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

	emitUpdateAllocationBlobberTerms(alloc, balances, t)

	return string(alloc.Encode()), nil
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
				zap.Any("OpenChallenges", ba.Stats.OpenChallenges),
				zap.Any("FailedChallenges", ba.Stats.FailedChallenges),
				zap.Any("SuccessChallenges", ba.Stats.SuccessChallenges))
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
func (sc *StorageSmartContract) canceledPassRates(alloc *StorageAllocation,
	now common.Timestamp, balances chainstate.StateContextI) (
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

			var expire = oc.CreatedAt + toSeconds(getMaxChallengeCompletionTime())
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
// of write pool back to himself. The cancel_allocation doesn't pay min_lock
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

	var passRates []float64
	passRates, err = sc.canceledPassRates(alloc, t.CreationDate, balances)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed",
			"calculating rest challenges success/fail rates: "+err.Error())
	}

	// can cancel
	// new values
	alloc.Expiration = t.CreationDate

	sps := make([]*stakePool, 0, len(alloc.BlobberAllocs))
	for _, d := range alloc.BlobberAllocs {
		var sp *stakePool
		if sp, err = sc.getStakePool(spenum.Blobber, d.BlobberID, balances); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"can't get stake pool of "+d.BlobberID+": "+err.Error())
		}
		if err := sp.reduceOffer(d.Offer()); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"error removing offer: "+err.Error())
		}
		sps = append(sps, sp)
	}

	err = sc.finishAllocation(t, alloc, passRates, sps, balances)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed", err.Error())
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

	// should be expired
	if alloc.Until() > t.CreationDate {
		return "", common.NewError("fini_alloc_failed",
			"allocation is not expired yet, or waiting a challenge completion")
	}

	var passRates []float64
	passRates, err = sc.finalizedPassRates(alloc)
	if err != nil {
		return "", common.NewError("fini_alloc_failed",
			"calculating rest challenges success/fail rates: "+err.Error())
	}

	var sps = []*stakePool{}
	for _, d := range alloc.BlobberAllocs {
		var sp *stakePool
		if sp, err = sc.getStakePool(spenum.Blobber, d.BlobberID, balances); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"can't get stake pool of "+d.BlobberID+": "+err.Error())
		}
		sps = append(sps, sp)
	}

	err = sc.finishAllocation(t, alloc, passRates, sps, balances)
	if err != nil {
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
	passRates []float64,
	sps []*stakePool,
	balances chainstate.StateContextI,
) (err error) {
	// we can use the i for the blobbers list above because of algorithm
	// of the getAllocationBlobbers method; also, we can use the i in the
	// passRates list above because of algorithm of the adjustChallenges
	for i, d := range alloc.BlobberAllocs {
		// min lock demand rest
		if d.MinLockDemand > d.Spent {
			delta, err := currency.MinusCoin(d.MinLockDemand, d.Spent)
			if err != nil {
				return err
			}
			if alloc.WritePool < delta {
				return fmt.Errorf("alloc_cancel_failed, paying min_lock for blobber %v"+
					"ammount was short by %v", d.BlobberID, d.MinLockDemand-d.Spent)
			}
			alloc.WritePool, err = currency.MinusCoin(alloc.WritePool, delta)
			if err != nil {
				return err
			}
			err = sps[i].DistributeRewards(delta, d.BlobberID, spenum.Blobber, balances)
			if err != nil {
				return fmt.Errorf("alloc_cancel_failed, paying min_lock %v for blobber "+
					"%v from write pool %v, minlock demand %v spent %v error %v",
					d.MinLockDemand-d.Spent, d.BlobberID, alloc.WritePool, d.MinLockDemand, d.Spent, err.Error())
			}
			d.Spent, err = currency.AddCoin(d.Spent, delta)
			if err != nil {
				return err
			}
		}
	}

	var blobbers []*StorageNode
	if blobbers, err = sc.getAllocationBlobbers(alloc, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"invalid state: can't get related blobbers: "+err.Error())
	}

	var cp *challengePool
	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"can't get related challenge pool: "+err.Error())
	}

	var passPayments currency.Coin
	for i, d := range alloc.BlobberAllocs {
		var b = blobbers[i]
		if b.ID != d.BlobberID {
			return common.NewErrorf("fini_alloc_failed",
				"blobber %s and %s don't match", b.ID, d.BlobberID)
		}
		if alloc.UsedSize > 0 && cp.Balance > 0 && passRates[i] > 0 && d.Stats != nil {
			ratio := float64(d.Stats.UsedSize) / float64(alloc.UsedSize)
			cpBalance, err := cp.Balance.Float64()
			if err != nil {
				return err
			}

			reward, err := currency.Float64ToCoin(cpBalance * ratio * passRates[i])
			if err != nil {
				return err
			}

			err = sps[i].DistributeRewards(reward, b.ID, spenum.Blobber, balances)
			if err != nil {
				return common.NewError("fini_alloc_failed",
					"paying reward to stake pool of "+d.BlobberID+": "+err.Error())
			}
			d.Spent, err = currency.AddCoin(d.Spent, reward)
			if err != nil {
				return fmt.Errorf("blobber alloc spent: %v", err)
			}
			passPayments, err = currency.AddCoin(passPayments, reward)
			if err != nil {
				return fmt.Errorf("pass payments: %v", err)
			}
		}

		if err = sps[i].save(spenum.Blobber, d.BlobberID, balances); err != nil {
			return common.NewError("fini_alloc_failed",
				"saving stake pool of "+d.BlobberID+": "+err.Error())
		}

		staked, err := sps[i].stake()
		if err != nil {
			return common.NewError("fini_alloc_failed",
				"getting stake of "+d.BlobberID+": "+err.Error())
		}

		tag, data := event.NewUpdateBlobberTotalStakeEvent(d.BlobberID, staked)
		balances.EmitEvent(event.TypeStats, tag, d.BlobberID, data)

		// update the blobber
		b.Allocated -= d.Size
		if _, err = balances.InsertTrieNode(b.GetKey(sc.ID), b); err != nil {
			return common.NewError("fini_alloc_failed",
				"saving blobber "+d.BlobberID+": "+err.Error())
		}
		// update the blobber in all (replace with existing one)
		if err := emitUpdateBlobber(b, balances); err != nil {
			return common.NewError("fini_alloc_failed",
				"emitting blobber "+b.ID+": "+err.Error())
		}
		err = removeAllocationFromBlobber(sc, d, alloc.ID, balances)
		if err != nil {
			return common.NewError("fini_alloc_failed",
				"removing allocation from blobber challenge partition "+b.ID+": "+err.Error())
		}
	}
	cp.Balance, err = currency.MinusCoin(cp.Balance, passPayments)
	if err != nil {
		return err
	}

	if cp.Balance > 0 {
		alloc.MovedBack, err = currency.AddCoin(alloc.MovedBack, cp.Balance)
		if err != nil {
			return err
		}

		err = alloc.moveFromChallengePool(cp, cp.Balance)
		if err != nil {
			return common.NewError("fini_alloc_failed",
				"moving challenge pool rest back to write pool: "+err.Error())
		}
	}

	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return common.NewErrorf("allocation_creation_failed",
			"can't get config: %v", err)
	}

	cancellationCharge, err := alloc.cancellationCharge(conf.CancellationCharge)
	if err != nil {
		return common.NewErrorf("allocation_creation_failed", err.Error())
	}
	if alloc.WritePool < cancellationCharge {
		cancellationCharge = alloc.WritePool
		logging.Logger.Error("insufficient funds, %v, for cancellation charge, %v. distributing the remaining write pool.")
	}
	reward, _, err := currency.DistributeCoin(cancellationCharge, int64(len(alloc.BlobberAllocs)))
	if err != nil {
		return common.NewErrorf("allocation_creation_failed", err.Error())
	}

	for i, ba := range alloc.BlobberAllocs {
		err = sps[i].DistributeRewards(reward, ba.BlobberID, spenum.Blobber, balances)
		if err != nil {
			return common.NewError("fini_alloc_failed",
				"paying cancellation charge to stake pool of "+ba.BlobberID+": "+err.Error())
		}
	}

	if err = cp.save(sc.ID, alloc, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"saving challenge pool: "+err.Error())
	}

	alloc.Finalized = true

	return nil
}

type transferAllocationInput struct {
	AllocationId      string `json:"allocation_id"`
	NewOwnerId        string `json:"new_owner_id"`
	NewOwnerPublicKey string `json:"new_owner_public_key"`
}

func (aci *transferAllocationInput) decode(input []byte) error {
	return json.Unmarshal(input, aci)
}

func (sc *StorageSmartContract) curatorTransferAllocation(
	txn *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (string, error) {
	var tai transferAllocationInput
	if err := tai.decode(input); err != nil {
		return "", common.NewError("curator_transfer_allocation_failed",
			"error unmarshalling input: "+err.Error())
	}

	alloc, err := sc.getAllocation(tai.AllocationId, balances)
	if err != nil {
		return "", common.NewError("curator_transfer_allocation_failed", err.Error())
	}

	if !alloc.isCurator(txn.ClientID) && alloc.Owner != txn.ClientID {
		return "", common.NewError("curator_transfer_allocation_failed",
			"only curators or the owner can transfer allocations; "+txn.ClientID+" is neither")
	}

	alloc.Owner = tai.NewOwnerId
	alloc.OwnerPublicKey = tai.NewOwnerPublicKey

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("curator_transfer_allocation_failed",
			"saving new allocation: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, alloc.ID, alloc.buildDbUpdates())

	// txn.Hash is the id of the new token pool
	return txn.Hash, nil
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
