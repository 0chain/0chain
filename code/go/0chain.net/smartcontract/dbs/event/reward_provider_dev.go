package event

import (
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/pkg/errors"
)

func (edb *EventDb) GetRewardToProviders(blockNumber, startBlockNumber, endBlockNumber string, rewardType int) ([]RewardProvider, error) {

	if blockNumber != "" {
		var rps []RewardProvider
		err := edb.Get().Where("block_number = ? AND reward_type = ?", blockNumber, rewardType).Find(&rps).Error
		return rps, err
	}

	if startBlockNumber != "" && endBlockNumber != "" {
		var rps []RewardProvider
		err := edb.Get().Where("block_number >= ? AND block_number <= ? AND reward_type = ?", startBlockNumber, endBlockNumber, rewardType).Find(&rps).Error

		return rps, err
	}

	return nil, errors.Errorf("start or end block number can't be empty")
}

func (edb *EventDb) GetChallengeRewardsToProviders(challengeID string) ([]RewardProvider, []RewardProvider, error) {

	var blobberRewards []RewardProvider
	err := edb.Get().Where("challenge_id = ? AND reward_type = ?", challengeID, spenum.ChallengePassReward).Find(&blobberRewards).Error
	if err != nil {
		return nil, nil, err
	}

	var validatorRewards []RewardProvider
	err = edb.Get().Where("challenge_id = ? AND reward_type = ?", challengeID, spenum.ValidationReward).Find(&validatorRewards).Error

	if err != nil {
		return nil, nil, err
	}

	return blobberRewards, validatorRewards, nil
}

func (edb *EventDb) GetAllocationCancellationRewardsToProviders(startBlock, endBlock string) ([]RewardProvider, error) {

	var rps []RewardProvider
	err := edb.Get().Where("block_number >= ? AND block_number <= ? AND reward_type = ?", startBlock, endBlock, spenum.CancellationChargeReward).Find(&rps).Error

	return rps, err
}
