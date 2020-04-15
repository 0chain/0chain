package vestingsc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

// lock / unlock request
type lockRequest struct {
	PoolID string `json:"pool_id"`
}

func (lr *lockRequest) encode() (b []byte) {
	var err error
	if b, err = json.Marshal(lr); err != nil {
		panic(err) // must not happen
	}
	return
}

func (lr *lockRequest) decode(b []byte) error {
	return json.Unmarshal(b, lr)
}

// add request
type addRequest struct {
	Description  string           `json:"description"`  // allow empty
	StartTime    common.Timestamp `json:"start_time"`   //
	Duration     time.Duration    `json:"duration"`     //
	Friquency    time.Duration    `json:"friquency"`    //
	Destinations []datastore.Key  `json:"destinations"` //
	Amount       state.Balance    `json:"amount"`       //
}

func (ar *addRequest) decode(b []byte) error {
	return json.Unmarshal(b, ar)
}

func toSeconds(dur time.Duration) common.Timestamp {
	return common.Timestamp(dur / time.Second)
}

// validate the addRequest
func (ar *addRequest) validate(now common.Timestamp, conf *config) (err error) {
	if ar.StartTime == 0 {
		ar.StartTime = now
	}
	switch {
	case len(ar.Description) > conf.MaxDescriptionLength:
		return errors.New("entry description is too long")
	case ar.StartTime < now:
		return errors.New("vesting starts before now")
	case ar.Duration < conf.MinDuration:
		return errors.New("vesting duration is too short")
	case ar.Duration > conf.MaxDuration:
		return errors.New("vesting duration is too long")
	case ar.Friquency < conf.MinFriquency:
		return errors.New("vesting friquency is too low")
	case ar.Friquency > conf.MaxFriquency:
		return errors.New("vesting friquency is too high")
	case len(ar.Destinations) == 0:
		return errors.New("no destinations")
	case len(ar.Destinations) > conf.MaxDestinations:
		return errors.New("too many destinations")
	}
	return
}

func poolKey(vscKey, poolID datastore.Key) datastore.Key {
	return vscKey + ":vestingpool:" + poolID
}

type vestingPool struct {
	*tokenpool.ZcnPool `json:"pool"`

	Description  string           `json:"description"`
	StartTime    common.Timestamp `json:"start_time"`
	ExpireAt     common.Timestamp `json:"expire_at"`
	Friquency    time.Duration    `json:"friquency"`
	Destinations []datastore.Key  `json:"destinations"`
	Amount       state.Balance    `json:"amount"`
	ClientID     datastore.Key    `json:"client_id"`

	// Last tokens transfer.
	Last common.Timestamp `json:"last"`
}

// newVestingPool returns new empty uninitialized vesting pool.
func newVestingPool() (vp *vestingPool) {
	vp = new(vestingPool)
	vp.ZcnPool = new(tokenpool.ZcnPool)
	return
}

// newVestingPoolFromRequest is the same as newVestingPool, but other fields
// set by the request. The request must be validated before.
func newVestingPoolFromReqeust(clientID datastore.Key, ar *addRequest) (
	vp *vestingPool) {

	vp = newVestingPool()
	vp.ClientID = clientID

	vp.Description = ar.Description
	vp.StartTime = ar.StartTime
	vp.ExpireAt = ar.StartTime + toSeconds(ar.Duration)
	vp.Friquency = ar.Friquency
	vp.Destinations = ar.Destinations
	vp.Amount = ar.Amount
	return
}

// Encode the vesting pool from JSON value. Implements
// required util.Serializale interface.
func (vp *vestingPool) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(vp); err != nil {
		panic(err) // must never happen
	}
	return
}

// Decode the vesting pool to JSON. Implements
// required util.Serializale interface.
func (vp *vestingPool) Decode(b []byte) error {
	return json.Unmarshal(b, vp)
}

//
// vest
//

