package storagesc

import (
	"encoding/json"
	"math/rand"
	"sort"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (sc *StorageSmartContract) completeChallengeForBlobber(blobberChallengeObj *BlobberChallenge, challengeCompleted *StorageChallenge, challengeResponse *ChallengeResponse) {
	found := false
	idx := -1
	for i, challenge := range blobberChallengeObj.Challenges {
		if challenge.ID == challengeCompleted.ID {
			found = true
			idx = i
			break
		}
	}
	if found && idx >= 0 && idx < len(blobberChallengeObj.Challenges) {
		blobberChallengeObj.Challenges = append(blobberChallengeObj.Challenges[:idx], blobberChallengeObj.Challenges[idx+1:]...)
		if len(blobberChallengeObj.LatestCompletedChallenges) >= 20 {
			blobberChallengeObj.LatestCompletedChallenges = blobberChallengeObj.LatestCompletedChallenges[1:]
		}
		challengeCompleted.Response = challengeResponse
		blobberChallengeObj.LatestCompletedChallenges = append(blobberChallengeObj.LatestCompletedChallenges, challengeCompleted)
	}

}

func (sc *StorageSmartContract) verifyChallenge(t *transaction.Transaction, input []byte) (string, error) {
	var challengeResponse ChallengeResponse
	err := json.Unmarshal(input, &challengeResponse)
	if err != nil {
		return "", err
	}
	if len(challengeResponse.ID) == 0 || len(challengeResponse.ValidationTickets) == 0 {
		return "", common.NewError("invalid_parameters", "Invalid parameters to challenge response")
	}

	var blobberChallengeObj BlobberChallenge
	blobberChallengeObj.BlobberID = t.ClientID

	blobberChallengeBytes, err := sc.DB.GetNode(blobberChallengeObj.GetKey())
	if err != nil {
		return "", common.NewError("blobber_challenge_read_err", "Error reading blobber challenge from DB")
	}
	if blobberChallengeBytes == nil {
		return "", common.NewError("invalid_parameters", "Cannot find the blobber challenge entity with ID "+t.ClientID)
	}

	err = blobberChallengeObj.Decode(blobberChallengeBytes)
	if err != nil {
		return "", common.NewError("blobber_challenge_decode_error", "Error decoding the blobber challenge")
	}

	challengeRequest, ok := blobberChallengeObj.ChallengeMap[challengeResponse.ID]

	if !ok {
		for _, completedChallenge := range blobberChallengeObj.LatestCompletedChallenges {
			if challengeResponse.ID == completedChallenge.ID && completedChallenge.Response != nil {
				return "Challenge Already redeemed by Blobber", nil
			}
		}
		return "", common.NewError("invalid_parameters", "Cannot find the challenge with ID "+challengeResponse.ID)
	}

	if challengeRequest.Blobber.ID != t.ClientID {
		return "", common.NewError("invalid_parameters", "Challenge response should be submitted by the same blobber as the challenge request")
	}

	allocationObj := &StorageAllocation{}
	allocationObj.ID = challengeRequest.AllocationID

	allocationBytes, err := sc.DB.GetNode(allocationObj.GetKey())
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_allocation", "Client state has invalid allocations")
	}

	err = allocationObj.Decode(allocationBytes)
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
		sc.completeChallengeForBlobber(&blobberChallengeObj, challengeRequest, &challengeResponse)
		allocationObj.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		allocationObj.Stats.SuccessChallenges++
		allocationObj.Stats.OpenChallenges--

		blobberAllocation.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		blobberAllocation.Stats.SuccessChallenges++
		blobberAllocation.Stats.OpenChallenges--

		defer sc.DB.PutNode(allocationObj.GetKey(), allocationObj.Encode())
		sc.DB.PutNode(blobberChallengeObj.GetKey(), blobberChallengeObj.Encode())
		return "Challenge Passed by Blobber", nil
	}

	if numFailure > (len(challengeRequest.Validators) / 2) {
		sc.completeChallengeForBlobber(&blobberChallengeObj, challengeRequest, &challengeResponse)
		//delete(blobberChallengeObj.ChallengeMap, challengeResponse.ID)
		//challengeRequest.Response = &challengeResponse
		allocationObj.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		allocationObj.Stats.FailedChallenges++
		allocationObj.Stats.OpenChallenges--

		blobberAllocation.Stats.LastestClosedChallengeTxn = challengeRequest.ID
		blobberAllocation.Stats.FailedChallenges++
		blobberAllocation.Stats.OpenChallenges--

		defer sc.DB.PutNode(allocationObj.GetKey(), allocationObj.Encode())
		sc.DB.PutNode(blobberChallengeObj.GetKey(), blobberChallengeObj.Encode())
		return "Challenge Failed by Blobber", nil
	}

	return "", common.NewError("not_enough_validations", "Not enough validations for the challenge")
}

func (sc *StorageSmartContract) addChallenge(t *transaction.Transaction, b *block.Block, input []byte) (string, error) {

	validatorList, _ := sc.getValidatorsList()

	if len(validatorList) == 0 {
		return "", common.NewError("no_validators", "Not enough validators for the challenge")
	}

	foundValidator := false
	for _, validator := range validatorList {
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

	allocationList, err := sc.getAllAllocationsList()
	if err != nil {
		return "", common.NewError("adding_challenge_error", "Error gettting the allocation list. "+err.Error())
	}
	if len(allocationList) == 0 {
		return "", common.NewError("adding_challenge_error", "No allocations at this time")
	}

	rand.Seed(b.RoundRandomSeed)
	allocationIndex := rand.Int63n(int64(len(allocationList)))
	allocationKey := allocationList[allocationIndex]

	allocationObj := &StorageAllocation{}
	allocationObj.ID = allocationKey

	allocationBytes, err := sc.DB.GetNode(allocationObj.GetKey())
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_allocation", "Client state has invalid allocations")
	}

	allocationObj.Decode(allocationBytes)
	sort.SliceStable(allocationObj.Blobbers, func(i, j int) bool {
		return allocationObj.Blobbers[i].ID < allocationObj.Blobbers[j].ID
	})
	storageChallenge.Validators = validatorList
	storageChallenge.Blobber = allocationObj.Blobbers[rand.Intn(len(allocationObj.Blobbers))]
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

	var blobberChallengeObj BlobberChallenge
	blobberChallengeObj.BlobberID = storageChallenge.Blobber.ID

	blobberChallengeBytes, err := sc.DB.GetNode(blobberChallengeObj.GetKey())
	if err != nil {
		return "", common.NewError("blobber_challenge_read_err", "Error reading blobber challenge from DB")
	}
	blobberChallengeObj.LatestCompletedChallenges = make([]*StorageChallenge, 0)
	if blobberChallengeBytes != nil {
		err = blobberChallengeObj.Decode(blobberChallengeBytes)
		if err != nil {
			return "", common.NewError("blobber_challenge_decode_error", "Error decoding the blobber challenge")
		}
	}

	storageChallenge.Created = t.CreationDate
	blobberChallengeObj.addChallenge(&storageChallenge)

	sc.DB.PutNode(blobberChallengeObj.GetKey(), blobberChallengeObj.Encode())

	allocationObj.Stats.OpenChallenges++
	allocationObj.Stats.TotalChallenges++
	blobberAllocation.Stats.OpenChallenges++
	blobberAllocation.Stats.TotalChallenges++
	sc.DB.PutNode(allocationObj.GetKey(), allocationObj.Encode())

	challengeBytes, err := json.Marshal(storageChallenge)
	return string(challengeBytes), err
}
