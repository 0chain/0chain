package event

import (
	"fmt"

	"0chain.net/smartcontract/dbs"
)

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
		reward, ok := spu.DelegateRewards[dp.PoolID]
		if !ok || reward == 0 {
			continue
		}
		err := edb.updateReward(reward, dp)
		if err != nil {
			return err
		}
	}

	return nil
}

func (edb *EventDb) rewardProvider(spu dbs.StakePoolReward) error {
	if spu.Reward == 0 {
		return nil
	}

	var err error
	update := dbs.NewDbUpdates(spu.ProviderId)

	switch dbs.Provider(spu.ProviderType) {
	case dbs.Blobber:
		{
			blobber, err := edb.blobberAggregateStats(spu.ProviderId)
			if err != nil {
				return err
			}
			update.Updates["reward"] = blobber.Reward + spu.Reward
			update.Updates["total_service_charge"] = blobber.TotalServiceCharge + spu.Reward
			err = edb.updateBlobber(*update)
		}
	case dbs.Validator:
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
