package event

import (
	"fmt"
	"time"

	"0chain.net/chaincore/currency"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/stakepool/spenum"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"0chain.net/smartcontract/dbs"
)

type providerAggregateStats struct {
	Rewards     currency.Coin `json:"rewards"`
	TotalReward currency.Coin `json:"total_reward"`
}

type providerRewardsDelegates struct {
	minerRewards     []Miner
	sharderRewards   []Sharder
	blobberRewards   []Blobber
	validatorRewards []Validator

	delegateRewards   []DelegatePool
	delegatePenalties []DelegatePool
}

func aggregateProviderRewards(spus []dbs.StakePoolReward) (*providerRewardsDelegates, error) {
	var (
		minerRewards     = make([]Miner, 0, len(spus))
		sharderRewards   = make([]Sharder, 0, len(spus))
		blobberRewards   = make([]Blobber, 0, len(spus))
		validatorRewards = make([]Validator, 0, len(spus))

		delegateRewards   = make([]DelegatePool, len(spus))
		delegatePenalties = make([]DelegatePool, len(spus))
	)

	for i, sp := range spus {
		if sp.Reward != 0 {
			switch spenum.Provider(sp.ProviderType) {
			case spenum.Miner:
				minerRewards = append(minerRewards,
					Miner{
						MinerID:     sp.ProviderId,
						Rewards:     sp.Reward,
						TotalReward: sp.Reward,
					})
			case spenum.Sharder:
				sharderRewards = append(sharderRewards,
					Sharder{
						SharderID:   sp.ProviderId,
						Rewards:     sp.Reward,
						TotalReward: sp.Reward,
					})
			case spenum.Blobber:
				blobberRewards = append(blobberRewards,
					Blobber{
						BlobberID:          sp.ProviderId,
						Reward:             sp.Reward,
						TotalServiceCharge: sp.Reward,
					})
			case spenum.Validator:
				validatorRewards = append(validatorRewards,
					Validator{
						ValidatorID: sp.ProviderId,
						Rewards:     int64(sp.Reward),
						TotalReward: int64(sp.Reward),
					})
			default:
				return nil, fmt.Errorf("unsupported provider type: %d", sp.ProviderType)
			}
		}

		for k, v := range spus[i].DelegateRewards {
			delegateRewards = append(delegateRewards, DelegatePool{
				ProviderID:   sp.ProviderId,
				ProviderType: sp.ProviderType,
				PoolID:       k,
				Reward:       currency.Coin(v),
				TotalReward:  currency.Coin(v),
			})
		}

		for k, v := range spus[i].DelegatePenalties {
			delegatePenalties = append(delegatePenalties, DelegatePool{
				ProviderID:   sp.ProviderId,
				ProviderType: sp.ProviderType,
				PoolID:       k,
				TotalPenalty: currency.Coin(v),
			})
		}
	}

	return &providerRewardsDelegates{
		minerRewards:     minerRewards,
		sharderRewards:   sharderRewards,
		blobberRewards:   blobberRewards,
		validatorRewards: validatorRewards,

		delegateRewards:   delegateRewards,
		delegatePenalties: delegatePenalties,
	}, nil
}

