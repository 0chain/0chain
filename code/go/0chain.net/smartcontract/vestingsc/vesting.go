package vestingsc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	sci "0chain.net/chaincore/smartcontractinterface"

	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/util"
)

//msgp:ignore info destInfo addRequest
//go:generate msgp -io=false -tests=false -unexported=true -v

// internal errors

var errZeroVesting = errors.New("zero vesting for this destination and period")

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
// stop vesting for destinations
//

type stopRequest struct {
	PoolID      string `json:"pool_id"`
	Destination string `json:"destination"`
}

func (sr *stopRequest) decode(b []byte) error {
	return json.Unmarshal(b, sr)
}

//
// a destination
//

type destination struct {
	ID     string        `json:"id"`     // destination ID
	Amount currency.Coin `json:"amount"` // amount to vest for the destination (initial)
	Vested currency.Coin `json:"vested"` // tokens already vested
	// Last tokens transfer time. The Last is for statistic and represent
	// last destination vesting (unlock / trigger).
	Last common.Timestamp `json:"last"`
	// Move is time of last non-zero transferring. Unlike the Last the Move
	// will neve updated if tokens transferring is zero because of a rounding
	// or division error. For example, triggering for a very short timeout
	// can produce zero tokens transfer (resolution is a second). The move
	// will be updated only if a triggering really moves tokens (non zero).
	Move common.Timestamp `json:"move"`
}

// tokens left for this destination
func (d *destination) left() (left currency.Coin, err error) {
	return currency.MinusCoin(d.Amount, d.Vested)
}

// full time range left for the destination based on last payment time and
// given ending time (the ExpireAt from vesting pool)
func (d *destination) full(end common.Timestamp) (full common.Timestamp) {
	return end - d.Move
}

// period is time range from last payment
func (d *destination) period(now common.Timestamp) (period common.Timestamp) {
	return now - d.Move
}

// move updates last vesting period
func (d *destination) move(now common.Timestamp, moved currency.Coin) error {
	d.Last = now
	if moved > 0 {
		d.Move = now
		newVested, err := currency.AddCoin(d.Vested, moved)
		if err != nil {
			return err
		}
		d.Vested = newVested
	}
	return nil
}

// The unlock returns amount of tokens to vest for current period.
// The dry argument leave all inside the destination as it was and
// used to obtain pool statistic. The now must not be later than the
// end. Also, the now must be greater or equal to start time of related
// vesting pool.
func (d *destination) unlock(now, end common.Timestamp, dry bool) (
	amount currency.Coin, err error) {

	var (
		full   = d.full(end)   // full time range left
		period = d.period(now) // current vesting period
		ending = now == end    // pool ending, should drain all

		ratio = 1.0 // vesting ratio for the period
	)
	left, err := d.left() // tokens left
	if err != nil {
		return 0, err
	}

	// also, the ending protects against zero division error
	if !ending {
		ratio = float64(period) / float64(full)
	}

	amount, err = currency.MultFloat64(left, ratio)
	if err != nil {
		return 0, err
	}

	if !dry {
		err = d.move(now, amount)
	}

	return
}

//
// destinations of a pool
//

type destinations []*destination

// start sets start time (the Last and the Move)
func (ds destinations) start(now common.Timestamp) {
	for _, d := range ds {
		d.Last = now // } setup start time
		d.Move = now // }
		d.Vested = 0 // clean possible request injection
	}
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
	tokenpool.ZcnPool `json:"pool"`

	Description  string           `json:"description"`  //
	StartTime    common.Timestamp `json:"start_time"`   //
	ExpireAt     common.Timestamp `json:"expire_at"`    //
	Destinations destinations     `json:"destinations"` //
	ClientID     string           `json:"client_id"`    // the pool owner
}

// newVestingPool returns new empty uninitialized vesting pool.
func newVestingPool() (vp *vestingPool) {
	vp = new(vestingPool)
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
	vp.Destinations.start(vp.StartTime)
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

	var balance currency.Coin
	balance, err = balances.GetClientBalance(t.ClientID)

	if err != nil && err != util.ErrValueNotPresent {
		return // unexpected error
	}

	if err == util.ErrValueNotPresent {
		return errors.New("no tokens to lock")
	}

	if t.Value > balance {
		return errors.New("lock amount is greater than balance")
	}

	return
}

