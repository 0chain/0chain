package vestingsc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"

	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

// add/replace request
type addRequest struct {
	Description  string           `json:"description"`  // allow empty
	StartTime    common.Timestamp `json:"start_time"`   //
	Duration     time.Duration    `json:"duration"`     //
	Friquency    time.Duration    `json:"friquency"`    //
	Destinations []datastore.Key  `json:"destinations"` //
	Amount       sate.Balance     `json:"amount"`       //
}

func (ar *addRequest) decode(b []byte) error {
	return json.Unmarshal(b, ar)
}

func toSeconds(dur time.Duration) common.Timestamp {
	return common.Timestamp(dur / time.Second)
}

// validate the addRequest
func (ar *addRequest) validate(now common.Timestamp, conf *config) (err error) {
	switch {
	case len(ar.Description) > conf.MaxDescriptionLength:
		return errors.New("entry description is too long")
	case ar.StartTime == 0:
		ar.StartTime = now
	case ar.StartTime < now:
		return errors.New("vesting starts before now")
	case ar.Duration < 0:
		return errors.New("negative vesting duration")
	case ar.Duration < conf.MinDuration:
		return errors.New("vesting duration is too short")
	case ar.Duration > conf.MaxDuration:
		return errors.New("vesting duration is too long")
	case ar.Friquency < 0:
		return errors.New("negative vesting friquency")
	case ar.Friquency < conf.MinFriquency:
		return errors.New("vesting friquency is too low")
	case ar.Friquency > conf.MaxFriquency:
		return errors.New("vesting friquency is too high")
	case len(ar.Destinations) == 0:
		return errors.New("no vesting destinations")
	case len(ar.Destinations) > conf.MaxDestinations:
		return errors.New("amount of destinations is too big")
	}
	return
}

func poolKey(vscKey string, poolID datastore.Key) datastore.Key {
	return vscKey + ":vestingpool:" + poolID
}

type vestingPool struct {
	*tokenpool.ZcnPool `json:"pool"`

	Description  string           `json:"description"`
	StartTime    common.Timestamp `json:"start_time"`
	ExpireAt     common.Timestamp `json:"expire_at"`
	Friquency    time.Duration    `json:"friquency"`
	Destinations []datastore.Key  `json:"destinations"`
	Amount       sate.Balance     `json:"amount"`
	ClientID     datastore.Key    `json:"client_id"`
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
	value state.Balance, balances chainstate.StateContextI) (err error) {

	var transfer *state.Transfer

	transfer, _, err = rp.DrainPool(vscKey, toClientID, value, nil)
	if err != nil {
		return fmt.Errorf("draining vesting pool: %v", err)
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return fmt.Errorf("adding vesting pool transfer: %v", err)
	}

	return
}

// vest moves configured amount of tokens to all destinations of the pool
func (vp *vestingPool) vest(vscKey string,
	balances chainstate.StateContextI) (err error) {

	var total = vp.Amount * int64(len(vp.Destinations))

	if vp.Balance < total {
		return errors.New("not enough tokens")
	}

	for _, dest := range vp.Destinations {
		if err = vp.vestTo(vscKey, dest, vp.Amount, balances); err != nil {
			return // vesting error
		}
	}

	return // success
}

//
// lock / unlock
//

func (vp *vestingPool) fill(balances chainstate.StateContextI) (err error) {
	//
	return
}

func (vp *vestingPool) drain(balances chainstate.StateContextI) (err error) {
	//
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

	return
}

type info struct {
	ID           datastore.Key    `json:"pool_id"`
	Balance      stat.Balance     `json:"balance"`
	Description  string           `json:"description"`
	StartTime    common.Timestamp `json:"start_time"`
	ExpireAt     common.Timestamp `json:"expire_at"`
	Friquency    time.Duration    `json:"friquency"`
	Destinations []datastore.Key  `json:"destinations"`
	Amount       sate.Balance     `json:"amount"`
	ClientID     datastore.Key    `json:"client_id"`
}

//
// helpers
//

func (vsc *VestingSmartContract) getPoolBytes(poolID datastore.Key,
	balances chainstate.StateContextI) (_ []byte, err error) {

	var val util.Serializable
	if val, err = balances.GetTrieNode(poolKey(vsc.ID, poolID)); err != nil {
		return
	}

	return val.Encode(), nil
}

func (vsc *VestingSmartContract) getPool(poolID datastore.Key,
	balances chainstate.StateContextI) (vp *vestingPool, err error) {

	var poolb []byte
	poolb, err = vsc.getPoolBytes(poolID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	vp = newVestingPool()
	if err = vp.Decode(poolb); err != nil {
		return nil, err
	}

	return
}

func (vsc *VestingSmartContract) checkFill(t *transaction.Transaction,
	balances chainState.StateContextI) (err error) {

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
// SC functions
//

func (vsc *VestingSmartContract) create(t *transaction.Transaction,
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
		if err = vsc.checkFill(t, balances); err != nil {
			return "", common.NewError("create_vesting_pool_failed",
				"can't fill pool: "+err.Error())
		}
		//
	}

	//

	return
}

func (vsc *VestingSmartContract) lock(t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (resp string, err error) {

	return
}

func (vsc *VestingSmartContract) unlock(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	return
}

func (vsc *VestingSmartContract) add(t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (resp string, err error) {

	return
}

func (vsc *VestingSmartContract) replace(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	return
}

func (vsc *VestingSmartContract) delete(t *transaction.Transaction,
	input []byte, balances chainstate.StateContextI) (resp string, err error) {

	return
}

//
// REST handlers
//

func (vsc *VestingSmartContract) getPoolInfo(ctx context.Context,
	params url.Values, balances chainstate.StateContextI) (
	resp interface{}, err error) {

	var poolID = datastore.Key(params.Get("pool_id"))

	vsc.getPool(poolID, balances)

	return
}
