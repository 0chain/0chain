package storagesc

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

// write pool lock configurations

type writePoolConfig struct {
	MinLock int64 `json:"min_lock"`
	// TODO (sfxdx): interests? other configs?
}

func writePoolConfigKey(scKey string) datastore.Key {
	return datastore.Key(scKey + ":writepool")
}

func (conf *writePoolConfig) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(conf); err != nil {
		panic(err) // must not happens
	}
	return
}

func (conf *writePoolConfig) Decode(b []byte) error {
	return json.Unmarshal(b, conf)
}

// write pool lock/unlock request

type writePoolRequest struct {
	AllocationID string `json:"allocation_id"`
}

func (req *writePoolRequest) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(req); err != nil {
		panic(err) // must not happen
	}
	return
}

func (req *writePoolRequest) decode(input []byte) error {
	return json.Unmarshal(input, req)
}

// write pool is a locked tokens for a duration for an allocation

type writePool struct {
	*tokenpool.ZcnLockingPool `json:"pool"`
	ClientID                  string `json:"client_id"`
}

func newWritePool(clientID string) *writePool {
	return &writePool{
		ZcnLockingPool: &tokenpool.ZcnLockingPool{},
		ClientID:       clientID,
	}
}

func writePoolKey(scKey, allocationID string) datastore.Key {
	return datastore.Key(scKey + ":writepool:" + allocationID)
}

func (wp *writePool) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(wp); err != nil {
		panic(err) // must never happens
	}
	return
}

func (wp *writePool) Decode(input []byte) (err error) {

	type writePoolJSON struct {
		Pool     json.RawMessage `json:"pool"`
		ClientID string          `json:"client_id"`
	}

	var writePoolVal writePoolJSON
	if err = json.Unmarshal(input, &writePoolVal); err != nil {
		return
	}

	if len(writePoolVal.Pool) == 0 {
		return // no data given
	}

	err = wp.ZcnLockingPool.Decode(writePoolVal.Pool, &tokenLock{})
	return
}

// save the write pool
func (wp *writePool) save(sscKey, allocationID string,
	balances chainState.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(writePoolKey(sscKey, allocationID), wp)
	return
}

// fill the pool by transaction
func (wp *writePool) fill(t *transaction.Transaction,
	balances chainState.StateContextI) (
	transfer *state.Transfer, resp string, err error) {

	if transfer, resp, err = wp.FillPool(t); err != nil {
		return
	}
	err = balances.AddTransfer(transfer)
	return
}

// stat

type writePoolStat struct {
	ID        datastore.Key    `json:"pool_id"`
	StartTime common.Timestamp `json:"start_time"`
	Duartion  time.Duration    `json:"duration"`
	TimeLeft  time.Duration    `json:"time_left"`
	Locked    bool             `json:"locked"`
	Balance   state.Balance    `json:"balance"`
}

func (stat *writePoolStat) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(stat); err != nil {
		panic(err) // must never happen
	}
	return
}

func (stat *writePoolStat) decode(input []byte) error {
	return json.Unmarshal(input, stat)
}

//
// smart contract methods
//

// getWritePoolBytes of a client
func (ssc *StorageSmartContract) getWritePoolBytes(allocationID datastore.Key,
	balances chainState.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(writePoolKey(ssc.ID, allocationID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// getWritePool of current client
func (ssc *StorageSmartContract) getWritePool(allocationID datastore.Key,
	balances chainState.StateContextI) (wp *writePool, err error) {

	var poolb []byte
	if poolb, err = ssc.getWritePoolBytes(allocationID, balances); err != nil {
		return
	}
	wp = newWritePool("")
	err = wp.Decode(poolb)
	return
}

// newWritePool SC function creates new write pool for a client don't saving it
func (ssc *StorageSmartContract) newWritePool(allocationID, clientID string,
	creationDate, expiresAt common.Timestamp,
	balances chainState.StateContextI) (wp *writePool, err error) {

	_, err = ssc.getWritePoolBytes(allocationID, balances)

	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewError("new_write_pool_failed", err.Error())
	}

	if err == nil {
		return nil, common.NewError("new_write_pool_failed", "already exist")
	}

	wp = newWritePool(clientID)
	wp.TokenLockInterface = &tokenLock{
		StartTime: creationDate,
		Duration:  common.ToTime(expiresAt).Sub(common.ToTime(creationDate)),
		Owner:     clientID,
	}
	wp.TokenPool.ID = writePoolKey(ssc.ID, allocationID)
	return
}

// lock tokens for write pool of transaction's client
func (ssc *StorageSmartContract) writePoolLock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// configurations

	var conf *writePoolConfig
	if conf, err = ssc.getWritePoolConfig(balances, true); err != nil {
		return "", common.NewError("write_pool_lock_failed",
			"can't get configs: "+err.Error())
	}

	// filter by configurations

	if t.Value < conf.MinLock {
		return "", common.NewError("write_pool_lock_failed",
			"insufficent amount to lock")
	}

	// write lock request & user balance

	var req writePoolRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}
	var balance state.Balance
	balance, err = balances.GetClientBalance(t.ClientID)

	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	if err == util.ErrValueNotPresent {
		return "", common.NewError("write_pool_lock_failed", "no tokens to lock")
	}

	if state.Balance(t.Value) > balance {
		return "", common.NewError("write_pool_lock_failed",
			"lock amount is greater than balance")
	}

	if req.AllocationID == "" {
		return "", common.NewError("write_pool_lock_failed",
			"missing allocation_id")
	}

	// user write pools

	var wp *writePool
	if wp, err = ssc.getWritePool(req.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	// lock more tokens

	if _, resp, err = wp.fill(t, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed",
			err.Error())
	}

	if err = wp.save(ssc.ID, req.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	return
}

// unlock tokens if expired
func (ssc *StorageSmartContract) writePoolUnlock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// the request

	var req writePoolRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	// write pool

	var wp *writePool
	if wp, err = ssc.getWritePool(req.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_lock_failed", err.Error())
	}

	if wp.ClientID != t.ClientID {
		return "", common.NewError("write_pool_lock_failed",
			"only owner can unlock the pool")
	}

	var transfer *state.Transfer
	transfer, resp, err = wp.EmptyPool(ssc.ID, t.ClientID,
		common.ToTime(t.CreationDate))
	if err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	// save pool
	if err = wp.save(ssc.ID, req.AllocationID, balances); err != nil {
		return "", common.NewError("write_pool_unlock_failed", err.Error())
	}

	return
}

