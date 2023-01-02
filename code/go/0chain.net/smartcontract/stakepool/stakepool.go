package stakepool

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

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

type AbstractStakePool interface {
	GetPools() map[string]*DelegatePool
	HasStakePool(user string) bool
	LockPool(txn *transaction.Transaction, providerType spenum.Provider, providerId datastore.Key, status spenum.PoolStatus, balances cstate.StateContextI) (string, error)
	EmitStakeEvent(providerType spenum.Provider, providerID string, balances cstate.StateContextI) error
	EmitUnStakeEvent(providerType spenum.Provider, providerID string, amount currency.Coin, balances cstate.StateContextI) error
	Save(providerType spenum.Provider, providerID string,
		balances cstate.StateContextI) error
	GetSettings() Settings
	Empty(sscID, poolID, clientID string, balances cstate.StateContextI) error
	UnlockPool(clientID string, providerType spenum.Provider, providerId datastore.Key, balances cstate.StateContextI) (string, error)
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
	ServiceChargeRatio float64       `json:"service_charge"`
}

type DelegatePool struct {
	Balance      currency.Coin     `json:"balance"`
	Reward       currency.Coin     `json:"reward"`
	Status       spenum.PoolStatus `json:"status"`
	RoundCreated int64             `json:"round_created"` // used for cool down
	DelegateID   string            `json:"delegate_id"`
	StakedAt     common.Timestamp  `json:"staked_at"`
}

// swagger:model stakePoolStat
type StakePoolStat struct {
	ID           string             `json:"pool_id"` // pool ID
	Balance      currency.Coin      `json:"balance"` // total balance
	StakeTotal   currency.Coin      `json:"stake_total"`
	UnstakeTotal currency.Coin      `json:"unstake_total"`
	Delegate     []DelegatePoolStat `json:"delegate"` // delegate pools
	Penalty      currency.Coin      `json:"penalty"`  // total for all
	Rewards      currency.Coin      `json:"rewards"`  // rewards
	Settings     Settings           `json:"settings"` // Settings of the stake pool
}

type DelegatePoolStat struct {
	ID           string          `json:"id"`            // blobber ID
	Balance      currency.Coin   `json:"balance"`       // current balance
	DelegateID   string          `json:"delegate_id"`   // wallet
	Rewards      currency.Coin   `json:"rewards"`       // total for all time
	UnStake      bool            `json:"unstake"`       // want to unstake
	ProviderId   string          `json:"provider_id"`   // id
	ProviderType spenum.Provider `json:"provider_type"` // ype

	TotalReward  currency.Coin `json:"total_reward"`
	TotalPenalty currency.Coin `json:"total_penalty"`
	Status       string        `json:"status"`
	RoundCreated int64         `json:"round_created"`
}

// swagger:model userPoolStat
type UserPoolStat struct {
	Pools map[datastore.Key][]*DelegatePoolStat `json:"pools"`
}

