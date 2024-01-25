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
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

type NewAllocationTxnOutput struct {
	ID          string   `json:"id"`
	Blobber_ids []string `json:"blobber_ids"`
}

func (sn *NewAllocationTxnOutput) Decode(input []byte) error {
	return json.Unmarshal(input, sn)
}

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

	blobberIds := make([]string, 0, len(alloc.BlobberAllocs))
	for _, v := range alloc.BlobberAllocs {
		blobberIds = append(blobberIds, v.BlobberID)
	}

	transactionOutput := NewAllocationTxnOutput{alloc.ID, blobberIds}
	buff, _ := json.Marshal(transactionOutput)
	return string(buff), nil
}

type newAllocationRequest struct {
	Name                 string     `json:"name"`
	DataShards           int        `json:"data_shards"`
	ParityShards         int        `json:"parity_shards"`
	Size                 int64      `json:"size"`
	Owner                string     `json:"owner_id"`
	OwnerPublicKey       string     `json:"owner_public_key"`
	Blobbers             []string   `json:"blobbers"`
	ReadPriceRange       PriceRange `json:"read_price_range"`
	WritePriceRange      PriceRange `json:"write_price_range"`
	ThirdPartyExtendable bool       `json:"third_party_extendable"`
	FileOptionsChanged   bool       `json:"file_options_changed"`
	FileOptions          uint16     `json:"file_options"`
}

// storageAllocation from the request
func (nar *newAllocationRequest) storageAllocation(conf *Config, now common.Timestamp) (sa *StorageAllocation) {
	sa = new(StorageAllocation)
	sa.DataShards = nar.DataShards
	sa.ParityShards = nar.ParityShards
	sa.Size = nar.Size
	sa.Expiration = common.Timestamp(common.ToTime(now).Add(conf.TimeUnit).Unix())
	sa.Owner = nar.Owner
	sa.OwnerPublicKey = nar.OwnerPublicKey
	sa.PreferredBlobbers = nar.Blobbers
	sa.ReadPriceRange = nar.ReadPriceRange
	sa.WritePriceRange = nar.WritePriceRange
	sa.ThirdPartyExtendable = nar.ThirdPartyExtendable
	sa.FileOptions = nar.FileOptions

	return
}

func (nar *newAllocationRequest) validate(conf *Config) error {
	if nar.DataShards <= 0 {
		return errors.New("invalid number of data shards")
	}

	if nar.ParityShards <= 0 {
		return errors.New("invalid number of parity shards")
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
		return "", common.NewErrorf("allocation_creation_failed ",
			"can't get config: %v", err)
	}

	resp, err := sc.newAllocationRequestInternal(t, input, conf, NewTokenTransfer(t.Value, t.ClientID, t.ToClientID, false), balances, timings)
	if err != nil {
		return "", err
	}

	return resp, err
}

