package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/partitions"
	"math/rand"
	"time"
)

func (ssc *StorageSmartContract) blobberBlockRewards(
	balances cstate.StateContextI,
) (err error) {
	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"cannot get smart contract configurations: "+err.Error())
	}

	if conf.BlockReward.BlockReward == 0 {
		return nil
	}

	allBlobbers, err := getActivePassedBlobbersList(balances)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"cannot get all blobbers list: "+err.Error())
	}

	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

	blobberPartition, err := allBlobbers.GetRandomSlice(r, balances)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"Error getting random partition: "+err.Error())
	}

	// filter out blobbers with stake too low to qualify for rewards
	var qualifyingBlobberIds []string
	var stakePools []*stakePool
	var stakeTotals []float64
	var totalQStake float64
	var weight []float64
	var totalWeight float64
	for _, b := range blobberPartition {
		var sp *stakePool
		var blobber partitions.BlobberRewardNode

		err = blobber.Decode(b.Encode())
		if err != nil {
			return common.NewError("blobber_block_rewards_failed",
				"can't decode blobber reward node: "+err.Error())
		}
		if sp, err = ssc.getStakePool(b.Name(), balances); err != nil {
			return common.NewError("blobber_block_rewards_failed",
				"can't get related stake pool: "+err.Error())
		}

		stake := float64(sp.stake())

		qualifyingBlobberIds = append(qualifyingBlobberIds, blobber.Id)
		stakePools = append(stakePools, sp)
		stakeTotals = append(stakeTotals, stake)
		totalQStake += stake
		blobberWeight := float64(blobber.WritePrice) * stake * float64(blobber.SuccessChallenges)
		weight = append(weight, blobberWeight)
		totalWeight += blobberWeight
	}

	if totalWeight == 0 {
		totalWeight = 1
	}

	for i, qsp := range stakePools {
		reward := float64(conf.BlockReward.BlockReward) * (weight[i] / totalWeight)

		if err := mintReward(qsp, reward, balances); err != nil {
			return common.NewError("blobber_block_rewards_failed", "minting capacity reward"+err.Error())
		}

		qsp.Rewards.Blobber += state.Balance(reward)
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
