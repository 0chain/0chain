package storagesc

import (
	"fmt"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"0chain.net/smartcontract/storagesc/blockrewards"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
)

func updateBlockRewards(
	deltaCapacity, deltaUsed int64,
	blobber *StorageNode,
	sp *stakePool,
	conf *scConfig,
	balances cstate.StateContextI,
	qtl blockrewards.QualifyingTotalsList,
) error {
	if deltaCapacity > 0 || deltaUsed > 0 { // todo what to do if sc.yaml block rewards changes
		if sp.stake() >= conf.BlockReward.QualifyingStake {
			balances.UpdateBlockRewardTotals(deltaCapacity, deltaUsed)
		}
	}

	if err := payBlobberRewards(blobber, sp, conf, qtl, balances); err != nil {
		return fmt.Errorf("paying blobber rewards: %v", err)
	}
	return nil
}

func blockRewardModifiedStakePool(
	newStake state.Balance,
	conf *scConfig,
	blobber *StorageNode,
	ssc *StorageSmartContract,
	balances cstate.StateContextI,
) error {
	var err error
	var sp *stakePool
	if sp, err = ssc.getStakePool(blobber.ID, balances); err != nil { // todo is ok to get twice?
		return fmt.Errorf("can't get stake pool: %v", err)
	}
	originalStake := sp.stake()

	qualify := conf.BlockReward.QualifyingStake
	if originalStake >= qualify && newStake < qualify {
		balances.UpdateBlockRewardTotals(-1*blobber.Capacity, -1*blobber.Used)
	} else if originalStake < qualify && newStake >= qualify {
		balances.UpdateBlockRewardTotals(blobber.Capacity, blobber.Used)
	}

	return payBlobberRewards(blobber, sp, conf, nil, balances)
}

func payBlobberRewards(
	blobber *StorageNode,
	sp *stakePool,
	conf *scConfig,
	qtl blockrewards.QualifyingTotalsList,
	balances cstate.StateContextI,
) error {
	if qtl == nil {
		var err error
		qtl, err = blockrewards.GetQualifyingTotalsList(balances)
		if err != nil {
			return fmt.Errorf("getting block reward totals: %v", err)
		}
	}

	logging.Logger.Info("piers piers2 start payBlobberRewards",
		zap.Any("list length", len(qtl)),
		zap.Any("blobber", blobber),
		zap.Any("stake pools", sp),
	)
	if int64(len(qtl)) < balances.GetBlock().Round-1 {
		return fmt.Errorf("block reward totals not saved, length %d, round %d",
			len(qtl), balances.GetBlock().Round)
	}
	if len(qtl) == 0 {
		return nil
	}
	var stakes = float64(sp.stake())
	if stakes == 0 {
		return nil
	}
	numRounds := balances.GetBlock().Round - blobber.LastBlockRewardPaymentRound
	if numRounds > int64(len(qtl)) {
		numRounds = int64(len(qtl) - 1)
	}
	var settings blockrewards.BlockReward = *conf.BlockReward
	var reward = blobber.BlockRewardCarry
	for i := int64(0); i < numRounds; i++ {
		index := blobber.LastBlockRewardPaymentRound + i
		if (qtl)[index].SettingsChange != nil {
			settings = *(qtl)[index].SettingsChange
		}

		var capRatio float64
		if (qtl)[index].Capacity > 0 {
			capRatio = float64(blobber.Capacity) / float64((qtl)[index].Capacity)
		}
		capacityReward := float64(settings.BlockReward) * settings.BlobberCapacityWeight * capRatio

		var usedRatio float64
		if (qtl)[index].Used > 0 {
			usedRatio = float64(blobber.Used) / float64((qtl)[index].Used)
		}
		usedReward := float64(settings.BlockReward) * settings.BlobberUsageWeight * usedRatio

		reward += capacityReward + usedReward
	}

	var totalRewardUsed state.Balance
	for _, pool := range sp.Pools {
		poolReward := state.Balance(reward * float64(pool.Balance) / stakes)
		if err := balances.AddMint(state.NewMint(ADDRESS, pool.DelegateID, poolReward)); err != nil {
			return fmt.Errorf(
				"error miniting block reward, mint: %v\terr: %v",
				state.NewMint(ADDRESS, pool.DelegateID, poolReward), err,
			)
		}
		totalRewardUsed += poolReward
	}
	blobber.BlockRewardCarry = reward - float64(totalRewardUsed)
	blobber.LastBlockRewardPaymentRound = balances.GetBlock().Round
	if len(qtl) > 3 {
		logging.Logger.Info("piers piers2 end payBlobberRewards",
			zap.Int("length qtl", len(qtl)),
			zap.Any("qtl last block", (qtl)[balances.GetBlock().Round-1]),
			zap.Any("blobber", blobber),
			zap.Any("stake pools", sp),
		)
	} else {
		logging.Logger.Info("piers piers2 end payBlobberRewards",
			zap.Int("length qtl", len(qtl)),
			zap.Any("list", qtl),
			zap.Any("blobber", blobber),
			zap.Any("stake pools", sp),
		)
	}

	return nil
}