//
// stat
//

func (ssc *StorageSmartContract) getWritePoolStat(wp *writePool, tp time.Time) (
	stat *writePoolStat, err error) {

	stat = new(writePoolStat)

	if err = stat.decode(wp.LockStats(tp)); err != nil {
		return nil, err
	}

	stat.ID = wp.ID
	stat.Locked = wp.IsLocked(tp)
	stat.Balance = wp.Balance

	return
}

// statistic for all locked tokens of a write pool
func (ssc *StorageSmartContract) getWritePoolStatHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = datastore.Key(params.Get("allocation_id"))
		wp           *writePool
	)
	if wp, err = ssc.getWritePool(allocationID, balances); err != nil {
		return
	}

	var (
		tp   = time.Now()
		stat *writePoolStat
	)
	if stat, err = ssc.getWritePoolStat(wp, tp); err != nil {
		return nil, common.NewError("write_pool_stats", err.Error())
	}

	return &stat, nil
}

//
// configure the write pool
//

// getWritePoolConfigBytes of a client
func (ssc *StorageSmartContract) getWritePoolConfigBytes(
	balances chainState.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(writePoolConfigKey(ssc.ID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

func getConfiguredWritePoolConfig() (conf *writePoolConfig) {
	conf = new(writePoolConfig)
	conf.MinLock = config.SmartContractConfig.GetInt64(
		"smart_contracts.storagesc.writepool.min_lock")
	return
}

func (ssc *StorageSmartContract) setupWritePoolConfig(
	balances chainState.StateContextI) (conf *writePoolConfig, err error) {

	conf = getConfiguredWritePoolConfig()
	_, err = balances.InsertTrieNode(writePoolConfigKey(ssc.ID), conf)
	if err != nil {
		return nil, err
	}
	return
}

// getWritePoolConfig
func (ssc *StorageSmartContract) getWritePoolConfig(
	balances chainState.StateContextI, setup bool) (
	conf *writePoolConfig, err error) {

	var confb []byte
	confb, err = ssc.getWritePoolConfigBytes(balances)
	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	conf = new(writePoolConfig)

	if err == util.ErrValueNotPresent {
		if !setup {
			return // value not present
		}
		return ssc.setupWritePoolConfig(balances)
	}

	if err = conf.Decode(confb); err != nil {
		return nil, err
	}
	return
}

func (ssc *StorageSmartContract) getWritePoolConfigHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var conf *writePoolConfig
	conf, err = ssc.getWritePoolConfig(balances, false)

	if err != nil && err != util.ErrValueNotPresent {
		return // unexpected error
	}

	// return configurations from sc.yaml not saving them
	if err == util.ErrValueNotPresent {
		return getConfiguredWritePoolConfig(), nil
	}

	return conf, nil // actual value
}

func (ssc *StorageSmartContract) writePoolUpdateConfig(
	t *transaction.Transaction, input []byte,
	balances chainState.StateContextI) (resp string, err error) {

	if t.ClientID != owner {
		return "", common.NewError("write_pool_update_config",
			"unauthorized access - only the owner can update the variables")
	}

	var conf, update *writePoolConfig
	if conf, err = ssc.getWritePoolConfig(balances, true); err != nil {
		return "", common.NewError("write_pool_update_config", err.Error())
	}

	update = new(writePoolConfig)
	if err = update.Decode(input); err != nil {
		return "", common.NewError("write_pool_update_config", err.Error())
	}

	if update.MinLock > 0 {
		conf.MinLock = update.MinLock
	}

	_, err = balances.InsertTrieNode(writePoolConfigKey(ssc.ID), conf)
	if err != nil {
		return "", common.NewError("write_pool_update_config", err.Error())
	}

	return string(conf.Encode()), nil
}
