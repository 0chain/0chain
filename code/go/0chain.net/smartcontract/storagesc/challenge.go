package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/partitions"

	"0chain.net/chaincore/block"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/util"

	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

const passedBlobbersPartitionSize = 50

func OngoingBlobberKey(startRound int64) datastore.Key {
	return ONGOING_PASSED_BLOBBERS_KEY + ":round:" + strconv.Itoa(int(startRound))
}

// getActivePassedBlobbersList gets blobbers passed challenge from last challenge period
func getActivePassedBlobbersList(balances c_state.StateContextI) (partitions.RandPartition, error) {
	all, err := partitions.GetRandomSelector(ACTIVE_PASSED_BLOBBERS_KEY, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		all = partitions.NewRandomSelector(
			ACTIVE_PASSED_BLOBBERS_KEY,
			passedBlobbersPartitionSize,
			nil,
			partitions.ItemBlobberReward,
		)
	}
	all.SetCallback(nil)
	return all, nil
}

// getOngoingPassedBlobbersList gets blobbers passed challenge from ongoing challenge period
func getOngoingPassedBlobbersList(balances c_state.StateContextI, startRound int64) (partitions.RandPartition, error) {
	key := OngoingBlobberKey(startRound)
	all, err := partitions.GetRandomSelector(key, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		all = newOngoingPassedBlobbersList(startRound)
	}
	all.SetCallback(nil)
	return all, nil
}

func newOngoingPassedBlobbersList(startRound int64) partitions.RandPartition {
	key := OngoingBlobberKey(startRound)
	return partitions.NewRandomSelector(
		key,
		passedBlobbersPartitionSize,
		nil,
		partitions.ItemBlobberReward,
	)
}

func (sc *StorageSmartContract) completeChallengeForBlobber(
	blobberChallengeObj *BlobberChallenge, challengeCompleted *StorageChallenge,
	challengeResponse *ChallengeResponse) bool {

	found := false
	if len(blobberChallengeObj.Challenges) > 0 {
		latestOpenChallenge := blobberChallengeObj.Challenges[0]
		if latestOpenChallenge.ID == challengeCompleted.ID {
			found = true
		}
	}
	idx := 0
	if found && idx >= 0 && idx < len(blobberChallengeObj.Challenges) {
		blobberChallengeObj.Challenges = append(blobberChallengeObj.Challenges[:idx], blobberChallengeObj.Challenges[idx+1:]...)
		challengeCompleted.Response = challengeResponse
		blobberChallengeObj.LatestCompletedChallenge = challengeCompleted
	}
	return found
}

func (sc *StorageSmartContract) getBlobberChallenge(blobberID string,
	balances c_state.StateContextI) (bc *BlobberChallenge, err error) {

	bc = new(BlobberChallenge)
	bc.BlobberID = blobberID
	err = balances.GetTrieNode(bc.GetKey(sc.ID), bc)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// move tokens from challenge pool to blobber's stake pool (to unlocked)
func (sc *StorageSmartContract) blobberReward(t *transaction.Transaction,
	alloc *StorageAllocation, prev common.Timestamp, bc *BlobberChallenge,
	details *BlobberAllocation, validators []string, partial float64,
	balances c_state.StateContextI) (err error) {

	var conf *Config
	if conf, err = sc.getConfig(balances, true); err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	// time of this challenge
	var tp = bc.LatestCompletedChallenge.Created

	if tp > alloc.Expiration+toSeconds(details.Terms.ChallengeCompletionTime) {
		return errors.New("late challenge response")
	}

	if tp > alloc.Expiration {
		tp = alloc.Expiration // last challenge
	}

	// pool
	var cp *challengePool
	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("can't get allocation's challenge pool: %v", err)
	}

	var (
		rdtu = alloc.restDurationInTimeUnits(prev)
		dtu  = alloc.durationInTimeUnits(tp - prev)
		move = float64(details.challenge(dtu, rdtu))
	)

	// part of this tokens goes to related validators
	var validatorsReward = conf.ValidatorReward * move
	move -= validatorsReward

	// for a case of a partial verification
	blobberReward := float64(move) * partial // blobber (partial) reward
	back := move - blobberReward             // return back to write pool

	if back > 0 {
		// move back to write pool
		var wp *writePool
		if wp, err = sc.getWritePool(alloc.Owner, balances); err != nil {
			return fmt.Errorf("can't get allocation's write pool: %v", err)
		}
		var until = alloc.Until()
		err = cp.moveToWritePool(alloc, details.BlobberID, until, wp, state.Balance(back))
		if err != nil {
			return fmt.Errorf("moving partial challenge to write pool: %v", err)
		}
		alloc.MovedBack += state.Balance(back)
		details.Returned += state.Balance(back)
		// save the write pool
		if err = wp.save(sc.ID, alloc.Owner, balances); err != nil {
			return fmt.Errorf("can't save allocation's write pool: %v", err)
		}
	}

	var sp *stakePool
	if sp, err = sc.getStakePool(bc.BlobberID, balances); err != nil {
		return fmt.Errorf("can't get stake pool: %v", err)
	}

	err = sp.DistributeRewards(blobberReward, bc.BlobberID, spenum.Blobber, balances)
	if err != nil {
		return fmt.Errorf("can't move tokens to blobber: %v", err)
	}

	details.ChallengeReward += state.Balance(blobberReward)

	// validators' stake pools
	var vsps []*stakePool
	if vsps, err = sc.validatorsStakePools(validators, balances); err != nil {
		return
	}

	err = cp.moveToValidators(sc.ID, validatorsReward, validators, vsps, balances)
	if err != nil {
		return fmt.Errorf("rewarding validators: %v", err)
	}
	alloc.MovedToValidators += state.Balance(validatorsReward)

	// save validators' stake pools
	if err = sc.saveStakePools(validators, vsps, balances); err != nil {
		return
	}

	// save the pools
	if err = sp.save(sc.ID, bc.BlobberID, balances); err != nil {
		return fmt.Errorf("can't save sake pool: %v", err)
	}

	if err = cp.save(sc.ID, alloc.ID, balances); err != nil {
		return fmt.Errorf("can't save allocation's challenge pool: %v", err)
	}

	return
}

