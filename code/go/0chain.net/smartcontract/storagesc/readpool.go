package storagesc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// read pool lock configs

type readPoolConfig struct {
	MinLock       int64         `json:"min_lock"`
	MinLockPeriod time.Duration `json:"min_lock_period"`
	MaxLockPeriod time.Duration `json:"max_lock_period"`
}

func readPoolConfigKey(scKey string) datastore.Key {
	return datastore.Key(scKey + ":readpool")
}

func (rpc *readPoolConfig) getKey(scKey string) datastore.Key {
	return readPoolConfigKey(scKey)
}

func (rpc *readPoolConfig) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(rpc); err != nil {
		panic(err) // must not happens
	}
	return
}

func (rpc *readPoolConfig) Decode(b []byte) error {
	return json.Unmarshal(b, rpc)
}

// lock request

type lockRequest struct {
	Duration time.Duration `json:"duration"`
}

func (lr *lockRequest) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(lr); err != nil {
		panic(err) // must not happen
	}
	return
}

func (lr *lockRequest) decode(input []byte) error {
	return json.Unmarshal(input, lr)
}

// unlock request

type unlockRequest struct {
	PoolID datastore.Key `json:"pool_id"`
}

func (ur *unlockRequest) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(ur); err != nil {
		panic(err) // must not happen
	}
	return
}

func (ur *unlockRequest) decode(input []byte) error {
	return json.Unmarshal(input, ur)
}

// read pool (a locked tokens for a duration)

type readPool struct {
	*tokenpool.ZcnLockingPool `json:"pool"`
}

func newReadPool() *readPool {
	return &readPool{ZcnLockingPool: &tokenpool.ZcnLockingPool{}}
}

func (rp *readPool) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(rp); err != nil {
		panic(err) // must never happens
	}
	return
}

func (rp *readPool) decode(input []byte) (err error) {

	type readPoolJSON struct {
		Pool json.RawMessage `json:"pool"`
	}

	var readPoolVal readPoolJSON
	if err = json.Unmarshal(input, &readPoolVal); err != nil {
		return
	}

	if len(readPoolVal.Pool) == 0 {
		return // no data given
	}

	err = rp.ZcnLockingPool.Decode(readPoolVal.Pool, &tokenLock{})
	return
}

// readPools -- set of locked tokens for a duration

// readPools of a user
type readPools struct {
	ClientID datastore.Key               `json:"client_id"`
	Pools    map[datastore.Key]*readPool `json:"pools"`
}

func newReadPools(clientID datastore.Key) (rps *readPools) {
	rps = new(readPools)
	rps.ClientID = clientID
	rps.Pools = make(map[datastore.Key]*readPool)
	return
}

func (rps *readPools) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(rps); err != nil {
		panic(err) // must never happens
	}
	return
}

func (rps *readPools) Decode(input []byte) (err error) {
	var objMap map[string]json.RawMessage
	if err = json.Unmarshal(input, &objMap); err != nil {
		return
	}
	var cid, ok = objMap["client_id"]
	if ok {
		if err = json.Unmarshal(cid, &rps.ClientID); err != nil {
			return err
		}
	}
	var p json.RawMessage
	if p, ok = objMap["pools"]; ok {
		var rawMessagesPools map[string]json.RawMessage
		if err = json.Unmarshal(p, &rawMessagesPools); err != nil {
			return
		}
		for _, raw := range rawMessagesPools {
			var tempPool = newReadPool()
			if err = tempPool.decode(raw); err != nil {
				return
			}
			rps.addPool(tempPool)
		}
	}
	return
}

func readPoolsKey(scKey, clientID string) datastore.Key {
	return datastore.Key(scKey + ":readpool:" + clientID)
}

func (rps *readPools) getKey(scKey string) datastore.Key {
	return readPoolsKey(scKey, rps.ClientID)
}

func (rps *readPools) addPool(rp *readPool) (err error) {
	if _, ok := rps.Pools[rp.ID]; ok {
		return errors.New("user already has read pool")
	}
	rps.Pools[rp.ID] = rp
	return
}

func (rps *readPools) delPool(id datastore.Key) {
	delete(rps.Pools, id)
}

// stat

type readPoolStats struct {
	Stats []*readPoolStat `json:"stats"`
}

func (stats *readPoolStats) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(stats); err != nil {
		panic(err) // must never happens
	}
	return
}

func (stats *readPoolStats) decode(input []byte) error {
	return json.Unmarshal(input, stats)
}

func (stats *readPoolStats) addStat(stat *readPoolStat) {
	stats.Stats = append(stats.Stats, stat)
}

type readPoolStat struct {
	ID        datastore.Key    `json:"pool_id"`
	StartTime common.Timestamp `json:"start_time"`
	Duartion  time.Duration    `json:"duration"`
	TimeLeft  time.Duration    `json:"time_left"`
	Locked    bool             `json:"locked"`
	Balance   state.Balance    `json:"balance"`
}

