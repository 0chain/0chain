package vestingsc

import (
	"0chain.net/smartcontract"
	"context"
	"encoding/json"
	"net/url"
	"sort"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//
// index of vesting pools of a client
//

func clientPoolsKey(vscKey, clientID datastore.Key) datastore.Key {
	return vscKey + ":clientvestingpools:" + clientID
}

type clientPools struct {
	Pools []datastore.Key `json:"pools"`
}

func (cp *clientPools) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(cp); err != nil {
		panic(err) // must not happen
	}
	return
}

func (cp *clientPools) Decode(b []byte) (err error) {
	return json.Unmarshal(b, cp)
}

func (cp *clientPools) getIndex(poolID datastore.Key) (i int, ok bool) {
	i = sort.Search(len(cp.Pools), func(i int) bool {
		return cp.Pools[i] >= poolID
	})
	if i == len(cp.Pools) {
		return // not found
	}
	if cp.Pools[i] == poolID {
		return i, true // found
	}
	return // not found
}

func (cp *clientPools) removeByIndex(i int) {
	cp.Pools = append(cp.Pools[:i], cp.Pools[i+1:]...)
}

func (cp *clientPools) remove(poolID datastore.Key) (ok bool) {
	var i int
	if i, ok = cp.getIndex(poolID); !ok {
		return // false
	}
	cp.removeByIndex(i)
	return true // removed
}

func (cp *clientPools) add(poolID datastore.Key) (ok bool) {
	if len(cp.Pools) == 0 {
		cp.Pools = append(cp.Pools, poolID)
		return true // added
	}
	var i = sort.Search(len(cp.Pools), func(i int) bool {
		return cp.Pools[i] >= poolID
	})
	// out of bounds
	if i == len(cp.Pools) {
		cp.Pools = append(cp.Pools, poolID)
		return true // added
	}
	// the same
	if cp.Pools[i] == poolID {
		return false // already have
	}
	// next
	cp.Pools = append(cp.Pools[:i],
		append([]string{poolID}, cp.Pools[i:]...)...)
	return true // added
}

func (cp *clientPools) save(vscKey, clientID datastore.Key,
	balances chainstate.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(clientPoolsKey(vscKey, clientID), cp)
	return
}

//
// SC helpers
//

func (vsc *VestingSmartContract) getClientPoolsBytes(clientID datastore.Key,
	balances chainstate.StateContextI) (_ []byte, err error) {

	var val util.Serializable
	val, err = balances.GetTrieNode(clientPoolsKey(vsc.ID, clientID))
	if err != nil {
		return
	}

	return val.Encode(), nil
}

func (vsc *VestingSmartContract) getClientPools(clientID datastore.Key,
	balances chainstate.StateContextI) (cp *clientPools, err error) {

	var listb []byte
	if listb, err = vsc.getClientPoolsBytes(clientID, balances); err != nil {
		return
	}

	cp = new(clientPools)
	if err = cp.Decode(listb); err != nil {
		return nil, smartcontract.NewError(smartcontract.DecodingErr, err)
	}

	return
}

func (vsc *VestingSmartContract) getOrCreateClientPools(clientID datastore.Key,
	balances chainstate.StateContextI) (cp *clientPools, err error) {

	cp, err = vsc.getClientPools(clientID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return // unexpected error
	}

	if err == util.ErrValueNotPresent {
		return new(clientPools), nil // create new
	}

	return // existing one, nil
}

//
// REST-handlers
//

func (vsc *VestingSmartContract) getClientPoolsHandler(ctx context.Context,
	params url.Values, balances chainstate.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID = params.Get("client_id")
		cp       *clientPools
	)

	// just return empty list if not found
	if cp, err = vsc.getOrCreateClientPools(clientID, balances); err != nil {
		err := smartcontract.NewError(smartcontract.FailGetOrCreateClientPoolsErr, err)
		return nil, smartcontract.WrapErrInternal(err)
	}

	return cp, nil
}
