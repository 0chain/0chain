package storagesc

import (
	"encoding/json"
	"math/rand"
	"sort"

	"0chain.net/chaincore/block"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

func (sc *StorageSmartContract) completeChallengeForBlobber(blobberChallengeObj *BlobberChallenge, challengeCompleted *StorageChallenge, challengeResponse *ChallengeResponse) bool {
	found := false
	idx := -1
	if len(blobberChallengeObj.Challenges) > 0 {
		latestOpenChallenge := blobberChallengeObj.Challenges[0]
		if latestOpenChallenge.ID == challengeCompleted.ID {
			found = true
		}
	}
	idx = 0
	if found && idx >= 0 && idx < len(blobberChallengeObj.Challenges) {
		blobberChallengeObj.Challenges = append(blobberChallengeObj.Challenges[:idx], blobberChallengeObj.Challenges[idx+1:]...)
		challengeCompleted.Response = challengeResponse
		blobberChallengeObj.LatestCompletedChallenge = challengeCompleted
	}
	return found
}

func (sc *StorageSmartContract) verifyChallenge(t *transaction.Transaction, input []byte, balances c_state.StateContextI) (string, error) {
	var challengeResponse ChallengeResponse
	err := json.Unmarshal(input, &challengeResponse)
	if err != nil {
		return "", err
	}
	if len(challengeResponse.ID) == 0 || len(challengeResponse.ValidationTickets) == 0 {
		return "", common.NewError("invalid_parameters", "Invalid parameters to challenge response")
	}

	blobberChallengeObj := &BlobberChallenge{}
	blobberChallengeObj.BlobberID = t.ClientID

	blobberChallengeBytes, err := balances.GetTrieNode(blobberChallengeObj.GetKey(sc.ID))
	if blobberChallengeBytes == nil {
		return "", common.NewError("invalid_parameters", "Cannot find the blobber challenge entity with ID "+t.ClientID)
	}

	err = blobberChallengeObj.Decode(blobberChallengeBytes.Encode())
	if err != nil {
		return "", common.NewError("blobber_challenge_decode_error", "Error decoding the blobber challenge")
	}

	challengeRequest, ok := blobberChallengeObj.ChallengeMap[challengeResponse.ID]

	if !ok {
		if blobberChallengeObj.LatestCompletedChallenge != nil && challengeResponse.ID == blobberChallengeObj.LatestCompletedChallenge.ID && blobberChallengeObj.LatestCompletedChallenge.Response != nil {
			return "Challenge Already redeemed by Blobber", nil
		}
		return "", common.NewError("invalid_parameters", "Cannot find the challenge with ID "+challengeResponse.ID)
	}

	if challengeRequest.Blobber.ID != t.ClientID {
		return "", common.NewError("invalid_parameters", "Challenge response should be submitted by the same blobber as the challenge request")
	}

	allocationObj := &StorageAllocation{}
	allocationObj.ID = challengeRequest.AllocationID

	allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(sc.ID))
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_allocation", "Client state has invalid allocations")
	}

	err = allocationObj.Decode(allocationBytes.Encode())
	if err != nil {
		return "", common.NewError("decode_error", "Error decoding the allocation")
	}

	blobberAllocation, ok := allocationObj.BlobberMap[t.ClientID]
	if !ok {
		return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation")
	}

	numSuccess := 0
	numFailure := 0
	for _, vt := range challengeResponse.ValidationTickets {
		if vt != nil {
			ok, err := vt.VerifySign()
			if !ok || err != nil {
				continue
			}
			if vt.Result {
				numSuccess++
			} else {
				numFailure++
			}
		}
	}

	if numSuccess > (len(challengeRequest.Validators) / 2) {
		//challengeRequest.Response = &challengeResponse
		//delete(blobberChallengeObj.ChallengeMap, challengeResponse.ID)
		completed := sc.completeChallengeForBlobber(blobberChallengeObj, challengeRequest, &challengeResponse)
		if !completed {
			return "", common.NewError("challenge_out_of_order", "First challenge on the list is not same as the one attempted to redeem")
		}
		allocationObj.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		allocationObj.Stats.SuccessChallenges++
		allocationObj.Stats.OpenChallenges--

		blobberAllocation.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		blobberAllocation.Stats.SuccessChallenges++
		blobberAllocation.Stats.OpenChallenges--

		balances.InsertTrieNode(allocationObj.GetKey(sc.ID), allocationObj)
		balances.InsertTrieNode(blobberChallengeObj.GetKey(sc.ID), blobberChallengeObj)
		Logger.Info("Challenge passed", zap.Any("challenge", challengeResponse.ID))
		return "Challenge Passed by Blobber", nil
	}

	if numFailure > (len(challengeRequest.Validators) / 2) || (numSuccess + numFailure) == len(challengeRequest.Validators) {
		completed := sc.completeChallengeForBlobber(blobberChallengeObj, challengeRequest, &challengeResponse)
		if !completed {
			return "", common.NewError("challenge_out_of_order", "First challenge on the list is not same as the one attempted to redeem")
		}
		//delete(blobberChallengeObj.ChallengeMap, challengeResponse.ID)
		//challengeRequest.Response = &challengeResponse
		allocationObj.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		allocationObj.Stats.FailedChallenges++
		allocationObj.Stats.OpenChallenges--

		blobberAllocation.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		blobberAllocation.Stats.FailedChallenges++
		blobberAllocation.Stats.OpenChallenges--

		balances.InsertTrieNode(allocationObj.GetKey(sc.ID), allocationObj)
		balances.InsertTrieNode(blobberChallengeObj.GetKey(sc.ID), blobberChallengeObj)
		Logger.Info("Challenge failed", zap.Any("challenge", challengeResponse.ID))
		return "Challenge Failed by Blobber", nil
	}

	return "", common.NewError("not_enough_validations", "Not enough validations for the challenge")
}

