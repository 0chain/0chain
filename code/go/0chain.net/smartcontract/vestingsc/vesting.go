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

//
// lock, unlock, trigger, delete a pool
//

type poolRequest struct {
	PoolID string `json:"pool_id"`
}

func (lr *poolRequest) decode(b []byte) error {
	return json.Unmarshal(b, lr)
}

//
// a destination
//

type destination struct {
	ID     datastore.Key    `json:"id"`     // destination ID
	Amount state.Balance    `json:"amount"` // amount to vest for the destination
	Last   common.Timestamp `json:"last"`   // last tokens transfer
}

func (d *destination) setLast(last common.Timestamp) {
	d.Last = last
}

func (d *destination) want(now, full common.Timestamp) (amount state.Balance) {
	var (
		period = now - d.Last
		ratio  = float64(period) / float64(full)
	)
	if period <= 0 {
		return // zero
	}
	return state.Balance(float64(d.Amount) * ratio)
}

//
// destinations of a pool
//

type destinations []*destination

func (ds destinations) setLast(last common.Timestamp) {
	for _, d := range ds {
		d.setLast(last)
	}
}

// want balance for all destinations for given time
func (ds destinations) want(now, full common.Timestamp) (
	amount state.Balance) {

	for _, d := range ds {
		amount += d.want(now, full)
	}
	return
}

//
// add (create) pool request
//

type addRequest struct {
	Description  string           `json:"description,omitempty"` // allow empty
	StartTime    common.Timestamp `json:"start_time"`            //
	Duration     time.Duration    `json:"duration"`              //
	Destinations destinations     `json:"destinations"`          //
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
	case len(ar.Destinations) == 0:
		return errors.New("no destinations")
	case len(ar.Destinations) > conf.MaxDestinations:
		return errors.New("too many destinations")
	}
	return
}

//
// vesting pool
//

func poolKey(vscKey, poolID datastore.Key) datastore.Key {
	return vscKey + ":vestingpool:" + poolID
}

type vestingPool struct {
	*tokenpool.ZcnPool `json:"pool"`

	Description  string           `json:"description"`  //
	StartTime    common.Timestamp `json:"start_time"`   //
	ExpireAt     common.Timestamp `json:"expire_at"`    //
	Destinations destinations     `json:"destinations"` //
	ClientID     datastore.Key    `json:"client_id"`    // the pool owner
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
	vp.Destinations = ar.Destinations
	vp.Destinations.setLast(vp.StartTime)
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

func checkFill(t *transaction.Transaction, balances chainstate.StateContextI) (
	err error) {

	var balance state.Balance
	balance, err = balances.GetClientBalance(t.ClientID)

	if err != nil && err != util.ErrValueNotPresent {
		return // unexpected error
	}

	if err == util.ErrValueNotPresent {
		return errors.New("no tokens to lock")
	}

	if state.Balance(t.Value) > balance {
		return errors.New("lock amount is greater than balance")
	}

	return
}

// fill the pool by client
func (vp *vestingPool) fill(t *transaction.Transaction,
	balances chainstate.StateContextI) (resp string, err error) {

	if err = checkFill(t, balances); err != nil {
		return
	}

	var transfer *state.Transfer
	if transfer, resp, err = vp.FillPool(t); err != nil {
		return
	}
	err = balances.AddTransfer(transfer)
	return
}

func (vp *vestingPool) left(now common.Timestamp) (left state.Balance) {

	var (
		end  = vp.ExpireAt
		full = end - vp.StartTime
	)

	if now > end {
		now = end
	}

	var want = vp.Destinations.want(now, full)

	if vp.Balance >= want {
		return vp.Balance - want
	}

	return // zero
}

func (vp *vestingPool) earned(id datastore.Key, now common.Timestamp) (
	found *destination, earned state.Balance, err error) {

	if vp.Balance == 0 {
		return nil, 0, errors.New("empty pool")
	}

	var (
		end  = vp.ExpireAt
		full = end - vp.StartTime
	)

	if now > end {
		now = end
	}

	var total state.Balance // total wanted

	for _, d := range vp.Destinations {
		var want = d.want(now, full)
		total += want
		if d.ID == id {
			earned, found = want, d
		}
	}

	if found == nil {
		return nil, 0, fmt.Errorf("destinations %q not found in pool", id)
	}

	if vp.Balance >= total {
		return // the pool has enough tokens to vest all wanted
	}

	// not enough tokens, recalculate

	var ratio = float64(vp.Balance) / float64(total)
	earned = state.Balance(float64(earned) * ratio) // truncated

	return // based on tokens left
}

type earn struct {
	d      *destination
	earned state.Balance
}

func (vp *vestingPool) earns(now common.Timestamp) (es []earn, err error) {

	if vp.Balance == 0 {
		return nil, errors.New("empty pool")
	}

	var (
		end  = vp.ExpireAt
		full = end - vp.StartTime
	)

	if now > end {
		now = end
	}

	var total state.Balance // total wanted

	es = make([]earn, 0, len(vp.Destinations))

	for _, d := range vp.Destinations {
		var want = d.want(now, full)
		total += want
		es = append(es, earn{d, want})
	}

	if vp.Balance >= total {
		return // the pool has enough tokens to vest all wanted
	}

	// not enough tokens, recalculate

	var ratio = float64(vp.Balance) / float64(total)
	for i := range es {
		es[i].earned = state.Balance(float64(es[i].earned) * ratio)
	}

	return // based on tokens left
}

func (vp *vestingPool) moveToDest(vscKey datastore.Key, d *destination,
	value state.Balance, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	if value == 0 {
		d.setLast(now)
		return
	}

	var transfer *state.Transfer
	transfer, resp, err = vp.DrainPool(vscKey, d.ID, value, nil)
	if err != nil {
		return "", fmt.Errorf("vesting destination: %v", err)
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", fmt.Errorf("adding transfer vesting_pool->destination: %v",
			err)
	}

	d.setLast(now) // update the last

	return
}

