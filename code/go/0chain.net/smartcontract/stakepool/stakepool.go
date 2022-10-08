package stakepool

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"0chain.net/smartcontract/zbig"

	"0chain.net/chaincore/currency"

	"0chain.net/core/maths"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/common"
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
	Reward   currency.Coin            `json:"rewards"`
	Settings Settings                 `json:"settings"`
	Minter   cstate.ApprovedMinter    `json:"minter"`
}

type Settings struct {
	DelegateWallet     string        `json:"delegate_wallet"`
	MinStake           currency.Coin `json:"min_stake"`
	MaxStake           currency.Coin `json:"max_stake"`
	MaxNumDelegates    int           `json:"num_delegates"`
	ServiceChargeRatio zbig.BigRat   `json:"service_charge" msg:"service_charge,extension"`
}

type DelegatePool struct {
	Balance      currency.Coin     `json:"balance"`
	Reward       currency.Coin     `json:"reward"`
	Status       spenum.PoolStatus `json:"status"`
	RoundCreated int64             `json:"round_created"` // used for cool down
	DelegateID   string            `json:"delegate_id"`
	StakedAt     common.Timestamp  `json:"staked_at"`
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

func (sp *StakePool) MintServiceCharge(balances cstate.StateContextI) (currency.Coin, error) {
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
	clientId, providerId string,
	providerType spenum.Provider,
	usp *UserStakePools,
	balances cstate.StateContextI,
) (currency.Coin, error) {
	var reward currency.Coin
	var err error
	if clientId == sp.Settings.DelegateWallet && sp.Reward > 0 {
		reward, err = sp.MintServiceCharge(balances)
		if err != nil {
			return 0, err
		}
		return reward, nil
	}

	dPool, ok := sp.Pools[clientId]
	if !ok {
		return 0, fmt.Errorf("cannot find rewards for %s", clientId)
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
		newReward, err := currency.AddCoin(reward, dPool.Reward)
		if err != nil {
			return 0, err
		}
		reward = newReward
		dPool.Reward = 0
	}

	var dpUpdate = newDelegatePoolUpdate(clientId, providerId, providerType)
	dpUpdate.Updates["reward"] = 0

	if dPool.Status == spenum.Deleting {
		delete(sp.Pools, clientId)
		dpUpdate.Updates["status"] = spenum.Deleted
		err := dpUpdate.emitUpdate(balances)
		if err != nil {
			return 0, err
		}
		usp.Del(providerId)
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
	value currency.Coin,
	providerId string,
	providerType spenum.Provider,
	balances cstate.StateContextI,
) (err error) {
	if value == 0 {
		return nil // nothing to move
	}
	var spUpdate = NewStakePoolReward(providerId, providerType)

	// if no stake pools pay all rewards to the provider
	if len(sp.Pools) == 0 {
		sp.Reward, err = currency.AddCoin(sp.Reward, value)
		if err != nil {
			return err
		}
		spUpdate.Reward = value

		if err := spUpdate.Emit(event.TagStakePoolReward, balances); err != nil {
			return err
		}

		return nil
	}

	serviceCharge, err := currency.MultBigRat(value, sp.Settings.ServiceChargeRatio.Rat)
	if err != nil {
		return err
	}
	if serviceCharge > 0 {
		reward := serviceCharge
		sr, err := currency.AddCoin(sp.Reward, reward)
		if err != nil {
			return err
		}
		sp.Reward = sr
		spUpdate.Reward = reward
	}

	valueLeft := value - serviceCharge
	if valueLeft == 0 {
		return nil
	}

	valueBalance := valueLeft
	stake, err := sp.stake()
	if err != nil {
		return err
	}
	if stake == 0 {
		return fmt.Errorf("no stake")
	}

	for id, pool := range sp.Pools {
		if valueBalance == 0 {
			break
		}
		ratio := float64(pool.Balance) / float64(stake)
		reward, err := currency.MultFloat64(valueLeft, ratio)
		if err != nil {
			return err
		}
		if reward > valueBalance {
			reward = valueBalance
			valueBalance = 0
		} else {
			valueBalance -= reward
		}
		pool.Reward, err = currency.AddCoin(pool.Reward, reward)
		if err != nil {
			return err
		}
		spUpdate.DelegateRewards[id], err = reward.Int64()
		if err != nil {
			return err
		}
	}

	if valueBalance > 0 {
		err = sp.equallyDistributeRewards(valueBalance, spUpdate)
		if err != nil {
			return err
		}
	}
	if err := spUpdate.Emit(event.TagStakePoolReward, balances); err != nil {
		return err
	}

	return nil
}

func (sp *StakePool) stake() (stake currency.Coin, err error) {
	for _, pool := range sp.Pools {
		newStake, err := currency.AddCoin(stake, pool.Balance)
		if err != nil {
			return 0, err
		}
		stake = newStake
	}
	return
}

func (sp *StakePool) equallyDistributeRewards(coins currency.Coin, spUpdate *StakePoolReward) error {

	var delegates []*DelegatePool
	for _, v := range sp.Pools {
		delegates = append(delegates, v)
	}
	sort.Slice(delegates, func(i, j int) bool {
		return strings.Compare(delegates[i].DelegateID, delegates[j].DelegateID) == -1
	})

	share, r, err := currency.DistributeCoin(coins, int64(len(delegates)))
	if err != nil {
		return err
	}
	c, err := coins.Int64()
	if err != nil {
		return err
	}
	if share == 0 {
		for i := int64(0); i < c; i++ {
			delegates[i].Reward++
			spUpdate.DelegateRewards[delegates[i].DelegateID]++
		}
		return nil
	}

	iShare, err := share.Int64()
	if err != nil {
		return err
	}
	for i := range delegates {
		delegates[i].Reward, err = currency.AddCoin(delegates[i].Reward, share)
		if err != nil {
			return err
		}

		spUpdate.DelegateRewards[delegates[i].DelegateID], err =
			maths.SafeAddInt64(spUpdate.DelegateRewards[delegates[i].DelegateID], iShare)
		if err != nil {
			return err
		}

	}

	if r > 0 {
		for i := 0; i < int(r); i++ {
			delegates[i].Reward++
			spUpdate.DelegateRewards[delegates[i].DelegateID]++
		}
	}

	return nil
}
