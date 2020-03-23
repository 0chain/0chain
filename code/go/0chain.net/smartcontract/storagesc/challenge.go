package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"0chain.net/chaincore/block"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/util"

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

func (sc *StorageSmartContract) getBlobberChallengeBytes(blobberID string,
	balances c_state.StateContextI) (b []byte, err error) {

	var (
		bc   BlobberChallenge
		seri util.Serializable
	)
	bc.BlobberID = blobberID
	if seri, err = balances.GetTrieNode(bc.GetKey(sc.ID)); err != nil {
		return
	}
	return seri.Encode(), nil
}

func (sc *StorageSmartContract) getBlobberChallenge(blobberID string,
	balances c_state.StateContextI) (bc *BlobberChallenge, err error) {

	var b []byte
	if b, err = sc.getBlobberChallengeBytes(blobberID, balances); err != nil {
		return
	}
	bc = new(BlobberChallenge)
	if err = bc.Decode(b); err != nil {
		return nil, fmt.Errorf("decoding blobber_challenge: %v", err)
	}
	return
}

// move tokens from challenge pool to blobber's stake pool (to unlocked)
func (sc *StorageSmartContract) blobberReward(alloc *StorageAllocation,
	prev common.Timestamp, bc *BlobberChallenge,
	balances c_state.StateContextI) (err error) {

	// time of this challenge
	var tp = bc.LatestCompletedChallenge.Created

	if tp > alloc.Expiration+toSeconds(alloc.ChallengeCompletionTime) {
		return errors.New("late challenge response")
	}

	// pools
	var sp *stakePool
	if sp, err = sc.getStakePool(bc.BlobberID, balances); err != nil {
		return fmt.Errorf("can't get blobber's stake pool: %v", err)
	}

	var cp *challengePool
	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("can't get allocation's challenge pool: %v", err)
	}

	var ratio = float64(tp-prev) / float64(alloc.Expiration-prev)

	if tp > alloc.Expiration {
		ratio = 1 // all left (allocation closed, challenge completion time)
	}

	var move = state.Balance(float64(cp.Balance) * ratio)

	if err = cp.moveToStakePool(sp, move); err != nil {
		return fmt.Errorf("can't move tokens to blobber: %v", err)
	}

	// save pools
	if err = sp.save(sc.ID, bc.BlobberID, balances); err != nil {
		return fmt.Errorf("can't save blobber's stake pool: %v", err)
	}

	if err = cp.save(sc.ID, alloc.ID, balances); err != nil {
		return fmt.Errorf("can't save allocation's challenge pool: %v", err)
	}

	return
}

// move tokens from challenge pool back to write pool
func (sc *StorageSmartContract) blobberPenalty(alloc *StorageAllocation,
	prev common.Timestamp, bc *BlobberChallenge,
	balances c_state.StateContextI) (err error) {

	// time of this challenge
	var tp = bc.LatestCompletedChallenge.Created

	if tp > alloc.Expiration+toSeconds(alloc.ChallengeCompletionTime) {
		return errors.New("late challenge response")
	}

	// pools
	var sp *stakePool
	if sp, err = sc.getStakePool(bc.BlobberID, balances); err != nil {
		return fmt.Errorf("can't get blobber's stake pool: %v", err)
	}

	var wp *writePool
	if wp, err = sc.getWritePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("can't get allocation's write pool: %v", err)
	}

	var ratio = float64(tp-prev) / float64(alloc.Expiration-prev)

	if tp > alloc.Expiration {
		ratio = 1 // all left (allocation closed, challenge completion time)
	}

	var offer = sp.findOffer(alloc.ID)
	if offer == nil {
		return errors.New("invalid state, can't find stake pool offer: " +
			alloc.ID)
	}

	var move = state.Balance(float64(offer.Lock) * ratio)
	offer.Lock -= move // subtract the offer stake

	if err = sp.moveToWritePool(wp, move); err != nil {
		return fmt.Errorf("can't move tokens to write pool: %v", err)
	}

	// save pools
	if err = sp.save(sc.ID, bc.BlobberID, balances); err != nil {
		return fmt.Errorf("can't save blobber's stake pool: %v", err)
	}

	if err = wp.save(sc.ID, alloc.ID, balances); err != nil {
		return fmt.Errorf("can't save allocation's write pool: %v", err)
	}

	return
}

