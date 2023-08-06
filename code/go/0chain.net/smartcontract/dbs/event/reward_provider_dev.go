package event

import (
	"0chain.net/smartcontract/stakepool/spenum"
	"fmt"
	"github.com/0chain/common/core/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/url"
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

	var result []int64
	var totals []int64

	var blockRewards []BlockReward

	err := edb.Get().Table("reward_providers").Select("provider_id, sum(amount) as amount").Where("reward_type = ? AND block_number >= ? AND block_number <= ?", spenum.BlockRewardBlobber, startBlock, endBlock).Group("provider_id").Order("provider_id").Scan(&blockRewards).Error
	if err != nil {
		return nil, err
	}

	for _, br := range blockRewards {
		result = append(result, br.Amount)
	}

	for _, br := range blockRewards {

		var delegateReward int64

		err = edb.Get().Table("reward_delegates").Select("sum(amount) as amount").Where("reward_type = ? AND provider_id = ? AND block_number >= ? AND block_number <= ?", spenum.BlockRewardBlobber, br.ProviderID, startBlock, endBlock).Scan(&delegateReward).Error
		if err != nil {
			return nil, err
		}

		result = append(result, delegateReward)

		totals = append(totals, br.Amount+delegateReward)
	}

	result = append(result, totals...)

	return result, err
}

func (edb *EventDb) GetQueryRewards(query string) (QueryReward, error) {
	var result QueryReward

	amount := 0

	whereQuery, err := url.QueryUnescape(query)
	if err != nil {
		return result, err
	}

	logging.Logger.Info("Jayash 1", zap.Any("query", "SELECT COALESCE(SUM(amount), 0) FROM reward_providers WHERE "+whereQuery))

	err = edb.Get().Raw("SELECT COALESCE(SUM(amount), 0) FROM reward_providers WHERE " + whereQuery).Scan(&amount).Error
	if err != nil {
		logging.Logger.Info("Jayash 1.1", zap.Any("err", err))
		return result, err
	}

	result.TotalProviderReward = int64(amount)

	err = edb.Get().Raw("SELECT COALESCE(SUM(amount), 0) FROM reward_delegates WHERE " + whereQuery).Scan(&amount).Error
	if err != nil {
		logging.Logger.Info("Jayash 1.2", zap.Any("err", err))
		return result, err
	}

	result.TotalDelegateReward += int64(amount)

	result.TotalReward = result.TotalProviderReward + result.TotalDelegateReward

	logging.Logger.Info("Jayash 6", zap.Any("result", result))

	return result, nil
}

func (edb *EventDb) GetPartitionSizeFrequency(startBlock, endBlock string) (map[int]int, error) {
	type CountFrequency struct {
		Cnt       int
		Frequency int
	}
	query := fmt.Sprintf(`SELECT cnt, COUNT(*) AS frequency FROM (SELECT COUNT(*) AS cnt FROM reward_providers WHERE reward_type = 3 AND block_number >= %s AND block_number < %s GROUP BY block_number) subquery GROUP BY cnt`, startBlock, endBlock)

	logging.Logger.Info("Jayash 2", zap.Any("query", query))

	// Create an empty slice to store the results
	var results []CountFrequency

	// Execute the query and directly scan the results into the slice
	err := edb.Get().Raw(query).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// Create a map to store the result
	result := make(map[int]int)

	// Populate the map based on the results
	for _, cf := range results {
		result[cf.Cnt] = cf.Frequency
	}

	logging.Logger.Info("Jayash 3", zap.Any("result", result), zap.Any("error", err))

	return result, nil
}

func (edb *EventDb) GetBlobberPartitionSelectionFrequency(startBlock, endBlock string) (map[string]int, error) {
	type ProviderFrequency struct {
		ProviderID string `gorm:"column:provider_id"`
		Frequency  int    `gorm:"column:frequency"`
	}

	query := fmt.Sprintf(`SELECT provider_id, COUNT(*) AS frequency FROM reward_providers WHERE reward_type = 3 AND block_number >= %s AND block_number < %s GROUP BY provider_id`, startBlock, endBlock)

	logging.Logger.Info("Jayash 4", zap.Any("query", query))

	// Create an empty slice to store the results
	var result []ProviderFrequency

	// Execute the query and directly scan the results into the slice of ProviderFrequency
	err := edb.Get().Raw(query).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	// Convert the slice to a map
	frequencyMap := make(map[string]int)
	for _, pf := range result {
		frequencyMap[pf.ProviderID] = pf.Frequency
	}

	logging.Logger.Info("Jayash 5", zap.Any("result", frequencyMap), zap.Error(err))

	return frequencyMap, nil
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

type QueryReward struct {
	TotalProviderReward int64 `json:"total_provider_reward"`
	TotalDelegateReward int64 `json:"total_delegate_reward"`
	TotalReward         int64 `json:"total_reward"`
}
