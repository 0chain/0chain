package event

import (
	"fmt"

	"0chain.net/smartcontract/stakepool/spenum"
)

func (edb *EventDb) rewardUpdate(spu StakePoolReward) error {

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

func (edb *EventDb) rewardProvider(spu StakePoolReward) error {
	if spu.Reward == 0 {
		return nil
	}

	var err error
	update := NewDbUpdates(spu.ProviderId)

	switch spenum.Provider(spu.ProviderType) {
	case spenum.Blobber:
		{
			blobber, err := edb.blobberAggregateStats(spu.ProviderId)
			if err != nil {
				return err
			}
			update.Updates["reward"] = blobber.Reward + spu.Reward
			update.Updates["total_service_charge"] = blobber.TotalServiceCharge + spu.Reward
			err = edb.updateBlobber(*update)
		}
	case spenum.Validator:
		{
			validator, err := edb.validatorAggregateStats(spu.ProviderId)
			if err != nil {
				return err
			}
			update.Updates["reward"] = validator.Reward + spu.Reward
			update.Updates["total_service_charge"] = validator.TotalReward + spu.Reward
			err = edb.updateValidator(*update)
		}
	default:
		return fmt.Errorf("not implented provider type %v", spu.ProviderType)
	}
	if err != nil {
		return err
	}

	return nil
}