func (sc *StorageSmartContract) verifyChallenge(t *transaction.Transaction,
	input []byte, balances c_state.StateContextI) (string, error) {

	var challengeResponse ChallengeResponse
	err := json.Unmarshal(input, &challengeResponse)
	if err != nil {
		return "", err
	}
	if len(challengeResponse.ID) == 0 || len(challengeResponse.ValidationTickets) == 0 {
		return "", common.NewError("invalid_parameters", "Invalid parameters to challenge response")
	}

	blobberChallengeObj, err := sc.getBlobberChallenge(t.ClientID, balances)
	if err != nil {
		return "", common.NewError("invalid_parameters",
			"can't get the blobber challenge "+t.ClientID+": "+err.Error())
	}

	challengeRequest, ok := blobberChallengeObj.ChallengeMap[challengeResponse.ID]

	if !ok {
		if blobberChallengeObj.LatestCompletedChallenge != nil &&
			challengeResponse.ID == blobberChallengeObj.LatestCompletedChallenge.ID &&
			blobberChallengeObj.LatestCompletedChallenge.Response != nil {

			return "Challenge Already redeemed by Blobber", nil
		}
		return "", common.NewError("invalid_parameters", "Cannot find the challenge with ID "+challengeResponse.ID)
	}

	if challengeRequest.Blobber.ID != t.ClientID {
		return "", common.NewError("invalid_parameters",
			"Challenge response should be submitted by the same blobber as the challenge request")
	}

	allocationObj, err := sc.getAllocation(challengeRequest.AllocationID,
		balances)
	if err != nil {
		return "", common.NewError("invalid_allocation",
			"can't get related allocation: "+err.Error())
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

	// time of previous complete challenge (not the current one)
	// or allocation start time if no challenges
	var prev = allocationObj.StartTime
	if last := blobberChallengeObj.LatestCompletedChallenge; last != nil {
		prev = last.Created
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
		sc.challengeResolved(balances, true)
		//Logger.Info("Challenge passed", zap.Any("challenge", challengeResponse.ID))

		err = sc.blobberReward(allocationObj, prev, blobberChallengeObj,
			balances)
		if err != nil {
			return "", common.NewError("challenge_reward_error", err.Error())
		}

		return "Challenge Passed by Blobber", nil
	}

	if numFailure > (len(challengeRequest.Validators)/2) || (numSuccess+numFailure) == len(challengeRequest.Validators) {
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
		sc.challengeResolved(balances, false)
		Logger.Info("Challenge failed", zap.Any("challenge", challengeResponse.ID))

		err = sc.blobberPenalty(allocationObj, prev, blobberChallengeObj,
			balances)
		if err != nil {
			return "", common.NewError("challenge_penalty_error", err.Error())
		}

		return "Challenge Failed by Blobber", nil
	}

	return "", common.NewError("not_enough_validations", "Not enough validations for the challenge")
}

func (sc *StorageSmartContract) generateChallenges(t *transaction.Transaction, b *block.Block, input []byte, balances c_state.StateContextI) error {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	statsBytes, err := balances.GetTrieNode(stats.GetKey(sc.ID))
	if statsBytes != nil {
		err = stats.Decode(statsBytes.Encode())
		if err != nil {
			Logger.Error("storage stats decode error")
			return err
		}
	}
	//Logger.Info("Stats for generating challenge", zap.Any("stats", stats))
	lastChallengeTime := stats.LastChallengedTime
	if lastChallengeTime == 0 {
		lastChallengeTime = t.CreationDate
	}
	numMins := int64((t.CreationDate - lastChallengeTime) / 60)
	sizeDiffMB := (stats.Stats.UsedSize - stats.LastChallengedSize) / (1024 * 1024)

	if numMins == 0 && sizeDiffMB == 0 {
		return nil
	}

	if numMins == 0 {
		numMins = 1
	}
	if sizeDiffMB == 0 {
		sizeDiffMB = 1
	}
	challengeGenerationRate := config.SmartContractConfig.GetInt64("smart_contracts.storagesc.challenge_rate_per_mb_min")
	numChallenges := int64(math.Min(float64(challengeGenerationRate*numMins*sizeDiffMB), float64(100)))
	//	Logger.Info("Generating challenges", zap.Any("mins_since_last", numMins), zap.Any("mb_size_diff", sizeDiffMB))
	hashString := encryption.Hash(t.Hash + b.PrevHash)
	randomSeed, err := strconv.ParseUint(hashString[0:16], 16, 64)
	if err != nil {
		Logger.Error("Error in creating seed for creating challenges", zap.Error(err))
		return err
	}
	r := rand.New(rand.NewSource(int64(randomSeed)))
	for i := int64(0); i < numChallenges; i++ {
		challengeID := encryption.Hash(hashString + strconv.FormatInt(i, 10))
		challengeSeed, err := strconv.ParseUint(challengeID[0:16], 16, 64)
		if err != nil {
			Logger.Error("Error in creating challenge seed", zap.Error(err), zap.Any("challengeID", challengeID))
			continue
		}
		challengeString, err := sc.addChallenge(challengeID, t.CreationDate, r, int64(challengeSeed), balances)
		if err != nil {
			Logger.Error("Error in adding challenge", zap.Error(err), zap.Any("challengeString", challengeString))
			continue
		}
	}
	return nil
}

func (sc *StorageSmartContract) addChallenge(challengeID string, creationDate common.Timestamp, r *rand.Rand, challengeSeed int64, balances c_state.StateContextI) (string, error) {

	validatorList, _ := sc.getValidatorsList(balances)

	if len(validatorList.Nodes) == 0 {
		return "", common.NewError("no_validators", "Not enough validators for the challenge")
	}

	allocationList, err := sc.getAllAllocationsList(balances)
	if err != nil {
		return "", common.NewError("adding_challenge_error", "Error gettting the allocation list. "+err.Error())
	}
	if len(allocationList.List) == 0 {
		return "", common.NewError("adding_challenge_error", "No allocations at this time")
	}

	allocationObj := &StorageAllocation{}
	allocationObj.Stats = &StorageAllocationStats{}

	allocationperm := r.Perm(len(allocationList.List))
	for _, v := range allocationperm {
		allocationKey := allocationList.List[v]
		allocationObj.ID = allocationKey

		allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(sc.ID))
		if allocationBytes == nil || err != nil {
			Logger.Error("Client state has invalid allocations", zap.Any("allocation_list", allocationList.List), zap.Any("selected_allocation", allocationKey))
			return "", common.NewError("invalid_allocation", "Client state has invalid allocations")
		}
		allocationObj.Decode(allocationBytes.Encode())
		sort.SliceStable(allocationObj.Blobbers, func(i, j int) bool {
			return allocationObj.Blobbers[i].ID < allocationObj.Blobbers[j].ID
		})
		if allocationObj.Stats.NumWrites > 0 {
			break
		}
	}

	if allocationObj.Stats.NumWrites == 0 {
		return "", common.NewError("no_allocation_writes", "No Allocation writes. challenge gemeration not possible")
	}

	selectedBlobberObj := &StorageNode{}

	blobberAllocation := &BlobberAllocation{}
	blobberAllocation.Stats = &StorageAllocationStats{}
	blobberperm := r.Perm(len(allocationObj.Blobbers))
	for _, v := range blobberperm {
		selectedBlobberObj = allocationObj.Blobbers[v]
		_, ok := allocationObj.BlobberMap[selectedBlobberObj.ID]
		if !ok {
			Logger.Error("Selected blobber not found in allocation state", zap.Any("selected_blobber", selectedBlobberObj), zap.Any("blobber_map", allocationObj.BlobberMap))
			return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation. Could not find blobber")
		}
		blobberAllocation = allocationObj.BlobberMap[selectedBlobberObj.ID]
		if len(blobberAllocation.AllocationRoot) > 0 {
			break
		}
	}

	selectedValidators := make([]*ValidationNode, 0)
	perm := r.Perm(allocationObj.DataShards + 1)
	for _, v := range perm {
		if strings.Compare(validatorList.Nodes[v].ID, selectedBlobberObj.ID) != 0 {
			selectedValidators = append(selectedValidators, validatorList.Nodes[v])
		}

	}

	//Logger.Info("Challenge blobber selected.", zap.Any("challenge", challengeID), zap.Any("selected_blobber", allocationObj.Blobbers[randIdx]), zap.Any("blobbers", allocationObj.Blobbers), zap.Any("random_index", randIdx))

	var storageChallenge StorageChallenge
	storageChallenge.ID = challengeID
	storageChallenge.Validators = selectedValidators
	storageChallenge.Blobber = selectedBlobberObj
	storageChallenge.RandomNumber = challengeSeed
	storageChallenge.AllocationID = allocationObj.ID

	if len(blobberAllocation.AllocationRoot) == 0 {
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

	storageChallenge.Created = creationDate
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
	sc.newChallenge(balances, storageChallenge.Created)
	return string(challengeBytes), err
}
