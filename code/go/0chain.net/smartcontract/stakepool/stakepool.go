package stakepool

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/common"
	"0chain.net/core/util"

	"0chain.net/core/datastore"

	cstate "0chain.net/chaincore/chain/state"

	"0chain.net/chaincore/state"
)

type Provider int

const (
	Miner Provider = iota
	Sharder
	Blobber
	Validator
	Authorizer
)

func (p Provider) String() string {
	return [...]string{"miner", "sharder", "blobber", "validator", "authorizer"}[p]
}

type PoolStatus int

const (
	Active PoolStatus = iota
	Pending
	Inactive
	Unstaking
	Deleting
)

func (p PoolStatus) String() string {
	return [...]string{"active", "pending", "inactive", "unstaking", "deleting"}[p]
}

func stakePoolKey(p Provider, id string) datastore.Key {
	return datastore.Key(p.String() + ":stakepool:" + id)
}

type StakePool struct {
	Pools    map[string]*DelegatePool `json:"pools"`
	Reward   state.Balance            `json:"rewards"`
	Settings StakePoolSettings        `json:"settings"`
	Minter   cstate.ApprovedMinters   `json:"minter"`
	mutex    *sync.RWMutex
}

type StakePoolSettings struct {
	DelegateWallet  string        `json:"delegate_wallet"`
	MinStake        state.Balance `json:"min_stake"`
	MaxStake        state.Balance `json:"max_stake"`
	MaxNumDelegates int           `json:"num_delegates"`
	ServiceCharge   float64       `json:"service_charge"`
}

type DelegatePool struct {
	Balance      state.Balance `json:"balance"`
	Reward       state.Balance `json:"reward"`
	Status       PoolStatus    `json:"status"`
	RoundCreated int64         `json:"round_created"`
}

func NewStakePool() *StakePool {
	return &StakePool{
		Pools: make(map[string]*DelegatePool),
	}
}

func (sp *StakePool) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(sp); err != nil {
		panic(err)
	}
	return
}

func (sp *StakePool) Decode(input []byte) error {
	return json.Unmarshal(input, sp)
}

func (sp *StakePool) OrderedPoolIds() []string {
	ids := make([]string, 0, len(sp.Pools))
	for id := range sp.Pools {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func GetStakePool(
	p Provider, id string, balances cstate.StateContextI,
) (*StakePool, error) {
	var poolBytes []byte

	var val util.Serializable
	val, err := balances.GetTrieNode(stakePoolKey(p, id))
	if err != nil {
		return nil, err
	}
	poolBytes = val.Encode()
	var sp = NewStakePool()
	err = sp.Decode(poolBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return sp, nil
}

func (sp *StakePool) Save(
	p Provider,
	id string,
	balances cstate.StateContextI,
) error {
	_, err := balances.InsertTrieNode(stakePoolKey(p, id), sp)
	return err
}

func (sp *StakePool) EmptyAccount(
	clientId,
	poolId, providerId string,
	providerType int,
	balances cstate.StateContextI,
) (state.Balance, bool, error) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	account, ok := sp.Pools[poolId]
	if !ok {
		return 0, false, fmt.Errorf("cannot find rewards for %s", poolId)
	}

	amount := account.Balance
	if amount > 0 {
		minter, err := cstate.GetMinter(sp.Minter)
		if err != nil {
			return 0, false, err
		}
		if err := balances.AddMint(&state.Mint{
			Minter:     minter,
			ToClientID: clientId,
			Amount:     amount,
		}); err != nil {
			return 0, false, fmt.Errorf("minting rewards: %v", err)
		}
	}
	account.Balance = 0
	dpId := DelegatePoolId{
		StakePoolId: StakePoolId{
			ProviderId:   providerId,
			ProviderType: providerType,
		},
		PoolId: poolId,
	}
	if account.Status == Deleting {
		delete(sp.Pools, poolId)
		err := dpId.emit(event.TagRemoveDelegatePool, balances)
		if err != nil {
			return 0, false, err
		}
		return amount, true, nil
	} else {
		err := dpId.emit(event.TagEmptyDelegatePool, balances)
		if err != nil {
			return 0, false, err
		}
		return amount, false, nil
	}

}

func (sp *StakePool) DistributeRewards(
	value float64,
	providerId string,
	providerType int,
	balances cstate.StateContextI,
) error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	if value == 0 {
		return nil // nothing to move
	}

	var serviceCharge float64
	serviceCharge = sp.Settings.ServiceCharge * value
	if state.Balance(serviceCharge) > 0 {
		sp.Reward += state.Balance(serviceCharge)
	}

	if state.Balance(value-serviceCharge) == 0 {
		return nil // nothing to move
	}
	reward := SpReward{
		StakePoolId: StakePoolId{
			ProviderId:   providerId,
			ProviderType: providerType,
		},
		SpReward:       int64(serviceCharge),
		DelegateReward: make(map[string]int64),
	}

	if len(sp.Pools) == 0 {
		return fmt.Errorf("no stake pools to move tokens to")
	}

	valueLeft := value - serviceCharge
	var stake = float64(sp.stake())
	if stake == 0 {
		return fmt.Errorf("no stake")
	}

	for _, id := range sp.OrderedPoolIds() {
		ratio := float64(sp.Pools[id].Balance) / stake
		sp.Pools[id].Reward += state.Balance(valueLeft * ratio)
		reward.DelegateReward[id] = int64(sp.Pools[id].Reward)
	}
	if err := reward.emit(balances); err != nil {
		return err
	}
	return nil
}

func (sp *StakePool) stake() (stake state.Balance) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	for _, id := range sp.OrderedPoolIds() {
		stake += sp.Pools[id].Balance
	}
	return
}
