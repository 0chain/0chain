package stakepool

import (
	"encoding/json"
	"fmt"
	"sync"

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

func stakePoolKey(p Provider, id string) datastore.Key {
	return datastore.Key(p.String() + ":stakepool:" + id)
}

type StakePool struct {
	Pools    map[string]*delegatePool `json:"pools"`
	Rewards  state.Balance            `json:"rewards"`
	Settings stakePoolSettings        `json:"settings"`
	Minter   cstate.ApprovedMinters   `json:"minter"`
	mutex    *sync.RWMutex
}

type stakePoolSettings struct {
	DelegateWallet  string        `json:"delegate_wallet"`
	MinStake        state.Balance `json:"min_stake"`
	MaxStake        state.Balance `json:"max_stake"`
	MaxNumDelegates int           `json:"num_delegates"`
	ServiceCharge   float64       `json:"service_charge"`
}

type delegatePool struct {
	Balance state.Balance `json:"balance"`
	Reward  state.Balance `json:"reward"`
	Status  PoolStatus    `json:"status"`
	Created int64         `json:"created"`
}

func NewStakePool() *StakePool {
	return &StakePool{
		Pools: make(map[string]*delegatePool),
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

func (sp *StakePool) save(
	p Provider,
	id string,
	balances cstate.StateContextI,
) error {
	_, err := balances.InsertTrieNode(stakePoolKey(p, id), sp)
	return err
}

func (sp *StakePool) AddReward(user string, amount state.Balance) error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	account, ok := sp.Pools[user]
	if !ok {
		return fmt.Errorf("cannot find rewards for %s", user)
	}
	account.Balance += amount
	return nil
}

func (sp *StakePool) EmptyAccount(
	clientId,
	poolId string,
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

	if account.Status == Deleting {
		delete(sp.Pools, poolId)
		return amount, true, nil
	}
	return amount, false, nil
}

// slash represents blobber penalty; it returns number of tokens moved in
// reality, with regards to division errors
func (sp *StakePool) slash(
	alloc *StorageAllocation,
	blobID string,
	until common.Timestamp,
	wp *writePool,
	offer, slash state.Balance,
) (move state.Balance, err error) {
	if offer == 0 || slash == 0 {
		return // nothing to move
	}

	if slash > offer {
		slash = offer // can't move the offer left
	}

	// the move is total movements, but it should be divided by all
	// related stake holders, that can loose some tokens due to
	// division error;

	var ap = wp.allocPool(alloc.ID, until)
	if ap == nil {
		ap = new(allocationPool)
		ap.AllocationID = alloc.ID
		ap.ExpireAt = 0
		alloc.addWritePoolOwner(alloc.Owner)
		wp.Pools.add(ap)
	}

	// offer ratio of entire stake; we are slashing only part of the offer
	// moving the tokens to allocation user; the ratio is part of entire
	// stake should be moved;
	var ratio = (float64(slash) / float64(sp.stake()))

	for _, dp := range sp.orderedPools() {
		var one = state.Balance(float64(dp.Balance) * ratio)
		if one == 0 {
			continue
		}
		if _, _, err = dp.TransferTo(ap, one, nil); err != nil {
			return 0, fmt.Errorf("transferring blobber slash: %v", err)
		}
		dp.Penalty += one
		move += one
	}

	// move
	if blobID != "" {
		var bp, ok = ap.Blobbers.get(blobID)
		if !ok {
			ap.Blobbers.add(&blobberPool{
				BlobberID: blobID,
				Balance:   move,
			})
		} else {
			bp.Balance += move
		}
	}

	return
}

func (sp *StakePool) PayRewards(value float64) error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	if value == 0 {
		return nil // nothing to move
	}

	var serviceCharge float64
	serviceCharge = sp.Settings.ServiceCharge * value
	if state.Balance(serviceCharge) > 0 {
		sp.Rewards += state.Balance(serviceCharge)
	}

	if state.Balance(value-serviceCharge) == 0 {
		return nil // nothing to move
	}

	if len(sp.Pools) == 0 {
		return fmt.Errorf("no stake pools to move tokens to")
	}

	valueLeft := value - serviceCharge
	var stake = float64(sp.stake())
	if stake == 0 {
		return fmt.Errorf("no stake")
	}

	for _, dp := range sp.Pools {
		ratio := float64(dp.Balance) / stake
		dp.Reward += state.Balance(valueLeft * ratio)
	}
	return nil
}

func (sp *StakePool) stake() (stake state.Balance) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	for _, dp := range sp.Pools {
		stake += dp.Balance
	}
	return
}
