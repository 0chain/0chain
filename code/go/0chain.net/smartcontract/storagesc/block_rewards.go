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
	qtl *blockrewards.QualifyingTotalsList,
) error {
	if sp.stake() < conf.BlockReward.QualifyingStake {
		blobber.LastBlockRewardPaymentRound = balances.GetBlock().Round
		return nil
	}

	if deltaCapacity > 0 || deltaUsed > 0 {
		balances.UpdateBlockRewardTotals(deltaCapacity, deltaUsed)
	}
	if err := payBlobberRewards(blobber, sp, qtl, balances); err != nil {
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

	if originalStake < qualify {
		if newStake >= qualify {
			balances.UpdateBlockRewardTotals(blobber.Capacity, blobber.Used)
		}
	} else {
		if newStake < qualify {
			balances.UpdateBlockRewardTotals(-1*blobber.Capacity, -1*blobber.Used)
		}
		err := payBlobberRewards(blobber, originalSp, nil, balances)
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
	for i := len(brc.Changes) - 1; i >= 0; i-- {
		settings := brc.Changes[i].Change
		for ; round > brc.Changes[i].Round && round >= end; round-- {
			var capRatio float64
			if qtl.Totals[round].Capacity > 0 {
				capRatio = float64(capacity) / float64(qtl.Totals[round].Capacity)
			}
			capacityReward := float64(settings.BlockReward) * settings.BlobberCapacityWeight * capRatio

			var usedRatio float64
			if qtl.Totals[round].Used > 0 {
				usedRatio = float64(used) / float64(qtl.Totals[round].Used)
			}
			usedReward := float64(settings.BlockReward) * settings.BlobberUsageWeight * usedRatio
			reward += capacityReward + usedReward
			fmt.Println("i", i, "round", round, "cap", capacityReward,
				"used", usedReward, "reward", reward,
				"qtl.cap", qtl.Totals[round].Capacity, "qtl.used", qtl.Totals[round].Used)
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
	balances cstate.StateContextI,
) error {
	//var round = balances.GetBlock().Round
	if qtl == nil {
		var err error
		qtl, err = blockrewards.GetQualifyingTotalsList(balances)
		if err != nil {
			return fmt.Errorf("getting block reward totals: %v", err)
		}
	}

	brc, err := getBlockRewardChanges(balances)
	if err != nil {
		return err
	}

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

	stakes := float64(sp.stake())
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
		logging.Logger.Info("piers piers2 payBlobberRewards paying stakeholder",
			zap.Int64("reward", int64(reward)),
			zap.Int64("poolReward", int64(poolReward)),
			zap.Float64("carry", pool.BlockRewardCarry),
			zap.Any("stake pools", pool),
		)
	}

	if len(qtl.Totals) > 3 {
		logging.Logger.Info("piers piers2 end payBlobberRewards",
			zap.Int("length qtl", len(qtl.Totals)),
			zap.Any("qtl last block", (qtl.Totals)[balances.GetBlock().Round-1]),
			zap.Any("blobber", blobber),
			zap.Any("stake pools", sp),
		)
	} else {
		logging.Logger.Info("piers piers2 end payBlobberRewards",
			zap.Int("length qtl", len(qtl.Totals)),
			zap.Any("list", qtl.Totals),
			zap.Any("blobber", blobber),
			zap.Any("stake pools", sp),
		)
	}

	return nil
}
