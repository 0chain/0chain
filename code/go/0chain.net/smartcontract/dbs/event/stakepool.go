package event

import (
	"fmt"
	"github.com/0chain/common/core/currency"
	"time"

	"github.com/0chain/common/core/logging"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type providerRewardsDelegates struct {
	rewards       map[string]currency.Coin
	delegatePools map[string]map[string]currency.Coin
}

func aggregateProviderRewards(spus []dbs.StakePoolReward) (*providerRewardsDelegates, error) {
	var (
		rewardsMap   = make(map[string]currency.Coin)
		dpRewardsMap = make(map[string]map[string]currency.Coin)
	)
	for i, sp := range spus {
		if sp.Reward != 0 {
			rewardsMap[sp.ProviderId] = rewardsMap[sp.ProviderId] + sp.Reward
		}
		for poolId := range spus[i].DelegateRewards {
			if _, found := dpRewardsMap[sp.ProviderId]; !found {
				dpRewardsMap[sp.ProviderId] = make(map[string]currency.Coin, len(spus[i].DelegateRewards))
			}
			dpRewardsMap[sp.ProviderId][poolId] = dpRewardsMap[sp.ProviderId][poolId] + spus[i].DelegateRewards[poolId]
		}
		// todo https://github.com/0chain/0chain/issues/2122
		// slash charges are no longer taken from rewards, but the stake pool. So related code has been removed.
	}

	return &providerRewardsDelegates{
		rewards:       rewardsMap,
		delegatePools: dpRewardsMap,
	}, nil
}

func mergeStakePoolRewardsEvents() *eventsMergerImpl[dbs.StakePoolReward] {
	return newEventsMerger[dbs.StakePoolReward](TagStakePoolReward, withProviderRewardsPenaltiesAdded())
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

	if len(rewards.rewards) > 0 {
		if err := edb.rewardProviders(rewards.rewards, round); err != nil {
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

func (edb *EventDb) rewardProviders(prRewards map[string]currency.Coin, round int64) error {
	var ids []string
	var rewards []uint64
	var lastUpdated []int64
	for id, r := range prRewards {
		ids = append(ids, id)
		rewards = append(rewards, uint64(r))
		lastUpdated = append(lastUpdated, round)
	}

	return CreateBuilder("provider_rewards", "provider_id", ids).
		AddUpdate("rewards", rewards, "provider_rewards.rewards + t.rewards").
		AddUpdate("total_rewards", rewards, "provider_rewards.total_rewards + t.rewards").
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
		AddUpdate("total_reward", reward, "delegate_pools.total_reward + t.reward").
		AddUpdate("round_pool_last_updated", lastUpdated).
		Exec(edb)
	return ret.Error
}

func (edb *EventDb) rewardProvider(spu dbs.StakePoolReward) error { //nolint: unused
	if spu.Reward == 0 {
		return nil
	}

	var provider interface{}
	switch spu.ProviderType {
	case spenum.Blobber:
		provider = &Blobber{Provider: Provider{ID: spu.ProviderId}}
	case spenum.Validator:
		provider = &Validator{Provider: Provider{ID: spu.ProviderId}}
	case spenum.Miner:
		provider = &Miner{Provider: Provider{ID: spu.ProviderId}}
	case spenum.Sharder:
		provider = &Sharder{Provider: Provider{ID: spu.ProviderId}}
	default:
		return fmt.Errorf("not implented provider type %v", spu.ProviderType)
	}

	vs := map[string]interface{}{
		"rewards":      gorm.Expr("rewards + ?", spu.Reward),
		"total_reward": gorm.Expr("total_reward + ?", spu.Reward),
	}

	return edb.Store.Get().Model(provider).Where(provider).Updates(vs).Error
}
