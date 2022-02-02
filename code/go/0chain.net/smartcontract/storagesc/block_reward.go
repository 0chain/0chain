package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
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

	allBlobbers, err := getBlobbersList(balances)
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
		var blobber *StorageNode
		if sp, err = ssc.getStakePool(b.Name(), balances); err != nil {
			return common.NewError("blobber_block_rewards_failed",
				"can't get related stake pool: "+err.Error())
		}

		if blobber, err = ssc.getBlobber(b.Name(), balances); err != nil {
			return common.NewError("blobber_block_rewards_failed",
				"can't get blobber detail: "+err.Error())
		}
		var stake float64
		for _, delegate := range sp.Pools {
			stake += float64(delegate.Balance)
		}
		bc, err := ssc.getBlobberChallenge(b.Name(), balances)
		if err != nil {
			return common.NewError("blobber_block_rewards_failed",
				"can't get blobbers challenge: "+err.Error())
		}
		successChallenge := bc.getSuccessCount(true, balances.GetTransaction().CreationDate,
			conf.BlockReward.ChallengePeriod, blobber.Terms.ChallengeCompletionTime)

		qualifyingBlobberIds = append(qualifyingBlobberIds, blobber.ID)
		stakePools = append(stakePools, sp)
		stakeTotals = append(stakeTotals, stake)
		totalQStake += stake
		blobberWeight := float64(blobber.Terms.WritePrice) * stake * float64(successChallenge)
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