func ToProviderStakePoolStats(provider *event.Provider, delegatePools []event.DelegatePool) (*StakePoolStat, error) {
	spStat := new(StakePoolStat)
	spStat.ID = provider.ID
	spStat.StakeTotal = provider.TotalStake
	spStat.UnstakeTotal = provider.UnstakeTotal
	spStat.Delegate = make([]DelegatePoolStat, 0, len(delegatePools))
	spStat.Settings = Settings{
		DelegateWallet:     provider.DelegateWallet,
		MinStake:           provider.MinStake,
		MaxStake:           provider.MaxStake,
		MaxNumDelegates:    provider.NumDelegates,
		ServiceChargeRatio: provider.ServiceCharge,
	}
	spStat.Rewards = provider.Rewards.TotalRewards
	for _, dp := range delegatePools {
		if spenum.PoolStatus(dp.Status) == spenum.Deleted {
			continue
		}
		dpStats := DelegatePoolStat{
			ID:           dp.PoolID,
			DelegateID:   dp.DelegateID,
			Status:       spenum.PoolStatus(dp.Status).String(),
			RoundCreated: dp.RoundCreated,
		}
		dpStats.Balance = dp.Balance

		dpStats.Rewards = dp.Reward

		dpStats.TotalPenalty = dp.TotalPenalty

		dpStats.TotalReward = dp.TotalReward

		newBal, err := currency.AddCoin(spStat.Balance, dpStats.Balance)
		if err != nil {
			return nil, err
		}
		spStat.Balance = newBal
		spStat.Delegate = append(spStat.Delegate, dpStats)
	}

	return spStat, nil
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

func (sp *StakePool) GetSettings() Settings {
	return sp.Settings
}
func (sp *StakePool) GetPools() map[string]*DelegatePool {
	return sp.Pools
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

func (sp *StakePool) HasStakePool(user string) bool {
	_, found := sp.Pools[user]
	return found
}

func (sp *StakePool) Save(
	p spenum.Provider,
	id string,
	balances cstate.StateContextI,
) error {
	_, err := balances.InsertTrieNode(stakePoolKey(p, id), sp)
	return err
}

func (sp *StakePool) Get(
	p spenum.Provider,
	id string,
	balances cstate.StateContextI,
) error {
	return balances.GetTrieNode(stakePoolKey(p, id), sp)
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
	balances cstate.StateContextI,
) (currency.Coin, error) {
	var delegateReward, serviceCharge currency.Coin
	var err error
	if clientId == sp.Settings.DelegateWallet && sp.Reward > 0 {
		serviceCharge, err = sp.MintServiceCharge(balances)
		if err != nil {
			return 0, err
		}
		balances.EmitEvent(event.TypeStats, event.TagCollectProviderReward, providerId, nil)

	}

	dPool, ok := sp.Pools[clientId]
	if !ok {
		if serviceCharge == 0 {
			return 0, fmt.Errorf("cannot find rewards for %s", clientId)
		}
		return serviceCharge, nil
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
		balances.EmitEvent(event.TypeStats, event.TagMintReward, clientId, event.RewardMint{
			Amount:       int64(dPool.Reward),
			BlockNumber:  balances.GetBlock().Round,
			ClientID:     clientId,
			ProviderType: providerType.String(),
			ProviderID:   providerId,
		})
		delegateReward = dPool.Reward
		dPool.Reward = 0
	}

	var dpUpdate = newDelegatePoolUpdate(clientId, providerId, providerType)
	dpUpdate.Updates["reward"] = 0
	dpUpdate.emitUpdate(balances)
	return delegateReward + serviceCharge, nil
}

func (sp *StakePool) Empty(sscID, poolID, clientID string, balances cstate.StateContextI) error {
	var dp, ok = sp.Pools[poolID]
	if !ok {
		return fmt.Errorf("no such delegate pool: %q", poolID)
	}

	if dp.DelegateID != clientID {
		return errors.New("trying to unlock not by delegate pool owner")
	}

	transfer := state.NewTransfer(sscID, clientID, dp.Balance)
	if err := balances.AddTransfer(transfer); err != nil {
		return err
	}

	sp.Pools[poolID].Balance = 0
	sp.Pools[poolID].Status = spenum.Deleted

	return nil
}

// DistributeRewardsRandN distributes rewards to randomly selected N delegate pools
func (sp *StakePool) DistributeRewardsRandN(
	value currency.Coin,
	providerId string,
	providerType spenum.Provider,
	seed int64,
	randN int,
	rewardType spenum.Reward,
	balances cstate.StateContextI,
) (err error) {
	if value == 0 {
		return nil // nothing to move
	}
	var spUpdate = NewStakePoolReward(providerId, providerType, rewardType)

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

	fValue, err := value.Float64()
	if err != nil {
		return err
	}
	serviceCharge, err := currency.Float64ToCoin(sp.Settings.ServiceChargeRatio * fValue)
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
	stake, pools, err := sp.getRandStakePools(seed, randN)
	if err != nil {
		return err
	}

	if stake == 0 {
		return nil
	}

	for _, pool := range pools {
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
		spUpdate.DelegateRewards[pool.DelegateID] = reward
		if err != nil {
			return err
		}
	}

	if valueBalance > 0 {
		err = equallyDistributeRewards(valueBalance, pools, spUpdate)
		if err != nil {
			return err
		}
	}
	if err := spUpdate.Emit(event.TagStakePoolReward, balances); err != nil {
		return err
	}
	return nil
}

func (sp *StakePool) getRandPools(seed int64, n int) []*DelegatePool {
	if len(sp.Pools) == 0 {
		return nil
	}

	pls := make([]*DelegatePool, 0, len(sp.Pools))
	for _, pool := range sp.Pools {
		pls = append(pls, pool)
	}

	// sort
	sort.SliceStable(pls, func(i, j int) bool {
		return pls[i].DelegateID < pls[j].DelegateID
	})

	if n >= len(pls) {
		return pls
	}

	// get random N from pools N
	plsIdxs := rand.New(rand.NewSource(seed)).Perm(n)
	selected := make([]*DelegatePool, 0, n)

	for _, idx := range plsIdxs {
		selected = append(selected, pls[idx])
	}

	return selected
}

func (sp *StakePool) getRandStakePools(seed int64, n int) (currency.Coin, []*DelegatePool, error) {
	pools := sp.getRandPools(seed, n)
	if len(pools) == 0 {
		return 0, nil, nil
	}

	var stake currency.Coin
	for _, p := range pools {
		var err error
		stake, err = currency.AddCoin(stake, p.Balance)
		if err != nil {
			return 0, nil, err
		}
	}

	return stake, pools, nil
}

func (sp *StakePool) DistributeRewards(
	value currency.Coin,
	providerId string,
	providerType spenum.Provider,
	rewardType spenum.Reward,
	balances cstate.StateContextI,
) (err error) {
	if value == 0 {
		return nil // nothing to move
	}
	var spUpdate = NewStakePoolReward(providerId, providerType, rewardType)

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

	fValue, err := value.Float64()
	if err != nil {
		return err
	}
	serviceCharge, err := currency.Float64ToCoin(sp.Settings.ServiceChargeRatio * fValue)
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
		spUpdate.DelegateRewards[id] = reward
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
	var pools []*DelegatePool
	for _, v := range sp.Pools {
		pools = append(pools, v)
	}
	sort.SliceStable(pools, func(i, j int) bool {
		return pools[i].DelegateID < pools[j].DelegateID
	})

	return equallyDistributeRewards(coins, pools, spUpdate)
}

func equallyDistributeRewards(coins currency.Coin, pools []*DelegatePool, spUpdate *StakePoolReward) error {
	share, r, err := currency.DistributeCoin(coins, int64(len(pools)))
	if err != nil {
		return err
	}
	c, err := coins.Int64()
	if err != nil {
		return err
	}
	if share == 0 {
		for i := int64(0); i < c; i++ {
			pools[i].Reward++
			spUpdate.DelegateRewards[pools[i].DelegateID]++
		}
		return nil
	}

	iShare, err := share.Int64()
	if err != nil {
		return err
	}
	for i := range pools {
		pools[i].Reward, err = currency.AddCoin(pools[i].Reward, share)
		if err != nil {
			return err
		}

		spUpdate.DelegateRewards[pools[i].DelegateID], err =
			currency.AddInt64(spUpdate.DelegateRewards[pools[i].DelegateID], iShare)
		if err != nil {
			return err
		}

	}

	if r > 0 {
		for i := 0; i < int(r); i++ {
			pools[i].Reward++
			spUpdate.DelegateRewards[pools[i].DelegateID]++
		}
	}

	return nil
}

type stakePoolRequest struct {
	ProviderType spenum.Provider `json:"provider_type,omitempty"`
	ProviderID   string          `json:"provider_id,omitempty"`
}

func (spr *stakePoolRequest) decode(p []byte) (err error) {
	return json.Unmarshal(p, spr)
}

func StakePoolLock(t *transaction.Transaction, input []byte, balances cstate.StateContextI,
	get func(providerType spenum.Provider, providerID string, balances cstate.CommonStateContextI) (AbstractStakePool, error)) (resp string, err error) {

	var spr stakePoolRequest
	if err = spr.decode(input); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"invalid request: %v", err)
	}

	var sp AbstractStakePool
	if sp, err = get(spr.ProviderType, spr.ProviderID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"can't get stake pool: %v", err)
	}

	if t.Value < sp.GetSettings().MinStake {
		return "", common.NewError("stake_pool_lock_failed",
			fmt.Sprintf("too small stake to lock: %v < %v", t.Value, sp.GetSettings().MinStake))
	}
	if t.Value > sp.GetSettings().MaxStake {
		return "", common.NewError("stake_pool_lock_failed",
			fmt.Sprintf("too large stake to lock: %v > %v", t.Value, sp.GetSettings().MaxStake))
	}

	logging.Logger.Info("stake_pool_lock", zap.Int("pools", len(sp.GetPools())), zap.Int("delegates", sp.GetSettings().MaxNumDelegates))
	if len(sp.GetPools()) >= sp.GetSettings().MaxNumDelegates && !sp.HasStakePool(t.ClientID) {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"max_delegates reached: %v, no more stake pools allowed",
			sp.GetSettings().MaxNumDelegates)
	}

	out, err := sp.LockPool(t, spr.ProviderType, spr.ProviderID, spenum.Active, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"stake pool digging error: %v", err)
	}

	if err = sp.Save(spr.ProviderType, spr.ProviderID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"saving stake pool: %v", err)
	}

	err = sp.EmitStakeEvent(spr.ProviderType, spr.ProviderID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_lock_failed",
			"stake pool staking error: %v", err)
	}

	return out, err
}

