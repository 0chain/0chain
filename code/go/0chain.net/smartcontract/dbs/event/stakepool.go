package event

import (
	"fmt"
	"time"

	"0chain.net/chaincore/currency"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/stakepool/spenum"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"0chain.net/smartcontract/dbs"
)

type providerAggregateStats struct {
	Rewards     currency.Coin `json:"rewards"`
	TotalReward currency.Coin `json:"total_reward"`
}

func (edb *EventDb) rewardUpdate(spu dbs.StakePoolReward) error {
	ts := time.Now()
	if spu.Reward != 0 {
		err := edb.rewardProvider(spu)
		if err != nil {
			return err
		}
		rpdu := time.Since(ts)
		if rpdu.Milliseconds() > 50 {
			logging.Logger.Debug("event db - reward provider slow",
				zap.Any("duration", rpdu),
				zap.Int("provider type", spu.ProviderType),
				zap.String("provider id", spu.ProviderId))
		}
	}

	if len(spu.DelegateRewards) == 0 {
		return nil
	}

	defer func() {
		du := time.Since(ts)
		if du > 50*time.Millisecond {
			logging.Logger.Debug("event db - update reward slow",
				zap.Any("duration", du),
				zap.Int("update items", len(spu.DelegateRewards)))
		}
	}()

	var (
		penalties = make([]rewardInfo, 0, len(spu.DelegateRewards))
		rewards   = make([]rewardInfo, 0, len(spu.DelegateRewards))
	)

	for pool, reward := range spu.DelegateRewards {
		// TODO: only blobbers have penalty?
		if reward < 0 && spu.ProviderType == int(spenum.Blobber) {
			penalties = append(penalties, rewardInfo{pool: pool, value: -reward})
		} else {
			rewards = append(rewards, rewardInfo{pool: pool, value: reward})
		}
	}

	if len(penalties) > 0 {
		if err := edb.bulkUpdatePenalty(spu.ProviderId, spu.ProviderType, penalties); err != nil {
			return err
		}
	}

	if len(rewards) > 0 {
		return edb.bulkUpdateRewards(spu.ProviderId, spu.ProviderType, rewards)
	}

	return nil
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