// vestTo given client given value
func (vp *vestingPool) vestTo(vscKey string, toClientID datastore.Key,
	value state.Balance, balances chainstate.StateContextI) (
	resp string, err error) {

	var transfer *state.Transfer
	transfer, resp, err = vp.DrainPool(vscKey, toClientID, value, nil)
	if err != nil {
		return "", fmt.Errorf("draining vesting pool: %v", err)
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", fmt.Errorf("adding vesting pool transfer: %v", err)
	}

	return
}

// vest moves configured amount of tokens to all destinations of the pool
func (vp *vestingPool) vest(vscKey string, now common.Timestamp,
	balances chainstate.StateContextI) (_ string, err error) {

	var total = vp.Amount * state.Balance(len(vp.Destinations))

	if vp.Balance < total {
		return "", errors.New("not enough tokens")
	}

	var (
		rb    strings.Builder
		oresp string
	)
	rb.WriteByte('[')
	for i, dest := range vp.Destinations {
		oresp, err = vp.vestTo(vscKey, dest, vp.Amount, balances)
		if err != nil {
			return // vesting error
		}
		if i > 0 {
			rb.WriteByte(',')
		}
		rb.WriteString(oresp)
	}
	rb.WriteByte(']')

	vp.Last = now

	return rb.String(), nil // success
}

//
// lock / unlock
//

func (vp *vestingPool) fill(t *transaction.Transaction,
	balances chainstate.StateContextI) (resp string, err error) {

	var transfer *state.Transfer
	if transfer, resp, err = vp.FillPool(t); err != nil {
		return
	}
	err = balances.AddTransfer(transfer)
	return
}

func (vp *vestingPool) empty(t *transaction.Transaction,
	balances chainstate.StateContextI) (resp string, err error) {

	if vp.Balance == 0 {
		return "", errors.New("nothing to unlock")
	}

	var transfer *state.Transfer
	transfer, resp, err = vp.EmptyPool(t.ToClientID, t.ClientID, nil)
	if err != nil {
		return "", fmt.Errorf("emptying vesting pool: %v", err)
	}
	if err = balances.AddTransfer(transfer); err != nil {
		return "", fmt.Errorf("adding transfer vesting_pool->client: %v", err)
	}

	return
}

func (vp *vestingPool) save(balances chainstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(vp.ID, vp)
	return
}

//
// info (stat)
//

func (vp *vestingPool) info() (i *info) {
	i = new(info)

	i.ID = vp.ID
	i.Balance = vp.Balance
	i.Description = vp.Description
	i.StartTime = vp.StartTime
	i.ExpireAt = vp.ExpireAt
	i.Friquency = vp.Friquency
	i.Destinations = vp.Destinations
	i.Amount = vp.Amount
	i.ClientID = vp.ClientID

	i.Last = vp.Last
	return
}

type info struct {
	ID           datastore.Key    `json:"pool_id"`
	Balance      state.Balance    `json:"balance"`
	Description  string           `json:"description"`
	StartTime    common.Timestamp `json:"start_time"`
	ExpireAt     common.Timestamp `json:"expire_at"`
	Friquency    time.Duration    `json:"friquency"`
	Destinations []datastore.Key  `json:"destinations"`
	Amount       state.Balance    `json:"amount"`
	ClientID     datastore.Key    `json:"client_id"`
	Last         common.Timestamp `json:"last"`
}

//
// helpers
//

func (vsc *VestingSmartContract) getPoolBytes(poolID datastore.Key,
	balances chainstate.StateContextI) (_ []byte, err error) {

	var val util.Serializable
	if val, err = balances.GetTrieNode(poolID); err != nil {
		return
	}

	return val.Encode(), nil
}

func (vsc *VestingSmartContract) getPool(poolID datastore.Key,
	balances chainstate.StateContextI) (vp *vestingPool, err error) {

	var poolb []byte
	if poolb, err = vsc.getPoolBytes(poolID, balances); err != nil {
		return
	}

	vp = newVestingPool()
	if err = vp.Decode(poolb); err != nil {
		return nil, err
	}

	return
}