func (edb *EventDb) rewardUpdate(spus []dbs.StakePoolReward) error {
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
		if du > 50*time.Millisecond {
			logging.Logger.Debug("event db - update reward slow",
				zap.Any("duration", du),
				zap.Int("update items", len(rewards.delegateRewards)+len(rewards.delegatePenalties)))
		}
	}()

	if len(rewards.minerRewards) > 0 {
		if err := rewardProvider(edb, "miners", "miner_id", rewards.minerRewards); err != nil {
			return fmt.Errorf("could not update miner rewards: %v", err)
		}
	}

	if len(rewards.sharderRewards) > 0 {
		if err := rewardProvider(edb, "sharders", "sharder_id", rewards.sharderRewards); err != nil {
			return fmt.Errorf("could not update sharder rewards: %v", err)
		}
	}

	if len(rewards.blobberRewards) > 0 {
		if err := rewardProvider(edb, "blobbers", "blobber_id", rewards.blobberRewards); err != nil {
			return fmt.Errorf("could not update blobber rewards: %v", err)
		}
	}

	if len(rewards.validatorRewards) > 0 {
		if err := rewardProvider(edb, "validators", "validator_id", rewards.validatorRewards); err != nil {
			return fmt.Errorf("could not update validator rewards: %v", err)
		}
	}

	rpdu := time.Since(ts)
	if rpdu.Milliseconds() > 50 {
		logging.Logger.Debug("event db - reward provider slow", zap.Any("duration", rpdu))
	}

	if len(rewards.delegateRewards) > 0 {
		if err := rewardProviderDelegates(edb, rewards.delegateRewards); err != nil {
			return fmt.Errorf("could not rewards delegate pool: %v", err)
		}
	}

	if len(rewards.delegatePenalties) > 0 {
		if err := penaltyProviderDelegates(edb, rewards.delegatePenalties); err != nil {
			return fmt.Errorf("could not penalty delegate pool: %v", err)
		}
	}

	return nil
}

func rewardProvider[T any](edb *EventDb, tableName, index string, providers []T) error {
	vs := map[string]interface{}{
		"rewards":      gorm.Expr(fmt.Sprintf("%s.rewards + excluded.rewards", tableName)),
		"total_reward": gorm.Expr(fmt.Sprintf("%s.total_reward + excluded.total_reward", tableName)),
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: index}},
		DoUpdates: clause.Assignments(vs),
	}).Create(&providers).Error
}

func rewardProviderDelegates(edb *EventDb, rewards []DelegatePool) error {
	vs := map[string]interface{}{
		"reward":       gorm.Expr("delegate_pools.reward + excluded.reward"),
		"total_reward": gorm.Expr("delegate_pools.total_reward + excluded.total_reward"),
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Where: clause.Where{
			Exprs: []clause.Expression{gorm.Expr("delegate_pools.status != ?", spenum.Deleted)},
		},
		Columns: []clause.Column{
			{Name: "provider_type"},
			{Name: "provider_id"},
			{Name: "pool_id"},
		},
		DoUpdates: clause.Assignments(vs),
	}).Create(&rewards).Error
}

func penaltyProviderDelegates(edb *EventDb, penalties []DelegatePool) error {
	vs := map[string]interface{}{
		"total_penalty": gorm.Expr("delegate_pools.total_penalty + excluded.total_penalty"),
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Where: clause.Where{
			Exprs: []clause.Expression{gorm.Expr("delegate_pools.status != ?", spenum.Deleted)},
		},
		Columns: []clause.Column{
			{Name: "provider_type"},
			{Name: "provider_id"},
			{Name: "pool_id"},
		},
		DoUpdates: clause.Assignments(vs),
	}).Create(&penalties).Error
}

type rewardInfo struct {
	pool  string
	value int64
}

func (edb *EventDb) rewardProvider(spu dbs.StakePoolReward) error {
	if spu.Reward == 0 {
		return nil
	}

	var provider interface{}
	switch spenum.Provider(spu.ProviderType) {
	case spenum.Blobber:
		provider = &Blobber{BlobberID: spu.ProviderId}
	case spenum.Validator:
		provider = &Validator{ValidatorID: spu.ProviderId}
	case spenum.Miner:
		provider = &Miner{MinerID: spu.ProviderId}
	case spenum.Sharder:
		provider = &Sharder{SharderID: spu.ProviderId}
	default:
		return fmt.Errorf("not implented provider type %v", spu.ProviderType)
	}

	vs := map[string]interface{}{
		"rewards":      gorm.Expr("rewards + ?", spu.Reward),
		"total_reward": gorm.Expr("total_reward + ?", spu.Reward),
	}

	return edb.Store.Get().Model(provider).Where(provider).Updates(vs).Error
}
