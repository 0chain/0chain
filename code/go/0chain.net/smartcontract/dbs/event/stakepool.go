package event

import (
	"0chain.net/smartcontract/stakepool/spenum"
	"fmt"
	"time"

	"github.com/0chain/common/core/currency"

	"github.com/0chain/common/core/logging"

	"0chain.net/smartcontract/dbs"
	"go.uber.org/zap"
)

type providerRewardsDelegates struct {
	rewards       map[string]currency.Coin
	totalRewards  map[string]currency.Coin
	delegatePools map[string]map[string]currency.Coin
}

type providerPenaltiesDelegates struct {
	delegatePools map[string]map[string]currency.Coin
}

func aggregateProviderRewards(spus []dbs.StakePoolReward) (*providerRewardsDelegates, error) {
	var (
		rewardsMap      = make(map[string]currency.Coin)
		totalRewardsMap = make(map[string]currency.Coin)
		dpRewardsMap    = make(map[string]map[string]currency.Coin)
	)
	for i, sp := range spus {
		if sp.Reward != 0 {
			rewardsMap[sp.ID] = rewardsMap[sp.ID] + sp.Reward
			totalRewardsMap[sp.ID] = totalRewardsMap[sp.ID] + sp.Reward
		}
		for poolId := range spus[i].DelegateRewards {
			if _, found := dpRewardsMap[sp.ID]; !found {
				dpRewardsMap[sp.ID] = make(map[string]currency.Coin, len(spus[i].DelegateRewards))
			}
			dpRewardsMap[sp.ID][poolId] = dpRewardsMap[sp.ID][poolId] + spus[i].DelegateRewards[poolId]
			totalRewardsMap[sp.ID] = totalRewardsMap[sp.ID] + spus[i].DelegateRewards[poolId]
		}
	}

	return &providerRewardsDelegates{
		rewards:       rewardsMap,
		totalRewards:  totalRewardsMap,
		delegatePools: dpRewardsMap,
	}, nil
}

func aggregateProviderPenalties(spus []dbs.StakePoolReward) (*providerPenaltiesDelegates, error) {
	var (
		dpPenaltiesMap = make(map[string]map[string]currency.Coin)
	)
	for i, sp := range spus {
		for poolId := range spus[i].DelegatePenalties {
			if _, found := dpPenaltiesMap[sp.ID]; !found {
				dpPenaltiesMap[sp.ID] = make(map[string]currency.Coin, len(spus[i].DelegatePenalties))
			}
			dpPenaltiesMap[sp.ID][poolId] = dpPenaltiesMap[sp.ID][poolId] + spus[i].DelegatePenalties[poolId]
		}
	}
	return &providerPenaltiesDelegates{
		delegatePools: dpPenaltiesMap,
	}, nil
}

func mergeStakePoolRewardsEvents() *eventsMergerImpl[dbs.StakePoolReward] {
	return newEventsMerger[dbs.StakePoolReward](TagStakePoolReward, withProviderRewardsPenaltiesAdded())
}

func mergeStakePoolPenaltyEvents() *eventsMergerImpl[dbs.StakePoolReward] {
	return newEventsMerger[dbs.StakePoolReward](TagStakePoolPenalty, withUniqueEventOverwrite())
}

// withProviderRewardsPenaltiesAdded is an event merger middleware that merge two
// StakePoolRewards
func withProviderRewardsPenaltiesAdded() eventMergeMiddleware {
	return withEventMerge(func(a, b *dbs.StakePoolReward) (*dbs.StakePoolReward, error) {
		a.Reward += b.Reward

		// merge delegate pool rewards
		for k, v := range b.DelegateRewards {
			_, ok := a.DelegateRewards[k]
			if !ok {
				a.DelegateRewards[k] = v
				continue
			}

			a.DelegateRewards[k] += v
		}

		// merge delegate pool penalties
		for k, v := range b.DelegatePenalties {
			_, ok := a.DelegatePenalties[k]
			if !ok {
				a.DelegatePenalties[k] = v
				continue
			}

			a.DelegatePenalties[k] += v
		}

		return a, nil
	})
}

