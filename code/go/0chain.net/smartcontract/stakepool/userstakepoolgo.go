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
	Pools map[string][]string `json:"pools"`
}

func UserStakePoolsKey(p spenum.Provider, clientID datastore.Key) datastore.Key {
	return datastore.Key(p.String() + ":stakepool:user_pools:" + clientID)
}

func NewUserStakePools() (usp *UserStakePools) {
	usp = new(UserStakePools)
	usp.Pools = make(map[datastore.Key][]datastore.Key)
	return
}

func (usp *UserStakePools) add(providerId, poolID datastore.Key) {
	usp.Pools[providerId] = append(usp.Pools[providerId], poolID)
}

func (usp *UserStakePools) FindProvider(searchId datastore.Key) datastore.Key {
	for providedId, provider := range usp.Pools {
		for _, poolId := range provider {
			if searchId == poolId {
				return providedId
			}
		}
	}
	return ""
}

func (usp *UserStakePools) Del(providerId, poolID datastore.Key) (empty bool) {
	var (
		list = usp.Pools[providerId]
		i    int
	)
	for _, id := range list {
		if id == poolID {
			continue
		}
		list[i], i = id, i+1
	}
	list = list[:i]
	if len(list) == 0 {
		delete(usp.Pools, providerId) // delete empty
	} else {
		usp.Pools[providerId] = list // update
	}
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
