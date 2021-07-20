package storagesc

import (
	"context"
	"encoding/json"
	"net/url"

	"0chain.net/smartcontract"
	"github.com/0chain/gosdk/core/common/errors"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

// challenge pool is a locked tokens for a duration for an allocation

type challengePool struct {
	*tokenpool.ZcnPool `json:"pool"`
}

func newChallengePool() *challengePool {
	return &challengePool{
		ZcnPool: &tokenpool.ZcnPool{},
	}
}

func challengePoolKey(scKey, allocationID string) datastore.Key {
	return datastore.Key(scKey + ":challengepool:" + allocationID)
}

func (cp *challengePool) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(cp); err != nil {
		panic(err) // must never happens
	}
	return
}

func (cp *challengePool) Decode(input []byte) (err error) {

	type challengePoolJSON struct {
		Pool json.RawMessage `json:"pool"`
	}

	var challengePoolVal challengePoolJSON
	if err = json.Unmarshal(input, &challengePoolVal); err != nil {
		return
	}

	if len(challengePoolVal.Pool) == 0 {
		return // no data given
	}

	err = cp.ZcnPool.Decode(challengePoolVal.Pool)
	return
}

// save the challenge pool
func (cp *challengePool) save(sscKey, allocationID string,
	balances cstate.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(challengePoolKey(sscKey, allocationID), cp)
	return
}

// moveToWritePool moves tokens back to write pool on data deleted
func (cp *challengePool) moveToWritePool(allocID, blobID string,
	until common.Timestamp, wp *writePool, value state.Balance) (err error) {

	if value == 0 {
		return // nothing to move
	}

	if cp.Balance < value {
		return errors.Newf("", "not enough tokens in challenge pool %s: %d < %d",
			cp.ID, cp.Balance, value)
	}

	var ap = wp.allocPool(allocID, until)
	if ap == nil {
		ap = new(allocationPool)
		ap.AllocationID = allocID
		ap.ExpireAt = 0
		wp.Pools.add(ap)
	}

	// move
	if blobID != "" {
		var bp, ok = ap.Blobbers.get(blobID)
		if !ok {
			ap.Blobbers.add(&blobberPool{
				BlobberID: blobID,
				Balance:   value,
			})
		} else {
			bp.Balance += value
		}
	}
	_, _, err = cp.TransferTo(ap, value, nil)
	return
}

func (cp *challengePool) moveToValidators(sscKey string, reward state.Balance,
	validatos []datastore.Key, vsps []*stakePool,
	balances cstate.StateContextI) (moved state.Balance, err error) {

	if len(validatos) == 0 || reward == 0 {
		return // nothing to move, or nothing to move to
	}

	var oneReward = state.Balance(float64(reward) / float64(len(validatos)))

	for i, sp := range vsps {
		if cp.Balance < oneReward {
			return 0, errors.Newf("", "not enough tokens in challenge pool: %v < %v",
				cp.Balance, oneReward)
		}
		var oneMove state.Balance
		oneMove, err = transferReward(sscKey, *cp.ZcnPool, sp, oneReward, balances)
		sp.Rewards.Validator += oneMove
		if err != nil {
			return 0, errors.Newf("", "moving to validator %s: %v",
				validatos[i], err)
		}
		moved += oneMove
	}

	return
}

func (cp *challengePool) stat(alloc *StorageAllocation) (
	stat *challengePoolStat) {

	stat = new(challengePoolStat)

	stat.ID = cp.ID
	stat.Balance = cp.Balance
	stat.StartTime = alloc.StartTime
	stat.Expiration = alloc.Until()
	stat.Finalized = alloc.Finalized

	return
}

type challengePoolStat struct {
	ID         string           `json:"id"`
	Balance    state.Balance    `json:"balance"`
	StartTime  common.Timestamp `json:"start_time"`
	Expiration common.Timestamp `json:"expiration"`
	Finalized  bool             `json:"finalized"`
}

//
// smart contract methods
//

// getChallengePoolBytes of a client
func (ssc *StorageSmartContract) getChallengePoolBytes(
	allocationID datastore.Key, balances cstate.StateContextI) (b []byte,
	err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(challengePoolKey(ssc.ID, allocationID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// getChallengePool of current client
func (ssc *StorageSmartContract) getChallengePool(allocationID datastore.Key,
	balances cstate.StateContextI) (cp *challengePool, err error) {

	var poolb []byte
	poolb, err = ssc.getChallengePoolBytes(allocationID, balances)
	if err != nil {
		return
	}
	cp = newChallengePool()
	err = cp.Decode(poolb)
	if err != nil {
		return nil, errors.Wrap(err, common.ErrDecoding())
	}
	return
}

// newChallengePool SC function creates new
// challenge pool for a client don't saving it
func (ssc *StorageSmartContract) newChallengePool(allocationID string,
	creationDate, expiresAt common.Timestamp, balances cstate.StateContextI) (
	cp *challengePool, err error) {

	_, err = ssc.getChallengePoolBytes(allocationID, balances)

	if err != nil && !errors.Is(err, util.ErrValueNotPresent()) {
		return nil, errors.Wrap(err, "new_challenge_pool_failed")
	}

	if err == nil {
		return nil, errors.New("new_challenge_pool_failed",
			"already exist")
	}

	err = nil // reset the util.ErrValueNotPresent()

	cp = newChallengePool()
	cp.TokenPool.ID = challengePoolKey(ssc.ID, allocationID)
	return
}

// create, fill and save challenge pool for new allocation
func (ssc *StorageSmartContract) createChallengePool(t *transaction.Transaction,
	alloc *StorageAllocation, balances cstate.StateContextI) (err error) {

	// create related challenge_pool expires with the allocation + challenge
	// completion time
	var cp *challengePool
	cp, err = ssc.newChallengePool(alloc.ID, t.CreationDate, alloc.Until(),
		balances)
	if err != nil {
		return errors.Newf("", "can't create challenge pool: %v", err)
	}

	// don't lock anything here

	// save the challenge pool
	if err = cp.save(ssc.ID, alloc.ID, balances); err != nil {
		return errors.Newf("", "can't save challenge pool: %v", err)
	}

	return
}

//
// stat
//

// statistic for all locked tokens of a challenge pool
func (ssc *StorageSmartContract) getChallengePoolStatHandler(
	ctx context.Context, params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = datastore.Key(params.Get("allocation_id"))
		alloc        *StorageAllocation
		cp           *challengePool
	)

	if allocationID == "" {
		err := errors.New("missing allocation_id URL query parameter")
		return nil, common.NewErrBadRequest(err, "")
	}

	if alloc, err = ssc.getAllocation(allocationID, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetAllocation)
	}

	if cp, err = ssc.getChallengePool(allocationID, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get challenge pool")
	}

	return cp.stat(alloc), nil
}