// StakePoolUnlock unlock tokens from provider, stake pool can return excess tokens from stake pool
func StakePoolUnlock(t *transaction.Transaction, input []byte, balances cstate.StateContextI,
	get func(providerType spenum.Provider, providerID string, balances cstate.CommonStateContextI) (AbstractStakePool, error),
) (resp string, err error) {
	var spr stakePoolRequest
	if err = spr.decode(input); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't decode request: %v", err)
	}
	var sp AbstractStakePool
	if sp, err = get(spr.ProviderType, spr.ProviderID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"can't get related stake pool: %v", err)
	}
	if err != nil {
		return "", err
	}
	dp, ok := sp.GetPools()[t.ClientID]
	if !ok {
		return "", common.NewErrorf("stake_pool_unlock_failed", "no such delegate pool: %v ", t.ClientID)
	}

	// if StakeAt has valid value and lock period is less than MinLockPeriod
	if dp.StakedAt > 0 {
		stakedAt := common.ToTime(dp.StakedAt)
		minLockPeriod := config.SmartContractConfig.GetDuration("stakepool.min_lock_period")
		if !stakedAt.Add(minLockPeriod).Before(time.Now()) {
			return "", common.NewErrorf("stake_pool_unlock_failed", "token can only be unstaked till: %s", stakedAt.Add(minLockPeriod))
		}
	}

	err = sp.Empty(t.ToClientID, t.ClientID, t.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"unlocking tokens: %v", err)
	}

	output, err := sp.UnlockPool(t.ClientID, spr.ProviderType, spr.ProviderID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed", "%v", err)
	}

	// Save the pool
	if err = sp.Save(spr.ProviderType, spr.ProviderID, balances); err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"saving stake pool: %v", err)
	}

	err = sp.EmitStakeEvent(spr.ProviderType, spr.ProviderID, balances)
	if err != nil {
		return "", common.NewErrorf("stake_pool_unlock_failed",
			"stake pool staking error: %v", err)
	}

	return output, nil
}

func toJson(val interface{}) string {
	var b, err = json.Marshal(val)
	if err != nil {
		panic(err) // must not happen
	}
	return string(b)
}
