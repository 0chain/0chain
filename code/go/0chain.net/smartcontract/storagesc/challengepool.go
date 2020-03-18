package storagesc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

// challenge pool lock/unlock request

type challengePoolRequest struct {
	AllocationID string `json:"allocation_id"`
}

func (req *challengePoolRequest) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(req); err != nil {
		panic(err) // must not happen
	}
	return
}

func (req *challengePoolRequest) decode(input []byte) error {
	return json.Unmarshal(input, req)
}

// challenge pool is a locked tokens for a duration for an allocation

type challengePool struct {
	*tokenpool.ZcnLockingPool `json:"pool"`
}

func newChallengePool() *challengePool {
	return &challengePool{
		ZcnLockingPool: &tokenpool.ZcnLockingPool{},
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

	err = cp.ZcnLockingPool.Decode(challengePoolVal.Pool, &tokenLock{})
	return
}

// save the challenge pool
func (cp *challengePool) save(sscKey, allocationID string,
	balances chainState.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(challengePoolKey(sscKey, allocationID), cp)
	return
}

// fill the pool by transaction
func (cp *challengePool) fill(t *transaction.Transaction,
	balances chainState.StateContextI) (
	transfer *state.Transfer, resp string, err error) {

	if transfer, resp, err = cp.FillPool(t); err != nil {
		return
	}
	err = balances.AddTransfer(transfer)
	return
}

// setExpiration of the locked tokens
func (cp *challengePool) setExpiation(set common.Timestamp) (err error) {
	if set == 0 {
		return // as is
	}
	tl, ok := cp.TokenLockInterface.(*tokenLock)
	if !ok {
		return fmt.Errorf(
			"invalid challenge pool state, invalid token lock type: %T",
			cp.TokenLockInterface)
	}
	tl.Duration = time.Duration(set-tl.StartTime) * time.Second
	return
}

// stat

type challengePoolStat struct {
	ID        datastore.Key    `json:"pool_id"`
	StartTime common.Timestamp `json:"start_time"`
	Duartion  time.Duration    `json:"duration"`
	TimeLeft  time.Duration    `json:"time_left"`
	Locked    bool             `json:"locked"`
	Balance   state.Balance    `json:"balance"`
}

func (stat *challengePoolStat) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(stat); err != nil {
		panic(err) // must never happen
	}
	return
}

func (stat *challengePoolStat) decode(input []byte) error {
	return json.Unmarshal(input, stat)
}

//
// smart contract methods
//

// getChallengePoolBytes of a client
func (ssc *StorageSmartContract) getChallengePoolBytes(
	allocationID datastore.Key, balances chainState.StateContextI) (
	b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(challengePoolKey(ssc.ID, allocationID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// getChallengePool of current client
func (ssc *StorageSmartContract) getChallengePool(allocationID datastore.Key,
	balances chainState.StateContextI) (cp *challengePool, err error) {

	var poolb []byte
	poolb, err = ssc.getChallengePoolBytes(allocationID, balances)
	if err != nil {
		return
	}
	cp = newChallengePool("")
	err = cp.Decode(poolb)
	return
}

// newChallengePool SC function creates new
// challenge pool for a client don't saving it
func (ssc *StorageSmartContract) newChallengePool(allocationID string,
	creationDate, expiresAt common.Timestamp,
	balances chainState.StateContextI) (cp *challengePool, err error) {

	_, err = ssc.getChallengePoolBytes(allocationID, balances)

	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewError("new_challenge_pool_failed", err.Error())
	}

	if err == nil {
		return nil, common.NewError("new_challenge_pool_failed",
			"already exist")
	}

	err = nil // reset the util.ErrValueNotPresent

	cp = newChallengePool()
	cp.TokenLockInterface = &tokenLock{
		StartTime: creationDate,
		Duration:  common.ToTime(expiresAt).Sub(common.ToTime(creationDate)),
	}
	cp.TokenPool.ID = challengePoolKey(ssc.ID, allocationID)
	return
}

// create, fill and save challenge pool for new allocation
func (ssc *StorageSmartContract) createChallengePool(t *transaction.Transaction,
	sa *StorageAllocation, balances chainState.StateContextI) (err error) {

	// create related challenge_pool expires with the allocation + challenge
	// completion time
	var cp *challengePool
	cp, err = ssc.newChallengePool(sa.GetKey(ssc.ID), t.CreationDate,
		sa.Expiration+toSeconds(sa.ChallengeCompletionTime), balances)
	if err != nil {
		return fmt.Errorf("can't create challenge pool: %v", err)
	}

	// lock required number of tokens

	if state.Balance(t.Value) < sa.MinLockDemand {
		return fmt.Errorf("not enough tokens to create allocation: %v < %v",
			t.Value, sa.MinLockDemand)
	}

	if _, _, err = cp.fill(t, balances); err != nil {
		return fmt.Errorf("can't fill challenge pool: %v", err)
	}

	// save the challenge pool
	if err = cp.save(ssc.ID, sa.ID, balances); err != nil {
		return fmt.Errorf("can't save challenge pool: %v", err)
	}

	return
}

// lock tokens for challenge pool of transaction's client
func (ssc *StorageSmartContract) challengePoolLock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// configurations

	var conf *challengePoolConfig
	if conf, err = ssc.getChallengePoolConfig(balances, true); err != nil {
		return "", common.NewError("challenge_pool_lock_failed",
			"can't get configs: "+err.Error())
	}

	// filter by configurations

	if t.Value < conf.MinLock {
		return "", common.NewError("challenge_pool_lock_failed",
			"insufficient amount to lock")
	}

	// challenge lock request & user balance

	var req challengePoolRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("challenge_pool_lock_failed", err.Error())
	}
	var balance state.Balance
	balance, err = balances.GetClientBalance(t.ClientID)

	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("challenge_pool_lock_failed", err.Error())
	}

	if err == util.ErrValueNotPresent {
		return "", common.NewError("challenge_pool_lock_failed",
			"no tokens to lock")
	}

	if state.Balance(t.Value) > balance {
		return "", common.NewError("challenge_pool_lock_failed",
			"lock amount is greater than balance")
	}

	if req.AllocationID == "" {
		return "", common.NewError("challenge_pool_lock_failed",
			"missing allocation_id")
	}

	// user challenge pools

	var cp *challengePool
	if cp, err = ssc.getChallengePool(req.AllocationID, balances); err != nil {
		return "", common.NewError("challenge_pool_lock_failed", err.Error())
	}

	// lock more tokens

	if _, resp, err = cp.fill(t, balances); err != nil {
		return "", common.NewError("challenge_pool_lock_failed",
			err.Error())
	}

	if err = cp.save(ssc.ID, req.AllocationID, balances); err != nil {
		return "", common.NewError("challenge_pool_lock_failed", err.Error())
	}

	return
}

