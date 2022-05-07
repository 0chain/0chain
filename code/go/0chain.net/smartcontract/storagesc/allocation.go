package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"go.uber.org/zap"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
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

func (sc *StorageSmartContract) getAllocationsList(clientID string,
	balances chainstate.StateContextI) (*Allocations, error) {

	allocationList := &Allocations{}
	var clientAlloc ClientAllocation
	clientAlloc.ClientID = clientID
	err := balances.GetTrieNode(clientAlloc.GetKey(sc.ID), &clientAlloc)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}

		return allocationList, nil
	}

	return clientAlloc.Allocations, nil
}

func (sc *StorageSmartContract) removeUserAllocation(
	oldUser string,
	alloc *StorageAllocation,
	balances chainstate.StateContextI,
) error {
	var err error

	clientAllocation := &ClientAllocation{}
	clientAllocation.ClientID = oldUser
	clientAllocation.Allocations, err = sc.getAllocationsList(alloc.Owner, balances)
	if err != nil {
		return fmt.Errorf("Failed to get allocation list: %v", err)
	}

	ok := clientAllocation.Allocations.List.remove(alloc.ID)
	if !ok {
		return fmt.Errorf("failed to remove allocation %s from client %s list", alloc.ID, oldUser)
	}

	_, err = balances.InsertTrieNode(clientAllocation.GetKey(sc.ID), clientAllocation)
	if err != nil {
		return fmt.Errorf("saving client allocations list (client: %s): %v", oldUser, err)
	}

	return nil
}

func (sc *StorageSmartContract) addUserAllocation(
	newUser string,
	alloc *StorageAllocation,
	balances chainstate.StateContextI,
) error {
	var err error

	clientAllocation := &ClientAllocation{}
	clientAllocation.ClientID = newUser
	clientAllocation.Allocations, err = sc.getAllocationsList(alloc.Owner, balances)
	if err != nil {
		return fmt.Errorf("Failed to get allocation list: %v", err)
	}

	ok := clientAllocation.Allocations.List.add(alloc.ID)
	if !ok {
		return fmt.Errorf("failed to add allocation %s to client %s list", alloc.ID, newUser)
	}

	r, err := balances.InsertTrieNode(clientAllocation.GetKey(sc.ID), clientAllocation)
	if err != nil {
		return fmt.Errorf("saving client allocations list (client: %s): %v", newUser, err)
	}
	logging.Logger.Debug("after client allocation save", zap.String("root", r))

	return nil
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

	if err := sc.addUserAllocation(alloc.Owner, alloc, balances); err != nil {
		return "", common.NewError("add_allocation_failed", err.Error())
	}

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("add_allocation_failed",
			"saving new allocation: %v", err)
	}

	err = emitAddOrOverwriteAllocation(alloc, balances)
	if err != nil {
		return "", common.NewErrorf("add_allocation_failed",
			"saving new allocation in db: %v", err)
	}

	buff := alloc.Encode()
	return string(buff), nil
}

type newAllocationRequest struct {
	Name                       string           `json:"name"`
	DataShards                 int              `json:"data_shards"`
	ParityShards               int              `json:"parity_shards"`
	Size                       int64            `json:"size"`
	Expiration                 common.Timestamp `json:"expiration_date"`
	Owner                      string           `json:"owner_id"`
	OwnerPublicKey             string           `json:"owner_public_key"`
	Blobbers                   []string         `json:"blobbers"`
	ReadPriceRange             PriceRange       `json:"read_price_range"`
	WritePriceRange            PriceRange       `json:"write_price_range"`
	MaxChallengeCompletionTime time.Duration    `json:"max_challenge_completion_time"`
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
	sa.WritePoolOwners = append(sa.WritePoolOwners, nar.Owner)
	sa.PreferredBlobbers = nar.Blobbers
	sa.ReadPriceRange = nar.ReadPriceRange
	sa.WritePriceRange = nar.WritePriceRange
	sa.MaxChallengeCompletionTime = nar.MaxChallengeCompletionTime
	return
}

func (nar *newAllocationRequest) validate(conf *Config) error {
	// TODO: uncomment this when we start to validate max challenge completion time
	//if nar.MaxChallengeCompletionTime > conf.MaxChallengeCompletionTime {
	//	return errors.New("max challenge completion time exceeded")
	//}

	return nil
}

func (nar *newAllocationRequest) decode(b []byte) error {
	return json.Unmarshal(b, nar)
}

func (nar *newAllocationRequest) encode() ([]byte, error) {
	return json.Marshal(nar)
}

/*
// (1) adjust blobber capacity used, (2) add offer (stake lock boundary),
// (3) save updated blobber
func (sc *StorageSmartContract) addBlobbersOffers(sa *StorageAllocation,
	blobbers []*StorageNode, balances chainstate.StateContextI) (err error) {

	// update blobbers' stakes and capacity used
	for i, b := range blobbers {
		b.Used += sa.BlobberAllocs[i].Size // adjust used size
		var sp *stakePool
		if sp, err = sc.getStakePool(b.ID, balances); err != nil {
			return fmt.Errorf("can't get blobber's stake pool: %v", err)
		}
		sp.addOffer(sa, sa.BlobberAllocs[i])

		// save blobber
		if _, err = balances.InsertTrieNode(b.GetKey(sc.ID), b); err != nil {
			return fmt.Errorf("can't save blobber: %v", err)
		}
		// save its stake pool
		if err = sp.save(sc.ID, b.ID, balances); err != nil {
			return fmt.Errorf("can't save blobber's stake pool: %v", err)
		}
	}

	return
}
*/
// update blobbers list in the all blobbers list
func updateBlobbersInAll(all *StorageNodes, update []*StorageNode,
	balances chainstate.StateContextI) (err error) {

	// update the blobbers in all blobbers list
	for _, b := range update {
		all.Nodes.update(b)
		// don't replace if blobber has removed from the all blobbers list;
		// for example, if the blobber has removed, then it shouldn't be
		// in the all blobbers list
		if err := emitUpdateBlobber(b, balances); err != nil {
			return fmt.Errorf("emmiting blobber %v: %v", b, err)
		}
	}

	// save
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, all)
	if err != nil {
		return fmt.Errorf("can't save all blobber list: %v", err)
	}

	return
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
	size int64, balances chainstate.StateContextI) (filter filterBlobberFunc) {

	return filterBlobberFunc(func(b *StorageNode) (kick bool) {
		var sp, err = sc.getStakePool(b.ID, balances)
		if err != nil {
			return true // kick off
		}
		if b.Terms.WritePrice == 0 {
			return false // keep, ok or already filtered by bid
		}
		// clean capacity (without delegate pools want to 'unstake')
		var free = sp.cleanCapacity(now, b.Terms.WritePrice)
		return free < size // kick off if it hasn't enough free space
	})
}

