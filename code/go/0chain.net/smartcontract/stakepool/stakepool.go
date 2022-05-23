package stakepool

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/datastore"

	cstate "0chain.net/chaincore/chain/state"

	"0chain.net/chaincore/state"
)

//go:generate msgp -v -io=false -tests=false

func stakePoolKey(p spenum.Provider, id string) datastore.Key {
	return p.String() + ":stakepool:" + id
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
	Balance      state.Balance     `json:"balance"`
	Reward       state.Balance     `json:"reward"`
	Status       spenum.PoolStatus `json:"status"`
	RoundCreated int64             `json:"round_created"` // used for cool down
	DelegateID   string            `json:"delegate_id"`
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
	p spenum.Provider, id string, balances cstate.StateContextI,
) (*StakePool, error) {
	var sp = NewStakePool()
	err := balances.GetTrieNode(stakePoolKey(p, id), sp)
	if err != nil {
		return nil, err
	}

	return sp, nil
}

func (sp *StakePool) Save(
	p spenum.Provider,
	id string,
	balances cstate.StateContextI,
) error {
	_, err := balances.InsertTrieNode(stakePoolKey(p, id), sp)
	return err
}

func (sp *StakePool) MintServiceCharge(balances cstate.StateContextI) (state.Balance, error) {
	minter, err := cstate.GetMinter(sp.Minter)
	if err != nil {
		return 0, err
	}
	if err := balances.AddMint(&state.Mint{
		Minter:     minter,
		ToClientID: sp.Settings.DelegateWallet,
		Amount:     sp.Reward,
	}); err != nil {
		return 0, fmt.Errorf("minting rewards: %v", err)
	}
	minted := sp.Reward
	sp.Reward = 0
	return minted, nil
}

func (sp *StakePool) MintRewards(
	clientId,
	poolId, providerId string,
	providerType spenum.Provider,
	usp *UserStakePools,
	balances cstate.StateContextI,
) (state.Balance, error) {
	var reward state.Balance
	var err error
	if clientId == sp.Settings.DelegateWallet && sp.Reward > 0 {
		reward, err = sp.MintServiceCharge(balances)
		if err != nil {
			return 0, err
		}
		if len(poolId) == 0 {
			return reward, nil
		}
	}
	if len(poolId) == 0 {
		return 0, errors.New("no pool id from which to release funds found")
	}

	dPool, ok := sp.Pools[poolId]
	if !ok {
		return 0, fmt.Errorf("cannot find rewards for %s", poolId)
	}

	if dPool.Reward > 0 {
		minter, err := cstate.GetMinter(sp.Minter)
		if err != nil {
			return 0, err
		}
		if err := balances.AddMint(&state.Mint{
			Minter:     minter,
			ToClientID: clientId,
			Amount:     dPool.Reward,
		}); err != nil {
			return 0, fmt.Errorf("minting rewards: %v", err)
		}
		reward += dPool.Reward
		dPool.Reward = 0
	}

	var dpUpdate = newDelegatePoolUpdate(providerId, providerType)
	dpUpdate.Updates["reward"] = 0

	if dPool.Status == spenum.Deleting {
		delete(sp.Pools, poolId)
		dpUpdate.Updates["status"] = spenum.Deleted
		err := dpUpdate.emitUpdate(balances)
		if err != nil {
			return 0, err
		}
		usp.Del(providerId, poolId)
		return reward, nil
	} else {
		err := dpUpdate.emitUpdate(balances)
		if err != nil {
			return 0, err
		}
		return reward, nil
	}
}

func (sp *StakePool) DistributeRewards(
	value float64,
	providerId string,
	providerType spenum.Provider,
	balances cstate.StateContextI,
) error {
	if value == 0 {
		return nil // nothing to move
	}
	var spUpdate = NewStakePoolReward(providerId, providerType)

	// if no stake pools pay all rewards to the provider
	if len(sp.Pools) == 0 {
		sp.Reward += state.Balance(value)
		spUpdate.Reward = int64(value)
		if err := spUpdate.Emit(event.TagStakePoolReward, balances); err != nil {
			return err
		}
		return nil
	}

	serviceCharge := sp.Settings.ServiceCharge * value
	if state.Balance(serviceCharge) > 0 {
		reward := state.Balance(serviceCharge)
		sp.Reward += reward
		spUpdate.Reward = int64(reward)
	}

	if state.Balance(value-serviceCharge) == 0 {
		return nil
	}

	valueLeft := value - serviceCharge
	var stake = float64(sp.stake())
	if stake == 0 {
		return fmt.Errorf("no stake")
	}

	for id, pool := range sp.Pools {
		ratio := float64(pool.Balance) / stake
		reward := state.Balance(valueLeft * ratio)
		pool.Reward += reward
		spUpdate.DelegateRewards[id] = int64(reward)
	}
	if err := spUpdate.Emit(event.TagStakePoolReward, balances); err != nil {
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