// vest unlocks tokens for a destination (and by the destination)
func (vp *vestingPool) vest(t *transaction.Transaction,
	balances chainstate.StateContextI) (resp string, err error) {

	var (
		d      *destination
		earned state.Balance
	)

	d, earned, err = vp.earned(t.ClientID, t.CreationDate)
	if err != nil {
		return // error
	}

	var now = t.CreationDate
	if now > vp.ExpireAt {
		now = vp.ExpireAt
	}

	return vp.moveToDest(t.ToClientID, d, earned, now, balances)
}

// move tokens to destinations' wallets
func (vp *vestingPool) trigger(t *transaction.Transaction,
	balances chainstate.StateContextI) (resp string, err error) {

	var es []earn
	if es, err = vp.earns(t.CreationDate); err != nil {
		return
	}

	var now = t.CreationDate
	if now > vp.ExpireAt {
		now = vp.ExpireAt
	}

	var (
		sb   strings.Builder
		drsp string
		i    int
		e    earn
	)

	sb.WriteByte('[')
	for i, e = range es {
		drsp, err = vp.moveToDest(t.ToClientID, e.d, e.earned, now, balances)
		if err != nil {
			return
		}
		if drsp == "" {
			continue
		}
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(drsp)
		i++
	}
	sb.WriteByte(']')

	return sb.String(), nil
}

// unlock all tokens left to owner
func (vp *vestingPool) drain(t *transaction.Transaction,
	balances chainstate.StateContextI) (resp string, err error) {

	var left = vp.left(t.CreationDate)
	if left == 0 {
		return "", errors.New("nothing to unlock")
	}

	var transfer *state.Transfer
	transfer, resp, err = vp.DrainPool(t.ToClientID, t.ClientID, left, nil)
	if err != nil {
		return "", fmt.Errorf("draining vesting pool: %v", err)
	}
	if err = balances.AddTransfer(transfer); err != nil {
		return "", fmt.Errorf("adding transfer vesting_pool->client: %v", err)
	}

	return
}

// unlock all tokens to owner
func (vp *vestingPool) empty(t *transaction.Transaction,
	balances chainstate.StateContextI) (resp string, err error) {

	var transfer *state.Transfer
	transfer, resp, err = vp.EmptyPool(t.ToClientID, t.ClientID, nil)
	if err != nil {
		return "", fmt.Errorf("draining vesting pool: %v", err)
	}
	if err = balances.AddTransfer(transfer); err != nil {
		return "", fmt.Errorf("adding transfer vesting_pool->client: %v", err)
	}

	return
}

// save the pool
func (vp *vestingPool) save(balances chainstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(vp.ID, vp)
	return
}

//
// info (stat)
//

func (vp *vestingPool) info(now common.Timestamp) (i *info) {
	i = new(info)

	i.ID = vp.ID
	i.Balance = vp.Balance
	i.Left = vp.left(now)
	i.Description = vp.Description
	i.StartTime = vp.StartTime
	i.ExpireAt = vp.ExpireAt

	var es, err = vp.earns(now)
	if err != nil {
		es = make([]earn, 0, len(vp.Destinations))
		for _, d := range vp.Destinations {
			es = append(es, earn{d: d})
		}
	}

	var dinfos = make([]*destInfo, 0, len(vp.Destinations))
	for _, e := range es {
		dinfos = append(dinfos, &destInfo{
			ID:     e.d.ID,
			Wanted: e.d.Amount,
			Earned: e.earned,
			Last:   e.d.Last,
		})
	}

	i.Destinations = dinfos
	i.ClientID = vp.ClientID
	return
}

