package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"math"
	"math/rand"
	"strconv"
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

	hashString := encryption.Hash(balances.GetTransaction().Hash + balances.GetBlock().Hash)
	randomSeed, err := strconv.ParseUint(hashString[0:16], 16, 64)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"Error in creating seed for creating challenges"+err.Error())
	}
	r := rand.New(rand.NewSource(int64(randomSeed)))

	blobberPartition, err := allBlobbers.GetRandomSlice(r, balances)

	// filter out blobbers with stake too low to qualify for rewards
	var qualifyingBlobberIds []string
	var stakePools []*stakePool
	var stakeTotals []float64
	var totalQStake float64
	var weight []float64
	var totalweight float64
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
		totalweight += blobberWeight
	}

	if totalweight == 0 {
		totalweight = 1
	}

	for i, qsp := range stakePools {
		reward := float64(conf.BlockReward.BlockReward) * (weight[i] / totalweight)
		totalDelegateRewards := reward
		for id, delegate := range qsp.Pools {
			var ratio float64
			if stakeTotals[i] > 0 {
				ratio = float64(delegate.Balance) / stakeTotals[i]
			} else {
				ratio = 0
			}
			dReward := float64(conf.BlockReward.BlockReward) * weight[i] * ratio
			finalReward := math.Min(totalDelegateRewards, dReward)
			qsp.Pools[id].Rewards += state.Balance(finalReward)
			totalDelegateRewards -= finalReward
			if totalDelegateRewards < 0 {
				totalDelegateRewards = 0
			}
		}

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