func (edb *EventDb) rewardUpdate(spus []dbs.StakePoolReward, round int64) error {
	if len(spus) == 0 {
		return nil
	}

	ts := time.Now()
	rewards, err := aggregateProviderRewards(spus)
	if err != nil {
		return err
	}

	defer func() {
		du := time.Since(ts)
		n := len(rewards.rewards) + len(rewards.delegatePools)
		if du > 50*time.Millisecond {
			logging.Logger.Debug("event db - update reward slow",
				zap.Int64("round", round),
				zap.Duration("duration", du),
				zap.Int("total update items", n),
				zap.Int("rewards num", len(rewards.rewards)),
				zap.Int("delegate pools num", len(rewards.delegatePools)))
		}
	}()

	logging.Logger.Debug("reward provider", zap.Any("rewards", rewards))

	if len(rewards.rewards) > 0 || len(rewards.totalRewards) > 0 {
		if err := edb.rewardProviders(rewards.rewards, rewards.totalRewards, round); err != nil {
			return fmt.Errorf("could not rewards providers: %v", err)
		}
	}

	rpdu := time.Since(ts)
	if rpdu.Milliseconds() > 50 {
		logging.Logger.Debug("event db - reward providers slow",
			zap.Duration("duration", rpdu),
			zap.Int64("round", round))
	}

	if len(rewards.delegatePools) > 0 {
		logging.Logger.Debug("reward provider pools", zap.Any("rewards", rewards))
		if err := edb.rewardProviderDelegates(rewards.delegatePools, round); err != nil {
			return fmt.Errorf("could not rewards delegate pool: %v", err)
		}
	}

	if edb.Debug() {
		if err := edb.insertProviderReward(spus, round); err != nil {
			return err
		}
		if err := edb.insertDelegateReward(spus, round); err != nil {
			return err
		}
	}

	return nil
}

func (edb *EventDb) penaltyUpdate(spus []dbs.StakePoolReward, round int64) error {
	if len(spus) == 0 {
		return nil
	}

	ts := time.Now()

	penalties, err := aggregateProviderPenalties(spus)
	if err != nil {
		return err
	}

	defer func() {
		du := time.Since(ts)
		n := len(penalties.delegatePools)
		if du > 50*time.Millisecond {
			logging.Logger.Debug("event db - update penalties slow",
				zap.Int64("round", round),
				zap.Duration("duration", du),
				zap.Int("total update items", n),
				zap.Int("delegate pools num", len(penalties.delegatePools)))
		}
	}()

	if len(penalties.delegatePools) > 0 {
		if err := edb.penaltyProviderDelegates(penalties.delegatePools, round); err != nil {
			return fmt.Errorf("could not penalise delegate pool: %v", err)
		}
	}

	if edb.Debug() {
		if err := edb.insertDelegateReward(spus, round); err != nil {
			return err
		}
	}
	return nil
}

func (edb *EventDb) rewardProviders(
	prRewards map[string]currency.Coin,
	prTotalRewards map[string]currency.Coin,
	round int64,
) error {
	var ids []string
	var rewards []uint64
	var totalRewards []uint64
	var lastUpdated []int64
	for id, tr := range prTotalRewards {
		// Adding provider id to the list of ids
		ids = append(ids, id)

		// Adding provider reward or setting to 0 if service charge is 0 and there is no provider reward
		r, ok := prRewards[id]
		if !ok {
			r = 0
		}
		rewards = append(rewards, uint64(r))

		// Adding provider total reward
		totalRewards = append(totalRewards, uint64(tr))

		// Last updated time stamp
		lastUpdated = append(lastUpdated, round)
	}

	return CreateBuilder("provider_rewards", "provider_id", ids).
		AddUpdate("rewards", rewards, "provider_rewards.rewards + t.rewards").
		AddUpdate("total_rewards", totalRewards, "provider_rewards.total_rewards + t.total_rewards").
		AddUpdate("round_service_charge_last_updated", lastUpdated).
		Exec(edb).Error
}

