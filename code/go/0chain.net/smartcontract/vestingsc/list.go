package vestingsc

import (
	"encoding/json"
	"sort"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

//
// index of vesting pools of a client
//

func ClientPoolsKey(vscKey, clientID datastore.Key) datastore.Key {
	return vscKey + ":clientvestingpools:" + clientID
}

// swagger:model VestingClientPools
type ClientPools struct {
	Pools []string `json:"pools"`
}

func (cp *ClientPools) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(cp); err != nil {
		panic(err) // must not happen
	}
	return
}

func (cp *ClientPools) Decode(b []byte) (err error) {
	return json.Unmarshal(b, cp)
}

func (cp *ClientPools) getIndex(poolID datastore.Key) (i int, ok bool) {
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

func (cp *ClientPools) removeByIndex(i int) {
	cp.Pools = append(cp.Pools[:i], cp.Pools[i+1:]...)
}

func (cp *ClientPools) remove(poolID datastore.Key) (ok bool) {
	var i int
	if i, ok = cp.getIndex(poolID); !ok {
		return // false
	}
	cp.removeByIndex(i)
	return true // removed
}

func (cp *ClientPools) add(poolID datastore.Key) (ok bool) {
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

func (cp *ClientPools) save(vscKey, clientID datastore.Key,
	balances chainstate.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(ClientPoolsKey(vscKey, clientID), cp)
	return
}

//
// SC helpers
//

func (vsc *VestingSmartContract) getClientPools(clientID datastore.Key,
	balances chainstate.StateContextI) (cp *ClientPools, err error) {

	cp = new(ClientPools)
	err = balances.GetTrieNode(ClientPoolsKey(vsc.ID, clientID), cp)
	if err != nil {
		return nil, err
	}

	return cp, nil
}

func (vsc *VestingSmartContract) getOrCreateClientPools(clientID datastore.Key,
	balances chainstate.StateContextI) (cp *ClientPools, err error) {

	cp, err = vsc.getClientPools(clientID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return // unexpected error
	}

	if err == util.ErrValueNotPresent {
		return new(ClientPools), nil // create new
	}

	return // existing one, nil
}
