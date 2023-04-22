package event

import "0chain.net/smartcontract/stakepool/spenum"

func (edb *EventDb) GetRewardToProviders(blockNumber, startBlockNumber, endBlockNumber string, rewardType int) []RewardProvider {

	if blockNumber != "" {
		var rps []RewardProvider
		edb.Get().Where("block_number = ? AND reward_type = ?", blockNumber, rewardType).Find(&rps)

		return rps
	}

	if startBlockNumber != "" && endBlockNumber != "" {
		var rps []RewardProvider
		edb.Get().Where("block_number >= ? AND block_number <= ? AND reward_type = ?", startBlockNumber, endBlockNumber, rewardType).Find(&rps)

		return rps
	}

	return nil
}

func (edb *EventDb) GetChallengeRewardsToProviders(challengeID string) ([]RewardProvider, []RewardProvider) {

	var blobberRewards []RewardProvider
	edb.Get().Where("challenge_id = ? AND reward_type = ?", challengeID, spenum.ChallengePassReward).Find(&blobberRewards)

	var validatorRewards []RewardProvider
	edb.Get().Where("challenge_id = ? AND reward_type = ?", challengeID, spenum.ValidationReward).Find(&validatorRewards)

	return blobberRewards, validatorRewards
}

func (edb *EventDb) GetAllocationCancellationRewardsToProviders(startBlock, endBlock string) []RewardProvider {

	var rps []RewardProvider
	edb.Get().Where("block_number >= ? AND block_number <= ? AND reward_type = ?", startBlock, endBlock, spenum.CancellationChargeReward).Find(&rps)

	return rps
}
