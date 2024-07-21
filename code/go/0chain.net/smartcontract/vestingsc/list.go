package vestingsc

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"

	sci "0chain.net/chaincore/smartcontractinterface"

	"0chain.net/smartcontract"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/util"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

//
// index of vesting pools of a client
//

func clientPoolsKey(vscKey, clientID datastore.Key) datastore.Key {
	return vscKey + ":clientvestingpools:" + clientID
}

// swagger:model vestingClientPools
type clientPools struct {
	Pools []string `json:"pools"`
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

func (vsc *VestingSmartContract) getClientPools(clientID datastore.Key,
	balances chainstate.CommonStateContextI) (cp *clientPools, err error) {

	cp = new(clientPools)
	err = balances.GetTrieNode(clientPoolsKey(vsc.ID, clientID), cp)
	if err != nil {
		return nil, err
	}

	return cp, nil
}

func getOrCreateClientPools(
	clientID datastore.Key,
	balances chainstate.CommonStateContextI,
) (cp *clientPools, err error) {
	var vsc = VestingSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	return vsc.getOrCreateClientPools(clientID, balances)
}

func (vsc *VestingSmartContract) getOrCreateClientPools(clientID datastore.Key,
	balances chainstate.CommonStateContextI) (cp *clientPools, err error) {

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
	params url.Values, balances chainstate.CommonStateContextI) (
	resp interface{}, err error) {

	var (
		clientID = params.Get("client_id")
		cp       *clientPools
	)

	// just return empty list if not found
	if cp, err = vsc.getOrCreateClientPools(clientID, balances); err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get or create client pools")
	}

	return cp, nil
}