func (edb *EventDb) rewardProviderDelegates(dps map[string]map[string]currency.Coin, round int64) error {
	var poolIds []string
	var providerIds []string
	var reward []uint64
	var lastUpdated []uint64
	for id, pools := range dps {
		for poolId, r := range pools {
			poolIds = append(poolIds, poolId)
			providerIds = append(providerIds, id)
			reward = append(reward, uint64(r))
			lastUpdated = append(lastUpdated, uint64(round))
		}
	}

	ret := CreateBuilder("delegate_pools", "pool_id", poolIds).
		AddCompositeId("provider_id", providerIds).
		AddUpdate("reward", reward, "delegate_pools.reward + t.reward").
		AddUpdate("total_reward", reward, "delegate_pools.total_reward + t.total_reward").
		AddUpdate("round_pool_last_updated", lastUpdated).
		Exec(edb)
	return ret.Error
}

func (edb *EventDb) penaltyProviderDelegates(dps map[string]map[string]currency.Coin, round int64) error {

	var poolIds []string
	var providerIds []string
	var slash []uint64
	var lastUpdated []uint64

	for id, pools := range dps {
		for poolId, s := range pools {
			poolIds = append(poolIds, poolId)
			providerIds = append(providerIds, id)
			slash = append(slash, uint64(s))
			lastUpdated = append(lastUpdated, uint64(round))
		}
	}

	ret := CreateBuilder("delegate_pools", "pool_id", poolIds).
		AddCompositeId("provider_id", providerIds).
		AddUpdate("balance", slash, "delegate_pools.balance - t.balance").
		AddUpdate("total_penalty", slash, "t.total_penalty + delegate_pools.total_penalty").
		AddUpdate("round_pool_last_updated", lastUpdated).
		Exec(edb)

	return ret.Error
}

func (edb *EventDb) feesSpecificRevenue(spus []dbs.StakePoolReward) error {
	var (
		minerIDs          []string
		sharderIDs        []string
		authorizerIDs     []string
		minerFeeRewards   []int64
		sharderFeeRewards []int64
		authorizerRewards []int64
	)

	for _, spu := range spus {
		if spu.Type != spenum.Miner && spu.Type != spenum.Sharder && spu.Type != spenum.Authorizer {
			continue
		}

		switch spu.RewardType {
		case spenum.FeeRewardMiner:
			minerIDs = append(minerIDs, spu.ProviderID.ID)
			minerFeeRewards = append(minerFeeRewards, int64(spu.TotalReward()))
		case spenum.FeeRewardSharder:
			sharderIDs = append(sharderIDs, spu.ProviderID.ID)
			sharderFeeRewards = append(sharderFeeRewards, int64(spu.TotalReward()))
		case spenum.FeeRewardAuthorizer:
			authorizerIDs = append(authorizerIDs, spu.ProviderID.ID)
			authorizerRewards = append(authorizerRewards, int64(spu.TotalReward()))
		}
	}

	if len(minerIDs) > 0 {
		err := CreateBuilder("miners", "id", minerIDs).
			AddUpdate("fees", minerFeeRewards, "miners.fees + t.fees").
			Exec(edb).Debug().Error
		if err != nil {
			return fmt.Errorf("could not update miner fee: %v", err)
		}
	}

	if len(sharderIDs) > 0 {
		err := CreateBuilder("sharders", "id", sharderIDs).
			AddUpdate("fees", sharderFeeRewards, "sharders.fees + t.fees").
			Exec(edb).Debug().Error
		if err != nil {
			return fmt.Errorf("could not update sharder fee: %v", err)
		}
	}

	if len(authorizerIDs) > 0 {
		err := CreateBuilder("authorizers", "id", authorizerIDs).
			AddUpdate("fee", authorizerRewards, "authorizers.fee + t.fee").
			Exec(edb).Debug().Error
		if err != nil {
			return fmt.Errorf("could not update authorizer fee: %v", err)
		}
	}

	return nil
}
