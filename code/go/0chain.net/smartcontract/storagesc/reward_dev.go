package storagesc

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool/spenum"
	"net/http"
)

func (srh *StorageRestHandler) getAllChallenges(w http.ResponseWriter, r *http.Request) {
	// read all data from challenges table and return
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	allocationID := r.URL.Query().Get("allocation_id")

	challenges, err := edb.GetAllChallengesByAllocationID(allocationID)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, challenges, nil)
}

func (srh *StorageRestHandler) getBlockRewards(w http.ResponseWriter, r *http.Request) {
	// read all data from block_rewards table and return
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	blockNumber := r.URL.Query().Get("block_number")
	startBlockNumber := r.URL.Query().Get("start_block_number")
	endBlockNumber := r.URL.Query().Get("end_block_number")

	providerRewards, err := edb.GetRewardToProviders(blockNumber, startBlockNumber, endBlockNumber, spenum.BlockRewardBlobber.Int())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	delegateRewards, err := edb.GetRewardsToDelegates(blockNumber, startBlockNumber, endBlockNumber, spenum.BlockRewardBlobber.Int())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	result := map[string]interface{}{
		"provider_rewards": providerRewards,
		"delegate_rewards": delegateRewards,
	}

	common.Respond(w, r, result, nil)
}

func (srh *StorageRestHandler) getReadRewards(w http.ResponseWriter, r *http.Request) {
	// read all data from block_rewards table and return
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	blockNumber := r.URL.Query().Get("block_number")
	startBlockNumber := r.URL.Query().Get("start_block_number")
	endBlockNumber := r.URL.Query().Get("end_block_number")

	providerRewards, err := edb.GetRewardToProviders(blockNumber, startBlockNumber, endBlockNumber, spenum.FileDownloadReward.Int())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	delegateRewards, err := edb.GetRewardsToDelegates(blockNumber, startBlockNumber, endBlockNumber, spenum.FileDownloadReward.Int())
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	result := map[string]interface{}{
		"provider_rewards": providerRewards,
		"delegate_rewards": delegateRewards,
	}

	common.Respond(w, r, result, nil)
}

func (srh *StorageRestHandler) getChallengeRewards(w http.ResponseWriter, r *http.Request) {
	// read all data from challenge_rewards table and return
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	challengeID := r.URL.Query().Get("challenge_id")

	blobberRewards, validatorRewards, err := edb.GetChallengeRewardsToProviders(challengeID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("error while getting challenge rewards"))
		return
	}
	blobberDelegateRewards, validatorDelegateRewards, err := edb.GetChallengeRewardsToDelegates(challengeID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("error while getting challenge rewards"))
		return
	}

	result := map[string]interface{}{
		"blobber_rewards":            blobberRewards,
		"validator_rewards":          validatorRewards,
		"blobber_delegate_rewards":   blobberDelegateRewards,
		"validator_delegate_rewards": validatorDelegateRewards,
	}

	common.Respond(w, r, result, nil)
}

func (srh *StorageRestHandler) getTotalChallengeRewards(w http.ResponseWriter, r *http.Request) {
	// read all data from challenge_rewards table and return
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	allocationID := r.URL.Query().Get("allocation_id")

	challenges, err := edb.GetAllChallengesByAllocationID(allocationID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("error while getting challenges"))
		return
	}

	totalBlobberRewards := map[string]int64{}
	totalValidatorRewards := map[string]int64{}

	for _, challenge := range challenges {
		blobberRewards, validatorRewards, err := edb.GetChallengeRewardsToProviders(challenge.ChallengeID)

		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("error while getting challenge rewards"))
			return
		}

		for _, reward := range blobberRewards {
			// check if the provider_id is already in the map totalBlobberRewards
			if _, ok := totalBlobberRewards[reward.ProviderId]; ok {
				cur, _ := reward.Amount.Int64()
				totalBlobberRewards[reward.ProviderId] += cur
			} else {
				cur, _ := reward.Amount.Int64()
				totalBlobberRewards[reward.ProviderId] = cur
			}
		}

		for _, reward := range validatorRewards {
			// check if the provider_id is already in the map totalChallengeRewards
			if _, ok := totalValidatorRewards[reward.ProviderId]; ok {
				cur, _ := reward.Amount.Int64()
				totalValidatorRewards[reward.ProviderId] += cur
			} else {
				cur, _ := reward.Amount.Int64()
				totalValidatorRewards[reward.ProviderId] = cur
			}
		}

		blobberDelegateRewards, validatorDelegateRewards, err := edb.GetChallengeRewardsToDelegates(challenge.ChallengeID)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("error while getting challenge rewards"))
			return
		}

		for _, reward := range blobberDelegateRewards {
			// check if the provider_id is already in the map totalBlobberRewards
			if _, ok := totalBlobberRewards[reward.ProviderID]; ok {
				cur, _ := reward.Amount.Int64()
				totalBlobberRewards[reward.ProviderID] += cur
			} else {
				cur, _ := reward.Amount.Int64()
				totalBlobberRewards[reward.ProviderID] = cur
			}
		}

		for _, reward := range validatorDelegateRewards {
			// check if the provider_id is already in the map totalChallengeRewards
			if _, ok := totalValidatorRewards[reward.ProviderID]; ok {
				cur, _ := reward.Amount.Int64()
				totalValidatorRewards[reward.ProviderID] += cur
			} else {
				cur, _ := reward.Amount.Int64()
				totalValidatorRewards[reward.ProviderID] = cur
			}
		}
	}

	result := map[string]interface{}{
		"blobber_rewards":   totalBlobberRewards,
		"validator_rewards": totalValidatorRewards,
	}

	common.Respond(w, r, result, nil)
}

