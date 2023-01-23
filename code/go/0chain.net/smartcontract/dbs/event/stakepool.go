package event

import (
	"fmt"
	"time"

	"github.com/0chain/common/core/logging"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type providerRewardsDelegates struct {
	rewards       []ProviderRewards
	delegatePools []DelegatePool
	desc          [][]string
}

func aggregateProviderRewards(
	spus []dbs.StakePoolReward, round int64,
) (*providerRewardsDelegates, error) {
	var (
		rewards       = make([]ProviderRewards, 0, len(spus))
		delegatePools = make([]DelegatePool, 0, len(spus))
		descs         = make([][]string, 0, len(spus))
	)
	for i, sp := range spus {
		if sp.Reward != 0 {
			rewards = append(rewards, ProviderRewards{
				ProviderID:                    sp.ProviderId,
				Rewards:                       sp.Reward,
				TotalRewards:                  sp.Reward,
				RoundServiceChargeLastUpdated: round,
			})
		}

		// merge delegate rewards and penalties
		for k, v := range spus[i].DelegateRewards {
			delegatePools = append(delegatePools, DelegatePool{
				ProviderID:           sp.ProviderId,
				ProviderType:         sp.ProviderType,
				PoolID:               k,
				Reward:               v,
				TotalReward:          v,
				TotalPenalty:         spus[i].DelegatePenalties[k],
				RoundPoolLastUpdated: round,
			})
		}

		// append remaining penalties if any
		for k, v := range spus[i].DelegatePenalties {
			if _, ok := sp.DelegateRewards[k]; !ok {
				delegatePools = append(delegatePools, DelegatePool{
					ProviderID:           sp.ProviderId,
					ProviderType:         sp.ProviderType,
					PoolID:               k,
					TotalPenalty:         v,
					RoundPoolLastUpdated: round,
				})
			}
		}
	}

	return &providerRewardsDelegates{
		rewards:       rewards,
		delegatePools: delegatePools,
		desc:          descs,
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
	rewards, err := aggregateProviderRewards(spus, round)
	if err != nil {
		return err
	}

	defer func() {
		du := time.Since(ts)
		n := len(rewards.rewards) + len(rewards.delegatePools)
		if du > 50*time.Millisecond {
			logging.Logger.Debug("event db - update reward slow",
				zap.Int64("round", round),
				zap.Any("duration", du),
				zap.Int("total update items", n),
				zap.Int("rewards num", len(rewards.rewards)),
				zap.Int("delegate pools num", len(rewards.delegatePools)),
				zap.Any("desc", rewards.desc))
		}
	}()

	if len(rewards.rewards) > 0 {
		if err := edb.rewardProviders(rewards.rewards); err != nil {
			return fmt.Errorf("could not rewards providers: %v", err)
		}
	}

	rpdu := time.Since(ts)
	if rpdu.Milliseconds() > 50 {
		logging.Logger.Debug("event db - reward providers slow",
			zap.Any("duration", rpdu),
			zap.Int64("round", round))
	}

	if len(rewards.delegatePools) > 0 {
		if err := edb.rewardProviderDelegates(rewards.delegatePools); err != nil {
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

func (edb *EventDb) rewardProviders(prRewards []ProviderRewards) error {
	var ids []string
	var rewards []uint64
	var lastUpdated []int64
	for _, r := range prRewards {
		ids = append(ids, r.ProviderID)
		rewards = append(rewards, uint64(r.Rewards))
		lastUpdated = append(lastUpdated, r.RoundServiceChargeLastUpdated)
	}

	return CreateBuilder("provider_rewards", "provider_id", ids).
		AddUpdate("rewards", rewards, "provider_rewards.rewards + t.rewards").
		AddUpdate("total_rewards", rewards, "provider_rewards.total_rewards + t.rewards").
		AddUpdate("round_service_charge_last_updated", lastUpdated).
		Exec(edb).Error
}

func (edb *EventDb) rewardProviderDelegates(dps []DelegatePool) error {
	var poolIds []string
	var reward []uint64
	var lastUpdated []uint64
	for _, r := range dps {
		poolIds = append(poolIds, r.PoolID)
		reward = append(reward, uint64(r.Reward))
		lastUpdated = append(lastUpdated, uint64(r.RoundPoolLastUpdated))
	}

	ret := CreateBuilder("delegate_pools", "pool_id", poolIds).
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
