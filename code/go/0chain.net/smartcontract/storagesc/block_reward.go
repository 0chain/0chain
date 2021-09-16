package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
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

	allBlobbers, err := ssc.getBlobbersList(balances)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"cannot get all blobbers list: "+err.Error())
	}

	// filter out blobbers with stake too low to qualify for rewards
	var qualifyingBlobberIds []string
	var stakePools []*stakePool
	var stakeTotals []float64
	var totalQStake float64
	for _, blobber := range allBlobbers.Nodes {
		var sp *stakePool
		if sp, err = ssc.getStakePool(blobber.ID, balances); err != nil {
			return common.NewError("blobber_block_rewards_failed",
				"can't get related stake pool: "+err.Error())
		}
		var stake float64
		for _, delegate := range sp.Pools {
			stake += float64(delegate.Balance)
		}
		if state.Balance(stake) >= conf.BlockReward.QualifyingStake {
			qualifyingBlobberIds = append(qualifyingBlobberIds, blobber.ID)
			stakePools = append(stakePools, sp)
			stakeTotals = append(stakeTotals, stake)
			totalQStake += stake
		}
	}

	for i, qsp := range stakePools {
		var ratio float64
		if totalQStake > 0 {
			ratio = stakeTotals[i] / totalQStake
		} else {
			ratio = 1.0 / float64(len(stakePools))
		}

		capacityReward := float64(conf.BlockReward.BlockReward) * conf.BlockReward.BlobberCapacityWeight * ratio
		if err := mintReward(qsp, capacityReward, balances); err != nil {
			return common.NewError("blobber_block_rewards_failed", "minting capacity reward"+err.Error())
		}
		usageReward := float64(conf.BlockReward.BlockReward) * conf.BlockReward.BlobberUsageWeight * ratio
		if err := mintReward(qsp, usageReward, balances); err != nil {
			return common.NewError("blobber_block_rewards_failed", "minting usage reward"+err.Error())
		}
		qsp.Rewards.Blobber += state.Balance(capacityReward + usageReward)
	}

	for i, qsp := range stakePools {
		if err = qsp.save(ssc.ID, qualifyingBlobberIds[i], balances); err != nil {
			return common.NewError("blobber_block_rewards_failed",
				"saving stake pool: "+err.Error())
		}
	}

	// save configuration (minted tokens)
	_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"saving configurations: "+err.Error())
	}

	return nil
}