// obtain stake pools of given validators
func (ssc *StorageSmartContract) validatorsStakePools(
	validators []datastore.Key, balances c_state.StateContextI) (
	sps []*stakePool, err error) {

	sps = make([]*stakePool, 0, len(validators))
	for _, id := range validators {
		var sp *stakePool
		if sp, err = ssc.getStakePool(id, balances); err != nil {
			return nil, fmt.Errorf("can't get validator %s stake pool: %v",
				id, err)
		}
		sps = append(sps, sp)
	}

	return
}

func (ssc *StorageSmartContract) saveStakePools(validators []datastore.Key,
	sps []*stakePool, balances c_state.StateContextI) (err error) {

	for i, sp := range sps {
		if err = sp.save(ssc.ID, validators[i], balances); err != nil {
			return fmt.Errorf("saving stake pool: %v", err)
		}
	}
	return
}

// move tokens from challenge pool back to write pool
func (sc *StorageSmartContract) blobberPenalty(t *transaction.Transaction,
	alloc *StorageAllocation, prev common.Timestamp, bc *BlobberChallenge,
	details *BlobberAllocation, validators []string,
	balances c_state.StateContextI) (err error) {

	var conf *Config
	if conf, err = sc.getConfig(balances, true); err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	// time of this challenge
	var tp = bc.LatestCompletedChallenge.Created

	if tp > alloc.Expiration+toSeconds(details.Terms.ChallengeCompletionTime) {
		return errors.New("late challenge response")
	}

	if tp > alloc.Expiration {
		tp = alloc.Expiration // last challenge
	}

	// pools
	var cp *challengePool
	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("can't get allocation's challenge pool: %v", err)
	}

	var wp *writePool
	if wp, err = sc.getWritePool(alloc.Owner, balances); err != nil {
		return fmt.Errorf("can't get allocation's write pool: %v", err)
	}

	var (
		rdtu = alloc.restDurationInTimeUnits(prev)
		dtu  = alloc.durationInTimeUnits(tp - prev)
		move = float64(details.challenge(dtu, rdtu))
	)

	// part of this tokens goes to related validators
	var validatorsReward = conf.ValidatorReward * move
	move -= validatorsReward

	// validators' stake pools
	var vsps []*stakePool
	if vsps, err = sc.validatorsStakePools(validators, balances); err != nil {
		return
	}

	// validators reward
	err = cp.moveToValidators(sc.ID, validatorsReward, validators, vsps, balances)
	if err != nil {
		return fmt.Errorf("rewarding validators: %v", err)
	}
	alloc.MovedToValidators += state.Balance(validatorsReward)

	// save validators' stake pools
	if err = sc.saveStakePools(validators, vsps, balances); err != nil {
		return
	}

	// move back to write pool
	var until = alloc.Until()
	err = cp.moveToWritePool(alloc, details.BlobberID, until, wp, state.Balance(move))
	if err != nil {
		return fmt.Errorf("moving failed challenge to write pool: %v", err)
	}
	alloc.MovedBack += state.Balance(move)
	details.Returned += state.Balance(move)

	// blobber stake penalty
	if conf.BlobberSlash > 0 && move > 0 &&
		state.Balance(conf.BlobberSlash*float64(move)) > 0 {

		var slash = state.Balance(conf.BlobberSlash * float64(move))

		// load stake pool
		var sp *stakePool
		if sp, err = sc.getStakePool(bc.BlobberID, balances); err != nil {
			return fmt.Errorf("can't get blobber's stake pool: %v", err)
		}

		var move state.Balance
		move, err = sp.slash(alloc, details.BlobberID, until, wp, details.Offer(), slash, balances)
		if err != nil {
			return fmt.Errorf("can't move tokens to write pool: %v", err)
		}

		sp.TotalOffers -= move  // subtract the offer stake
		details.Penalty += move // penalty statistic

		// save stake pool
		if err = sp.save(sc.ID, bc.BlobberID, balances); err != nil {
			return fmt.Errorf("can't save blobber's stake pool: %v", err)
		}
	}

	// save pools
	if err = wp.save(sc.ID, alloc.Owner, balances); err != nil {
		return fmt.Errorf("can't save allocation's write pool: %v", err)
	}

	if err = cp.save(sc.ID, alloc.ID, balances); err != nil {
		return fmt.Errorf("can't save allocation's challenge pool: %v", err)
	}

	return
}

