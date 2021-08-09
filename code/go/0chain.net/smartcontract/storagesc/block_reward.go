package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"fmt"
	"go.uber.org/zap"
)

func (ssc *StorageSmartContract) payBlobberBlockRewards(
	balances cstate.StateContextI,
) (err error) {
	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return common.NewError("blobber_block_rewards",
			"cannot get smart contract configurations: "+err.Error())
	}

	if conf.BlockReward.BlobberCapacityWeight+conf.BlockReward.BlobberUsageWeight == 0 ||
		conf.BlockReward.BlockReward == 0 {
		return nil
	}

	bs, err := getBlobberStakeTotals(balances)
	if err != nil {
		return common.NewErrorf("blobber_block_rewards",
			"cannot get blobbers stakes: %v", err)
	}
	logging.Logger.Info("blobber_block_rewards", zap.Any("blobberStakes", bs))

	// filter out blobbers with stake too low to qualify for rewards
	var qualifyingBlobberIds = make([]string, 0, len(bs.StakeTotals))
	var totalQCapacity float64
	var totalQUsage float64
	for blobberId, stakes := range bs.StakeTotals {
		if state.Balance(stakes) >= conf.BlockReward.QualifyingStake {
			qualifyingBlobberIds = append(qualifyingBlobberIds, blobberId)
			totalQCapacity += float64(bs.Capacities[blobberId])
			totalQUsage += float64(bs.Used[blobberId])
		}
		logging.Logger.Info("blobber_block_rewards",
			zap.Any("for loop blobber id", blobberId),
			zap.Any("stakes", stakes),
			zap.Any("QualifyingStake", conf.BlockReward.QualifyingStake),
		)
	}
	logging.Logger.Info("blobber_block_rewards",
		zap.Any("qualifyingBlobberIds", qualifyingBlobberIds),
		zap.Any("totalQCapacity", totalQCapacity),
		zap.Any("totalQUsage", totalQUsage),
	)
	logging.Logger.Info("blobber_block_rewards weights",
		zap.Any("BlockReward", conf.BlockReward.BlockReward),
		zap.Any("BlobberCapacityWeight", conf.BlockReward.BlobberCapacityWeight),
		zap.Any("BlobberUsageWeight", conf.BlockReward.BlobberUsageWeight),
	)
	mints, err := getBlockRewardMints(ssc, balances)
	if err != nil {
		return fmt.Errorf("Error getting mint info: %v", err)
	}
	logging.Logger.Info("blobber_block_rewards", zap.Any("mints before", mints))
	for _, id := range qualifyingBlobberIds {
		var capRatio float64
		if totalQCapacity > 0 {
			capRatio = float64(bs.Capacities[id]) / totalQCapacity
		} else {
			capRatio = 1.0 / float64(len(qualifyingBlobberIds))
		}
		var useRatio float64
		if totalQUsage > 0 {
			useRatio = float64(bs.Used[id]) / totalQUsage
		} else {
			useRatio = 1.0 / float64(len(qualifyingBlobberIds))
		}

		capacityReward := float64(conf.BlockReward.BlockReward) * conf.BlockReward.BlobberCapacityWeight * capRatio
		usageReward := float64(conf.BlockReward.BlockReward) * conf.BlockReward.BlobberUsageWeight * useRatio
		logging.Logger.Info("blobber_block_rewards for loop qualifying blobbers",
			zap.Any("blobber id", id),
			zap.Any("capacityReward", capacityReward),
			zap.Any("usageReward", usageReward),
		)
		if err := mints.addMint(id, capacityReward+usageReward, conf); err != nil {
			return common.NewErrorf("blobber_block_rewards_failed",
				"error minting for blobber %v: %v", id, err)
		}
	}
	logging.Logger.Info("blobber_block_rewards", zap.Any("mints after", mints))
	// block rewards for each blobber
	// will be minted next time getStakePool is called.
	if err := mints.save(balances); err != nil {
		return common.NewErrorf("blobber_block_rewards_failed",
			"cannot save block reward mints: %v", err)
	}

	// save configuration (minted tokens)
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"saving configurations: "+err.Error())
	}

	return nil
}
