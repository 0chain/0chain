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
	if deltaCapacity > 0 || deltaUsed > 0 { // todo what to do if sc.yaml block rewards changes
		if sp.stake() >= conf.BlockReward.QualifyingStake {
			balances.UpdateBlockRewardTotals(deltaCapacity, deltaUsed)
		}
	}

	if err := payBlobberRewards(blobber, sp, qtl, balances); err != nil {
		return fmt.Errorf("paying blobber rewards: %v", err)
	}
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
	newStake := originalStake - change

	qualify := conf.BlockReward.QualifyingStake
	if originalStake >= qualify && newStake < qualify {
		balances.UpdateBlockRewardTotals(-1*blobber.Capacity, -1*blobber.Used)
	} else if originalStake < qualify && newStake >= qualify {
		balances.UpdateBlockRewardTotals(blobber.Capacity, blobber.Used)
	}

	return payBlobberRewards(blobber, originalSp, nil, balances)
}

func payBlobberRewards(
	blobber *StorageNode,
	sp *stakePool,
	qtl *blockrewards.QualifyingTotalsList,
	balances cstate.StateContextI,
) error {
	var round = balances.GetBlock().Round
	if qtl == nil {
		var err error
		qtl, err = blockrewards.GetQualifyingTotalsList(balances)
		if err != nil {
			return fmt.Errorf("getting block reward totals: %v", err)
		}
	}

	logging.Logger.Info("piers piers2 start payBlobberRewards",
		zap.Any("list length", len(qtl.Totals)),
		zap.Any("blobber", blobber),
		zap.Any("stake pools", sp),
	)
	if int64(len(qtl.Totals)) < round-1 {
		logging.Logger.Info("piers big payBlobberRewards lengh does not match round",
			zap.Int64("length", int64(len(qtl.Totals))),
			zap.Int64("round", round),
			zap.Any("qtl", qtl.Totals),
		)
		//return fmt.Errorf("block reward totals missing, length %d, exopected %d",
		//	len(qtl.Totals), round)
	}
	if len(qtl.Totals) == 0 {
		return nil
	}
	var stakes = float64(sp.stake())
	if stakes == 0 {
		return nil
	}

	if blobber.LastBlockRewardPaymentRound == 0 {
		blobber.LastBlockRewardPaymentRound = balances.GetBlock().Round // todo should do this in proper place
	}

	var startSettingsWereSet = qtl.Totals[blobber.LastBlockRewardPaymentRound].LastSettingsChange
	var settings = qtl.Totals[startSettingsWereSet].SettingsChange
	if settings == nil {
		return fmt.Errorf("cannot find inital block rewards settings, "+
			"not found on round %d, qtl %v", startSettingsWereSet, qtl)
	}
	var reward float64
	logging.Logger.Info("piers piers2 payBlobberRewards about to calculate reward",
		zap.Int64("blobber.LastBlockRewardPaymentRound", blobber.LastBlockRewardPaymentRound),
		zap.Any("qtl LastBlockRewardPaymentRound", qtl.Totals[blobber.LastBlockRewardPaymentRound]),
		zap.Int64("round", round),
		zap.Int64("startSettingsWereSet", startSettingsWereSet),
		zap.Any("new setting round", qtl.Totals[startSettingsWereSet]),
	)
	for i := blobber.LastBlockRewardPaymentRound; i < round; i++ {
		if qtl.Totals[i].SettingsChange != nil {
			settings = qtl.Totals[i].SettingsChange
			logging.Logger.Info("piers piers2 payBlobberRewards setting change",
				zap.Any("i", i),
				zap.Any("config block rewards settings", settings),
			)
		}

		var capRatio float64
		if qtl.Totals[i].Capacity > 0 {
			capRatio = float64(blobber.Capacity) / float64(qtl.Totals[i].Capacity)
		}
		capacityReward := float64(settings.BlockReward) * settings.BlobberCapacityWeight * capRatio

		var usedRatio float64
		if qtl.Totals[i].Used > 0 {
			usedRatio = float64(blobber.Used) / float64(qtl.Totals[i].Used)
		}
		usedReward := float64(settings.BlockReward) * settings.BlobberUsageWeight * usedRatio
		reward += capacityReward + usedReward

		//logging.Logger.Info("piers piers2 payBlobberRewards calculating reward",
		//	zap.Any("i", i),
		//	zap.Float64("reward", reward),
		//	zap.Float64("capacityReward", capacityReward),
		//	zap.Float64("usedReward", usedReward),
		//	zap.Any("config block rewards settings", settings),
		//)
	}

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

	blobber.LastBlockRewardPaymentRound = balances.GetBlock().Round
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