type destInfo struct {
	ID     datastore.Key    `json:"id"`     // identifier
	Wanted state.Balance    `json:"wanted"` // wanted amount for entire period
	Earned state.Balance    `json:"earned"` // can unlock
	Last   common.Timestamp `json:"last"`   // last time unlocked
}

type info struct {
	ID           datastore.Key    `json:"pool_id"`      // pool ID
	Balance      state.Balance    `json:"balance"`      // real pool balance
	Left         state.Balance    `json:"left"`         // owner can unlock
	Description  string           `json:"description"`  // description
	StartTime    common.Timestamp `json:"start_time"`   // from
	ExpireAt     common.Timestamp `json:"expire_at"`    // until
	Destinations []*destInfo      `json:"destinations"` // receivers
	ClientID     datastore.Key    `json:"client_id"`    // owner
}

/*

add ->                                                       [+]
	- fill pool
	- set last to start time
lock ->                                                      [+]
	- add tokens to pool
unlock (owner) ->                                            [+]
	-> calculate destinations amount
	-> calculate pool amount left
	-> move the left to owner
unlock (destination) ->                                      [+]
	-> calculate destinations amount
	-> calculate pool amount left
	-> move destination amount to destination wallet
delete ->                                                    [+]
	-> calculate destinations amount
	-> calculate pool amount left
	-> move destinations amount to destinations
	-> move left to owner
	-> delete pool
trigger ->                                                   [+]
	-> calculate destinations amount
	-> calculate pool amount left
	-> move destinations amount to destinations
*/

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
	if conf, err = getConfig(); err != nil {
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

	return string(vp.Encode()), nil
}

func (vsc *VestingSmartContract) delete(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var dr poolRequest
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
			"only pool owner can delete the pool")
	}

	// move tokens to destinations
	if vp.Balance > 0 {
		if _, err = vp.trigger(t, balances); err != nil {
			return "", common.NewError("delete_vesting_pool_failed",
				"moving tokens to destinations: "+err.Error())
		}
	}

	// move left to owner
	if vp.Balance > 0 {
		if _, err = vp.empty(t, balances); err != nil {
			return "", common.NewError("delete_vesting_pool_failed",
				"draining pool: "+err.Error())
		}
	}

	var cp *clientPools
	if cp, err = vsc.getOrCreateClientPools(t.ClientID, balances); err != nil {
		return "", common.NewError("delete_vesting_pool_failed",
			"unexpected error: "+err.Error())
	}

	if len(cp.Pools) > 0 {
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
	}

	if _, err = balances.DeleteTrieNode(vp.ID); err != nil {
		return "", common.NewError("delete_vesting_pool_failed",
			"can't delete vesting pool: "+err.Error())
	}

	return `{"pool_id":` + vp.ID + `,"action":"deleted"}`, nil
}

func (vsc *VestingSmartContract) lock(t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (resp string, err error) {

	var lr poolRequest
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
	if conf, err = getConfig(); err != nil {
		return "", common.NewError("lock_vesting_pool_failed",
			"can't get SC configurations: "+err.Error())
	}

	if state.Balance(t.Value) < conf.MinLock {
		return "", common.NewError("lock_vesting_pool_failed",
			"insufficient amount to lock")
	}

	if resp, err = vp.fill(t, balances); err != nil {
		return "", common.NewError("lock_vesting_pool_failed",
			"filling pool: "+err.Error())
	}

	if err = vp.save(balances); err != nil {
		return "", common.NewError("lock_vesting_pool_failed",
			"saving pool: "+err.Error())
	}

	return
}

// unlock by owner, unlock by a destination
func (vsc *VestingSmartContract) unlock(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var ur poolRequest
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

	if vp.ClientID == t.ClientID {
		// owner
		if resp, err = vp.drain(t, balances); err != nil {
			return "", common.NewError("unlock_vesting_pool_failed",
				"draining pool: "+err.Error())
		}
	} else {
		// a destination
		if resp, err = vp.vest(t, balances); err != nil {
			return "", common.NewError("unlock_vesting_pool_failed",
				"vesting pool: "+err.Error())
		}
	}

	if err = vp.save(balances); err != nil {
		return "", common.NewError("unlock_vesting_pool_failed",
			"saving pool: "+err.Error())
	}

	return
}

//
// function triggered by server
//

// trigger next vesting and return all transfers in transaction's response
func (vsc *VestingSmartContract) trigger(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var tr poolRequest
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

	if vp.ClientID != t.ClientID {
		return "", common.NewError("trigger_vesting_pool_failed",
			"only owner can trigger the pool")
	}

	if resp, err = vp.trigger(t, balances); err != nil {
		return "", common.NewError("trigger_vesting_pool_failed",
			"triggering pool: "+err.Error())
	}

	if err = vp.save(balances); err != nil {
		return "", common.NewError("trigger_vesting_pool_failed",
			"saving pool: "+err.Error())
	}

	return //
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

	return vp.info(common.Now()), nil
}
