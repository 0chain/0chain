package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"fmt"
)

func (ssc *StorageSmartContract) payBlobberBlockRewards(
	balances cstate.StateContextI,
) (err error) {
	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"cannot get smart contract configurations: "+err.Error())
	}

	if conf.BlockReward.BlobberCapacityWeight+conf.BlockReward.BlobberUsageWeight == 0 ||
		conf.BlockReward.BlockReward == 0 {
		return nil
	}

	blobberStakes, err := getBlobberStakes(balances)
	if err != nil {
		return common.NewErrorf("blobber_block_rewards_failed",
			"cannot get blobbers stakes: %v", err)
	}

	// filter out blobbers with stake too low to qualify for rewards
	var qualifyingBlobberIds = make([]string, 0, len(blobberStakes))
	var stakeTotals = make([]float64, 0, len(blobberStakes))
	var totalQStake float64
	for blobberId, stakes := range blobberStakes {
		if state.Balance(stakes) >= conf.BlockReward.QualifyingStake {
			qualifyingBlobberIds = append(qualifyingBlobberIds, blobberId)
			stakeTotals = append(stakeTotals, float64(stakes))
			totalQStake += float64(stakes)
		}
	}

	mints, err := getBlockRewardMints(ssc, balances)
	if err != nil {
		return fmt.Errorf("Error getting mint info: %v", err)
	}

	for i, blobberId := range qualifyingBlobberIds {
		var ratio float64
		if totalQStake > 0 {
			ratio = stakeTotals[i] / totalQStake
		} else {
			ratio = 1.0 / float64(len(blobberStakes))
		}

		capacityReward := float64(conf.BlockReward.BlockReward) * conf.BlockReward.BlobberCapacityWeight * ratio
		usageReward := float64(conf.BlockReward.BlockReward) * conf.BlockReward.BlobberUsageWeight * ratio
		if err := mints.addMint(blobberId, capacityReward+usageReward); err != nil {
			return common.NewErrorf("blobber_block_rewards_failed",
				"error minting for blobber %v: %v", blobberId, err)
		}
	}

	// block rewards for each blobber
	// will be minted next time getStakePool is called.
	if err := mints.save(balances); err != nil {
		return common.NewErrorf("blobber_block_rewards_failed",
			"cannot save block reward mints: %v", err)
	}

	return nil
}