// newAllocationRequest creates new allocation
func (sc *StorageSmartContract) newAllocationRequest(
	t *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (string, error) {
	var conf *Config
	var err error
	if conf, err = sc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("allocation_creation_failed",
			"can't get config: %v", err)
	}

	resp, err := sc.newAllocationRequestInternal(t, input, conf, false, balances)
	if err != nil {
		return "", err
	}

	return resp, err
}

type blobberWithPool struct {
	*StorageNode
	Pool *stakePool
}

// newAllocationRequest creates new allocation
func (sc *StorageSmartContract) newAllocationRequestInternal(
	t *transaction.Transaction,
	input []byte,
	conf *Config,
	mintNewTokens bool,
	balances chainstate.StateContextI,
) (resp string, err error) {
	if err != nil {
		return "", common.NewErrorf("allocation_creation_failed",
			"getting blobber list: %v", err)
	}

	if t.ClientID == "" {
		return "", common.NewError("allocation_creation_failed",
			"Invalid client in the transaction. No client id in transaction")
	}

	var request newAllocationRequest
	logging.Logger.Debug("new_allocation_request", zap.String("request", string(input)))
	if err = request.decode(input); err != nil {
		return "", common.NewErrorf("allocation_creation_failed",
			"malformed request: %v", err)
	}

	if len(request.Blobbers) < (request.DataShards + request.ParityShards) {
		return "", common.NewErrorf("allocation_creation_failed",
			"Blobbers provided are not enough to honour the allocation")
	}

	//if more than limit blobbers sent, just cut them
	if len(request.Blobbers) > conf.MaxBlobbersPerAllocation {
		logging.Logger.Info("Too many blobbers selected, max available", zap.Int("max_blobber_size", conf.MaxBlobbersPerAllocation))
		request.Blobbers = request.Blobbers[:conf.MaxBlobbersPerAllocation]
	}

	inputBlobbers := sc.getBlobbers(request.Blobbers, balances)
	if len(inputBlobbers.Nodes) < (request.DataShards + request.ParityShards) {
		return "", common.NewErrorf("allocation_creation_failed",
			"Not enough provided blobbers found in mpt")
	}

	if request.Owner == "" {
		request.Owner = t.ClientID
		request.OwnerPublicKey = t.PublicKey
	}

	logging.Logger.Debug("new_allocation_request", zap.String("t_hash", t.Hash), zap.Strings("blobbers", request.Blobbers))
	var sa = request.storageAllocation() // (set fields, including expiration)
	blobbers, err := sc.fetchPools(inputBlobbers, balances)
	if err != nil {
		return "", err
	}
	blobberNodes, bSize, err := sc.validateBlobbers(common.ToTime(t.CreationDate), sa, balances, blobbers)
	bi := make([]string, 0, len(blobberNodes))
	for _, b := range blobberNodes {
		bi = append(bi, b.ID)
	}
	logging.Logger.Debug("new_allocation_request", zap.Int64("size", bSize), zap.Strings("blobbers", bi))

	if err != nil {
		return "", common.NewErrorf("allocation_creation_failed", "%v", err)
	}

	sa.ID = t.Hash
	for _, b := range blobberNodes {
		balloc := newBlobberAllocation(bSize, sa, b.StorageNode, t.CreationDate)
		sa.BlobberAllocs = append(sa.BlobberAllocs, balloc)

		if b.Terms.ChallengeCompletionTime > sa.ChallengeCompletionTime {
			sa.ChallengeCompletionTime = b.Terms.ChallengeCompletionTime
		}

		b.Used += bSize
		_, err := balances.InsertTrieNode(b.GetKey(sc.ID), b)
		if err != nil {
			return "", fmt.Errorf("can't save blobber: %v", err)
		}

		if err := b.Pool.addOffer(balloc.Offer()); err != nil {
			return "", fmt.Errorf("ading offer: %v", err)
		}
		if err = b.Pool.save(sc.ID, b.ID, balances); err != nil {
			return "", fmt.Errorf("can't save blobber's stake pool: %v", err)
		}
	}

	sa.StartTime = t.CreationDate
	sa.Tx = t.Hash

	// create write pool and lock tokens
	if err = sc.createWritePool(t, sa, mintNewTokens, balances); err != nil {
		return "", common.NewError("allocation_creation_failed", err.Error())
	}

	if err = sc.createChallengePool(t, sa, balances); err != nil {
		return "", common.NewError("allocation_creation_failed", err.Error())
	}

	if resp, err = sc.addAllocation(sa, balances); err != nil {
		return "", common.NewErrorf("allocation_creation_failed", "%v", err)
	}

	return resp, err
}