func (stat *readPoolStat) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(stat); err != nil {
		panic(err) // must never happens
	}
	return
}

func (stat *readPoolStat) decode(input []byte) error {
	return json.Unmarshal(input, stat)
}

type tokenLock struct {
	StartTime common.Timestamp `json:"start_time"`
	Duration  time.Duration    `json:"duration"`
	Owner     datastore.Key    `json:"owner"`
}

func (tl tokenLock) IsLocked(entity interface{}) bool {
	if tm, ok := entity.(time.Time); ok {
		return tm.Sub(common.ToTime(tl.StartTime)) < tl.Duration
	}
	return true
}

func (tl tokenLock) LockStats(entity interface{}) []byte {
	if tm, ok := entity.(time.Time); ok {
		var stat readPoolStat
		stat.StartTime = tl.StartTime
		stat.Duartion = tl.Duration
		stat.TimeLeft = (tl.Duration - tm.Sub(common.ToTime(tl.StartTime)))
		stat.Locked = tl.IsLocked(tm)
		return stat.encode()
	}
	return nil
}

//
// smart contract methods
//

// getReadPoolsBytes of a client
func (ssc *StorageSmartContract) getReadPoolsBytes(clientID datastore.Key,
	balances chainState.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(readPoolsKey(ssc.ID, clientID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// getReadPools of current client
func (ssc *StorageSmartContract) getReadPools(clientID datastore.Key,
	balances chainState.StateContextI) (rps *readPools, err error) {

	var poolb []byte
	if poolb, err = ssc.getReadPoolsBytes(clientID, balances); err != nil {
		return
	}
	rps = newReadPools(clientID)
	err = rps.Decode(poolb)
	return
}

// newReadPool SC function creates new read pool for a client.
func (ssc *StorageSmartContract) newReadPool(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	_, err = ssc.getReadPoolsBytes(t.ClientID, balances)

	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("new_read_pool_failed", err.Error())
	}

	if err == nil {
		return "", common.NewError("new_read_pool_failed", "already exist")
	}

	var rps = newReadPools(t.ClientID)
	if _, err = balances.InsertTrieNode(rps.getKey(ssc.ID), rps); err != nil {
		return "", common.NewError("new_read_pool_failed", err.Error())
	}

	return string(rps.Encode()), nil
}

// lock tokens for read pool of transaction's client
func (ssc *StorageSmartContract) readPoolLock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// configs

	var conf *readPoolConfig
	if conf, err = ssc.getReadPoolConfig(balances, true); err != nil {
		return "", common.NewError("read_pool_lock_failed",
			"can't get configs: "+err.Error())
	}

	// user read pools

	var rps *readPools
	if rps, err = ssc.getReadPools(t.ClientID, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	// lock request & user balance

	var lr lockRequest
	if err = lr.decode(input); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}
	var balance state.Balance
	balance, err = balances.GetClientBalance(t.ClientID)

	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	if err == util.ErrValueNotPresent {
		return "", common.NewError("read_pool_lock_failed", "no tokens to lock")
	}

	if state.Balance(t.Value) > balance {
		return "", common.NewError("read_pool_lock_failed",
			"lock amount is greater than balance")
	}

	// filter by configs

	if t.Value < conf.MinLock {
		return "", common.NewError("read_pool_lock_failed",
			"insufficent amount to lock")
	}

	if lr.Duration < conf.MinLockPeriod {
		return "", common.NewError("read_pool_lock_failed",
			fmt.Sprintf("duration (%s) is shorter than min lock period (%s)",
				lr.Duration.String(), conf.MinLockPeriod.String()))
	}

	if lr.Duration > conf.MaxLockPeriod {
		return "", common.NewError("read_pool_lock_failed",
			fmt.Sprintf("duration (%s) is longer than max lock period (%v)",
				lr.Duration.String(), conf.MaxLockPeriod.String()))
	}

	// lock

	var rp = newReadPool()
	rp.TokenLockInterface = &tokenLock{
		StartTime: t.CreationDate,
		Duration:  lr.Duration,
		Owner:     t.ClientID,
	}

	var transfer *state.Transfer
	if transfer, resp, err = rp.DigPool(t.Hash, t); err != nil {
		return "", common.NewError("read_pool_lock_failed",
			err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	if err = rps.addPool(rp); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	_, err = balances.InsertTrieNode(rps.getKey(ssc.ID), rps)
	if err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	return
}

// unlock tokens if expired
func (ssc *StorageSmartContract) readPoolUnlock(t *transaction.Transaction,
	input []byte, balances chainState.StateContextI) (resp string, err error) {

	// user read pools

	var rps *readPools
	if rps, err = ssc.getReadPools(t.ClientID, balances); err != nil {
		return "", common.NewError("read_pool_lock_failed", err.Error())
	}

	// the request

	var (
		transfer *state.Transfer
		req      unlockRequest
	)

	if err = req.decode(input); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	var pool, ok = rps.Pools[req.PoolID]
	if !ok {
		return "", common.NewError("read_pool_unlock_failed", "pool not found")
	}

	transfer, resp, err = pool.EmptyPool(ssc.ID, t.ClientID,
		common.ToTime(t.CreationDate))
	if err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	// save pools
	rps.delPool(pool.ID)
	if _, err = balances.InsertTrieNode(rps.getKey(ssc.ID), rps); err != nil {
		return "", common.NewError("read_pool_unlock_failed", err.Error())
	}

	return
}

//
// stat
//

func (ssc *StorageSmartContract) getPoolStat(rp *readPool, tp time.Time) (
	stat *readPoolStat, err error) {

	stat = new(readPoolStat)

	if err = stat.decode(rp.LockStats(tp)); err != nil {
		return nil, err
	}

	stat.ID = rp.ID
	stat.Locked = rp.IsLocked(tp)
	stat.Balance = rp.Balance

	return
}

// statistic for all locked tokens of the read pool
func (ssc *StorageSmartContract) getReadPoolsStatsHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID = datastore.Key(params.Get("client_id"))
		rps      *readPools
	)
	if rps, err = ssc.getReadPools(clientID, balances); err != nil {
		return
	}

	if len(rps.Pools) == 0 {
		return nil, common.NewError("read_pool_stats", "no pools exist")
	}

	var (
		tp    = time.Now()
		stats readPoolStats
	)

	for _, rp := range rps.Pools {
		stat, err := ssc.getPoolStat(rp, tp)
		if err != nil {
			return nil, common.NewError("read_pool_stats", err.Error())
		}
		stats.addStat(stat)
	}

	return &stats, nil
}

//
// configure the read pool
//

// getReadPoolConfigBytes of a client
func (ssc *StorageSmartContract) getReadPoolConfigBytes(
	balances chainState.StateContextI) (b []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(readPoolConfigKey(ssc.ID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

func getConfiguredReadPoolConfig() (conf *readPoolConfig) {
	conf = new(readPoolConfig)
	conf.MinLockPeriod = config.SmartContractConfig.GetDuration(
		"smart_contracts.storagesc.readpool.min_lock_period")
	conf.MaxLockPeriod = config.SmartContractConfig.GetDuration(
		"smart_contracts.storagesc.readpool.max_lock_period")
	conf.MinLock = config.SmartContractConfig.GetInt64(
		"smart_contracts.storagesc.readpool.min_lock")
	return
}

func (ssc *StorageSmartContract) setupReadPoolConfig(
	balances chainState.StateContextI) (rpc *readPoolConfig, err error) {

	rpc = getConfiguredReadPoolConfig()
	_, err = balances.InsertTrieNode(readPoolConfigKey(ssc.ID), rpc)
	if err != nil {
		return nil, err
	}
	return
}

// getReadPoolConfig
func (ssc *StorageSmartContract) getReadPoolConfig(
	balances chainState.StateContextI, setup bool) (
	conf *readPoolConfig, err error) {

	var confb []byte
	confb, err = ssc.getReadPoolConfigBytes(balances)
	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	conf = new(readPoolConfig)

	if err == util.ErrValueNotPresent {
		if !setup {
			return // value not present
		}
		return ssc.setupReadPoolConfig(balances)
	}

	if err = conf.Decode(confb); err != nil {
		return nil, err
	}
	return
}

func (ssc *StorageSmartContract) getReadPoolsConfigHandler(ctx context.Context,
	params url.Values, balances chainState.StateContextI) (
	resp interface{}, err error) {

	var conf *readPoolConfig
	conf, err = ssc.getReadPoolConfig(balances, false)

	if err != nil && err != util.ErrValueNotPresent {
		return // unexpected error
	}

	// return configurations from sc.yaml not saving them
	if err == util.ErrValueNotPresent {
		return getConfiguredReadPoolConfig(), nil
	}

	return conf, nil // actual value
}

func (ssc *StorageSmartContract) readPoolUpdateConfig(
	t *transaction.Transaction, input []byte,
	balances chainState.StateContextI) (resp string, err error) {

	if t.ClientID != owner {
		return "", common.NewError("read_pool_update_config",
			"unauthorized access - only the owner can update the variables")
	}

	var conf, update *readPoolConfig
	if conf, err = ssc.getReadPoolConfig(balances, true); err != nil {
		return "", common.NewError("read_pool_update_config", err.Error())
	}

	update = new(readPoolConfig)
	if err = update.Decode(input); err != nil {
		return "", common.NewError("read_pool_update_config", err.Error())
	}

	if update.MinLock > 0 {
		conf.MinLock = update.MinLock
	}

	if update.MinLockPeriod > 0 {
		conf.MinLockPeriod = update.MinLockPeriod
	}

	if update.MaxLockPeriod > 0 {
		conf.MaxLockPeriod = update.MaxLockPeriod
	}

	_, err = balances.InsertTrieNode(readPoolConfigKey(ssc.ID), conf)
	if err != nil {
		return "", common.NewError("read_pool_update_config", err.Error())
	}

	return string(conf.Encode()), nil
}
