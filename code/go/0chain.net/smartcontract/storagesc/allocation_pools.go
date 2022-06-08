package storagesc

import (
	"encoding/json"
	"fmt"
	"sort"

	"0chain.net/chaincore/transaction"

	"0chain.net/core/datastore"

	"0chain.net/chaincore/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

func allocationPoolKey(allocationId string) datastore.Key {
	return datastore.Key(ADDRESS + ":allocation write pool:" + allocationId)
}

// Created using StorageAllocation.getAllocationPools
type allocationPools struct {
	Pools map[string]*allocationPool `json:"allocation_pools,omitempty"`
}

func newAllocationPools() *allocationPools {
	return &allocationPools{
		Pools: make(map[string]*allocationPool),
	}
}

// Encode implements util.Serializable interface.
func (aps *allocationPools) Encode() []byte {
	var b, err = json.Marshal(aps)
	if err != nil {
		panic(err)
	}
	return b
}

// Decode implements util.Serializable interface.
func (aps *allocationPools) Decode(p []byte) error {
	return json.Unmarshal(p, aps)
}

func (aps *allocationPools) save(allocationId string, balances cstate.StateContextI) error {
	_, err := balances.InsertTrieNode(allocationPoolKey(allocationId), aps)
	return err
}

func createAllocationPools(
	txn *transaction.Transaction,
	alloc *StorageAllocation,
	mintTokens bool,
	balances cstate.StateContextI,
) (*allocationPools, error) {
	aps := newAllocationPools()
	var mld = alloc.restMinLockDemand()
	if txn.Value < int64(mld) || txn.Value <= 0 {
		return nil, fmt.Errorf("not enough tokens to honor the min lock demand"+
			" (%d < %d)", txn.Value, mld)
	}

	var until = alloc.Until()
	ap, err := newAllocationPool(txn, until, mintTokens, balances)
	if err != nil {
		return nil, err
	}
	aps.Pools[alloc.Owner] = ap

	return aps, nil
}

func getAllocationPools(
	allocationID string,
	balances cstate.CommonStateContextI,
) (*allocationPools, error) {
	aps := newAllocationPools()
	err := balances.GetTrieNode(allocationPoolKey(allocationID), aps)
	if err != nil {
		return nil, err
	}
	return aps, nil
}

func (aps *allocationPools) sortExpiry() []string {
	var clients []string
	for client := range aps.Pools {
		clients = append(clients, client)
	}
	sort.Slice(clients, func(i, j int) bool {
		return aps.Pools[clients[i]].ExpireAt < aps.Pools[clients[j]].ExpireAt
	})
	return clients
}

func (aps *allocationPools) addToOrCreateAllocationPool(
	txn *transaction.Transaction,
	until common.Timestamp,
	conf *Config,
	mintTokens bool,
	balances cstate.StateContextI,
) error {
	var err error
	ap, found := aps.Pools[txn.ClientID]
	if found {
		if ap.ExpireAt < until {
			ap.ExpireAt = until
		}
		ap.Balance += currency.Coin(txn.Value)
		return nil
	}
	if len(aps.Pools) >= conf.MaxPoolsPerAllocation {
		return fmt.Errorf("max allocation pools %v exceeded", conf.MaxPoolsPerAllocation)
	}
	ap, err = newAllocationPool(txn, until, mintTokens, balances)
	if err != nil {
		return err
	}
	aps.Pools[txn.ClientID] = ap
	return nil
}

func (aps *allocationPools) getExpiresAfter(
	now common.Timestamp,
) []*allocationPool {
	var pools []*allocationPool
	for _, ap := range aps.Pools {
		if ap.ExpireAt >= now && ap.Balance > 0 {
			pools = append(pools, ap)
		}
	}
	return pools
}

func (aps *allocationPools) moveTo(client string, value currency.Coin) error {
	ap, found := aps.Pools[client]
	if !found {
		return fmt.Errorf("cannot find clinet %s pool to transfer funds", client)
	}
	ap.Balance += value
	return nil
}

func (aps *allocationPools) moveFromCP(
	owner string,
	cp *challengePool,
	value currency.Coin,
) error {
	ap, found := aps.Pools[owner]
	if !found {
		return fmt.Errorf("cannot find owner %s of allocation", owner)
	}
	return ap.moveToAllocationPool(cp, value)
}

func (aps *allocationPools) moveToChallenge(
	allocID string,
	owner string,
	cp *challengePool,
	now common.Timestamp,
	value currency.Coin,
) error {
	var err error
	if value == 0 {
		return err
	}

	for _, ap := range aps.Pools {
		if value == 0 {
			break // all required tokens has moved to the blobber
		}
		var move currency.Coin
		if value >= ap.Balance {
			move = ap.Balance
		} else {
			move = value
		}
		cp.Balance += value
		ap.Balance -= value
		value -= move
	}

	if value != 0 {
		return fmt.Errorf("not enough tokens for allocation: %s,", allocID)
	}

	// remove empty allocation pools
	aps.removeSpentPools(owner, now)
	return nil
}

// remove empty pools of an allocation (all given pools should belongs to
// one allocation)
func (aps *allocationPools) removeSpentPools(owner string, now common.Timestamp) {
	for id, ap := range aps.Pools {
		if ap.ExpireAt < now || ap.Balance == 0 {
			if id != owner {
				delete(aps.Pools, id)
			}
		}
	}
}

func (aps *allocationPools) allocUntil(
	until common.Timestamp,
) currency.Coin {
	aps.getExpiresAfter(until)
	var value currency.Coin
	for _, ap := range aps.Pools {
		value += ap.Balance
	}
	return value
}