func (vsc *VestingSmartContract) checkFill(t *transaction.Transaction,
	balances chainstate.StateContextI) (err error) {

	var balance state.Balance
	balance, err = balances.GetClientBalance(t.ClientID)

	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	if err == util.ErrValueNotPresent {
		return errors.New("no tokens to lock")
	}

	if state.Balance(t.Value) > balance {
		return errors.New("lock amount is greater than balance")
	}

	return
}

//
// transaction outputs
//

type Out struct {
	Function string          `json:"function"`
	Output   json.RawMessage `json:"output"`
}

func (o *Out) toJSON() string {
	var b, err = json.Marshal(o)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func addOut(vp *vestingPool) string {
	var o Out
	o.Function = "add"
	o.Output = json.RawMessage(vp.Encode())
	return o.toJSON()
}

func delOut(dr *lockRequest) string {
	var o Out
	o.Function = "delete"
	o.Output = json.RawMessage(dr.encode())
	return o.toJSON()
}

func lockOut(resp string) string {
	var o Out
	o.Function = "lock"
	o.Output = json.RawMessage(resp)
	return o.toJSON()
}

func unlockOut(resp string) string {
	var o Out
	o.Function = "unlock"
	o.Output = json.RawMessage(resp)
	return o.toJSON()
}

func triggerOut(resp string) string {
	var o Out
	o.Function = "trigger"
	o.Output = json.RawMessage(resp)
	return o.toJSON()
}

//
// SC functions
//

func (vsc *VestingSmartContract) add(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var ar addRequest
	if err = ar.decode(input); err != nil {
		return "", common.NewError("create_vesting_pool_failed",
			"malformed request: "+err.Error())
	}

	var conf *config
	if conf, err = vsc.getConfig(balances, true); err != nil {
		return "", common.NewError("create_vesting_pool_failed",
			"can't get SC configurations: "+err.Error())
	}

	if err = ar.validate(t.CreationDate, conf); err != nil {
		return "", common.NewError("create_vesting_pool_failed",
			"invalid request: "+err.Error())
	}

	if t.ClientID == "" {
		return "", common.NewError("create_vesting_pool_failed",
			"empty client_id of transaction")
	}

	var vp = newVestingPoolFromReqeust(t.ClientID, &ar)
	vp.ID = poolKey(vsc.ID, t.Hash) // set ID by this transaction

	// lock tokens if provided

	if t.Value > 0 {
		if state.Balance(t.Value) < conf.MinLock {
			return "", common.NewError("create_vesting_pool_failed",
				"insufficient amount to lock")
		}
		if err = vsc.checkFill(t, balances); err != nil {
			return "", common.NewError("create_vesting_pool_failed",
				"can't fill pool: "+err.Error())
		}
		if _, err = vp.fill(t, balances); err != nil {
			return "", common.NewError("create_vesting_pool_failed",
				"can't fill pool: "+err.Error())
		}
	}

	var cp *clientPools
	if cp, err = vsc.getOrCreateClientPools(t.ClientID, balances); err != nil {
		return "", common.NewError("create_vesting_pool_failed",
			"unexpected error: "+err.Error())
	}

	cp.add(vp.ID)
	if err = cp.save(vsc.ID, t.ClientID, balances); err != nil {
		return "", common.NewError("create_vesting_pool_failed",
			"can't save client's pools list: "+err.Error())
	}

	if err = vp.save(balances); err != nil {
		return "", common.NewError("create_vesting_pool_failed",
			"can't save pool: "+err.Error())
	}

	if err = vsc.addTxnToVestingLog(t.Hash, balances); err != nil {
		return "", common.NewError("create_vesting_pool_failed",
			"saving transaction in log: "+err.Error())
	}

	return addOut(vp), nil
}

func (vsc *VestingSmartContract) delete(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var dr lockRequest
	if err = dr.decode(input); err != nil {
		return "", common.NewError("delete_vesting_pool_failed",
			"invalid request: "+err.Error())
	}

	if dr.PoolID == "" {
		return "", common.NewError("delete_vesting_pool_failed",
			"invalid request: missing pool id")
	}

	if t.ClientID == "" {
		return "", common.NewError("delete_vesting_pool_failed",
			"empty client id of transaction")
	}

	var vp *vestingPool
	if vp, err = vsc.getPool(dr.PoolID, balances); err != nil {
		return "", common.NewError("delete_vesting_pool_failed",
			"can't get pool: "+err.Error())
	}

	if vp.ClientID != t.ClientID {
		return "", common.NewError("delete_vesting_pool_failed",
			"only pool owner can do that")
	}

	if vp.Balance > 0 {
		if _, err = vp.empty(t, balances); err != nil {
			return "", common.NewError("delete_vesting_pool_failed",
				"emptying pool: "+err.Error())
		}
	}

	var cp *clientPools
	if cp, err = vsc.getOrCreateClientPools(t.ClientID, balances); err != nil {
		return "", common.NewError("delete_vesting_pool_failed",
			"unexpected error: "+err.Error())
	}

	cp.remove(vp.ID)

	if len(cp.Pools) == 0 {
		_, err = balances.DeleteTrieNode(clientPoolsKey(vsc.ID, t.ClientID))
		if err != nil {
			return "", common.NewError("delete_vesting_pool_failed",
				"can't delete client's pools list: "+err.Error())
		}
	} else {
		if err = cp.save(vsc.ID, t.ClientID, balances); err != nil {
			return "", common.NewError("delete_vesting_pool_failed",
				"can't save client's pools list: "+err.Error())
		}
	}

	if _, err = balances.DeleteTrieNode(vp.ID); err != nil {
		return "", common.NewError("delete_vesting_pool_failed",
			"can't delete vesting pool: "+err.Error())
	}

	if err = vsc.addTxnToVestingLog(t.Hash, balances); err != nil {
		return "", common.NewError("delete_vesting_pool_failed",
			"saving transaction in log: "+err.Error())
	}

	return delOut(&dr), nil
}

func (vsc *VestingSmartContract) lock(t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (resp string, err error) {

	var lr lockRequest
	if err = lr.decode(input); err != nil {
		return "", common.NewError("lock_vesting_pool_failed",
			"invalid request: "+err.Error())
	}

	if lr.PoolID == "" {
		return "", common.NewError("lock_vesting_pool_failed",
			"invalid request: missing pool id")
	}

	var vp *vestingPool
	if vp, err = vsc.getPool(lr.PoolID, balances); err != nil {
		return "", common.NewError("lock_vesting_pool_failed",
			"can't get pool: "+err.Error())
	}

	if vp.ClientID != t.ClientID {
		return "", common.NewError("lock_vesting_pool_failed",
			"only owner can lock more tokens to the pool")
	}

	var conf *config
	if conf, err = vsc.getConfig(balances, true); err != nil {
		return "", common.NewError("lock_vesting_pool_failed",
			"can't get SC configurations: "+err.Error())
	}

	if state.Balance(t.Value) < conf.MinLock {
		return "", common.NewError("lock_vesting_pool_failed",
			"insufficient amount to lock")
	}

	if err = vsc.checkFill(t, balances); err != nil {
		return "", common.NewError("lock_vesting_pool_failed", err.Error())
	}

	if resp, err = vp.fill(t, balances); err != nil {
		return "", common.NewError("lock_vesting_pool_failed",
			"filling pool: "+err.Error())
	}

	if err = vp.save(balances); err != nil {
		return "", common.NewError("lock_vesting_pool_failed",
			"saving pool: "+err.Error())
	}

	return lockOut(resp), nil
}

func (vsc *VestingSmartContract) unlock(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var ur lockRequest
	if err = ur.decode(input); err != nil {
		return "", common.NewError("unlock_vesting_pool_failed",
			"invalid request: "+err.Error())
	}

	if ur.PoolID == "" {
		return "", common.NewError("unlock_vesting_pool_failed",
			"invalid request: missing pool id")
	}

	var vp *vestingPool
	if vp, err = vsc.getPool(ur.PoolID, balances); err != nil {
		return "", common.NewError("unlock_vesting_pool_failed",
			"can't get pool: "+err.Error())
	}

	if vp.ClientID != t.ClientID {
		return "", common.NewError("unlock_vesting_pool_failed",
			"only owner can unlock tokens from the pool")
	}

	if resp, err = vp.empty(t, balances); err != nil {
		return "", common.NewError("unlock_vesting_pool_failed",
			"draining pool: "+err.Error())
	}

	if err = vp.save(balances); err != nil {
		return "", common.NewError("unlock_vesting_pool_failed",
			"saving pool: "+err.Error())
	}

	return unlockOut(resp), nil
}

//
// function triggered by server
//

type triggerResp struct {
	PoolID  string          `json:"pool_id"` //
	Vesting json.RawMessage `json:"vesting"` //
}

func (tr *triggerResp) toJSON() string {
	var b, err = json.Marshal(tr)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// trigger next vesting and return all transfers in transaction's response
func (vsc *VestingSmartContract) trigger(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var conf *config
	if conf, err = vsc.getConfig(balances, true); err != nil {
		return "", common.NewError("trigger_vesting_pool_failed",
			"can't get config: "+err.Error())
	}

	if !conf.isValidTrigger(t.ClientID) {
		return "", common.NewError("trigger_vesting_pool_failed",
			"not allowed for this client")
	}

	var tr lockRequest
	if err = tr.decode(input); err != nil {
		return "", common.NewError("trigger_vesting_pool_failed",
			"invalid request: "+err.Error())
	}

	if tr.PoolID == "" {
		return "", common.NewError("trigger_vesting_pool_failed",
			"invalid request: missing pool id")
	}

	var vp *vestingPool
	if vp, err = vsc.getPool(tr.PoolID, balances); err != nil {
		return "", common.NewError("trigger_vesting_pool_failed",
			"can't get pool: "+err.Error())
	}

	// next (this) time to vest
	var next common.Timestamp
	if vp.Last == 0 {
		next = vp.StartTime + toSeconds(vp.Friquency)
	} else {
		next = vp.Last + toSeconds(vp.Friquency)
	}

	if next > vp.ExpireAt {
		return "", common.NewError("trigger_vesting_pool_failed",
			"expired pool")
	}

	if next > t.CreationDate {
		return "", common.NewError("trigger_vesting_pool_failed",
			"early vesting")
	}

	// the time has come and is before the expire_at

	// vest
	if resp, err = vp.vest(vsc.ID, next, balances); err != nil {
		return "", common.NewError("trigger_vesting_pool_failed",
			"vesting: "+err.Error())
	}

	if err = vp.save(balances); err != nil {
		return "", common.NewError("trigger_vesting_pool_failed",
			"saving pool: "+err.Error())
	}

	// build transaction response

	var trsp triggerResp
	trsp.PoolID = tr.PoolID
	trsp.Vesting = json.RawMessage(resp)

	return triggerOut(trsp.toJSON()), nil
}

//
// REST handlers
//

func (vsc *VestingSmartContract) getPoolInfoHandler(ctx context.Context,
	params url.Values, balances chainstate.StateContextI) (
	resp interface{}, err error) {

	var (
		poolID = datastore.Key(params.Get("pool_id"))
		vp     *vestingPool
	)

	if vp, err = vsc.getPool(poolID, balances); err != nil {
		return
	}

	return vp.info(), nil
}
