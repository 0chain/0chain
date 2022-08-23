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
	Providers []string `json:"providers"` // provider ids
}

func UserStakePoolsKey(p spenum.Provider, clientID datastore.Key) datastore.Key {
	return p.String() + ":stakepool:user_pools:" + clientID
}

func NewUserStakePools() *UserStakePools {
	return &UserStakePools{}
}

func (usp *UserStakePools) Add(providerID datastore.Key) {
	if _, exist := usp.Find(providerID); exist {
		return
	}

	usp.Providers = append(usp.Providers, providerID)
}

func (usp *UserStakePools) Find(providerID datastore.Key) (int, bool) {
	for i, p := range usp.Providers {
		if p == providerID {
			return i, true
		}
	}
	return -1, false
}

func (usp *UserStakePools) Del(providerID datastore.Key) (empty bool) {
	i, ok := usp.Find(providerID)
	if !ok {
		return len(usp.Providers) == 0
	}

	l := len(usp.Providers)
	if i == l-1 {
		usp.Providers = usp.Providers[:l-1]
		return len(usp.Providers) == 0
	}

	usp.Providers[i] = usp.Providers[l-1]
	usp.Providers = usp.Providers[:l-1]
	return len(usp.Providers) == 0
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

// GetUserStakePools of given client and provider type
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