func (sc *StorageSmartContract) verifyChallenge(t *transaction.Transaction,
	input []byte, balances c_state.StateContextI) (resp string, err error) {

	var challResp ChallengeResponse

	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewError("verify_challenge",
			"cannot get smart contract configurations: "+err.Error())
	}

	startRound := getStartRound(balances.GetBlock().Round, conf.BlockReward.ChallengePeriod)

	var ongoingList partitions.RandPartition

	if balances.GetBlock().Round%conf.BlockReward.ChallengePeriod == 0 {

		if balances.GetBlock().Round != 0 {

			ongoingList, err = getOngoingPassedBlobbersList(balances, startRound-conf.BlockReward.ChallengePeriod)
			if err != nil {
				return "", common.NewError("verify_challenge",
					"cannot get ongoing partition: "+err.Error())
			}

			_, err = balances.InsertTrieNode(ACTIVE_PASSED_BLOBBERS_KEY, ongoingList)
			if err != nil {
				return "", common.NewError("verify_challenge",
					"error updating active passed partition: "+err.Error())
			}

			ongoingList = newOngoingPassedBlobbersList(startRound)
			_, err = balances.InsertTrieNode(OngoingBlobberKey(startRound), ongoingList)
			if err != nil {
				return "", common.NewError("verify_challenge",
					"cannot reset ongoing partition: "+err.Error())
			}

		} else {
			ongoingList, err = getOngoingPassedBlobbersList(balances, startRound)
			if err != nil {
				return "", common.NewError("verify_challenge",
					"cannot get ongoing partition: "+err.Error())
			}
		}

	} else {
		ongoingList, err = getOngoingPassedBlobbersList(balances, startRound)
		if err != nil {
			return "", common.NewError("verify_challenge",
				"cannot get ongoing partition: "+err.Error())
		}
	}

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
		return "", common.NewErrorf("verify_challenge",
			"can't get the blobber challenge %s: %v", t.ClientID, err)
	}

	var challReq, ok = blobberChall.ChallengeMap[challResp.ID]
	if !ok {
		if blobberChall.LatestCompletedChallenge != nil &&
			challResp.ID == blobberChall.LatestCompletedChallenge.ID &&
			blobberChall.LatestCompletedChallenge.Response != nil {

			return "Challenge Already redeemed by Blobber", nil
		}
		return "", common.NewErrorf("verify_challenge",
			"Cannot find the challenge with ID %s", challResp.ID)
	}

	if challReq.Blobber.ID != t.ClientID {
		return "", common.NewError("verify_challenge",
			"Challenge response should be submitted by the same blobber"+
				" as the challenge request")
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(challReq.AllocationID, balances)
	if err != nil {
		return "", common.NewErrorf("verify_challenge",
			"can't get related allocation: %v", err)
	}

	details, ok := alloc.BlobberMap[t.ClientID]
	if !ok {
		return "", common.NewError("verify_challenge",
			"Blobber is not part of the allocation")
	}

	var (
		success, failure int
		validators       []string // validators for rewards
	)
	for _, vt := range challResp.ValidationTickets {
		if vt != nil {
			if ok, err := vt.VerifySign(balances); !ok || err != nil {
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

	pass, fresh, threshold := challReq.isChallengePassed(
		success, failure, details.Terms.ChallengeCompletionTime, t.CreationDate)

	// verification, or partial verification
	if pass && fresh {

		blobber, err := sc.getBlobber(t.ClientID, balances)
		if err != nil {
			return "", common.NewError("verify_challenge",
				"can't get blobber"+err.Error())
		}

		// this expiry of blobber needs to be corrected once logic is finalized

		if blobber.RewardPartition.StartRound != startRound ||
			balances.GetBlock().Round == 0 {

			partIndex, err := ongoingList.Add(
				&partitions.BlobberRewardNode{
					Id:                blobber.ID,
					SuccessChallenges: 0,
					WritePrice:        blobber.Terms.WritePrice,
					ReadPrice:         blobber.Terms.ReadPrice,
					TotalData:         sizeInGB(blobber.BytesWritten),
					DataRead:          blobber.DataRead,
				}, balances)
			if err != nil {
				return "", common.NewError("verify_challenge",
					"can't add to ongoing partition list"+err.Error())
			}

			blobber.RewardPartition = partitionLocation{
				Index:      partIndex,
				StartRound: startRound,
			}

			_, err = balances.InsertTrieNode(blobber.GetKey(sc.ID), blobber)
			if err != nil {
				return "", common.NewError("verify_challenge",
					"error inserting blobber to chain"+err.Error())
			}
		}

		blobberRewardItem, err := ongoingList.GetItem(blobber.RewardPartition.Index, blobber.ID, balances)
		if err != nil {
			return "", common.NewError("verify_challenge",
				"can't get blobber reward from partition list: "+err.Error())
		}

		var brStats partitions.BlobberRewardNode
		err = brStats.Decode(blobberRewardItem.Encode())
		if err != nil {
			return "", common.NewError("verify_challenge",
				"can't decode blobber reward item"+err.Error())
		}

		brStats.SuccessChallenges++

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

		details.Stats.LastestClosedChallengeTxn = challReq.ID
		details.Stats.SuccessChallenges++
		details.Stats.OpenChallenges--

		err = ongoingList.UpdateItem(blobber.RewardPartition.Index, &brStats, balances)
		if err != nil {
			return "", common.NewError("verify_challenge",
				"error updating blobber reward item")
		}

		err = ongoingList.Save(balances)
		if err != nil {
			return "", common.NewError("verify_challenge",
				"error saving ongoing blobber reward partition")
		}

		_, err = balances.InsertTrieNode(blobberChall.GetKey(sc.ID), blobberChall)
		if err != nil {
			return "", common.NewError("verify_challenge", err.Error())
		}
		sc.challengeResolved(balances, true)

		var partial = 1.0
		if success < threshold {
			partial = float64(success) / float64(threshold)
		}

		err = sc.blobberReward(t, alloc, prev, blobberChall, details,
			validators, partial, balances)
		if err != nil {
			return "", common.NewError("challenge_reward_error", err.Error())
		}

		// save allocation object
		_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
		if err != nil {
			return "", common.NewError("challenge_reward_error", err.Error())
		}

		if success < threshold {
			return "challenge passed partially by blobber", nil
		}

		return "challenge passed by blobber", nil
	}

	var enoughFails = failure > (len(challReq.Validators)/2) ||
		(success+failure) == len(challReq.Validators)

	if enoughFails || (pass && !fresh) {

		completed := sc.completeChallengeForBlobber(blobberChall, challReq,
			&challResp)
		if !completed {
			return "", common.NewError("challenge_out_of_order",
				"First challenge on the list is not same as the one"+
					" attempted to redeem")
		}
		alloc.Stats.LastestClosedChallengeTxn = challReq.ID
		alloc.Stats.FailedChallenges++
		alloc.Stats.OpenChallenges--

		details.Stats.LastestClosedChallengeTxn = challReq.ID
		details.Stats.FailedChallenges++
		details.Stats.OpenChallenges--

		balances.InsertTrieNode(blobberChall.GetKey(sc.ID), blobberChall)
		sc.challengeResolved(balances, false)
		Logger.Info("Challenge failed", zap.Any("challenge", challResp.ID))

		err = sc.blobberPenalty(t, alloc, prev, blobberChall, details,
			validators, balances)
		if err != nil {
			return "", common.NewError("challenge_penalty_error", err.Error())
		}

		// save allocation object
		_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
		if err != nil {
			return "", common.NewError("challenge_reward_error", err.Error())
		}

		if pass && !fresh {
			return "late challenge (failed)", nil
		}

		return "Challenge Failed by Blobber", nil
	}

	return "", common.NewError("not_enough_validations",
		"Not enough validations, no successful validations")
}

func getStartRound(currentRound, changePeriod int64) int64 {
	extra := currentRound % changePeriod
	return currentRound - extra
}

func (sc *StorageSmartContract) addGenerateChallengesStat(tp time.Time,
	err *error) {

	if (*err) != nil {
		return // failed call, don't calculate stat
	}

	var tm = sc.SmartContractExecutionStats["generate_challenges"]
	if tm == nil {
		return // missing timer (unexpected)
	}

	if timer, ok := tm.(metrics.Timer); ok {
		timer.Update(time.Since(tp))
	}
}

func (sc *StorageSmartContract) generateChallenges(t *transaction.Transaction,
	b *block.Block, _ []byte, balances c_state.StateContextI) (err error) {

	var tp = time.Now()
	defer sc.addGenerateChallengesStat(tp, &err)

	var stats = &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	//var statsBytes util.Serializable

	err = balances.GetTrieNode(stats.GetKey(sc.ID), stats)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		return nil
	default:
		// unexpected MPT error
		return err
	}

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

	// SC configurations
	var conf *Config
	if conf, err = sc.getConfig(balances, false); err != nil {
		return common.NewErrorf("generate_challenges",
			"can't get SC configurations: %v", err)
	}

	var rated = conf.ChallengeGenerationRate * float64(numMins*sizeDiffMB)
	if rated < 1 {
		rated = 1
	}
	numChallenges := int64(math.Min(rated,
		float64(conf.MaxChallengesPerGeneration)))
	hashString := encryption.Hash(t.Hash + b.PrevHash)
	var randomSeed uint64
	randomSeed, err = strconv.ParseUint(hashString[0:16], 16, 64)
	if err != nil {
		Logger.Error("Error in creating seed for creating challenges",
			zap.Error(err))
		return err
	}
	r := rand.New(rand.NewSource(int64(randomSeed)))

	// select allocations for the challenges

	validators, err := getValidatorsList(balances)
	if err != nil {
		return common.NewErrorf("adding_challenge_error",
			"error getting the validators list: %v", err)
	}

	listLen, err := validators.Size(balances)
	if listLen == 0 {
		return common.NewErrorf("adding_challenge_error",
			"no available validators")
	}

	var all *Allocations
	if all, err = sc.getAllAllocationsList(balances); err != nil {
		return common.NewErrorf("adding_challenge_error",
			"error getting the allocation list: %v", err)
	}

	if len(all.List) == 0 {
		return common.NewError("adding_challenge_error",
			"no allocations at this time")
	}

	var selectAlloc = func(i int) (alloc *StorageAllocation, err error) {
		alloc, err = sc.getAllocation(all.List[i], balances)
		if err != nil && err != util.ErrValueNotPresent {
			return nil, common.NewErrorf("adding_challenge_error",
				"unexpected error getting allocation: %v", err)
		}
		if err == util.ErrValueNotPresent {
			Logger.Error("client state has invalid allocations",
				zap.Any("allocation_list", all.List),
				zap.Any("selected_allocation", all.List[i]))
			return nil, common.NewErrorf("invalid_allocation",
				"client state has invalid allocations")
		}
		if alloc.Expiration < t.CreationDate {
			return nil, nil
		}
		if alloc.Stats == nil {
			return nil, nil
		}
		if alloc.Stats.NumWrites > 0 {
			return alloc, nil // found
		}
		return nil, nil
	}

	//
	//
	//

	var alloc *StorageAllocation

	for i := int64(0); i < numChallenges; i++ {

		// looking for allocation with NumWrites > 0

		alloc, err = selectAlloc(r.Intn(len(all.List)))
		if err != nil {
			return err
		}

		if alloc == nil {
			continue // try another one
		}

		// found

		challengeID := encryption.Hash(hashString + strconv.FormatInt(i, 10))
		var challengeSeed uint64
		challengeSeed, err = strconv.ParseUint(challengeID[0:16], 16, 64)
		if err != nil {
			Logger.Error("Error in creating challenge seed", zap.Error(err),
				zap.Any("challengeID", challengeID))
			continue
		}
		// statistics
		var (
			tp              = time.Now()
			challengeString string
		)
		challengeString, err = sc.addChallenge(alloc, validators, challengeID,
			t.CreationDate, r, int64(challengeSeed), balances)
		if err != nil {
			Logger.Error("Error in adding challenge", zap.Error(err),
				zap.Any("challengeString", challengeString))
			continue
		}
		if tm := sc.SmartContractExecutionStats["challenge_request"]; tm != nil {
			if timer, ok := tm.(metrics.Timer); ok {
				timer.Update(time.Since(tp))
			}
		}
	}
	return nil
}

func (sc *StorageSmartContract) addChallenge(alloc *StorageAllocation,
	validators partitions.RandPartition, challengeID string,
	creationDate common.Timestamp, r *rand.Rand, challengeSeed int64,
	balances c_state.StateContextI) (resp string, err error) {

	sort.SliceStable(alloc.Blobbers, func(i, j int) bool {
		return alloc.Blobbers[i].ID < alloc.Blobbers[j].ID
	})

	selectedBlobberObj := &StorageNode{}

	blobberAllocation := &BlobberAllocation{}
	blobberAllocation.Stats = &StorageAllocationStats{}

	for _, ri := range r.Perm(len(alloc.Blobbers)) {
		selectedBlobberObj = alloc.Blobbers[ri]
		_, ok := alloc.BlobberMap[selectedBlobberObj.ID]
		if !ok {
			Logger.Error("Selected blobber not found in allocation state",
				zap.Any("selected_blobber", selectedBlobberObj),
				zap.Any("blobber_map", alloc.BlobberMap))
			return "", common.NewError("invalid_parameters",
				"Blobber is not part of the allocation. Could not find blobber")
		}
		blobberAllocation = alloc.BlobberMap[selectedBlobberObj.ID]
		if blobberAllocation.AllocationRoot != "" {
			break // found
		}
	}

	if blobberAllocation.AllocationRoot == "" {
		return "", common.NewErrorf("no_blobber_writes", "no blobber writes, "+
			"challenge generation not possible, allocation %s, blobber: %s",
			alloc.ID, blobberAllocation.BlobberID)
	}

	selectedValidators := make([]*ValidationNode, 0)
	randSlice, err := validators.GetRandomSlice(r, balances)

	perm := r.Perm(len(randSlice))
	for i := 0; i < minInt(len(randSlice), alloc.DataShards+1); i++ {
		if randSlice[perm[i]].Name() != selectedBlobberObj.ID {
			selectedValidators = append(selectedValidators,
				&ValidationNode{
					ID:      randSlice[perm[i]].Name(),
					BaseURL: randSlice[perm[i]].Data(),
				})
		}
		if len(selectedValidators) >= alloc.DataShards {
			break
		}
	}

	var storageChallenge StorageChallenge
	storageChallenge.ID = challengeID
	storageChallenge.Validators = selectedValidators
	storageChallenge.Blobber = selectedBlobberObj
	storageChallenge.RandomNumber = challengeSeed
	storageChallenge.AllocationID = alloc.ID

	storageChallenge.AllocationRoot = blobberAllocation.AllocationRoot

	blobberChallengeObj := &BlobberChallenge{}
	blobberChallengeObj.BlobberID = storageChallenge.Blobber.ID

	err = balances.GetTrieNode(blobberChallengeObj.GetKey(sc.ID), blobberChallengeObj)
	switch err {
	case nil, util.ErrValueNotPresent:
	default:
		return "", err
	}

	storageChallenge.Created = creationDate
	addedChallege := blobberChallengeObj.addChallenge(&storageChallenge)
	if !addedChallege {
		challengeBytes, err := json.Marshal(storageChallenge)
		return string(challengeBytes), err
	}

	balances.InsertTrieNode(blobberChallengeObj.GetKey(sc.ID), blobberChallengeObj)

	alloc.Stats.OpenChallenges++
	alloc.Stats.TotalChallenges++
	blobberAllocation.Stats.OpenChallenges++
	blobberAllocation.Stats.TotalChallenges++
	balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	//Logger.Info("Adding a new challenge", zap.Any("blobberChallengeObj", blobberChallengeObj), zap.Any("challenge", storageChallenge.ID))
	challengeBytes, err := json.Marshal(storageChallenge)
	if err := sc.newChallenge(balances, storageChallenge.Created); err != nil {
		return "", err
	}
	return string(challengeBytes), err
}
