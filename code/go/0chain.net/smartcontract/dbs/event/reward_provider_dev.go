package event

import (
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/logging"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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

func (edb *EventDb) GetAllocationCancellationRewardsToProviders(startBlock, endBlock string) ([]RewardProvider, error) {

	var rps []RewardProvider
	err := edb.Get().Where("block_number >= ? AND block_number <= ? AND reward_type = ?", startBlock, endBlock, spenum.CancellationChargeReward).Find(&rps).Error

	return rps, err
}

func (edb *EventDb) GetAllocationChallengeRewards(allocationID string) (map[string]ProviderAllocationRewards, error) {

	var result = make(map[string]ProviderAllocationRewards)

	var rps []ProviderAllocationReward

	err := edb.Get().Table("reward_providers").Select("provider_id, reward_type,  sum(amount) as amount").Where("allocation_id = ? AND reward_type IN (?, ?)", allocationID, spenum.ValidationReward, spenum.ChallengePassReward).Group("provider_id, reward_type").Scan(&rps).Error
	if err != nil {
		return nil, err
	}

	for _, rp := range rps {
		amount := rp.Amount

		var deleagateRewards []DelegateAllocationReward
		err = edb.Get().Table("reward_delegates").Select("pool_id as delegate_id, sum(amount) as amount").Where("provider_id = ? AND allocation_id = ? AND reward_type IN (?, ?)", rp.ProviderId, allocationID, spenum.ValidationReward, spenum.ChallengePassReward).Group("pool_id").Scan(&deleagateRewards).Error

		if err != nil {
			return nil, err
		}

		result[rp.ProviderId] = ProviderAllocationRewards{
			Amount:       amount,
			Total:        0,
			ProviderType: rp.RewardType,
		}

		totalProviderReward := amount

		var providerDelegateRewards = make(map[string]int64)

		for _, dr := range deleagateRewards {
			providerDelegateRewards[dr.DelegateID] = dr.Amount
			totalProviderReward += dr.Amount
		}

		providerReward := result[rp.ProviderId]
		providerReward.Total = totalProviderReward
		providerReward.DelegateRewards = providerDelegateRewards
		result[rp.ProviderId] = providerReward
	}

	return result, nil
}

func (edb *EventDb) GetAllocationReadRewards(allocationID string) (map[string]ProviderAllocationRewards, error) {
	var result = make(map[string]ProviderAllocationRewards)

	var rps []ProviderAllocationReward

	err := edb.Get().Table("reward_providers").Select("provider_id, sum(amount) as amount").Where("allocation_id = ? AND reward_type = ?", allocationID, spenum.FileDownloadReward).Group("provider_id, reward_type").Scan(&rps).Error
	if err != nil {
		return nil, err
	}

	for _, rp := range rps {
		amount := rp.Amount

		var deleagateRewards []DelegateAllocationReward
		err = edb.Get().Table("reward_delegates").Select("pool_id as delegate_id, sum(amount) as amount").Where("provider_id = ? AND allocation_id = ? AND reward_type = ?", rp.ProviderId, allocationID, spenum.FileDownloadReward).Group("pool_id").Scan(&deleagateRewards).Error

		if err != nil {
			return nil, err
		}

		result[rp.ProviderId] = ProviderAllocationRewards{
			Amount:       amount,
			Total:        0,
			ProviderType: rp.RewardType,
		}

		totalProviderReward := amount

		var providerDelegateRewards = make(map[string]int64)

		for _, dr := range deleagateRewards {
			providerDelegateRewards[dr.DelegateID] = dr.Amount
			totalProviderReward += dr.Amount
		}

		providerReward := result[rp.ProviderId]
		providerReward.Total = totalProviderReward
		providerReward.DelegateRewards = providerDelegateRewards
		result[rp.ProviderId] = providerReward
	}

	return result, nil
}

func (edb *EventDb) GetAllocationCancellationRewards(allocationID string) (map[string]ProviderAllocationRewards, error) {
	var result = make(map[string]ProviderAllocationRewards)

	var rps []ProviderAllocationReward

	err := edb.Get().Table("reward_providers").Select("provider_id, sum(amount) as amount").Where("allocation_id = ? AND reward_type = ?", allocationID, spenum.CancellationChargeReward).Group("provider_id, reward_type").Scan(&rps).Error
	if err != nil {
		return nil, err
	}

	for _, rp := range rps {
		amount := rp.Amount

		var deleagateRewards []DelegateAllocationReward
		err = edb.Get().Table("reward_delegates").Select("pool_id as delegate_id, sum(amount) as amount").Where("provider_id = ? AND allocation_id = ? AND reward_type = ?", rp.ProviderId, allocationID, spenum.CancellationChargeReward).Group("pool_id").Scan(&deleagateRewards).Error

		if err != nil {
			return nil, err
		}

		result[rp.ProviderId] = ProviderAllocationRewards{
			Amount:       amount,
			Total:        0,
			ProviderType: rp.RewardType,
		}

		totalProviderReward := amount

		var providerDelegateRewards = make(map[string]int64)

		for _, dr := range deleagateRewards {
			providerDelegateRewards[dr.DelegateID] = dr.Amount
			totalProviderReward += dr.Amount
		}

		providerReward := result[rp.ProviderId]
		providerReward.Total = totalProviderReward
		providerReward.DelegateRewards = providerDelegateRewards
		result[rp.ProviderId] = providerReward
	}

	return result, nil
}

func (edb *EventDb) GetBlockRewards(startBlock, endBlock string) ([]int64, error) {

	uniqueIdForLogging := uuid.New().String()

	var result []int64
	var totals []int64

	var blockRewards []BlockReward

	err := edb.Get().Table("reward_providers").Select("provider_id, sum(amount) as amount").Where("block_number >= ? AND block_number <= ?", startBlock, endBlock).Group("provider_id").Scan(&blockRewards).Error
	if err != nil {
		return nil, err
	}

	logging.Logger.Info("jayash blockRewards : "+uniqueIdForLogging, zap.Any("blockRewards", blockRewards))

	for _, br := range blockRewards {
		result = append(result, br.Amount)
	}

	logging.Logger.Info("jayash result : "+uniqueIdForLogging, zap.Any("result", result))

	for _, br := range blockRewards {

		var delegateRewards BlockReward
		err = edb.Get().Table("reward_delegates").Select("provider_id, sum(amount) as amount").Where("block_number >= ? AND block_number <= ?", startBlock, endBlock).Group("provider_id").Scan(&delegateRewards).Error
		if err != nil {
			return nil, err
		}

		logging.Logger.Info("jayash delegateRewards : "+uniqueIdForLogging, zap.Any("delegateRewards", delegateRewards))

		result = append(result, delegateRewards.Amount)

		logging.Logger.Info("jayash result2 : "+uniqueIdForLogging, zap.Any("result", result))

		totals = append(totals, br.Amount+delegateRewards.Amount)

		logging.Logger.Info("jayash totals : "+uniqueIdForLogging, zap.Any("totals", totals))
	}

	result = append(result, totals...)

	logging.Logger.Info("jayash result3 : "+uniqueIdForLogging, zap.Any("result", result))

	return result, err
}

type ProviderAllocationRewards struct {
	DelegateRewards map[string]int64 `json:"delegate_rewards"`
	Amount          int64            `json:"amount"`
	Total           int64            `json:"total"`
	ProviderType    int64            `json:"provider_type"`
}

type DelegateAllocationReward struct {
	DelegateID string `json:"delegate_id"`
	Amount     int64  `json:"amount"`
}

type BlockReward struct {
	ProviderID string `json:"provider_id"`
	Amount     int64  `json:"amount"`
}

type ProviderAllocationReward struct {
	ProviderId string `json:"provider_id"`
	Amount     int64  `json:"amount"`
	RewardType int64  `json:"reward_type"`
}
