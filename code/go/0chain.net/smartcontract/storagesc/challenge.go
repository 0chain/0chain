package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
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

const allBlobbersChallengePartitionSize = 50
const blobberChallengeAllocationPartitionSize = 100
const passedBlobbersPartitionSize = 5

func getBlobbersChallengeList(balances c_state.StateContextI) (partitions.RandPartition, error) {
	all, err := partitions.GetRandomSelector(ALL_BLOBBERS_CHALLENGE_KEY, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		all = partitions.NewRandomSelector(
			ALL_BLOBBERS_CHALLENGE_KEY,
			allBlobbersChallengePartitionSize,
			nil,
			partitions.ItemBlobberChallenge,
		)
	}
	all.SetCallback(nil)
	return all, nil
}

func BlobberRewardKey(round int64) datastore.Key {
	var sb strings.Builder
	sb.WriteString(BLOBBER_REWARD_KEY)
	sb.WriteString(":round:")
	sb.WriteString(strconv.Itoa(int(round)))

	return sb.String()
}

// getActivePassedBlobbersList gets blobbers passed challenge from last challenge period
func getActivePassedBlobbersList(balances c_state.StateContextI, period int64) (partitions.RandPartition, error) {
	key := BlobberRewardKey(GetPreviousRewardRound(balances.GetBlock().Round, period))
	all, err := partitions.GetRandomSelector(key, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		all = partitions.NewRandomSelector(
			key,
			passedBlobbersPartitionSize,
			nil,
			partitions.ItemBlobberReward,
		)
	}
	all.SetCallback(nil)
	return all, nil
}

// getOngoingPassedBlobbersList gets blobbers passed challenge from ongoing challenge period
func getOngoingPassedBlobbersList(balances c_state.StateContextI, period int64) (partitions.RandPartition, error) {
	key := BlobberRewardKey(GetCurrentRewardRound(balances.GetBlock().Round, period))
	all, err := partitions.GetRandomSelector(key, balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		all = partitions.NewRandomSelector(
			key,
			passedBlobbersPartitionSize,
			nil,
			partitions.ItemBlobberReward,
		)
	}
	all.SetCallback(nil)
	return all, nil
}

func getBlobbersChallengeAllocationList(blobberID string, balances c_state.StateContextI) (partitions.RandPartition, error) {
	all, err := partitions.GetRandomSelector(getBlobberChallengeAllocationKey(blobberID), balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		all = partitions.NewRandomSelector(
			getBlobberChallengeAllocationKey(blobberID),
			blobberChallengeAllocationPartitionSize,
			nil,
			partitions.ItemBlobberChallengeAllocation,
		)
	}
	all.SetCallback(nil)
	return all, nil
}