func (sc *StorageSmartContract) fetchPools(inputBlobbers *StorageNodes, balances chainstate.StateContextI) ([]*blobberWithPool, error) {
	blobbers := make([]*blobberWithPool, 0, len(inputBlobbers.Nodes))
	pools := make(chan *blobberWithPool, len(inputBlobbers.Nodes))
	errs := make(chan error, len(inputBlobbers.Nodes))

	for _, b := range inputBlobbers.Nodes {
		go func(blob *StorageNode) {
			var sp *stakePool
			var err error
			if sp, err = sc.getStakePool(blob.ID, balances); err != nil {
				errs <- common.NewErrorf("allocation_creation_failed", "can't get blobber's stake pool: %v", err)
				return
			}
			pools <- &blobberWithPool{blob, sp}
		}(b)
	}

	for {
		if len(blobbers) == len(inputBlobbers.Nodes) {
			return blobbers, nil
		}

		select {
		case err := <-errs:
			return nil, err
		case p := <-pools:
			blobbers = append(blobbers, p)
		}
	}
}

func (sc *StorageSmartContract) selectBlobbers(
	creationDate time.Time,
	allBlobbersList StorageNodes,
	sa *StorageAllocation,
	randomSeed int64,
	balances chainstate.StateContextI,
) ([]*StorageNode, int64, error) {
	var err error
	var conf *Config
	if conf, err = sc.getConfig(balances, true); err != nil {
		return nil, 0, fmt.Errorf("can't get config: %v", err)
	}

	sa.TimeUnit = conf.TimeUnit // keep the initial time unit

	// number of blobbers required
	var size = sa.DataShards + sa.ParityShards
	// size of allocation for a blobber
	var bSize = sa.bSize()
	timestamp := common.Timestamp(creationDate.Unix())
	var list = sa.filterBlobbers(allBlobbersList.Nodes.copy(), timestamp,
		bSize, filterHealthyBlobbers(timestamp),
		sc.filterBlobbersByFreeSpace(timestamp, bSize, balances))

	if len(list) < size {
		return nil, 0, errors.New("Not enough blobbers to honor the allocation")
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

// getBlobbers get blobbers from MPT concurrently based on input blobber ids (TODO: We need to remove as much pointers as much to reduce load on garbage collector, this function was made to keep things simple and backward code compatible)
func (sc *StorageSmartContract) getBlobbers(blobberIDs []string,
	balances chainstate.StateContextI) *StorageNodes {
	blobbers := sc.getBlobbersByIDs(blobberIDs, balances)
	return &StorageNodes{
		Nodes: blobbers,
	}
}

func (sc *StorageSmartContract) validateBlobbers(
	creationDate time.Time,
	sa *StorageAllocation,
	balances chainstate.StateContextI,
	blobbers []*blobberWithPool,
) ([]*blobberWithPool, int64, error) {
	var err error
	var conf *Config
	if conf, err = sc.getConfig(balances, true); err != nil {
		return nil, 0, fmt.Errorf("can't get config: %v", err)
	}

	sa.TimeUnit = conf.TimeUnit // keep the initial time unit

	if err = sa.validate(creationDate, conf); err != nil {
		return nil, 0, fmt.Errorf("invalid request: %v", err)
	}

	// number of blobbers required
	var size = sa.DataShards + sa.ParityShards
	// size of allocation for a blobber
	var bSize = sa.bSize()
	var list, errs = sa.validateEachBlobber(sc, blobbers, common.Timestamp(creationDate.Unix()),
		balances)

	if len(list) < size {
		return nil, 0, errors.New("Not enough blobbers to honor the allocation: " + strings.Join(errs, ", "))
	}

	sa.BlobberDetails = make([]*BlobberAllocation, 0)
	sa.Stats = &StorageAllocationStats{}

	return list[:size], bSize, nil
}

type updateAllocationRequest struct {
	ID              string           `json:"id"`              // allocation id
	Name            string           `json:"name"`            // allocation name
	OwnerID         string           `json:"owner_id"`        // Owner of the allocation
	Size            int64            `json:"size"`            // difference
	Expiration      common.Timestamp `json:"expiration_date"` // difference
	SetImmutable    bool             `json:"set_immutable"`
	UpdateTerms     bool             `json:"update_terms"`
	AddBlobberId    string           `json:"add_blobber_id"`
	RemoveBlobberId string           `json:"remove_blobber_id"`
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

	return nil
}

// calculate size difference for every blobber of the allocations
func (uar *updateAllocationRequest) getBlobbersSizeDiff(
	alloc *StorageAllocation) (diff int64) {

	var size = alloc.DataShards + alloc.ParityShards
	if uar.Size > 0 {
		diff = (uar.Size + int64(size-1)) / int64(size)
	} else if uar.Size < 0 {
		diff = (uar.Size - int64(size-1)) / int64(size)
	}
	// else -> (0), no changes, avoid unnecessary calculation

	return
}

// new size of blobbers' allocation
func (uar *updateAllocationRequest) getNewBlobbersSize(
	alloc *StorageAllocation) (newSize int64) {

	return alloc.BlobberAllocs[0].Size + uar.getBlobbersSizeDiff(alloc)
}

func (sc *StorageSmartContract) getBlobbersByIDs(ids []string, balances chainstate.StateContextI) []*StorageNode {
	type blobberResp struct {
		index   int
		blobber *StorageNode
	}

	blobberCh := make(chan blobberResp, len(ids))
	var wg sync.WaitGroup
	for i, details := range ids {
		wg.Add(1)
		go func(index int, blobberId string) {
			defer wg.Done()
			blobber, err := sc.getBlobber(blobberId, balances)
			if err != nil || blobber == nil {
				logging.Logger.Debug("can't get blobber", zap.String("blobberId", blobberId), zap.Error(err))
				return
			}
			blobberCh <- blobberResp{
				index:   index,
				blobber: blobber,
			}
		}(i, details)
	}
	wg.Wait()
	close(blobberCh)

	//ensure original ordering
	blobbers := make([]*StorageNode, len(ids))
	for resp := range blobberCh {
		blobbers[resp.index] = resp.blobber
	}
	filtered := make([]*StorageNode, 0, len(ids))
	for _, b := range blobbers {
		if b != nil {
			filtered = append(filtered, b)
		}
	}

	return filtered
}

// getAllocationBlobbers loads blobbers of an allocation from store
func (sc *StorageSmartContract) getAllocationBlobbers(alloc *StorageAllocation,
	balances chainstate.StateContextI) (blobbers []*StorageNode, err error) {

	blobbers = make([]*StorageNode, len(alloc.BlobberAllocs))
	type blobberResp struct {
		index   int
		blobber *StorageNode
	}

	blobberCh := make(chan blobberResp, len(alloc.BlobberAllocs))
	errorCh := make(chan error)
	var wg sync.WaitGroup
	for i, details := range alloc.BlobberAllocs {
		wg.Add(1)
		go func(index int, blobberId string) {
			defer wg.Done()
			var blobber *StorageNode
			blobber, err = sc.getBlobber(blobberId, balances)
			if err != nil {
				errorCh <- fmt.Errorf("can't get blobber %q: %v", blobberId, err)
			}
			blobberCh <- blobberResp{
				index:   index,
				blobber: blobber,
			}
		}(i, details.BlobberID)
	}
	wg.Wait()
	close(errorCh)
	close(blobberCh)
	for err := range errorCh {
		if err != nil {
			return nil, err
		}
	}

	for resp := range blobberCh {
		blobbers[resp.index] = resp.blobber
	}

	return
}

// closeAllocation making it expired; the allocation will be alive the
// challenge_completion_time and be closed then
func (sc *StorageSmartContract) closeAllocation(t *transaction.Transaction,
	alloc *StorageAllocation, balances chainstate.StateContextI) (
	resp string, err error) {

	if alloc.Expiration-t.CreationDate <
		toSeconds(alloc.ChallengeCompletionTime) {
		return "", common.NewError("allocation_closing_failed",
			"doesn't need to close allocation is about to expire")
	}

	// mark as expired, but it will be alive at least chellenge_competion_time
	alloc.Expiration = t.CreationDate

	for _, ba := range alloc.BlobberAllocs {
		var sp *stakePool
		if sp, err = sc.getStakePool(ba.BlobberID, balances); err != nil {
			return "", fmt.Errorf("can't get stake pool of %s: %v", ba.BlobberID,
				err)
		}
		if err := sp.removeOffer(ba.Offer()); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"error removing offer: "+err.Error())
		}
		if err = sp.save(sc.ID, ba.BlobberID, balances); err != nil {
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

	err = emitAddOrOverwriteAllocation(alloc, balances)
	if err != nil {
		return "", common.NewErrorf("allocation_closing_failed",
			"saving allocation in db: %v", err)
	}

	return string(alloc.Encode()), nil // closing
}

func (sc *StorageSmartContract) saveUpdatedAllocation(
	alloc *StorageAllocation,
	blobbers []*StorageNode,
	balances chainstate.StateContextI,
) (err error) {
	for _, b := range blobbers {
		if _, err = balances.InsertTrieNode(b.GetKey(sc.ID), b); err != nil {
			return
		}
		if err := emitUpdateBlobber(b, balances); err != nil {
			return fmt.Errorf("emmiting blobber %v: %v", b, err)
		}
	}

	// save allocation
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return
	}

	err = emitAddOrOverwriteAllocation(alloc, balances)
	if err != nil {
		return
	}

	return
}

// allocation period used to calculate weighted average prices
type allocPeriod struct {
	read   state.Balance    // read price
	write  state.Balance    // write price
	period common.Timestamp // period (duration)
	size   int64            // size for period
}

func (ap *allocPeriod) weight() float64 {
	return float64(ap.period) * float64(ap.size)
}

// returns weighted average read and write prices
func (ap *allocPeriod) join(np *allocPeriod) (avgRead, avgWrite state.Balance) {
	var (
		apw, npw = ap.weight(), np.weight() // weights
		ws       = apw + npw                // weights sum
		rp, wp   float64                    // read sum, write sum (weighted)
	)

	rp = (float64(ap.read) * apw) + (float64(np.read) * npw)
	wp = (float64(ap.write) * apw) + (float64(np.write) * npw)

	avgRead = state.Balance(rp / ws)
	avgWrite = state.Balance(wp / ws)
	return
}

func weightedAverage(prev, next *Terms, tx, pexp, expDiff common.Timestamp,
	psize, sizeDiff int64) (avg Terms) {

	// allocation periods
	var left, added allocPeriod
	left.read, left.write = prev.ReadPrice, prev.WritePrice   // } prices
	added.read, added.write = next.ReadPrice, next.WritePrice // }
	left.size, added.size = psize, psize+sizeDiff             // sizes
	left.period, added.period = pexp-tx, pexp+expDiff-tx      // periods
	// join
	avg.ReadPrice, avg.WritePrice = left.join(&added)

	// just copy from next
	avg.MinLockDemand = next.MinLockDemand
	avg.MaxOfferDuration = next.MaxOfferDuration
	avg.ChallengeCompletionTime = next.ChallengeCompletionTime
	return
}

// The adjustChallengePool moves more or moves some tokens back from or to
// challenge pool during allocation extending or reducing.
func (sc *StorageSmartContract) adjustChallengePool(
	alloc *StorageAllocation,
	awp *allocationWritePools,
	odr, ndr common.Timestamp,
	oterms []Terms,
	now common.Timestamp,
	balances chainstate.StateContextI,
) (err error) {

	var (
		changes = alloc.challengePoolChanges(odr, ndr, oterms)
		cp      *challengePool
	)

	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("adjust_challenge_pool: %v", err)
	}

	var changed bool

	for i, ch := range changes {
		var blobID = alloc.BlobberAllocs[i].BlobberID
		switch {
		case ch > 0:
			err = awp.moveToChallenge(alloc.ID, blobID, cp, now, ch)
			changed = true
		case ch < 0:
			// only if the challenge pool has the tokens; all the tokens
			// can be moved back already, or moved to a blobber due to
			// challenge process
			if cp.Balance >= -ch {
				wp, err := awp.getOwnerWP()
				if err != nil {
					return fmt.Errorf("adjust_challenge_pool: %v", err)
				}
				err = cp.moveToWritePool(alloc, blobID, alloc.Until(), wp, -ch)
				if err != nil {
					logging.Logger.Error("moveToWritePool faliled", zap.Error(err))
				}
				changed = true
			}
		default:
			// no changes for the blobber
		}
		if err != nil {
			return fmt.Errorf("adjust_challenge_pool: %v", err)
		}
	}

	if changed {
		err = cp.save(sc.ID, alloc.ID, balances)
	}

	return
}