// newAllocationRequest creates new allocation
func (sc *StorageSmartContract) newAllocationRequestInternal(
	txn *transaction.Transaction,
	input []byte,
	conf *Config,
	transfer *Transfer,
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
	if err := request.validate(conf); err != nil {
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

	logging.Logger.Debug("new_allocation_request", zap.String("t_hash", txn.Hash), zap.Strings("blobbers", request.Blobbers), zap.Any("amount", txn.Value))
	sa := request.storageAllocation(conf, txn.CreationDate) // (set fields, ignore expiration)
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
		snr := StoragNodeToStorageNodeResponse(*blobbers[i])
		snr.TotalOffers = spMap[blobbers[i].ID].TotalOffers
		snr.TotalStake = stake
		stakedCapacity, err := spMap[blobbers[i].ID].stakedCapacity(blobbers[i].Terms.WritePrice)
		if err != nil {
			return "", common.NewErrorf("allocation_creation_failed", "can not get total staked capacity for blobber %s: %v", blobbers[i].ID, err)
		}
		snr.StakedCapacity = stakedCapacity

		sns = append(sns, &snr)
	}

	sa, blobberNodes, err := setupNewAllocation(request, sns, m, txn.CreationDate, conf, txn.Hash)
	if err != nil {
		return "", err
	}

	for _, b := range blobberNodes {
		_, err = balances.InsertTrieNode(b.GetKey(), b)
		if err != nil {
			logging.Logger.Error("new_allocation_request_failed: error inserting blobber",
				zap.String("txn", txn.Hash),
				zap.String("blobber", b.ID),
				zap.Error(err))
			return "", fmt.Errorf("can't Save blobber: %v", err)
		}

		if err := spMap[b.ID].addOffer(sa.BlobberAllocsMap[b.ID].Offer()); err != nil {
			logging.Logger.Error("new_allocation_request_failed: error adding offer to blobber",
				zap.String("txn", txn.Hash),
				zap.String("blobber", b.ID),
				zap.Error(err))
			return "", fmt.Errorf("ading offer: %v", err)
		}

		if err = spMap[b.ID].Save(spenum.Blobber, b.ID, balances); err != nil {
			logging.Logger.Error("new_allocation_request_failed: error saving blobber pool",
				zap.String("txn", txn.Hash),
				zap.String("blobber", b.ID),
				zap.Error(err))
			return "", fmt.Errorf("can't Save blobber's stake pool: %v", err)
		}

		emitUpdateBlobberAllocatedSavedHealth(b, balances)
	}

	// create write pool and lock tokens
	if err := sa.addToWritePool(txn, balances, transfer); err != nil {
		logging.Logger.Error("new_allocation_request_failed: error adding to allocation write pool",
			zap.String("txn", txn.Hash),
			zap.Error(err))
		return "", common.NewError("allocation_creation_failed", err.Error())
	}

	if err := sa.checkFunding(); err != nil {
		return "", common.NewError("allocation_creation_failed", err.Error())
	}
	m.tick("create_write_pool")

	if err = sc.createChallengePool(txn, sa, balances, conf); err != nil {
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
	sa := request.storageAllocation(conf, now) // (set fields, ignore expiration)
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
		bAlloc := newBlobberAllocation(bSize, sa, b, conf, now)
		sa.BlobberAllocs = append(sa.BlobberAllocs, bAlloc)
		sa.BlobberAllocsMap[b.ID] = bAlloc
		b.Allocated += bSize
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
	var list, errs = sa.validateEachBlobber(blobbers, common.Timestamp(creationDate.Unix()), conf)

	if len(list) < size {
		return nil, 0, errors.New("Not enough blobbers to honor the allocation: " + strings.Join(errs, ", "))
	}

	sa.BlobberAllocs = make([]*BlobberAllocation, 0)
	sa.Stats = &StorageAllocationStats{}

	return list[:size], bSize, nil
}

type updateAllocationRequest struct {
	ID                      string `json:"id"`               // allocation id
	Name                    string `json:"name"`             // allocation name
	OwnerID                 string `json:"owner_id"`         // Owner of the allocation
	OwnerPublicKey          string `json:"owner_public_key"` // Owner Public Key of the allocation
	Size                    int64  `json:"size"`             // difference
	Extend                  bool   `json:"extend"`
	AddBlobberId            string `json:"add_blobber_id"`
	RemoveBlobberId         string `json:"remove_blobber_id"`
	SetThirdPartyExtendable bool   `json:"set_third_party_extendable"`
	FileOptionsChanged      bool   `json:"file_options_changed"`
	FileOptions             uint16 `json:"file_options"`
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
		uar.Extend == false &&
		len(uar.AddBlobberId) == 0 &&
		len(uar.Name) == 0 &&
		(!uar.SetThirdPartyExtendable || (uar.SetThirdPartyExtendable && alloc.ThirdPartyExtendable)) &&
		(!uar.FileOptionsChanged || uar.FileOptions == alloc.FileOptions) &&
		(alloc.Owner == uar.OwnerID) {
		return errors.New("update allocation changes nothing")
	} else {
		if uar.Size < 0 {
			return fmt.Errorf("allocation can't be reduced")
		}
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

func (sa *StorageAllocation) saveUpdatedAllocation(
	blobbers []*StorageNode,
	balances chainstate.StateContextI,
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
	return
}

// The adjustChallengePool moves more or moves some tokens back from or to
// challenge pool during allocation extending.
func (sc *StorageSmartContract) adjustChallengePool(
	alloc *StorageAllocation,
	odr, ndr common.Timestamp,
	oterms []Terms,
	timeUnit time.Duration,
	balances chainstate.StateContextI,
) error {
	if alloc.Stats.UsedSize == 0 {
		return nil // no written data
	}

	changes, err := alloc.challengePoolChanges(odr, ndr, timeUnit, oterms)
	if err != nil {
		return fmt.Errorf("adjust_challenge_pool: %v", err)
	}

	cp, err := sc.getChallengePool(alloc.ID, balances)
	if err != nil {
		return fmt.Errorf("adjust_challenge_pool: %v", err)
	}

	totalChanges := 0

	addedToCP, removedFromCP := currency.Coin(0), currency.Coin(0)
	for i, ch := range changes {
		changeValueInInt64, err := ch.Value.Int64()
		if err != nil {
			return err
		}

		switch {
		case !ch.isNegative && ch.Value > 0:
			err = alloc.moveToChallengePool(cp, ch.Value)
			addedToCP += ch.Value

			alloc.BlobberAllocs[i].ChallengePoolIntegralValue += ch.Value
			alloc.MovedToChallenge += ch.Value
			totalChanges += int(changeValueInInt64)
		case ch.isNegative && ch.Value > 0:
			err = alloc.moveFromChallengePool(cp, ch.Value)
			removedFromCP += ch.Value

			alloc.BlobberAllocs[i].ChallengePoolIntegralValue -= ch.Value
			alloc.MovedBack += ch.Value
			totalChanges -= int(changeValueInInt64)
		default:
			// no changes for the blobber
		}
		if err != nil {
			return fmt.Errorf("adjust_challenge_pool: %v", err)
		}
	}

	if totalChanges > 0 {
		err = cp.save(sc.ID, alloc, balances)
		if err != nil {
			return err
		}

		i := int64(0)
		i, err = addedToCP.Int64()
		if err != nil {
			return err
		}
		balances.EmitEvent(event.TypeStats, event.TagToChallengePool, cp.ID, event.ChallengePoolLock{
			Client:       alloc.Owner,
			AllocationId: alloc.ID,
			Amount:       i,
		})
	} else if totalChanges < 0 {
		err = cp.save(sc.ID, alloc, balances)
		if err != nil {
			return err
		}

		i := int64(0)
		i, err = removedFromCP.Int64()
		if err != nil {
			return err
		}
		balances.EmitEvent(event.TypeStats, event.TagFromChallengePool, cp.ID, event.ChallengePoolLock{
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
	req *updateAllocationRequest,
	balances chainstate.StateContextI,
) (err error) {
	var (
		diff = req.getBlobbersSizeDiff(alloc) // size difference
		size = req.getNewBlobbersSize(alloc)  // blobber size

		// keep original terms to adjust challenge pool value
		originalTerms = make([]Terms, 0, len(alloc.BlobberAllocs))
		// original allocation duration remains
		originalRemainingDuration = alloc.Expiration - txn.CreationDate
	)

	alloc.Expiration = common.Timestamp(common.ToTime(txn.CreationDate).Add(conf.TimeUnit).Unix()) // new expiration

	alloc.Size += req.Size // new size

	// 1. update terms
	for i, details := range alloc.BlobberAllocs {
		var sp *stakePool
		if sp, err = sc.getStakePool(spenum.Blobber, details.BlobberID, balances); err != nil {
			return fmt.Errorf("can't get stake pool of %s: %v", details.BlobberID, err)
		}

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
			err = nil
			chainstate.WithActivation(balances, "hard_fork_1", func() {
				if b.IsShutDown() || b.IsKilled() {
					err = common.NewErrorf("allocation_extending_failed",
						"blobber %s is not active", b.ID)
				}
			}, func() {
				if b.IsKilled() {
					err = common.NewErrorf("allocation",
						"blobber %s is not active", b.ID)
				}
			})
			if err != nil {
				return err
			}

			stakedCapacity, err := sp.stakedCapacity(b.Terms.WritePrice)
			if err != nil {
				return common.NewErrorf("allocation_extending_failed",
					"can't get staked capacity: %v", err)
			}

			if b.Capacity-b.Allocated-diff < 0 || stakedCapacity-b.Allocated-diff < 0 {
				return common.NewErrorf("allocation_extending_failed",
					"blobber %s doesn't have enough free space", b.ID)
			}
		}

		b.Allocated += diff // new capacity used

		// update terms using weighted average
		setCappedPrices(details, b, conf)

		if err != nil {
			return err
		}

		details.Size = size // new size

		// update blobber's offer
		newOffer := details.Offer()
		if newOffer != oldOffer {
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
					return fmt.Errorf("reduce offer: %v", err)
				}
			}
			if err = sp.Save(spenum.Blobber, details.BlobberID, balances); err != nil {
				return fmt.Errorf("can't save stake pool of %s: %v", details.BlobberID,
					err)
			}

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

	// Always extend if size is increased
	if request.Size > 0 {
		request.Extend = true
	}

	if request.OwnerID == "" {
		request.OwnerID = t.ClientID
	}

	var alloc *StorageAllocation
	if alloc, err = sc.getAllocation(request.ID, balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get existing allocation: "+err.Error())
	}
	if err != nil {
		return "", err
	}

	if t.ClientID != alloc.Owner {
		if !alloc.ThirdPartyExtendable || !request.Extend {
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
	if blobbers, err = sc.getAllocationBlobbers(alloc, balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			err.Error())
	}

	// If the txn client_id is not the owner of the allocation, should just be able to extend the allocation if permissible
	// This way, even if an atttacker of an innocent user incorrectly tries to modify any other part of the allocation, it will not have any effect
	if t.ClientID != alloc.Owner /* Third-party actions */ {
		err = sc.extendAllocation(t, conf, alloc, blobbers, &request, balances)
		if err != nil {
			return "", err
		}
	} else /* Owner Actions */ {

		// update allocation transaction hash
		alloc.Tx = t.Hash

		if len(request.AddBlobberId) > 0 {
			blobbers, err = alloc.changeBlobbers(
				conf, blobbers, request.AddBlobberId, request.RemoveBlobberId, t.CreationDate, balances, sc, t.ClientID,
			)
			if err != nil {
				return "", common.NewError("allocation_updating_failed", err.Error())
			}
		}

		if len(blobbers) != len(alloc.BlobberAllocs) {
			return "", common.NewError("allocation_updating_failed",
				"error allocation blobber size mismatch")
		}

		// if size or expiration increased, then we use new terms
		// otherwise, we use the same terms
		if request.Extend {
			err = sc.extendAllocation(t, conf, alloc, blobbers, &request, balances)
			if err != nil {
				return "", err
			}
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
					AllocationIdHash: alloc.ID,
					BlobberID:        request.RemoveBlobberId,
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

	cp, err := sc.getChallengePool(alloc.ID, balances)
	if err != nil {
		return "", common.NewError("allocation_updating_failed", err.Error())
	}

	tokensRequiredToLock, err := alloc.requiredTokensForUpdateAllocation(cp.Balance, request.Extend, t.CreationDate)
	if err != nil {
		return "", common.NewError("allocation_updating_failed", err.Error())
	}

	if t.Value < tokensRequiredToLock {
		return "", common.NewError("allocation_updating_failed",
			fmt.Sprintf("not enough tokens to cover update allocation cost (locked : %d < required : %d)", t.Value, tokensRequiredToLock))
	}

	// lock tokens if this transaction provides them
	if t.Value > 0 {
		if err = alloc.addToWritePool(t, balances, NewTokenTransfer(t.Value, t.ClientID, t.ToClientID, false)); err != nil {
			return "", common.NewError("allocation_updating_failed", err.Error())
		}
	}

	err = alloc.saveUpdatedAllocation(blobbers, balances)
	if err != nil {
		return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

	emitAddOrOverwriteAllocationBlobberTerms(alloc, balances, t)

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

// a blobber can not send a challenge response, thus we have to check out
// challenge requests and their expiration
func (sc *StorageSmartContract) settleOpenChallengesAndGetPassRates(
	alloc *StorageAllocation,
	now,
	maxChallengeCompletionRounds int64,
	balances chainstate.StateContextI,
) (
	passRates []float64, err error) {

	if alloc.Stats == nil {
		alloc.Stats = &StorageAllocationStats{}
	}
	passRates = make([]float64, 0, len(alloc.BlobberAllocs))

	var removedChallengeIds []string
	allocChallenges, err := sc.getAllocationChallenges(alloc.ID, balances)
	switch err {
	case util.ErrValueNotPresent:
		for i := 0; i < len(alloc.BlobberAllocs); i++ {
			passRates = append(passRates, 1.0)
		}
		return passRates, nil
	case util.ErrNodeNotFound:
		return nil, err
	case nil:
		for _, oc := range allocChallenges.OpenChallenges {
			ba, ok := alloc.BlobberAllocsMap[oc.BlobberID]
			if !ok {
				continue
			}

			if ba.Stats == nil {
				ba.Stats = new(StorageAllocationStats) // make sure
			}

			var expire = oc.RoundCreatedAt + maxChallengeCompletionRounds

			ba.Stats.OpenChallenges--
			alloc.Stats.OpenChallenges--

			if expire < now {
				ba.Stats.FailedChallenges++
				alloc.Stats.FailedChallenges++

				err := emitUpdateChallenge(&StorageChallenge{
					ID:           oc.ID,
					AllocationID: alloc.ID,
					BlobberID:    oc.BlobberID,
				}, false, ChallengeRespondedLate, balances, alloc.Stats)
				if err != nil {
					return nil, err
				}

			} else {
				ba.Stats.SuccessChallenges++
				alloc.Stats.SuccessChallenges++

				err := emitUpdateChallenge(&StorageChallenge{
					ID:           oc.ID,
					AllocationID: alloc.ID,
					BlobberID:    oc.BlobberID,
				}, true, ChallengeResponded, balances, alloc.Stats)
				if err != nil {
					return nil, err
				}
			}

			removedChallengeIds = append(removedChallengeIds, oc.ID)
		}
	default:
		return nil, common.NewError("finish_allocation",
			"error fetching allocation challenge: "+err.Error())
	}

	allocChallenges.OpenChallenges = make([]*AllocOpenChallenge, 0)

	// Save the allocation challenges to MPT
	if err := allocChallenges.Save(balances, sc.ID); err != nil {
		return nil, common.NewErrorf("add_challenge",
			"error storing alloc challenge: %v", err)
	}

	for _, challengeID := range removedChallengeIds {
		_, err := balances.DeleteTrieNode(storageChallengeKey(sc.ID, challengeID))
		if err != nil {
			return nil, common.NewErrorf("remove_expired_challenges", "could not delete challenge node: %v", err)
		}
	}

	var blobbersSettledChallengesCount []int64

	for idx, ba := range alloc.BlobberAllocs {
		blobbersSettledChallengesCount = append(blobbersSettledChallengesCount, 0)
		if ba.Stats.OpenChallenges > 0 {
			logging.Logger.Warn("not all challenges canceled", zap.Int64("remaining", ba.Stats.OpenChallenges))

			blobbersSettledChallengesCount[idx] = ba.Stats.OpenChallenges

			ba.Stats.SuccessChallenges += ba.Stats.OpenChallenges
			alloc.Stats.SuccessChallenges += ba.Stats.OpenChallenges

			ba.Stats.OpenChallenges = 0
		}

		if ba.Stats.TotalChallenges == 0 {
			passRates = append(passRates, 1.0)
			continue
		}
		// success rate for the blobber allocation
		passRates = append(passRates, float64(ba.Stats.SuccessChallenges)/float64(ba.Stats.TotalChallenges))
	}

	alloc.Stats.OpenChallenges = 0

	emitUpdateAllocationAndBlobberStatsOnAllocFinalization(alloc, blobbersSettledChallengesCount, balances)

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
	passRates, err = sc.settleOpenChallengesAndGetPassRates(alloc, balances.GetBlock().Round, conf.MaxChallengeCompletionRounds, balances)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed",
			"calculating rest challenges success/fail rates: "+err.Error())
	}

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

	err = sc.finishAllocation(t, alloc, passRates, sps, balances, conf)
	if err != nil {
		return "", common.NewError("alloc_cancel_failed", err.Error())
	}

	alloc.Expiration = t.CreationDate
	alloc.Finalized, alloc.Canceled = true, true

	_, err = balances.DeleteTrieNode(alloc.GetKey(sc.ID))
	if err != nil {
		return "", common.NewErrorf("alloc_cancel_failed", "could not delete allocation: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, alloc.ID, alloc.buildDbUpdates())

	return "canceled", nil
}

//
// finalize an allocation (after expire + challenge completion time)
//

// 1. challenge pool                  -> blobbers or write pool
// 2. remove offer from blobber (stake pool)
// 3. update blobbers used and in all blobbers list too
// 4. write pool                      -> client
func (sc *StorageSmartContract) finalizeAllocation(
	t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (resp string, err error) {

	alloc, err := sc.finalizeAllocationInternal(t, input, balances)
	if err != nil {
		return "", err
	}

	_, err = balances.DeleteTrieNode(alloc.GetKey(sc.ID))
	if err != nil {
		return "", common.NewErrorf("fini_alloc_failed", "could not delete allocation: %v", err)
	}

	return "finalized", nil
}

// finalizeAllocationInternal finalize allocation without deleting it, which
// could be used in unit test to verify the challenges pass rate, rewards, etc.
func (sc *StorageSmartContract) finalizeAllocationInternal(
	t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (*StorageAllocation, error) {
	var (
		req lockRequest
		err error
	)
	if err = req.decode(input); err != nil {
		return nil, common.NewError("fini_alloc_failed", err.Error())
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(req.AllocationID, balances)
	if err != nil {
		return nil, common.NewError("fini_alloc_failed", err.Error())
	}

	// should be owner or one of blobbers of the allocation
	if !alloc.IsValidFinalizer(t.ClientID) {
		return nil, common.NewError("fini_alloc_failed",
			"not allowed, unknown finalization initiator")
	}

	// should not be finalized
	if alloc.Finalized {
		return nil, common.NewError("fini_alloc_failed",
			"allocation already finalized")
	}

	conf, err := getConfig(balances)
	if err != nil {
		return nil, common.NewError("can't get config", err.Error())
	}

	// should be expired
	if alloc.Expiration > t.CreationDate {
		return nil, common.NewError("fini_alloc_failed",
			"allocation is not expired yet")
	}

	var passRates []float64
	passRates, err = sc.settleOpenChallengesAndGetPassRates(alloc, balances.GetBlock().Round, conf.MaxChallengeCompletionRounds, balances)
	if err != nil {
		return nil, common.NewError("fini_alloc_failed",
			"calculating rest challenges success/fail rates: "+err.Error())
	}

	var sps []*stakePool
	for _, d := range alloc.BlobberAllocs {
		var sp *stakePool
		if sp, err = sc.getStakePool(spenum.Blobber, d.BlobberID, balances); err != nil {
			return nil, common.NewError("fini_alloc_failed",
				"can't get stake pool of "+d.BlobberID+": "+err.Error())
		}
		if err := sp.reduceOffer(d.Offer()); err != nil {
			return nil, common.NewError("fini_alloc_failed",
				"error removing offer: "+err.Error())
		}
		sps = append(sps, sp)
	}

	err = sc.finishAllocation(t, alloc, passRates, sps, balances, conf)
	if err != nil {
		return nil, common.NewError("fini_alloc_failed", err.Error())
	}

	alloc.Finalized = true
	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, alloc.ID, alloc.buildDbUpdates())

	return alloc, nil
}

func (sc *StorageSmartContract) finishAllocation(
	t *transaction.Transaction,
	alloc *StorageAllocation,
	passRates []float64,
	sps []*stakePool,
	balances chainstate.StateContextI,
	conf *Config,
) (err error) {

	var cp *challengePool
	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("could not get challenge pool of alloc: %s, err: %v", alloc.ID, err)
	}

	if err = alloc.payChallengePoolPassPayments(sps, balances, cp, passRates, conf, sc, t.CreationDate); err != nil {
		return fmt.Errorf("error paying challenge pool pass payments: %v", err)
	}

	if err = alloc.payCancellationCharge(sps, balances, passRates, conf, sc, t); err != nil {
		return fmt.Errorf("4 error paying cancellation charge: %v", err)
	}

	for _, d := range alloc.BlobberAllocs {
		if d.Stats.UsedSize > 0 {
			if err := removeAllocationFromBlobberPartitions(balances, d.BlobberID, d.AllocationID); err != nil {
				return err
			}
		}

		blobber, err := sc.getBlobber(d.BlobberID, balances)
		if err != nil {
			return common.NewError("fini_alloc_failed",
				"can't get blobber "+d.BlobberID+": "+err.Error())
		}
		blobber.SavedData += -d.Stats.UsedSize
		blobber.Allocated += -d.Size
		_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
		if err != nil {
			return common.NewError("fini_alloc_failed",
				"saving blobber "+d.BlobberID+": "+err.Error())
		}

		// Update saved data on events_db
		emitUpdateBlobberAllocatedSavedHealth(blobber, balances)
	}

	for i, sp := range sps {
		blobberAlloc := alloc.BlobberAllocs[i]
		if err = sp.Save(spenum.Blobber, blobberAlloc.BlobberID, balances); err != nil {
			return fmt.Errorf("can't save stake pool of %s: %v", blobberAlloc.BlobberID, err)
		}
	}

	err = sc.deleteChallengePool(alloc, balances)
	if err != nil {
		return fmt.Errorf("could not delete challenge pool of alloc: %s, err: %v", alloc.ID, err)
	}

	transfer := state.NewTransfer(sc.ID, alloc.Owner, alloc.WritePool)
	if err = balances.AddTransfer(transfer); err != nil {
		return fmt.Errorf("could not refund lock token: %v", err)
	}

	alloc.WritePool = 0
	return nil
}

func emitUpdateAllocationStatEvent(allocation *StorageAllocation, balances chainstate.StateContextI) {

	alloc := event.Allocation{
		AllocationID:     allocation.ID,
		UsedSize:         allocation.Stats.UsedSize,
		NumWrites:        allocation.Stats.NumWrites,
		MovedToChallenge: allocation.MovedToChallenge,
		MovedBack:        allocation.MovedBack,
		WritePool:        allocation.WritePool,
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationStat, alloc.AllocationID, &alloc)
}
