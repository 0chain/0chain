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
			logging.Logger.Debug("event db - reward provider",
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

	switch spenum.Provider(spu.ProviderType) {
	case spenum.Blobber:
		return edb.addBlobberRewards(spu.ProviderId, spu.Reward)
	case spenum.Validator:
		return edb.addValidatorRewards(spu.ProviderId, spu.Reward)
	case spenum.Miner:
		return edb.addMinerRewards(spu.ProviderId, spu.Reward)
	case spenum.Sharder:
		return edb.addSharderRewards(spu.ProviderId, spu.Reward)
	default:
		return fmt.Errorf("not implented provider type %v", spu.ProviderType)
	}

}

func (edb *EventDb) addBlobberRewards(blobberID string, reward currency.Coin) error {
	vs := map[string]interface{}{
		"reward":               gorm.Expr("reward + ?", reward),
		"total_service_charge": gorm.Expr("total_service_charge + ?", reward),
	}
	return edb.Store.Get().Model(&Blobber{}).Where(&Blobber{BlobberID: blobberID}).Updates(vs).Error
}

func (edb *EventDb) addValidatorRewards(validatorID string, reward currency.Coin) error {
	vs := map[string]interface{}{
		"rewards":      gorm.Expr("rewards + ?", reward),
		"total_reward": gorm.Expr("total_reward + ?", reward),
	}
	return edb.Store.Get().Model(&Validator{}).Where(&Validator{ValidatorID: validatorID}).Updates(vs).Error
}

func (edb *EventDb) addMinerRewards(minerID string, reward currency.Coin) error {
	vs := map[string]interface{}{
		"rewards":      gorm.Expr("rewards + ?", reward),
		"total_reward": gorm.Expr("total_reward + ?", reward),
	}
	return edb.Store.Get().Model(&Miner{}).Where(&Miner{MinerID: minerID}).Updates(vs).Error
}

func (edb *EventDb) addSharderRewards(sharderID string, reward currency.Coin) error {
	vs := map[string]interface{}{
		"rewards":      gorm.Expr("rewards + ?", reward),
		"total_reward": gorm.Expr("total_reward + ?", reward),
	}
	return edb.Store.Get().Model(&Sharder{}).Where(&Sharder{SharderID: sharderID}).Updates(vs).Error
}