// extendAllocation extends size or/and expiration (one of them can be reduced);
// here we use new terms of blobbers
func (sc *StorageSmartContract) extendAllocation(
	t *transaction.Transaction,
	alloc *StorageAllocation,
	blobbers []*StorageNode,
	uar *updateAllocationRequest,
	mintTokens bool,
	balances chainstate.StateContextI,
) (err error) {
	var (
		diff   = uar.getBlobbersSizeDiff(alloc) // size difference
		size   = uar.getNewBlobbersSize(alloc)  // blobber size
		gbSize = sizeInGB(size)                 // blobber size in GB
		cct    time.Duration                    // new challenge_completion_time

		// keep original terms to adjust challenge pool value
		oterms = make([]Terms, 0, len(alloc.BlobberAllocs))
		// original allocation duration remains
		odr = alloc.Expiration - t.CreationDate
	)

	// adjust the expiration if changed, boundaries has already checked
	var prevExpiration = alloc.Expiration
	alloc.Expiration += uar.Expiration // new expiration
	alloc.Size += uar.Size             // new size

	// 1. update terms
	for i, details := range alloc.BlobberAllocs {
		oterms = append(oterms, details.Terms) // keep original terms will be changed
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
		if uar.Size > 0 {
			if b.Capacity-b.Used-diff < 0 {
				return common.NewErrorf("allocation_extending_failed",
					"blobber %s doesn't have enough free space", b.ID)
			}
		}

		b.Used += diff // new capacity used

		// update terms using weighted average
		details.Terms = weightedAverage(&details.Terms, &b.Terms,
			t.CreationDate, prevExpiration, alloc.Expiration, details.Size,
			diff)

		details.Size = size // new size

		if uar.Expiration > toSeconds(b.Terms.MaxOfferDuration) {
			return common.NewErrorf("allocation_extending_failed",
				"blobber %s doesn't allow so long offers", b.ID)
		}

		if b.Terms.ChallengeCompletionTime > cct {
			cct = b.Terms.ChallengeCompletionTime // seek max CCT
		}

		// since, new terms is weighted average based on previous terms and
		// past allocation time and new terms and new allocation time; then
		// we can easily recalculate new min_lock_demand value from allocation
		// start to its new end using the new weighted average terms; but, we
		// can't reduce the min_lock_demand_value; that's all;

		// new blobber's min lock demand (alloc.Expiration is already updated
		// and we can use restDurationInTimeUnits method here)
		var nbmld = details.Terms.minLockDemand(gbSize,
			alloc.restDurationInTimeUnits(alloc.StartTime))

		// min_lock_demand can be increased only
		if nbmld > details.MinLockDemand {
			details.MinLockDemand = nbmld
		}

		newOffer := details.Offer()
		if newOffer != oldOffer {
			var sp *stakePool
			if sp, err = sc.getStakePool(details.BlobberID, balances); err != nil {
				return fmt.Errorf("can't get stake pool of %s: %v", details.BlobberID, err)
			}
			if err := sp.addOffer(newOffer - oldOffer); err != nil {
				return fmt.Errorf("adding offer: %v", err)
			}
			if err = sp.save(sc.ID, details.BlobberID, balances); err != nil {
				return fmt.Errorf("can't save stake pool of %s: %v", details.BlobberID,
					err)
			}

		}
	}

	// update max challenge_completion_time
	alloc.ChallengeCompletionTime = cct

	var until = alloc.Until()
	wps, err := alloc.getAllocationPools(sc, balances)
	if err != nil {
		return common.NewErrorf("allocation_extending_failed", "%v", err)
	}

	// lock tokens if this transaction provides them
	if t.Value > 0 {
		ap, err := newAllocationPool(t, alloc, until, mintTokens, balances)
		if err != nil {
			return common.NewErrorf("allocation_extending_failed",
				"write pool filling: %v", err)
		}
		if err := wps.addOwnerWritePool(ap); err != nil {
			return common.NewErrorf("allocation_extending_failed",
				"add write pool: %v", err)
		}
	}

	// is it about size increasing? if so, we should make sure the write
	// pool has enough tokens
	if diff > 0 {
		if mldLeft := alloc.restMinLockDemand(); mldLeft > 0 {
			if wps.allocUntil(alloc.ID, until) < mldLeft {
				return common.NewError("allocation_extending_failed",
					"not enough tokens in write pool to extend allocation")
			}
		}
	}

	// add more tokens to related challenge pool, or move some tokens back
	var ndr = alloc.Expiration - t.CreationDate
	err = sc.adjustChallengePool(alloc, wps, odr, ndr, oterms, t.CreationDate, balances)
	if err != nil {
		return common.NewErrorf("allocation_extending_failed", "%v", err)
	}

	if err := wps.saveWritePools(sc.ID, balances); err != nil {
		return common.NewErrorf("allocation_extending_failed", "%v", err)
	}

	return nil
}