func (srh *StorageRestHandler) getAllocationCancellationReward(w http.ResponseWriter, r *http.Request) {
	// read all data from allocation_cancellation_rewards table and return
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	startBlock := r.URL.Query().Get("start_block")
	endBlock := r.URL.Query().Get("end_block")

	providerRewards, err := edb.GetAllocationCancellationRewardsToProviders(startBlock, endBlock)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("error while getting allocation cancellation rewards"))
		return
	}

	delegateRewards, err := edb.GetAllocationCancellationRewardsToDelegates(startBlock, endBlock)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("error while getting allocation cancellation rewards"))
		return
	}

	result := map[string]interface{}{
		"provider_rewards": providerRewards,
		"delegate_rewards": delegateRewards,
	}

	common.Respond(w, r, result, nil)
}

func (srh *StorageRestHandler) getAllocationChallengeRewards(w http.ResponseWriter, r *http.Request) {
	// read all data from challenge_rewards table and return
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	allocationID := r.URL.Query().Get("allocation_id")

	challenges, err := edb.GetAllChallengesByAllocationID(allocationID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("error while getting challenges"))
		return
	}

	var blobberChallengeRewardsMap map[string]BlobberChallengeRewards
	blobberChallengeRewardsMap = make(map[string]BlobberChallengeRewards)

	var validatorChallengeRewardsMap map[string]ValidatorChallengeRewards
	validatorChallengeRewardsMap = make(map[string]ValidatorChallengeRewards)

	for _, challenge := range challenges {
		blobberRewards, validatorRewards, err := edb.GetChallengeRewardsToProviders(challenge.ChallengeID)

		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("error while getting challenge rewards"))
			return
		}

		for _, reward := range blobberRewards {
			var blobberReward BlobberChallengeRewards
			blobberReward.BlobberID = reward.ProviderId
			blobberReward.Amount, _ = reward.Amount.Int64()
			blobberReward.Total += blobberReward.Amount

			blobberChallengeRewardsMap[reward.ProviderId] = blobberReward
		}

		for _, reward := range validatorRewards {
			var validatorReward ValidatorChallengeRewards
			validatorReward.ValidatorID = reward.ProviderId
			validatorReward.Amount, _ = reward.Amount.Int64()
			validatorReward.Total += validatorReward.Amount

			validatorChallengeRewardsMap[reward.ProviderId] = validatorReward
		}

		blobberDelegateRewards, validatorDelegateRewards, err := edb.GetChallengeRewardsToDelegates(challenge.ChallengeID)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("error while getting challenge rewards"))
			return
		}

		for _, reward := range blobberDelegateRewards {
			rewardAmount, _ := reward.Amount.Int64()

			blobberReward := blobberChallengeRewardsMap[reward.ProviderID]

			curBlobberDelegateReward := int64(0)
			if _, ok := blobberReward.DelegateRewards[reward.PoolID]; ok {
				curBlobberDelegateReward = blobberReward.DelegateRewards[reward.PoolID]
			}

			blobberReward.DelegateRewards[reward.PoolID] = curBlobberDelegateReward + rewardAmount
			blobberReward.Total += rewardAmount

			blobberChallengeRewardsMap[reward.ProviderID] = blobberReward
		}

		for _, reward := range validatorDelegateRewards {
			rewardAmount, _ := reward.Amount.Int64()

			validatorReward := validatorChallengeRewardsMap[reward.ProviderID]

			curValidatorDelegateReward := int64(0)
			if _, ok := validatorReward.DelegateRewards[reward.PoolID]; ok {
				curValidatorDelegateReward = validatorReward.DelegateRewards[reward.PoolID]
			}

			validatorReward.DelegateRewards[reward.PoolID] = curValidatorDelegateReward + rewardAmount
			validatorReward.Total += rewardAmount

			validatorChallengeRewardsMap[reward.ProviderID] = validatorReward
		}
	}

	var blobberChallengeRewards []BlobberChallengeRewards
	for _, reward := range blobberChallengeRewardsMap {
		blobberChallengeRewards = append(blobberChallengeRewards, reward)
	}

	var validatorChallengeRewards []ValidatorChallengeRewards
	for _, reward := range validatorChallengeRewardsMap {
		validatorChallengeRewards = append(validatorChallengeRewards, reward)
	}

	result := ChallengeRewards{
		BlobberRewards:   blobberChallengeRewards,
		ValidatorRewards: validatorChallengeRewards,
	}

	common.Respond(w, r, result, nil)
}

type ChallengeRewards struct {
	BlobberRewards   []BlobberChallengeRewards
	ValidatorRewards []ValidatorChallengeRewards
}

type BlobberChallengeRewards struct {
	BlobberID       string
	DelegateRewards map[string]int64
	Amount          int64
	Total           int64
}

type ValidatorChallengeRewards struct {
	ValidatorID     string
	DelegateRewards map[string]int64
	Amount          int64
	Total           int64
}
