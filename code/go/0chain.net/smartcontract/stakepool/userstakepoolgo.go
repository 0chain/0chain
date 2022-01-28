package stakepool

import (
	"encoding/json"
	"fmt"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type userStakePools struct {
	Pools map[datastore.Key][]datastore.Key `json:"pools"`
}

func userStakePoolsKey(p Provider, clientID datastore.Key) datastore.Key {
	return datastore.Key(p.String() + ":stakepool:user_pools:" + clientID)
}

func newUserStakePools() (usp *userStakePools) {
	usp = new(userStakePools)
	usp.Pools = make(map[datastore.Key][]datastore.Key)
	return
}

func (usp *userStakePools) add(providerId, poolID datastore.Key) {
	usp.Pools[providerId] = append(usp.Pools[providerId], poolID)
}

func (usp *userStakePools) find(searchId datastore.Key) datastore.Key {
	for providedId, provider := range usp.Pools {
		for _, poolId := range provider {
			if searchId == poolId {
				return providedId
			}
		}
	}
	return ""
}

func (usp *userStakePools) del(providerId, poolID datastore.Key) (empty bool) {
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

func (usp *userStakePools) Encode() []byte {
	var p, err = json.Marshal(usp)
	if err != nil {
		panic(err) // must never happen
	}
	return p
}

func (usp *userStakePools) Decode(p []byte) error {
	return json.Unmarshal(p, usp)
}

// save the user stake pools
func (usp *userStakePools) save(
	p Provider,
	clientID datastore.Key,
	balances chainstate.StateContextI,
) (err error) {
	_, err = balances.InsertTrieNode(userStakePoolsKey(p, clientID), usp)
	return
}

// remove the entire user stake pools node
func (usp *userStakePools) remove(
	p Provider,
	clientID datastore.Key,
	balances chainstate.StateContextI,
) (err error) {
	_, err = balances.DeleteTrieNode(userStakePoolsKey(p, clientID))
	return
}

func getUserStakePoolBytes(
	p Provider,
	clientID datastore.Key,
	balances chainstate.StateContextI,
) (b []byte, err error) {
	var val util.Serializable
	val, err = balances.GetTrieNode(userStakePoolsKey(p, clientID))
	if err != nil {
		return
	}
	return val.Encode(), nil
}

// getUserStakePool of given client
func GetUserStakePool(
	p Provider,
	clientID datastore.Key,
	balances chainstate.StateContextI,
) (usp *userStakePools, err error) {
	var poolb []byte
	if poolb, err = getUserStakePoolBytes(p, clientID, balances); err != nil {
		return
	}
	usp = newUserStakePools()
	err = usp.Decode(poolb)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return
}

// getOrCreateUserStakePool of given client
func getOrCreateUserStakePool(
	p Provider,
	clientID datastore.Key,
	balances chainstate.StateContextI,
) (usp *userStakePools, err error) {
	var poolb []byte
	poolb, err = getUserStakePoolBytes(p, clientID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return
	}

	if err == util.ErrValueNotPresent {
		return newUserStakePools(), nil
	}

	usp = newUserStakePools()
	err = usp.Decode(poolb)
	return
}
