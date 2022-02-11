package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/maths"
	"0chain.net/smartcontract/partitions"
	"math"
	"math/rand"
	"time"
)

func (ssc *StorageSmartContract) blobberBlockRewards(
	balances cstate.StateContextI,
) (err error) {
	var (
		qualifyingBlobberIds []string
		stakePools           []*stakePool
		stakeTotals          []float64
		totalQStake          float64
		weight               []float64
		totalWeight          float64
		conf                 *scConfig
	)

	const (
		// constants for gamma
		alpha = 1
		A     = 1
		B     = 1

		// constants for zeta
		I  = 1
		K  = 1
		mu = 1
	)

	if conf, err = ssc.getConfig(balances, true); err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"cannot get smart contract configurations: "+err.Error())
	}

	if conf.BlockReward.BlockReward == 0 {
		return nil
	}

	bbr := getBlockReward(conf.BlockReward.BlockReward, balances.GetBlock().Round,
		conf.BlockReward.BlockRewardChangePeriod, conf.BlockReward.BlockRewardChangeRatio, conf.BlockReward.BlobberWeight)

	allBlobbers, err := getActivePassedBlobbersList(balances)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"cannot get all blobbers list: "+err.Error())
	}

	random := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

	blobberPartition, err := allBlobbers.GetRandomSlice(random, balances)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"Error getting random partition: "+err.Error())
	}

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

		gamma := maths.GetGamma(A, B, alpha, blobber.TotalData, blobber.DataRead)
		zeta := maths.GetZeta(I, K, mu, float64(blobber.WritePrice), float64(blobber.ReadPrice))
		qualifyingBlobberIds = append(qualifyingBlobberIds, blobber.Id)
		stakePools = append(stakePools, sp)
		stakeTotals = append(stakeTotals, stake)
		totalQStake += stake
		blobberWeight := (gamma*zeta*float64(blobber.SuccessChallenges) + 1) * stake
		weight = append(weight, blobberWeight)
		totalWeight += blobberWeight
	}

	if totalWeight == 0 {
		totalWeight = 1
	}

	for i, qsp := range stakePools {
		reward := bbr * (weight[i] / totalWeight)

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

func getBlockReward(br state.Balance, currentRound, brChangePeriod int64, brChangeRatio, blobberWeight float64) float64 {
	changeBalance := 1 - brChangeRatio
	changePeriods := currentRound % brChangePeriod
	return float64(br) * math.Pow(changeBalance, float64(changePeriods)) * blobberWeight
}
