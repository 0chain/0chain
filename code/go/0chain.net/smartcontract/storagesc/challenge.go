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
func (sc *StorageSmartContract) blobberReward(t *transaction.Transaction,
	alloc *StorageAllocation, prev common.Timestamp, bc *BlobberChallenge,
	details *BlobberAllocation, validators []string, partial float64,
	balances c_state.StateContextI) (err error) {

	var conf *scConfig
	if conf, err = sc.getConfig(balances, true); err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	// time of this challenge
	var tp = bc.LatestCompletedChallenge.Created

	if tp > alloc.Expiration+toSeconds(alloc.ChallengeCompletionTime) {
		return errors.New("late challenge response")
	}

	// pool
	var cp *challengePool
	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("can't get allocation's challenge pool: %v", err)
	}

	var ratio = float64(tp-prev) / float64(alloc.Expiration-prev)

	if tp > alloc.Expiration {
		ratio = 1 // all left (allocation expired, challenge completion time)
	}

	// blobber ratio (of all blobbers)
	ratio *= float64(details.Stats.UsedSize) / float64(alloc.UsedSize)

	var move = state.Balance(float64(cp.Balance) * ratio)

	// part of this tokens goes to related validators
	var validatorsReward = state.Balance(conf.ValidatorReward * float64(move))
	move -= validatorsReward

	// for a case of a partial verification
	move = state.Balance(float64(move) * partial)

	if err = cp.moveToBlobber(sc.ID, bc.BlobberID, move, balances); err != nil {
		return fmt.Errorf("can't move tokens to blobber: %v", err)
	}

	err = cp.moveToValidatos(sc.ID, validatorsReward, validators, balances)
	if err != nil {
		return fmt.Errorf("rewarding validators: %v", err)
	}

	// save the pool
	if err = cp.save(sc.ID, alloc.ID, balances); err != nil {
		return fmt.Errorf("can't save allocation's challenge pool: %v", err)
	}

	return
}

// move tokens from challenge pool back to write pool
func (sc *StorageSmartContract) blobberPenalty(t *transaction.Transaction,
	alloc *StorageAllocation, prev common.Timestamp, bc *BlobberChallenge,
	details *BlobberAllocation, validators []string,
	balances c_state.StateContextI) (err error) {

	var conf *scConfig
	if conf, err = sc.getConfig(balances, true); err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	// time of this challenge
	var tp = bc.LatestCompletedChallenge.Created

	if tp > alloc.Expiration+toSeconds(alloc.ChallengeCompletionTime) {
		return errors.New("late challenge response")
	}

	// pools
	var cp *challengePool
	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("can't get allocation's challenge pool: %v", err)
	}

	var ratio = float64(tp-prev) / float64(alloc.Expiration-prev)

	if tp > alloc.Expiration {
		ratio = 1 // all left (allocation closed, challenge completion time)
	}

	// rewards for validators
	var (
		sizeRatio = ratio *
			(float64(details.Stats.UsedSize) / float64(alloc.UsedSize))
		fictMove = state.Balance(float64(cp.Balance) * sizeRatio)
		reward   = state.Balance(conf.ValidatorReward * float64(fictMove))
	)
	err = cp.moveToValidatos(sc.ID, reward, validators, balances)
	if err != nil {
		return fmt.Errorf("rewarding validators: %v", err)
	}

	// blobber stake penalty
	if conf.BlobberSlash > 0 {

		// load pools

		var sp *stakePool
		if sp, err = sc.getStakePool(bc.BlobberID, balances); err != nil {
			return fmt.Errorf("can't get blobber's stake pool: %v", err)
		}

		var wp *writePool
		if wp, err = sc.getWritePool(alloc.ID, balances); err != nil {
			return fmt.Errorf("can't get allocation's write pool: %v", err)
		}

		// move blobber's stake tokens to allocation's write pool

		// used size ratio
		ratio *= float64(details.Stats.UsedSize) / float64(details.Size)

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

		if err = wp.save(sc.ID, alloc.ID, balances); err != nil {
			return fmt.Errorf("can't save allocation's write pool: %v", err)
		}

		if err = sp.save(sc.ID, bc.BlobberID, balances); err != nil {
			return fmt.Errorf("can't save blobber's stake pool: %v", err)
		}

	}

	// save challenge pool
	if err = cp.save(sc.ID, alloc.ID, balances); err != nil {
		return fmt.Errorf("can't save allocation's challenge pool: %v", err)
	}

	return
}