func (sc *StorageSmartContract) addChallenge(t *transaction.Transaction, b *block.Block, input []byte, balances c_state.StateContextI) (string, error) {

	validatorList, _ := sc.getValidatorsList(balances)

	if len(validatorList.Nodes) == 0 {
		return "", common.NewError("no_validators", "Not enough validators for the challenge")
	}

	foundValidator := false
	for _, validator := range validatorList.Nodes {
		if validator.ID == t.ClientID {
			foundValidator = true
			break
		}
	}

	if !foundValidator {
		return "", common.NewError("invalid_challenge_request", "Challenge can be requested only by validators")
	}

	var storageChallenge StorageChallenge
	storageChallenge.ID = t.Hash

	allocationList, err := sc.getAllAllocationsList(balances)
	if err != nil {
		return "", common.NewError("adding_challenge_error", "Error gettting the allocation list. "+err.Error())
	}
	if len(allocationList.List) == 0 {
		return "", common.NewError("adding_challenge_error", "No allocations at this time")
	}

	rand.Seed(b.RoundRandomSeed)
	allocationIndex := rand.Int63n(int64(len(allocationList.List)))
	allocationKey := allocationList.List[allocationIndex]

	allocationObj := &StorageAllocation{}
	allocationObj.ID = allocationKey

	allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(sc.ID))
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_allocation", "Client state has invalid allocations")
	}

	allocationObj.Decode(allocationBytes.Encode())
	sort.SliceStable(allocationObj.Blobbers, func(i, j int) bool {
		return allocationObj.Blobbers[i].ID < allocationObj.Blobbers[j].ID
	})

	rand.Seed(b.RoundRandomSeed)
	randIdx := rand.Int63n(int64(len(allocationObj.Blobbers)))
	Logger.Info("Challenge blobber selected.", zap.Any("challenge", t.Hash), zap.Any("selected_blobber", allocationObj.Blobbers[randIdx]), zap.Any("blobbers", allocationObj.Blobbers), zap.Any("random_index", randIdx), zap.Any("seed", b.RoundRandomSeed))

	storageChallenge.Validators = validatorList.Nodes
	storageChallenge.Blobber = allocationObj.Blobbers[randIdx]
	storageChallenge.RandomNumber = b.RoundRandomSeed
	storageChallenge.AllocationID = allocationObj.ID

	blobberAllocation, ok := allocationObj.BlobberMap[storageChallenge.Blobber.ID]

	if !ok {
		return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation. Could not find blobber")
	}

	if len(blobberAllocation.AllocationRoot) == 0 || blobberAllocation.Stats.UsedSize == 0 {
		return "", common.NewError("blobber_no_wm", "Blobber does not have any data for the allocation. "+allocationObj.ID+" blobber: "+blobberAllocation.BlobberID)
	}

	storageChallenge.AllocationRoot = blobberAllocation.AllocationRoot

	blobberChallengeObj := &BlobberChallenge{}
	blobberChallengeObj.BlobberID = storageChallenge.Blobber.ID

	blobberChallengeBytes, err := balances.GetTrieNode(blobberChallengeObj.GetKey(sc.ID))
	//blobberChallengeObj.LatestCompletedChallenges = make([]*StorageChallenge, 0)
	if blobberChallengeBytes != nil {
		err = blobberChallengeObj.Decode(blobberChallengeBytes.Encode())
		if err != nil {
			return "", common.NewError("blobber_challenge_decode_error", "Error decoding the blobber challenge")
		}
	}

	storageChallenge.Created = t.CreationDate
	addedChallege := blobberChallengeObj.addChallenge(&storageChallenge)
	if !addedChallege {
		challengeBytes, err := json.Marshal(storageChallenge)
		return string(challengeBytes), err
	}

	balances.InsertTrieNode(blobberChallengeObj.GetKey(sc.ID), blobberChallengeObj)

	allocationObj.Stats.OpenChallenges++
	allocationObj.Stats.TotalChallenges++
	blobberAllocation.Stats.OpenChallenges++
	blobberAllocation.Stats.TotalChallenges++
	balances.InsertTrieNode(allocationObj.GetKey(sc.ID), allocationObj)
	//Logger.Info("Adding a new challenge", zap.Any("blobberChallengeObj", blobberChallengeObj), zap.Any("challenge", storageChallenge.ID))
	challengeBytes, err := json.Marshal(storageChallenge)
	return string(challengeBytes), err
}