// unlock tokens if expired
func (ssc *StorageSmartContract) challengePoolUnlock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// the request

	var req challengePoolRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("challenge_pool_unlock_failed", err.Error())
	}

	// challenge pool

	var cp *challengePool
	if cp, err = ssc.getChallengePool(req.AllocationID, balances); err != nil {
		return "", common.NewError("challenge_pool_unlock_failed", err.Error())
	}

	if cp.ClientID != t.ClientID {
		return "", common.NewError("challenge_pool_unlock_failed",
			"only owner can unlock the pool")
	}

	var transfer *state.Transfer
	transfer, resp, err = cp.EmptyPool(ssc.ID, t.ClientID,
		common.ToTime(t.CreationDate))
	if err != nil {
		return "", common.NewError("challenge_pool_unlock_failed", err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("challenge_pool_unlock_failed", err.Error())
	}

	// save pool
	if err = cp.save(ssc.ID, req.AllocationID, balances); err != nil {
		return "", common.NewError("challenge_pool_unlock_failed", err.Error())
	}

	return
}

// update challenge pool expiration
func (ssc *StorageSmartContract) updateChallengePoolExpiration(
	t *transaction.Transaction, allocID string, expiried common.Timestamp,
	balances chainState.StateContextI) (err error) {

	var cp *challengePool
	if cp, err = ssc.getChallengePool(allocID, balances); err != nil {
		return
	}

	// lock tokens if this transaction provides them
	if t.Value > 0 {
		if _, _, err = cp.fill(t, balances); err != nil {
			return
		}
	}

	if err = cp.setExpiation(expiried); err != nil {
		return
	}

	if err = cp.save(ssc.ID, allocID, balances); err != nil {
		return
	}

	return
}

//
// stat
//

func (ssc *StorageSmartContract) getChallengePoolStat(cp *challengePool,
	tp time.Time) (stat *challengePoolStat, err error) {

	stat = new(challengePoolStat)

	if err = stat.decode(cp.LockStats(tp)); err != nil {
		return nil, err
	}

	stat.ID = cp.ID
	stat.Locked = cp.IsLocked(tp)
	stat.Balance = cp.Balance

	return
}

// statistic for all locked tokens of a challenge pool
func (ssc *StorageSmartContract) getChallengePoolStatHandler(
	ctx context.Context, params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = datastore.Key(params.Get("allocation_id"))
		cp           *challengePool
	)
	if cp, err = ssc.getChallengePool(allocationID, balances); err != nil {
		return
	}

	var (
		tp   = time.Now()
		stat *challengePoolStat
	)
	if stat, err = ssc.getChallengePoolStat(cp, tp); err != nil {
		return nil, common.NewError("challenge_pool_stats", err.Error())
	}

	return &stat, nil
}