// reduceAllocation reduces size or/and expiration (no one can be increased);
// here we use the same terms of related blobbers
func (sc *StorageSmartContract) reduceAllocation(t *transaction.Transaction,
	alloc *StorageAllocation, blobbers []*StorageNode,
	uar *updateAllocationRequest, balances chainstate.StateContextI,
) (err error) {
	var (
		diff = uar.getBlobbersSizeDiff(alloc) // size difference
		size = uar.getNewBlobbersSize(alloc)  // blobber size

		// original allocation duration remains
		odr = alloc.Expiration - t.CreationDate
	)

	// adjust the expiration if changed, boundaries has already checked
	alloc.Expiration += uar.Expiration
	alloc.Size += uar.Size

	// 1. update terms
	for i, ba := range alloc.BlobberAllocs {
		var b = blobbers[i]
		oldOffer := ba.Offer()
		b.Used += diff // new capacity used
		ba.Size = size // new size
		// update stake pool
		newOffer := ba.Offer()
		if newOffer != oldOffer {
			var sp *stakePool
			if sp, err = sc.getStakePool(ba.BlobberID, balances); err != nil {
				return fmt.Errorf("can't get stake pool of %s: %v", ba.BlobberID,
					err)
			}
			if newOffer < oldOffer {
				if err := sp.removeOffer(oldOffer - newOffer); err != nil {
					return fmt.Errorf("removing offer: %v", err)
				}
			} else {
				// if we are adding a blobber then we will want to add a new offer for that blobber
				if err := sp.addOffer(newOffer - oldOffer); err != nil {
					return fmt.Errorf("adding offer: %v", err)
				}
			}

			if err = sp.save(sc.ID, ba.BlobberID, balances); err != nil {
				return fmt.Errorf("can't save stake pool of %s: %v", ba.BlobberID,
					err)
			}
			if err := emitUpdateBlobber(b, balances); err != nil {
				return fmt.Errorf("emitting blobber %s, error:%v", b.ID, err)
			}
		}
	}

	wps, err := alloc.getAllocationPools(sc, balances)
	if err != nil {
		return common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

	// lock tokens if this transaction provides them
	if t.Value > 0 {
		if err = stakepool.CheckClientBalance(t, balances); err != nil {
			return common.NewErrorf("allocation_reducing_failed", "%v", err)
		}
		var until = alloc.Until()

		ap, err := newAllocationPool(t, alloc, until, false, balances)
		if err != nil {
			return common.NewErrorf("allocation_reducing_failed", "%v", err)
		}
		if err := wps.addOwnerWritePool(ap); err != nil {
			return common.NewErrorf("allocation_extending_failed",
				"add write pool: %v", err)
		}
	}

	// new allocation duration remains
	var ndr = alloc.Expiration - t.CreationDate
	err = sc.adjustChallengePool(alloc, wps, odr, ndr, nil, t.CreationDate,
		balances)
	if err != nil {
		return common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

	if err := wps.saveWritePools(sc.ID, balances); err != nil {
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
	return sc.updateAllocationRequestInternal(txn, input, conf, false, balances)
}

func (sc *StorageSmartContract) updateAllocationRequestInternal(
	t *transaction.Transaction,
	input []byte,
	conf *Config,
	mintTokens bool,
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

	var clist *Allocations // client allocations list
	if clist, err = sc.getAllocationsList(request.OwnerID, balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get client's allocations list: "+err.Error())
	}

	if !clist.has(request.ID) {
		return "", common.NewErrorf("allocation_updating_failed",
			"can't find allocation in client's allocations list: %s (%d)",
			request.ID, len(clist.List))
	}

	var alloc *StorageAllocation
	if alloc, err = sc.getAllocation(request.ID, balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			"can't get existing allocation: "+err.Error())
	}

	if err = request.validate(conf, alloc); err != nil {
		return "", common.NewError("allocation_updating_failed", err.Error())
	}

	// can't update expired allocation
	if alloc.Expiration < t.CreationDate {
		return "", common.NewError("allocation_updating_failed",
			"can't update expired allocation")
	}
	// adjust expiration
	var newExpiration = alloc.Expiration + request.Expiration
	var newSize = request.Size + alloc.Size

	// get blobber of the allocation to update them
	var blobbers []*StorageNode
	if blobbers, err = sc.getAllocationBlobbers(alloc, balances); err != nil {
		return "", common.NewError("allocation_updating_failed",
			err.Error())
	}

	if len(request.AddBlobberId) > 0 {
		blobbers, err = alloc.changeBlobbers(
			blobbers, request.AddBlobberId, request.RemoveBlobberId, sc, t.CreationDate, balances,
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
			bd.Terms.ChallengeCompletionTime = blobbers[i].Terms.ChallengeCompletionTime
			bd.Terms.MaxOfferDuration = blobbers[i].Terms.MaxOfferDuration
		}
	}

	// update allocation transaction hash
	alloc.Tx = t.Hash
	if len(request.Name) > 0 {
		alloc.Name = request.Name
	}

	// close allocation now
	if newExpiration <= t.CreationDate {
		return sc.closeAllocation(t, alloc, balances)
	}

	// an allocation can't be shorter than configured in SC
	// (prevent allocation shortening for entire period)
	if newExpiration < 0 ||
		newExpiration-t.CreationDate < toSeconds(conf.MinAllocDuration) {

		return "", common.NewError("allocation_updating_failed",
			"allocation duration becomes too short")
	}

	if newSize < conf.MinAllocSize || newSize < alloc.UsedSize {
		return "", common.NewError("allocation_updating_failed",
			"allocation size becomes too small")
	}

	// if size or expiration increased, then we use new terms
	// otherwise, we use the same terms
	if request.Size > 0 || request.Expiration > 0 {
		err = sc.extendAllocation(t, alloc, blobbers, &request, mintTokens, balances)
	} else if request.Size != 0 || request.Expiration != 0 {
		if mintTokens {
			return "", common.NewError("allocation_updating_failed",
				"cannot reduce when minting tokens")
		}
		err = sc.reduceAllocation(t, alloc, blobbers, &request, balances)
	} else if len(request.AddBlobberId) > 0 {
		err = sc.extendAllocation(t, alloc, blobbers, &request, mintTokens, balances)
	}
	if err != nil {
		return "", err
	}

	if request.SetImmutable {
		alloc.IsImmutable = true
	}

	err = sc.saveUpdatedAllocation(alloc, blobbers, balances)
	if err != nil {
		return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

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
		}
		ba.Stats.FailedChallenges += ba.Stats.OpenChallenges
		ba.Stats.OpenChallenges = 0
		ba.Stats.TotalChallenges = ba.Stats.FailedChallenges + ba.Stats.SuccessChallenges
		if ba.Stats.TotalChallenges == 0 {
			passRates = append(passRates, 1.0)
			continue
		}
		passRates = append(passRates, float64(ba.Stats.SuccessChallenges)/float64(ba.Stats.TotalChallenges))
		succesful += ba.Stats.SuccessChallenges
		failed += ba.Stats.FailedChallenges
	}
	alloc.Stats.SuccessChallenges = succesful
	alloc.Stats.FailedChallenges = failed
	alloc.Stats.TotalChallenges = alloc.Stats.FailedChallenges + alloc.Stats.FailedChallenges
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
	var failed, successful int64 = 0, 0

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

			//if c.Responded || c.AllocationID != alloc.ID {
			//	continue // already accepted, already rewarded/penalized
			//}
			var expire = oc.CreatedAt + toSeconds(ba.Terms.ChallengeCompletionTime)
			if expire < now {
				ba.Stats.FailedChallenges++
				alloc.Stats.FailedChallenges++
			} else {
				ba.Stats.SuccessChallenges++
				alloc.Stats.SuccessChallenges++
			}
		}

	default:
		return nil, fmt.Errorf("getting allocation challenge: %v", err)
	}

	for _, ba := range alloc.BlobberAllocs {

		ba.Stats.OpenChallenges = 0
		ba.Stats.TotalChallenges = ba.Stats.SuccessChallenges + ba.Stats.FailedChallenges
		if ba.Stats.TotalChallenges == 0 {
			passRates = append(passRates, 1.0)
			continue
		}
		// success rate for the blobber allocation
		//fmt.Println("pass rate i", i, "successful", d.Stats.SuccessChallenges, "failed", d.Stats.FailedChallenges)
		passRates = append(passRates, float64(ba.Stats.SuccessChallenges)/float64(ba.Stats.TotalChallenges))
		successful += ba.Stats.SuccessChallenges
		failed += ba.Stats.FailedChallenges
	}

	alloc.Stats.SuccessChallenges = successful
	alloc.Stats.FailedChallenges = failed
	alloc.Stats.TotalChallenges = alloc.Stats.SuccessChallenges + alloc.Stats.FailedChallenges
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
	alloc.Expiration, alloc.ChallengeCompletionTime = t.CreationDate, 0

	sps := make([]*stakePool, 0, len(alloc.BlobberAllocs))
	for _, d := range alloc.BlobberAllocs {
		var sp *stakePool
		if sp, err = sc.getStakePool(d.BlobberID, balances); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"can't get stake pool of "+d.BlobberID+": "+err.Error())
		}
		if err := sp.removeOffer(d.Offer()); err != nil {
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

	err = emitAddOrOverwriteAllocation(alloc, balances)
	if err != nil {
		return "", common.NewErrorf("alloc_cancel_failed",
			"saving allocation to db: %v", err)
	}

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
		if sp, err = sc.getStakePool(d.BlobberID, balances); err != nil {
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

	err = emitAddOrOverwriteAllocation(alloc, balances)
	if err != nil {
		return "", common.NewErrorf("alloc_cancel_failed",
			"saving allocation to db: %v", err)
	}

	return "finalized", nil
}

func (sc *StorageSmartContract) finishAllocation(
	t *transaction.Transaction,
	alloc *StorageAllocation,
	passRates []float64,
	sps []*stakePool,
	balances chainstate.StateContextI,
) (err error) {
	wps, err := alloc.getAllocationPools(sc, balances)
	if err != nil {
		return common.NewErrorf("allocation_extending_failed", "%v", err)
	}
	aps := wps.activeAllocationPools(alloc.ID, t.CreationDate)
	if len(aps) == 0 {
		return common.NewError("fini_alloc_failed",
			"no allocation pools to pay min lock demand")
	}

	aps.sortExpiry()
	apIndex := 0
	// we can use the i for the blobbers list above because of algorithm
	// of the getAllocationBlobbers method; also, we can use the i in the
	// passRates list above because of algorithm of the adjustChallenges
	for i, d := range alloc.BlobberAllocs {
		// min lock demand rest
		var paid state.Balance = 0
		if lack := d.MinLockDemand - d.Spent; lack > 0 {
			for apIndex < len(aps) && lack > 0 {
				pay := lack
				if pay > aps[apIndex].Balance {
					pay = aps[apIndex].Balance
				}
				aps[apIndex].Balance -= state.Balance(pay)
				if aps[apIndex].Balance == 0 {
					apIndex++
				}

				paid += pay
				lack -= pay
			}
			if lack > 0 {
				return fmt.Errorf("alloc_cancel_failed, paying min_lock for blobber %v"+
					"ammount was short by %v", d.BlobberID, lack)
			}

			err = sps[i].DistributeRewards(float64(paid), d.BlobberID, spenum.Blobber, balances)
			if err != nil {
				return fmt.Errorf("alloc_cancel_failed, paying min_lock lack %v for blobber "+
					"%v from alocation poosl %v, minlock demand %v spent %v error %v",
					lack, d.BlobberID, aps, d.MinLockDemand, d.Spent, err.Error())
			}
		}
		d.Spent += paid
		d.FinalReward += paid
	}

	if err := wps.saveWritePools(sc.ID, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"saving allocation write pools: "+err.Error())
	}

	var blobbers []*StorageNode
	if blobbers, err = sc.getAllocationBlobbers(alloc, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"invalid state: can't get related blobbers: "+err.Error())
	}

	var allb *StorageNodes
	if allb, err = sc.getBlobbersList(balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"can't get all blobbers list: "+err.Error())
	}

	var cp *challengePool
	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"can't get related challenge pool: "+err.Error())
	}

	var passPayments float64 = 0.0
	for i, d := range alloc.BlobberAllocs {
		var b = blobbers[i]
		if b.ID != d.BlobberID {
			return common.NewErrorf("fini_alloc_failed",
				"blobber %s and %s don't match", b.ID, d.BlobberID)
		}
		if alloc.UsedSize > 0 && cp.Balance > 0 && passRates[i] > 0 && d.Stats != nil {
			var (
				ratio  = float64(d.Stats.UsedSize) / float64(alloc.UsedSize)
				reward = float64(cp.Balance) * ratio * passRates[i]
			)
			err = sps[i].DistributeRewards(reward, b.ID, spenum.Blobber, balances)
			if err != nil {
				return common.NewError("fini_alloc_failed",
					"paying reward to stake pool of "+d.BlobberID+": "+err.Error())
			}
			d.Spent += state.Balance(reward)
			d.FinalReward += state.Balance(reward)
			passPayments += reward
		}

		if err = sps[i].save(sc.ID, d.BlobberID, balances); err != nil {
			return common.NewError("fini_alloc_failed",
				"saving stake pool of "+d.BlobberID+": "+err.Error())
		}

		data, _ := json.Marshal(dbs.DbUpdates{
			Id: d.BlobberID,
			Updates: map[string]interface{}{
				"total_stake": int64(sps[i].stake()),
			},
		})
		balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, d.BlobberID, string(data))

		// update the blobber
		b.Used -= d.Size
		if _, err = balances.InsertTrieNode(b.GetKey(sc.ID), b); err != nil {
			return common.NewError("fini_alloc_failed",
				"saving blobber "+d.BlobberID+": "+err.Error())
		}
		// update the blobber in all (replace with existing one)
		allb.Nodes.update(b)
		if err := emitUpdateBlobber(b, balances); err != nil {
			return common.NewError("fini_alloc_failed",
				"emitting blobber "+b.ID+": "+err.Error())
		}
	}
	cp.Balance -= state.Balance(passPayments)
	// move challenge pool rest to write pool
	alloc.MovedBack += cp.Balance

	// write pool
	var wp *writePool
	if wp, err = sc.getWritePool(alloc.Owner, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"can't get user's write pools: "+err.Error())
	}
	err = cp.moveToWritePool(alloc, "", alloc.Until(), wp, cp.Balance)
	if err != nil {
		return common.NewError("fini_alloc_failed",
			"moving challenge pool rest back to write pool: "+err.Error())
	}

	// save all blobbers list
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, allb)
	if err != nil {
		return common.NewError("fini_alloc_failed",
			"saving all blobbers list: "+err.Error())
	}

	// save all rest and remove allocation from all allocations list

	if err = cp.save(sc.ID, alloc.ID, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"saving challenge pool: "+err.Error())
	}

	if err = wp.save(sc.ID, alloc.Owner, balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"saving write pool: "+err.Error())
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

	if err := sc.removeUserAllocation(alloc.Owner, alloc, balances); err != nil {
		return "", common.NewError("curator_transfer_allocation_failed", err.Error())
	}

	alloc.Owner = tai.NewOwnerId
	alloc.OwnerPublicKey = tai.NewOwnerPublicKey

	if err := sc.addUserAllocation(alloc.Owner, alloc, balances); err != nil {
		return "", common.NewError("curator_transfer_allocation_failed", err.Error())
	}

	if !alloc.hasWritePool(sc, tai.NewOwnerId, balances) {
		if err = sc.createEmptyWritePool(txn, alloc, balances); err != nil {
			return "", common.NewError("curator_transfer_allocation_failed",
				"error creating write pool: "+err.Error())
		}
	}

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf("curator_transfer_allocation_failed",
			"saving new allocation: %v", err)
	}

	err = emitAddOrOverwriteAllocation(alloc, balances)
	if err != nil {
		return "", common.NewErrorf("curator_transfer_allocation_failed",
			"saving new allocation to db: %v", err)
	}

	// txn.Hash is the id of the new token pool
	return txn.Hash, nil
}

func (sa StorageAllocation) hasWritePool(
	ssc *StorageSmartContract,
	id string,
	balances chainstate.StateContextI,
) bool {
	wp, err := ssc.getWritePool(sa.Owner, balances)
	if err != nil {
		return false
	}
	for _, pool := range wp.Pools {
		if pool.AllocationID == sa.ID {
			return true
		}
	}
	return false
}
