package storagesc

import (
	"fmt"

	"0chain.net/core/util"

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
	qtl *blockrewards.QualifyingTotalsList,
) error {
	if sp.stake() < conf.BlockReward.QualifyingStake {
		blobber.LastBlockRewardPaymentRound = balances.GetBlock().Round
		return nil
	}

	if deltaCapacity > 0 || deltaUsed > 0 {
		balances.UpdateBlockRewardTotals(deltaCapacity, deltaUsed)
	}
	if err := payBlobberRewards(blobber, sp, qtl, conf, balances); err != nil {
		return fmt.Errorf("paying blobber rewards: %v", err)
	}
	blobber.LastBlockRewardPaymentRound = balances.GetBlock().Round
	return nil
}

func blockRewardModifiedStakePool(
	change state.Balance,
	originalSp *stakePool,
	conf *scConfig,
	blobber *StorageNode,
	ssc *StorageSmartContract,
	balances cstate.StateContextI,
) error {
	originalStake := originalSp.stake()
	newStake := originalStake + change
	qualify := conf.BlockReward.QualifyingStake
	logging.Logger.Info("piers piers2 blockRewardModifiedStakePool start",
		zap.Int64("round", balances.GetBlock().Round),
	)
	if originalStake < qualify {
		if newStake >= qualify {
			balances.UpdateBlockRewardTotals(blobber.Capacity, blobber.Used)
		}
	} else {
		if newStake < qualify {
			balances.UpdateBlockRewardTotals(-1*blobber.Capacity, -1*blobber.Used)
		}
		err := payBlobberRewards(blobber, originalSp, nil, conf, balances)
		if err != nil {
			return err
		}
	}
	blobber.LastBlockRewardPaymentRound = balances.GetBlock().Round
	return nil
}

func calculateReward(
	qtl *blockrewards.QualifyingTotalsList,
	brc *blockRewardChanges,
	start, end int64,
	capacity, used int64,
) float64 {
	var reward float64
	round := start - 1
	logging.Logger.Info("piers2 calculateReward",
		zap.Int64("start", start),
		zap.Int64("end", end),
		zap.Int("qtl lenght", len(qtl.Totals)),
		zap.Int64("capacity", capacity),
		zap.Int64("used", used),
		zap.Any("brc", brc),
	)
	for change, i := brc.getLatestChange(); change != nil; change, i = brc.getPreviousChange(i) {
		settings := change.Change
		for ; round >= brc.Changes[i].Round && round >= end; round-- {
			var capRatio float64
			if qtl.GetCapacity(round) > 0 {
				capRatio = float64(capacity) / float64(qtl.GetCapacity(round))
			}
			capacityReward := float64(settings.BlockReward) * settings.BlobberCapacityWeight * capRatio

			var usedRatio float64
			if qtl.GetUsed(round) > 0 {
				usedRatio = float64(used) / float64(qtl.GetUsed(round))
			}
			usedReward := float64(settings.BlockReward) * settings.BlobberUsageWeight * usedRatio
			reward += capacityReward + usedReward
			//fmt.Println("piers3 i", i, "round", round, "cap", capacityReward,
			//	"used", usedReward, "reward", reward)
		}
		if brc.Changes[i].Round < end {
			return reward
		}
	}
	return reward
}

func payBlobberRewards(
	blobber *StorageNode,
	sp *stakePool,
	qtl *blockrewards.QualifyingTotalsList,
	conf *scConfig,
	balances cstate.StateContextI,
) error {
	logging.Logger.Info("piers piers2 payBlobberRewards start",
		zap.Int64("round", balances.GetBlock().Round),
	)
	if qtl == nil {
		var err error
		qtl, err = blockrewards.GetQualifyingTotalsList(blobber.LastBlockRewardPaymentRound, balances)
		if err != nil {
			return fmt.Errorf("getting block reward totals: %v", err)
		}
	}
	logging.Logger.Info("piers piers2 payBlobberRewards",
		zap.Int64("round", balances.GetBlock().Round),
		zap.Int("qtl length", len(qtl.Totals)),
	)

	brc, err := getBlockRewardChanges(balances)
	logging.Logger.Info("piers2 calculateReward",
		zap.Int64("round", balances.GetBlock().Round),
		zap.Int64("LastBlockRewardPaymentRound", blobber.LastBlockRewardPaymentRound),
		zap.Any("block reward changes", brc),
		zap.Error(err),
	)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		brc = newBlockRewardChanges(conf)
	}
	logging.Logger.Info("piers2 calculateReward after startBlockRewardChanges",
		zap.Int64("round", balances.GetBlock().Round),
		zap.Int64("LastBlockRewardPaymentRound", blobber.LastBlockRewardPaymentRound),
		zap.Any("block reward changes", brc),
		zap.Error(err),
	)
	if blobber.LastBlockRewardPaymentRound == 0 {
		panic("should not happen") // todo remove
		blobber.LastBlockRewardPaymentRound = balances.GetBlock().Round
		return nil
	}

	var reward = calculateReward(
		qtl,
		brc,
		balances.GetBlock().Round,
		blobber.LastBlockRewardPaymentRound,
		blobber.Capacity, blobber.Used,
	)
	logging.Logger.Info("piers2 calculated Reward",
		zap.Int64("round", balances.GetBlock().Round),
		zap.Float64("reward", reward),
	)

	stakes := float64(sp.stake())
	fmt.Println("reward", reward, "stakes", stakes)
	for _, pool := range sp.Pools {
		poolReward := pool.BlockRewardCarry + reward*float64(pool.Balance)/stakes
		toMint := state.Balance(poolReward)
		pool.BlockRewardCarry = poolReward - float64(toMint)
		if err := balances.AddMint(state.NewMint(ADDRESS, pool.DelegateID, toMint)); err != nil {
			return fmt.Errorf(
				"error miniting block reward, mint: %v\terr: %v",
				state.NewMint(ADDRESS, pool.DelegateID, toMint), err,
			)
		}
		logging.Logger.Info("piers2 payBlobberRewards paying stakeholder",
			zap.Int64("reward", int64(reward)),
			zap.Int64("poolReward", int64(poolReward)),
			zap.Float64("carry", pool.BlockRewardCarry),
			zap.Any("stake pools", pool),
			zap.Any("minted", state.NewMint(ADDRESS, pool.DelegateID, toMint)),
		)
	}

	if len(qtl.Totals) > 3 {
		logging.Logger.Info("piers2 end payBlobberRewards",
			zap.Int("length qtl", len(qtl.Totals)),
			zap.Any("qtl last block", (qtl.Totals)[balances.GetBlock().Round-1]),
			zap.Any("blobber", blobber),
			zap.Any("stake pools", sp),
		)
	} else {
		logging.Logger.Info("piers2 end payBlobberRewards",
			zap.Int("length qtl", len(qtl.Totals)),
			zap.Any("list", qtl.Totals),
			zap.Any("blobber", blobber),
			zap.Any("stake pools", sp),
		)
	}

	return nil
}
