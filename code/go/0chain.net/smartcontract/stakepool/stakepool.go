package stakepool

import (
	"encoding/json"
	"fmt"
	"sort"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/datastore"

	cstate "0chain.net/chaincore/chain/state"

	"0chain.net/chaincore/state"
)

//go:generate msgp -io=false -tests=false -v

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

var poolString = []string{"active", "pending", "inactive", "unstaking", "deleting"}

func (p PoolStatus) String() string {
	return poolString[p]
}

func stakePoolKey(p Provider, id string) datastore.Key {
	return datastore.Key(p.String() + ":stakepool:" + id)
}

// StakePool holds delegate information for an 0chain providers
type StakePool struct {
	Pools    map[string]*DelegatePool `json:"pools"`
	Reward   state.Balance            `json:"rewards"`
	Settings StakePoolSettings        `json:"settings"`
	Minter   cstate.ApprovedMinter    `json:"minter"`
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
	RoundCreated int64         `json:"round_created"` // used for cool down
	DelegateID   string        `json:"delegate_id"`
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
	var sp = NewStakePool()
	err := balances.GetTrieNode(stakePoolKey(p, id), sp)
	if err != nil {
		return nil, err
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

func (sp *StakePool) MintRewards(
	clientId,
	poolId, providerId string,
	providerType Provider,
	usp *UserStakePools,
	balances cstate.StateContextI,
) (state.Balance, error) {

	dPool, ok := sp.Pools[poolId]
	if !ok {
		return 0, fmt.Errorf("cannot find rewards for %s", poolId)
	}
	reward := dPool.Reward

	if reward > 0 {
		minter, err := cstate.GetMinter(sp.Minter)
		if err != nil {
			return 0, err
		}
		if err := balances.AddMint(&state.Mint{
			Minter:     minter,
			ToClientID: clientId,
			Amount:     reward,
		}); err != nil {
			return 0, fmt.Errorf("minting rewards: %v", err)
		}
		dPool.Reward = 0
	}

	dpId := DelegatePoolId{
		StakePoolId: StakePoolId{
			ProviderId:   providerId,
			ProviderType: providerType,
		},
		PoolId: poolId,
	}
	if dPool.Status == Deleting {
		delete(sp.Pools, poolId)
		err := dpId.emit(event.TagRemoveDelegatePool, balances)
		if err != nil {
			return 0, err
		}
		usp.Del(providerId, poolId)
		return reward, nil
	} else {
		err := dpId.emit(event.TagEmptyDelegatePool, balances)
		if err != nil {
			return 0, err
		}
		return reward, nil
	}
}

func (sp *StakePool) DistributeRewards(
	value float64,
	providerId string,
	providerType Provider,
	balances cstate.StateContextI,
) error {
	if value == 0 {
		return nil // nothing to move
	}

	serviceCharge := sp.Settings.ServiceCharge * value
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

	for id, pool := range sp.Pools {
		ratio := float64(pool.Balance) / stake
		pool.Reward += state.Balance(valueLeft * ratio)
		reward.DelegateReward[id] = int64(pool.Reward)
	}
	if err := reward.emit(balances); err != nil {
		return err
	}
	return nil
}

func (sp *StakePool) stake() (stake state.Balance) {
	for _, pool := range sp.Pools {
		stake += pool.Balance
	}
	return
}