func (sc *StorageSmartContract) completeChallengeForBlobber(
	blobberChallengeObj *BlobberChallenge, challengeCompleted *StorageChallenge,
	challengeResponse *ChallengeResponse, balances c_state.StateContextI) bool {

	found := false
	if len(blobberChallengeObj.ChallengeIDs) > 0 {
		latestOpenChallengeID := blobberChallengeObj.ChallengeIDs[0]
		if latestOpenChallengeID == challengeCompleted.ID {
			found = true
		}
	}
	idx := 0
	if found && idx < len(blobberChallengeObj.ChallengeIDs) {

		blobberChallengeObj.ChallengeIDs = blobberChallengeObj.ChallengeIDs[1:]
		allocChallenge, err := sc.getAllocationChallenge(challengeCompleted.AllocationID, balances)
		if err != nil {
			Logger.Error("error fetching allocation challenge (complete_challenge)",
				zap.String("allocation id", challengeCompleted.AllocationID))
			return false
		}

		if _, ok := allocChallenge.ChallengeMap[challengeCompleted.ID]; ok {
			for i := range allocChallenge.Challenges {
				if allocChallenge.Challenges[i].ID == challengeCompleted.ID {
					allocChallenge.Challenges = append(
						allocChallenge.Challenges[:i], allocChallenge.Challenges[i+1:]...)
					break
				}
			}

			_, err = balances.InsertTrieNode(allocChallenge.GetKey(sc.ID), allocChallenge)
			if err != nil {
				Logger.Error("error inserting allocation challenge (complete_challenge)",
					zap.String("allocation id", challengeCompleted.AllocationID))
				return false
			}

		}

		if challengeResponse != nil {
			challengeCompleted.Responded = true
		}
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

func (sc *StorageSmartContract) getStorageChallenge(challengeID string,
	balances c_state.StateContextI) (challenge *StorageChallenge, err error) {

	challenge = new(StorageChallenge)
	challenge.ID = challengeID
	err = balances.GetTrieNode(challenge.GetKey(sc.ID), challenge)
	if err != nil {
		return nil, err
	}

	return challenge, nil
}

func (sc *StorageSmartContract) getAllocationChallenge(allocID string,
	balances c_state.StateContextI) (ac *AllocationChallenge, err error) {

	ac = new(AllocationChallenge)
	ac.AllocationID = allocID
	err = balances.GetTrieNode(ac.GetKey(sc.ID), ac)
	if err != nil {
		return nil, err
	}

	return ac, nil
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
	blobberReward := move * partial // blobber (partial) reward
	back := move - blobberReward    // return back to write pool

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

	rewardRound := GetCurrentRewardRound(balances.GetBlock().Round, conf.BlockReward.TriggerPeriod)

	ongoingList, err := getOngoingPassedBlobbersList(balances, conf.BlockReward.TriggerPeriod)
	if err != nil {
		return "", common.NewError("verify_challenge",
			"cannot get ongoing partition: "+err.Error())
	}

	if err = json.Unmarshal(input, &challResp); err != nil {
		return
	}

	if len(challResp.ID) == 0 ||
		len(challResp.ValidationTickets) == 0 {

		return "", common.NewError("verify_challenge",
			"Invalid parameters to challenge response")
	}

	blobberChall, err := sc.getBlobberChallenge(t.ClientID, balances)
	if err != nil {
		return "", common.NewErrorf("verify_challenge",
			"can't get the blobber challenge %s: %v", t.ClientID, err)
	}

	var _, ok = blobberChall.ChallengeIDMap[challResp.ID]
	if !ok {
		if blobberChall.LatestCompletedChallenge != nil &&
			challResp.ID == blobberChall.LatestCompletedChallenge.ID &&
			blobberChall.LatestCompletedChallenge.Responded {

			return "Challenge Already redeemed by Blobber", nil
		}
		return "", common.NewErrorf("verify_challenge",
			"Cannot find the challenge with ID %s", challResp.ID)
	}

	if blobberChall.BlobberID != t.ClientID {
		return "", common.NewError("verify_challenge",
			"Challenge response should be submitted by the same blobber"+
				" as the challenge request")
	}

	challReq, err := sc.getStorageChallenge(challResp.ID, balances)
	if err != nil {
		return "", common.NewErrorf("verify_challenge",
			"Cannot fetch the challenge with ID %s", challResp.ID)
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

	var (
		threshold = challReq.TotalValidators / 2
		pass      = success > threshold ||
			(success > failure && success+failure < threshold)
		cct   = toSeconds(details.Terms.ChallengeCompletionTime)
		fresh = challReq.Created+cct >= t.CreationDate
	)

	// verification, or partial verification
	if pass && fresh {
		blobber, err := sc.getBlobber(t.ClientID, balances)
		if err != nil {
			return "", common.NewError("verify_challenge",
				"can't get blobber"+err.Error())
		}

		// this expiry of blobber needs to be corrected once logic is finalized

		if blobber.RewardPartition.StartRound != rewardRound ||
			balances.GetBlock().Round == 0 {

			var dataRead float64 = 0
			if blobber.LastRewardDataReadRound >= rewardRound {
				dataRead = blobber.DataReadLastRewardRound
			}

			partIndex, err := ongoingList.Add(
				&partitions.BlobberRewardNode{
					ID:                blobber.ID,
					SuccessChallenges: 0,
					WritePrice:        blobber.Terms.WritePrice,
					ReadPrice:         blobber.Terms.ReadPrice,
					TotalData:         sizeInGB(blobber.BytesWritten),
					DataRead:          dataRead,
				}, balances)
			if err != nil {
				return "", common.NewError("verify_challenge",
					"can't add to ongoing partition list "+err.Error())
			}

			blobber.RewardPartition = RewardPartitionLocation{
				Index:      partIndex,
				StartRound: rewardRound,
				Timestamp:  t.CreationDate,
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

		completed := sc.completeChallengeForBlobber(blobberChall, challReq, &challResp, balances)
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

		_, err = balances.InsertTrieNode(challReq.GetKey(sc.ID), challReq)
		if err != nil {
			return "", common.NewError("verify_challenge_error", err.Error())
		}
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

	var enoughFails = failure > (challReq.TotalValidators/2) ||
		(success+failure) == challReq.TotalValidators

	if enoughFails || (pass && !fresh) {

		completed := sc.completeChallengeForBlobber(blobberChall, challReq, &challResp, balances)
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

		_, err := balances.InsertTrieNode(blobberChall.GetKey(sc.ID), blobberChall)
		if err != nil {
			return "", common.NewError("challenge_penalty_error", err.Error())
		}

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

func (sc *StorageSmartContract) getAllocationForChallenge(
	t *transaction.Transaction,
	allocID string,
	balances c_state.StateContextI) (alloc *StorageAllocation, err error) {

	alloc, err = sc.getAllocation(allocID, balances)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		Logger.Error("client state has invalid allocations",
			zap.Any("selected_allocation", allocID))
		return nil, common.NewErrorf("invalid_allocation",
			"client state has invalid allocations")
	default:
		return nil, common.NewErrorf("adding_challenge_error",
			"unexpected error getting allocation: %v", err)
	}

	if alloc.Expiration < t.CreationDate {
		return nil, common.NewError("adding_challenge_error",
			"allocation is already expired")
	}
	if alloc.Stats == nil {
		return nil, common.NewError("adding_challenge_error",
			"found empty allocation stats")
	}
	if alloc.Stats.NumWrites > 0 {
		return alloc, nil // found
	}
	return nil, nil
}

type challengeInput struct {
	cr          *rand.Rand
	challengeID string
}

type challengeOutput struct {
	alloc            *StorageAllocation
	storageChallenge *StorageChallenge
	blobberChallenge *BlobberChallenge
	allocChallenge   *AllocationChallenge
	blobberAlloc     *BlobberAllocation
	error            error
}

func (sc *StorageSmartContract) populateGenerateChallenge(
	blobberChallengeList partitions.RandPartition,
	challRand *rand.Rand,
	validators partitions.RandPartition,
	t *transaction.Transaction,
	challengeID string,
	balances c_state.StateContextI,
) (*challengeOutput, error) {

	bcPartition, err := blobberChallengeList.GetRandomSlice(challRand, balances)
	if err != nil {
		return nil, common.NewError("generate_challenges",
			"error getting random slice from blobber challenge partition")
	}

	randomIndex := challRand.Intn(len(bcPartition))
	bcItem := bcPartition[randomIndex]

	blobberID := bcItem.Name()
	if blobberID == "" {
		return nil, common.NewError("add_challenges",
			"empty blobber id")
	}

	bcAllocList, err := getBlobbersChallengeAllocationList(blobberID, balances)
	if err != nil {
		return nil, common.NewError("generate_challenges",
			"error getting blobber_challenge_allocation list: "+err.Error())
	}

	// maybe we should use another random seed
	bcAllocPartition, err := bcAllocList.GetRandomSlice(challRand, balances)
	if err != nil {
		return nil, common.NewError("generate_challenges",
			"error getting random slice from blobber challenge allocation partition")
	}

	randomIndex = challRand.Intn(len(bcAllocPartition))
	bcAllocItem := bcAllocPartition[randomIndex]

	allocID := bcAllocItem.Name()

	alloc, err := sc.getAllocationForChallenge(t, allocID, balances)
	if err != nil {
		return nil, err
	}

	if alloc == nil {
		return nil, errors.New("empty allocation")
	}

	blobber := &StorageNode{}

	for _, b := range alloc.Blobbers {
		if b.ID == blobberID {
			blobber = b
			break
		}
	}

	blobberAllocation, ok := alloc.BlobberMap[blobber.ID]
	if !ok {
		return nil, common.NewError("add_challenges",
			"blobber allocation doesn't exists in allocation")
	}

	if blobberAllocation.Stats == nil {
		blobberAllocation.Stats = new(StorageAllocationStats)
	}

	selectedValidators := make([]*ValidationNode, 0)
	randValidators, err := validators.GetRandomSlice(challRand, balances)
	if err != nil {
		return nil, common.NewError("add_challenge",
			"error getting validators random slice: "+err.Error())
	}

	perm := challRand.Perm(len(randValidators))
	for i := 0; i < minInt(len(randValidators), alloc.DataShards+1); i++ {
		randValidator := randValidators[perm[i]]
		if randValidator.Name() != blobber.ID {
			selectedValidators = append(selectedValidators,
				&ValidationNode{
					ID:      randValidator.Name(),
					BaseURL: randValidator.Data(),
				})
		}
		if len(selectedValidators) >= alloc.DataShards {
			break
		}
	}

	var storageChallenge = new(StorageChallenge)
	storageChallenge.ID = challengeID
	storageChallenge.TotalValidators = len(selectedValidators)
	storageChallenge.BlobberID = blobberID
	storageChallenge.AllocationID = alloc.ID
	storageChallenge.Created = t.CreationDate

	blobberChallengeObj, err := sc.getBlobberChallenge(blobberID, balances)
	if err != nil {
		if err == util.ErrValueNotPresent {
			blobberChallengeObj = &BlobberChallenge{}
			blobberChallengeObj.BlobberID = blobberID
		} else {
			return nil, common.NewError("add_challenge",
				"error fetching blobber challenge: "+err.Error())
		}
	}

	allocChallengeObj, err := sc.getAllocationChallenge(alloc.ID, balances)
	if err != nil {
		if err == util.ErrValueNotPresent {
			allocChallengeObj = &AllocationChallenge{}
			allocChallengeObj.AllocationID = alloc.ID
		} else {
			return nil, common.NewError("add_challenge",
				"error fetching allocation challenge: "+err.Error())
		}
	}

	return &challengeOutput{
		alloc:            alloc,
		storageChallenge: storageChallenge,
		blobberChallenge: blobberChallengeObj,
		allocChallenge:   allocChallengeObj,
		blobberAlloc:     blobberAllocation,
	}, nil
}

func (sc *StorageSmartContract) asyncGenerateChallenges(
	blobberChallengeList partitions.RandPartition,
	validators partitions.RandPartition,
	data <-chan challengeInput,
	result chan<- challengeOutput,
	t *transaction.Transaction,
	balances c_state.StateContextI) {

	defer close(result)

	for d := range data {
		resp, err := sc.populateGenerateChallenge(
			blobberChallengeList,
			d.cr,
			validators,
			t,
			d.challengeID,
			balances)
		if err != nil {
			result <- challengeOutput{
				error: err,
			}
		} else {
			result <- *resp
		}
	}
}

func (sc *StorageSmartContract) generateChallenge(t *transaction.Transaction,
	b *block.Block, _ []byte, balances c_state.StateContextI) (err error) {

	hashString := encryption.Hash(t.Hash + b.PrevHash)

	validators, err := getValidatorsList(balances)
	if err != nil {
		return common.NewErrorf("generate_challenge",
			"error getting the validators list: %v", err)
	}

	blobberChallengeList, err := getBlobbersChallengeList(balances)
	if err != nil {
		return common.NewErrorf("generate_challenge",
			"error getting the blobber challenge list: %v", err)
	}
	if listSize, err := blobberChallengeList.Size(balances); err == nil && listSize == 0 {
		Logger.Info("skipping generate challenge: empty blobber challenge partition")
		return nil
	}

	challengeID := encryption.Hash(hashString + strconv.FormatInt(1, 10))
	var challengeSeed uint64
	challengeSeed, err = strconv.ParseUint(challengeID[0:16], 16, 64)
	if err != nil {
		return common.NewErrorf("generate_challenge",
			"Error in creating challenge seed: %v", err)
	}
	cr := rand.New(rand.NewSource(int64(challengeSeed)))

	result, err := sc.populateGenerateChallenge(
		blobberChallengeList,
		cr,
		validators,
		t,
		challengeID,
		balances)
	if err != nil {
		return common.NewErrorf("adding_challenge_error", err.Error())
	}

	var alloc = result.alloc
	_, err = sc.addChallenge(
		alloc,
		result.storageChallenge,
		result.blobberChallenge,
		result.allocChallenge,
		result.blobberAlloc,
		balances,
	)
	if err != nil {
		return common.NewErrorf("adding_challenge_error",
			"Error in adding challenge: %v", err)
	}

	return nil
}

func (sc *StorageSmartContract) generateChallenges(t *transaction.Transaction,
	b *block.Block, _ []byte, balances c_state.StateContextI) (err error) {

	// SC configurations
	var conf *Config
	if conf, err = sc.getConfig(balances, false); err != nil {
		return common.NewErrorf("generate_challenges",
			"can't get SC configurations: %v", err)
	}

	numChallenges := conf.MaxChallengesPerGeneration
	hashString := encryption.Hash(t.Hash + b.PrevHash)

	validators, err := getValidatorsList(balances)
	if err != nil {
		return common.NewErrorf("adding_challenge_error",
			"error getting the validators list: %v", err)
	}

	listLen, err := validators.Size(balances)
	if err != nil {
		return common.NewErrorf("adding_challenge_error",
			"error checking validator size: %v", err)
	}
	if listLen == 0 {
		return common.NewErrorf("adding_challenge_error",
			"no available validators")
	}

	blobberChallengeList, err := getBlobbersChallengeList(balances)
	if err != nil {
		return common.NewErrorf("adding_challenge_error",
			"error getting the blobber challenge list: %v", err)
	}

	var (
		data   = make(chan challengeInput, numChallenges)
		output = make(chan challengeOutput, numChallenges)
	)

	for i := 0; i < 8; i++ {
		go sc.asyncGenerateChallenges(blobberChallengeList, validators, data, output, t, balances)
	}

	for i := 0; i < numChallenges; i++ {

		challengeID := encryption.Hash(hashString + strconv.FormatInt(int64(i), 10))
		var challengeSeed uint64
		challengeSeed, err = strconv.ParseUint(challengeID[0:16], 16, 64)
		if err != nil {
			Logger.Error("Error in creating challenge seed", zap.Error(err),
				zap.Any("challengeID", challengeID))
			continue
		}
		cr := rand.New(rand.NewSource(int64(challengeSeed)))
		data <- challengeInput{
			cr:          cr,
			challengeID: challengeID,
		}
	}
	close(data)

	for result := range output {

		if result.error != nil {
			return result.error
		}
		var (
			tp              = time.Now()
			challengeString string
			alloc           = result.alloc
		)

		challengeString, err = sc.addChallenge(alloc, result.storageChallenge,
			result.blobberChallenge,
			result.allocChallenge,
			result.blobberAlloc,
			balances)

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
	storageChallenge *StorageChallenge,
	blobberChallengeObj *BlobberChallenge,
	allocChallengeObj *AllocationChallenge,
	blobberAllocation *BlobberAllocation,
	balances c_state.StateContextI) (resp string, err error) {

	if storageChallenge.BlobberID == "" {
		return "", common.NewError("add_challenge",
			"no blobber to add challenge to")
	}

	if blobberAllocation == nil {
		return "", common.NewError("add_challenge",
			"no blobber Allocation to add challenge to")
	}

	addedChallenge := blobberChallengeObj.addChallenge(storageChallenge)
	if !addedChallenge {
		challengeBytes, err := json.Marshal(storageChallenge)
		return string(challengeBytes), err
	}

	addedAllocChallenge := allocChallengeObj.addChallenge(storageChallenge)
	if !addedAllocChallenge {
		challengeBytes, err := json.Marshal(storageChallenge)
		return string(challengeBytes), err
	}

	_, err = balances.InsertTrieNode(allocChallengeObj.GetKey(sc.ID), allocChallengeObj)
	if err != nil {
		return "", common.NewError("add_challenge",
			"error storing alloc challenge: "+err.Error())
	}

	_, err = balances.InsertTrieNode(storageChallenge.GetKey(sc.ID), storageChallenge)
	if err != nil {
		return "", common.NewError("add_challenge",
			"error storing challenge: "+err.Error())
	}

	_, err = balances.InsertTrieNode(blobberChallengeObj.GetKey(sc.ID), blobberChallengeObj)
	if err != nil {
		return "", common.NewError("add_challenge",
			"error storing blobber challenge: "+err.Error())
	}

	alloc.Stats.OpenChallenges++
	alloc.Stats.TotalChallenges++
	blobberAllocation.Stats.OpenChallenges++
	blobberAllocation.Stats.TotalChallenges++

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("add_challenge",
			"error storing allocation: "+err.Error())
	}

	challengeBytes, err := json.Marshal(storageChallenge)
	return string(challengeBytes), err
}
