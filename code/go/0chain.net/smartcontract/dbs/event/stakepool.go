package event

import (
	"fmt"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs"
)

type providerAggregateStats struct {
	Rewards     int64 `json:"rewards"`
	TotalReward int64 `json:"total_reward"`
}

func (edb *EventDb) rewardUpdate(spu dbs.StakePoolReward) error {
	if spu.Reward != 0 {
		err := edb.rewardProvider(spu)
		if err != nil {
			return err
		}
	}

	dps, err := edb.GetDelegatePools(spu.ProviderId, spu.ProviderType)
	if err != nil {
		return err
	}

	for _, dp := range dps {
		if reward, ok := spu.DelegateRewards[dp.PoolID]; ok {
			err := edb.updateReward(reward, dp)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (edb *EventDb) rewardProvider(spu dbs.StakePoolReward) error {
	if spu.Reward == 0 {
		return nil
	}

	update := dbs.NewDbUpdates(spu.ProviderId)

	switch spenum.Provider(spu.ProviderType) {
	case spenum.Blobber:
		blobber, err := edb.blobberAggregateStats(spu.ProviderId)
		if err != nil {
			return err
		}
		update.Updates["reward"] = blobber.Reward + spu.Reward
		update.Updates["total_service_charge"] = blobber.TotalServiceCharge + spu.Reward
		return edb.updateBlobber(*update)
	case spenum.Validator:
		validator, err := edb.validatorAggregateStats(spu.ProviderId)
		if err != nil {
			return err
		}
		update.Updates["rewards"] = validator.Rewards + spu.Reward
		update.Updates["total_reward"] = validator.TotalReward + spu.Reward
		return edb.updateValidator(*update)
	case spenum.Miner:
		miner, err := edb.minerAggregateStats(spu.ProviderId)
		if err != nil {
			return err
		}
		update.Updates["rewards"] = miner.Rewards + spu.Reward
		update.Updates["total_reward"] = miner.TotalReward + spu.Reward
		return edb.updateMiner(*update)
	case spenum.Sharder:
		sharder, err := edb.sharderAggregateStats(spu.ProviderId)
		if err != nil {
			return err
		}
		update.Updates["rewards"] = sharder.Rewards + spu.Reward
		update.Updates["total_reward"] = sharder.TotalReward + spu.Reward
		return edb.updateSharder(*update)
	default:
		return fmt.Errorf("not implented provider type %v", spu.ProviderType)
	}

}
