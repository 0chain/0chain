package storagesc

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool/spenum"
	"encoding/json"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
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
	// read all data from challenge_rewards table and return
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	allocationID := r.URL.Query().Get("allocation_id")

	result, err := edb.GetAllocationReadRewards(allocationID)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, resultJSON, nil)
}

func (srh *StorageRestHandler) getTotalChallengeRewards(w http.ResponseWriter, r *http.Request) {
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	allocationID := r.URL.Query().Get("allocation_id")

	totalBlobberRewards := map[string]int64{}
	totalValidatorRewards := map[string]int64{}

	challengeRewards, err := edb.GetAllocationChallengeRewards(allocationID)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	for i, j := range challengeRewards {
		if j.ProviderType == spenum.ChallengePassReward.Int() {
			totalBlobberRewards[i] = j.Total
		} else {
			totalValidatorRewards[i] = j.Total
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
	logging.Logger.Info("getAllocationChallengeRewards 1")
	// read all data from challenge_rewards table and return
	edb := srh.GetQueryStateContext().GetEventDB()
	if edb == nil {
		common.Respond(w, r, nil, common.NewErrInternal("no db connection"))
		return
	}

	logging.Logger.Info("getAllocationChallengeRewards")

	allocationID := r.URL.Query().Get("allocation_id")

	logging.Logger.Info("getAllocationChallengeRewards 2", zap.Any("allocationID", allocationID))

	result, err := edb.GetAllocationChallengeRewards(allocationID)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	logging.Logger.Info("getAllocationChallengeRewards 3", zap.Any("result", result))

	resultJSON, err := json.Marshal(result)
	if err != nil {
		common.Respond(w, r, nil, err)
		return
	}

	logging.Logger.Info("getAllocationChallengeRewards 4", zap.Any("resultJSON", resultJSON))

	common.Respond(w, r, resultJSON, err)
}