func (sc *StorageSmartContract) verifyChallenge(t *transaction.Transaction,
	input []byte, balances c_state.StateContextI) (resp string, err error) {

	var challResp ChallengeResponse
	if err = json.Unmarshal(input, &challResp); err != nil {
		return
	}

	if len(challResp.ID) == 0 ||
		len(challResp.ValidationTickets) == 0 {

		return "", common.NewError("verify_challenge",
			"Invalid parameters to challenge response")
	}

	var blobberChall *BlobberChallenge
	blobberChall, err = sc.getBlobberChallenge(t.ClientID, balances)
	if err != nil {
		return "", common.NewError("verify_challenge",
			"can't get the blobber challenge "+t.ClientID+": "+err.Error())
	}

	var challReq, ok = blobberChall.ChallengeMap[challResp.ID]
	if !ok {
		if blobberChall.LatestCompletedChallenge != nil &&
			challResp.ID == blobberChall.LatestCompletedChallenge.ID &&
			blobberChall.LatestCompletedChallenge.Response != nil {

			return "Challenge Already redeemed by Blobber", nil
		}
		return "", common.NewError("verify_challenge",
			"Cannot find the challenge with ID "+challResp.ID)
	}

	if challReq.Blobber.ID != t.ClientID {
		return "", common.NewError("verify_challenge",
			"Challenge response should be submitted by the same blobber"+
				" as the challenge request")
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(challReq.AllocationID, balances)
	if err != nil {
		return "", common.NewError("verify_challenge",
			"can't get related allocation: "+err.Error())
	}

	blobberAlloc, ok := alloc.BlobberMap[t.ClientID]
	if !ok {
		return "", common.NewError("verify_challenge", "Blobber is not part of the allocation")
	}

	var (
		success, failure int
		validators       []string // validators for rewards
	)
	for _, vt := range challResp.ValidationTickets {
		if vt != nil {
			if ok, err := vt.VerifySign(); !ok || err != nil {
				continue
			}

			validators = append(validators, vt.ValidatorID)

			if !vt.Result {
				failure++
				continue
			}
			success++
		}
	}

	// time of previous complete challenge (not the current one)
	// or allocation start time if no challenges
	var prev = alloc.StartTime
	if last := blobberChall.LatestCompletedChallenge; last != nil {
		prev = last.Created
	}

	var threshold = len(challReq.Validators) / 2

	// verification, or partial verification
	if success > threshold ||
		(success > failure && success+failure < threshold) {

		//challReq.Response = &challResp
		//delete(blobberChall.ChallengeMap, challResp.ID)
		completed := sc.completeChallengeForBlobber(blobberChall, challReq,
			&challResp)
		if !completed {
			return "", common.NewError("challenge_out_of_order",
				"First challenge on the list is not same as the one"+
					" attempted to redeem")
		}
		alloc.Stats.LastestClosedChallengeTxn = challReq.ID
		alloc.Stats.SuccessChallenges++
		alloc.Stats.OpenChallenges--

		blobberAlloc.Stats.LastestClosedChallengeTxn = challReq.ID
		blobberAlloc.Stats.SuccessChallenges++
		blobberAlloc.Stats.OpenChallenges--

		balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
		balances.InsertTrieNode(blobberChall.GetKey(sc.ID), blobberChall)
		sc.challengeResolved(balances, true)
		//Logger.Info("Challenge passed", zap.Any("challenge", challResp.ID))

		var partial = 1.0
		if success < threshold {
			partial = float64(success) / float64(threshold)
		}

		err = sc.blobberReward(t, alloc, prev, blobberChall, blobberAlloc,
			validators, partial, balances)
		if err != nil {
			return "", common.NewError("challenge_reward_error", err.Error())
		}

		if success < threshold {
			return "challenge passed partially by blobber", nil
		}

		return "challenge passed by blobber", nil
	}

	if failure > (len(challReq.Validators)/2) ||
		(success+failure) == len(challReq.Validators) {

		completed := sc.completeChallengeForBlobber(blobberChall, challReq,
			&challResp)
		if !completed {
			return "", common.NewError("challenge_out_of_order",
				"First challenge on the list is not same as the one"+
					" attempted to redeem")
		}
		//delete(blobberChall.ChallengeMap, challResp.ID)
		//challReq.Response = &challResp
		alloc.Stats.LastestClosedChallengeTxn = challReq.ID
		alloc.Stats.FailedChallenges++
		alloc.Stats.OpenChallenges--

		blobberAlloc.Stats.LastestClosedChallengeTxn = challReq.ID
		blobberAlloc.Stats.FailedChallenges++
		blobberAlloc.Stats.OpenChallenges--

		balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
		balances.InsertTrieNode(blobberChall.GetKey(sc.ID), blobberChall)
		sc.challengeResolved(balances, false)
		Logger.Info("Challenge failed", zap.Any("challenge", challResp.ID))

		err = sc.blobberPenalty(t, alloc, prev, blobberChall, blobberAlloc,
			validators, balances)
		if err != nil {
			return "", common.NewError("challenge_penalty_error", err.Error())
		}

		return "Challenge Failed by Blobber", nil
	}

	return "", common.NewError("not_enough_validations",
		"Not enough validations, no successful validations")
}

func (sc *StorageSmartContract) generateChallenges(t *transaction.Transaction,
	b *block.Block, input []byte, balances c_state.StateContextI) error {

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
