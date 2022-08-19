package stakepool

import (
	"encoding/json"

	"0chain.net/smartcontract/stakepool/spenum"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -v

type UserStakePools struct {
	Pools map[string]spenum.Provider `json:"pools"` // key: provider id, value: provider type
}

func UserStakePoolsKey(p spenum.Provider, clientID datastore.Key) datastore.Key {
	return p.String() + ":stakepool:user_pools:" + clientID
}

func NewUserStakePools() *UserStakePools {
	return &UserStakePools{
		Pools: map[string]spenum.Provider{},
	}
}

func (usp *UserStakePools) add(providerId datastore.Key, providerType spenum.Provider) {
	_, ok := usp.Pools[providerId]
	if ok {
		// already exist, one stake pool per user
		return
	}

	usp.Pools[providerId] = providerType
}

func (usp *UserStakePools) FindProviderById(providerId datastore.Key) bool {
	_, ok := usp.Pools[providerId]
	return ok
}

func (usp *UserStakePools) FindProvidersByType(providerType spenum.Provider) []datastore.Key {
	ids := make([]datastore.Key, 0, len(usp.Pools))
	for id, tp := range usp.Pools {
		if tp == providerType {
			ids = append(ids, id)
		}
	}
	return ids
}

func (usp *UserStakePools) Del(providerId datastore.Key) (empty bool) {
	delete(usp.Pools, providerId)
	return len(usp.Pools) == 0
}

func (usp *UserStakePools) Encode() []byte {
	var p, err = json.Marshal(usp)
	if err != nil {
		panic(err) // must never happen
	}
	return p
}

func (usp *UserStakePools) Decode(p []byte) error {
	return json.Unmarshal(p, usp)
}

// save the user stake pools
func (usp *UserStakePools) Save(
	p spenum.Provider,
	clientID datastore.Key,
	balances chainstate.StateContextI,
) (err error) {
	_, err = balances.InsertTrieNode(UserStakePoolsKey(p, clientID), usp)
	return
}

// GetUserStakePool of given client
func GetUserStakePools(
	p spenum.Provider,
	clientID datastore.Key,
	balances chainstate.StateContextI,
) (usp *UserStakePools, err error) {

	usp = NewUserStakePools()
	err = balances.GetTrieNode(UserStakePoolsKey(p, clientID), usp)
	if err != nil {
		return nil, err
	}

	return usp, nil
}

// getOrCreateUserStakePool of given client
func getOrCreateUserStakePool(
	p spenum.Provider,
	clientID datastore.Key,
	balances chainstate.StateContextI,
) (usp *UserStakePools, err error) {
	usp, err = GetUserStakePools(p, clientID, balances)
	switch err {
	case nil:
		return usp, nil
	case util.ErrValueNotPresent:
		return NewUserStakePools(), nil
	default:
		return nil, err
	}
}
