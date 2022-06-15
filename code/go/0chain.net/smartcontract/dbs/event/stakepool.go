package event

import (
	"fmt"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs"
)

type providerAggregateStats struct {
	Rewards     currency.Coin `json:"rewards"`
	TotalReward currency.Coin `json:"total_reward"`
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
		update.Updates["reward"], err = currency.AddCoin(blobber.Reward, spu.Reward)
		if err != nil {
			return err
		}
		update.Updates["total_service_charge"], err = currency.AddCoin(blobber.TotalServiceCharge, spu.Reward)
		if err != nil {
			return err
		}
		return edb.updateBlobber(*update)
	case spenum.Validator:
		validator, err := edb.validatorAggregateStats(spu.ProviderId)
		if err != nil {
			return err
		}
		update.Updates["rewards"], err = currency.AddCoin(validator.Rewards, spu.Reward)
		if err != nil {
			return err
		}
		update.Updates["total_reward"], err = currency.AddCoin(validator.TotalReward, spu.Reward)
		if err != nil {
			return err
		}
		return edb.updateValidator(*update)
	case spenum.Miner:
		miner, err := edb.minerAggregateStats(spu.ProviderId)
		if err != nil {
			return err
		}
		update.Updates["rewards"], err = currency.AddCoin(miner.Rewards, spu.Reward)
		if err != nil {
			return err
		}
		update.Updates["total_reward"], err = currency.AddCoin(miner.TotalReward, spu.Reward)
		if err != nil {
			return err
		}
		return edb.updateMiner(*update)
	case spenum.Sharder:
		sharder, err := edb.sharderAggregateStats(spu.ProviderId)
		if err != nil {
			return err
		}
		update.Updates["rewards"], err = currency.AddCoin(sharder.Rewards, spu.Reward)
		if err != nil {
			return err
		}
		update.Updates["total_reward"], err = currency.AddCoin(sharder.TotalReward, spu.Reward)
		if err != nil {
			return err
		}
		return edb.updateSharder(*update)
	default:
		return fmt.Errorf("not implented provider type %v", spu.ProviderType)
	}

}