// required starting pool amount
func (vp *vestingPool) want() (want currency.Coin, err error) {
	for _, d := range vp.Destinations {
		newWant, err := currency.AddCoin(want, d.Amount)
		if err != nil {
			return 0, err
		}
		want = newWant
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

// the tokens transfer
func (vp *vestingPool) moveToDest(vscKey, destID datastore.Key,
	value currency.Coin, balances chainstate.StateContextI) (
	resp string, err error) {

	var transfer *state.Transfer
	transfer, resp, err = vp.DrainPool(vscKey, destID, value, nil)
	if err != nil {
		return "", fmt.Errorf("vesting destination %s: %v", destID, err)
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", fmt.Errorf("adding transfer vesting_pool->destination %s: %v",
			destID, err)
	}

	return
}

// trigger sends all required for transaction's time (now) for all
// destinations, updating them
func (vp *vestingPool) trigger(t *transaction.Transaction,
	balances chainstate.StateContextI) (resp string, err error) {

	if vp.Balance == 0 {
		return "", errors.New("empty pool")
	}

	var (
		now = t.CreationDate
		end = vp.ExpireAt
	)

	if now > end {
		now = end
	} else if now < vp.StartTime {
		now = vp.StartTime
	}

	var (
		sb strings.Builder
		i  int
	)
	sb.WriteByte('[')
	for _, d := range vp.Destinations {
		value, err := d.unlock(now, end, false)
		if err != nil {
			return "", err
		}
		if value == 0 {
			continue
		}
		var mrsp string
		mrsp, err = vp.moveToDest(t.ToClientID, d.ID, value, balances)
		if err != nil {
			return "", fmt.Errorf("transferring to %s: %v", d.ID, err)
		}
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(mrsp)
		i++
	}
	sb.WriteByte(']')

	return sb.String(), nil
}

// excess returns amount of tokens over the vesting pool requires
func (vp *vestingPool) excess() (amount currency.Coin, err error) {
	var need currency.Coin
	for _, d := range vp.Destinations {
		destLeft, err := d.left()
		if err != nil {
			return 0, err
		}
		newNeed, err := currency.AddCoin(need, destLeft)
		if err != nil {
			return 0, err
		}
		need = newNeed
	}
	return vp.Balance - need, nil
}

func (vp *vestingPool) delete(destID string) (err error) {
	var (
		i     int
		found bool
	)
	for _, d := range vp.Destinations {
		if d.ID == destID {
			found = true
			continue
		}
		vp.Destinations[i], i = d, i+1
	}
	if !found {
		return fmt.Errorf("destination %s not found in the pool", destID)
	}
	vp.Destinations = vp.Destinations[:i]
	return
}

func (vp *vestingPool) find(destID string) (d *destination, err error) {
	for _, x := range vp.Destinations {
		if x.ID != destID {
			continue
		}
		d = x
		break
	}
	if d == nil {
		return nil, fmt.Errorf("destination %s not found in the pool", destID)
	}
	return
}

// vest is trigger for one destination
func (vp *vestingPool) vest(vscID, destID datastore.Key, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	var end = vp.ExpireAt

	if now > end {
		now = end
	} else if now < vp.StartTime {
		now = vp.StartTime
	}

	var d *destination
	if d, err = vp.find(destID); err != nil {
		return
	}

	value, err := d.unlock(now, end, false)
	if err != nil {
		return "", err
	}
	if value == 0 {
		return "", errZeroVesting
	}
	resp, err = vp.moveToDest(vscID, d.ID, value, balances)
	if err != nil {
		return "", fmt.Errorf("transferring to %s: %v", d.ID, err)
	}

	return
}

func (vp *vestingPool) drain(t *transaction.Transaction,
	balances chainstate.StateContextI) (resp string, err error) {

	if t.ClientID != vp.ClientID {
		return "", errors.New("only owner can unlock the excess tokens")
	}

	over, err := vp.excess()
	if err != nil {
		return "", err
	}
	if over == 0 {
		return "", errors.New("no excess tokens to unlock")
	}

	var transfer *state.Transfer
	transfer, resp, err = vp.DrainPool(t.ToClientID, t.ClientID, over, nil)
	if err != nil {
		return "", fmt.Errorf("draining vesting pool: %v", err)
	}
	if err = balances.AddTransfer(transfer); err != nil {
		return "", fmt.Errorf("adding transfer vesting_pool->owner: %v", err)
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

func (vp *vestingPool) info(now common.Timestamp) (i *info, err error) {
	i = new(info)

	i.ID = vp.ID
	i.Balance = vp.Balance
	i.Left, err = vp.excess()
	if err != nil {
		return nil, err
	}
	i.Description = vp.Description
	i.StartTime = vp.StartTime
	i.ExpireAt = vp.ExpireAt

	var end = i.ExpireAt

	if now < vp.StartTime {
		now = vp.StartTime
	}

	if now > end {
		now = end
	}

	var dinfos = make([]*destInfo, 0, len(vp.Destinations))
	for _, d := range vp.Destinations {
		value, err := d.unlock(now, end, true)
		if err != nil {
			return nil, err
		}
		dinfos = append(dinfos, &destInfo{
			ID:     d.ID,
			Wanted: d.Amount,
			Earned: value,
			Vested: d.Vested,
			Last:   d.Last,
		})
	}

	i.Destinations = dinfos
	i.ClientID = vp.ClientID
	return
}

type destInfo struct {
	ID     datastore.Key    `json:"id"`     // identifier
	Wanted currency.Coin    `json:"wanted"` // wanted amount for entire period
	Earned currency.Coin    `json:"earned"` // can unlock
	Vested currency.Coin    `json:"vested"` // tokens already vested
	Last   common.Timestamp `json:"last"`   // last time unlocked
}

// swagger:model vestingInfo
type info struct {
	ID           datastore.Key    `json:"pool_id"`      // pool ID
	Balance      currency.Coin    `json:"balance"`      // real pool balance
	Left         currency.Coin    `json:"left"`         // owner can unlock
	Description  string           `json:"description"`  // description
	StartTime    common.Timestamp `json:"start_time"`   // from
	ExpireAt     common.Timestamp `json:"expire_at"`    // until
	Destinations []*destInfo      `json:"destinations"` // receivers
	ClientID     datastore.Key    `json:"client_id"`    // owner
}

//
// helpers
//

func getPool(
	poolID datastore.Key,
	balances chainstate.CommonStateContextI,
) (vp *vestingPool, err error) {
	var vsc = VestingSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	return vsc.getPool(poolID, balances)
}

func (vsc *VestingSmartContract) getPool(
	poolID datastore.Key,
	balances chainstate.CommonStateContextI,
) (vp *vestingPool, err error) {

	vp = newVestingPool()
	err = balances.GetTrieNode(poolID, vp)
	if err != nil {
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
	if conf, err = vsc.getConfig(balances); err != nil {
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

	amtWanted, err := vp.want()
	if err != nil {
		return "", common.NewError("create_vesting_pool_failed",
			"couldn't calculate wanted amount: "+err.Error())
	}
	if t.Value < amtWanted {
		return "", common.NewError("create_vesting_pool_failed",
			"not enough tokens to create pool provided")
	}

	if t.Value < conf.MinLock {
		return "", common.NewError("create_vesting_pool_failed",
			"insufficient amount to lock")
	}
	if _, err = vp.fill(t, balances); err != nil {
		return "", common.NewError("create_vesting_pool_failed",
			"can't fill pool: "+err.Error())
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

func (vsc *VestingSmartContract) stop(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	var sr stopRequest
	if err = sr.decode(input); err != nil {
		return "", common.NewError("stop_vesting_failed",
			"malformed request: "+err.Error())
	}

	if sr.Destination == "" {
		return "", common.NewError("stop_vesting_failed",
			"missing destination to stop vesting")
	}

	var vp *vestingPool
	if vp, err = vsc.getPool(sr.PoolID, balances); err != nil {
		return "", common.NewError("stop_vesting_failed",
			"can't get vesting pool: "+err.Error())
	}

	if vp.ClientID != t.ClientID {
		return "", common.NewError("stop_vesting_failed",
			"only owner can stop a vesting")
	}

	if t.CreationDate > vp.ExpireAt {
		return "", common.NewError("stop_vesting_failed", "expired pool")
	}

	_, err = vp.vest(t.ToClientID, sr.Destination, t.CreationDate, balances)
	if err != nil && err != errZeroVesting {
		return "", common.NewError("stop_vesting_failed", err.Error())
	}

	if err = vp.delete(sr.Destination); err != nil {
		return "", common.NewError("stop_vesting_failed",
			"deleting destination: "+err.Error())
	}

	if err = vp.save(balances); err != nil {
		return "", common.NewError("trigger_vesting_pool_failed",
			"saving pool: "+err.Error())
	}

	return sr.Destination + " has deleted from the vesting pool", nil
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

	vp.Destinations = nil // reset

	// move left to owner
	if vp.Balance > 0 {
		if _, err = vp.drain(t, balances); err != nil {
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

	return `{"pool_id":"` + vp.ID + `","action":"deleted"}`, nil
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
		resp, err = vp.vest(t.ToClientID, t.ClientID, t.CreationDate, balances)
		if err != nil {
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

	if len(vp.Destinations) == 0 {
		return "", common.NewError("trigger_vesting_pool_failed",
			"no destinations in the pool")
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
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get pool")
	}

	return vp.info(common.Now())
}
