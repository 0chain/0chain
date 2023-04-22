package event

import "0chain.net/smartcontract/stakepool/spenum"

func (edb *EventDb) GetRewardsToDelegates(blockNumber, startBlockNumber, endBlockNumber string, rewardType int) []RewardDelegate {

	if blockNumber != "" {
		var rds []RewardDelegate
		edb.Get().Where("block_number = ? AND reward_type = ?", blockNumber, rewardType).Find(&rds)

		return rds
	}

	if startBlockNumber != "" && endBlockNumber != "" {
		var rds []RewardDelegate
		edb.Get().Where("block_number >= ? AND block_number <= ? AND reward_type = ?", startBlockNumber, endBlockNumber, rewardType).Find(&rds)

		return rds
	}

	return nil

}

func (edb *EventDb) GetChallengeRewardsToDelegates(challengeID string) ([]RewardDelegate, []RewardDelegate) {

	var blobberRewards []RewardDelegate
	edb.Get().Where("challenge_id = ? AND reward_type = ?", challengeID, spenum.ChallengePassReward).Find(&blobberRewards)

	var validatorRewards []RewardDelegate
	edb.Get().Where("challenge_id = ? AND reward_type = ?", challengeID, spenum.ValidationReward).Find(&validatorRewards)

	return blobberRewards, validatorRewards
}

func (edb *EventDb) GetAllocationCancellationRewardsToDelegates(startBlock, endBlock string) []RewardDelegate {

	var rps []RewardDelegate
	edb.Get().Where("block_number >= ? AND block_number <= ? AND reward_type = ?", startBlock, endBlock, spenum.CancellationChargeReward).Find(&rps)

	return rps
}
