package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"0chain.net/smartcontract/provider"

	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/partitions"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"

	"go.uber.org/zap"
)

// TODO: add back after fixing the chain stuck
// const blobberAllocationPartitionSize = 100

type BlobberChallengeResponded int

const (
	ChallengeNotResponded BlobberChallengeResponded = iota
	ChallengeResponded
	ChallengeRespondedLate
	ChallengeRespondedInvalid
)

const blobberAllocationPartitionSize = 10

// completeChallenge complete the challenge
func (sc *StorageSmartContract) completeChallenge(cab *challengeAllocBlobberPassResult) bool {
	if !cab.allocChallenges.removeChallenge(cab.challenge) {
		return false
	}

	// update to latest challenge
	cab.blobAlloc.LatestCompletedChallenge = cab.challenge
	return true
}

func (sc *StorageSmartContract) getStorageChallenge(challengeID string,
	balances cstate.StateContextI) (challenge *StorageChallenge, err error) {

	challenge = new(StorageChallenge)
	challenge.ID = challengeID
	err = balances.GetTrieNode(challenge.GetKey(sc.ID), challenge)
	if err != nil {
		return nil, err
	}
	challenge.ValidatorIDMap = make(map[string]struct{}, len(challenge.ValidatorIDs))
	for _, vID := range challenge.ValidatorIDs {
		challenge.ValidatorIDMap[vID] = struct{}{}
	}

	return challenge, nil
}

func (sc *StorageSmartContract) getAllocationChallenges(allocID string,
	balances cstate.StateContextI) (ac *AllocationChallenges, err error) {

	ac = new(AllocationChallenges)
	ac.AllocationID = allocID
	err = balances.GetTrieNode(ac.GetKey(sc.ID), ac)
	if err != nil {
		return nil, err
	}

	return ac, nil
}

// move tokens from challenge pool to blobber's stake pool (to unlocked)
func (sc *StorageSmartContract) blobberReward(
	alloc *StorageAllocation,
	latestCompletedChallTime common.Timestamp,
	blobAlloc *BlobberAllocation,
	validators []string,
	partial float64,
	balances cstate.StateContextI,
	allocationID string,
) error {
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	// time of this challenge
	challengeCompletedTime := blobAlloc.LatestCompletedChallenge.Created
	if challengeCompletedTime > alloc.Expiration {
		return errors.New("late challenge response")
	}

	if challengeCompletedTime < latestCompletedChallTime {
		logging.Logger.Debug("old challenge response - blobber reward",
			zap.Int64("latestCompletedChallTime", int64(latestCompletedChallTime)),
			zap.Int64("challenge time", int64(challengeCompletedTime)))
		return errors.New("old challenge response on blobber rewarding")
	}

	if challengeCompletedTime > alloc.Expiration {
		challengeCompletedTime = alloc.Expiration // last challenge
	}

	// pool
	var cp *challengePool
	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("can't get allocation's challenge pool: %v", err)
	}

	rdtu, err := alloc.restDurationInTimeUnits(latestCompletedChallTime, conf.TimeUnit)
	if err != nil {
		return fmt.Errorf("blobber reward failed: %v", err)
	}

	dtu, err := alloc.durationInTimeUnits(challengeCompletedTime-latestCompletedChallTime, conf.TimeUnit)
	if err != nil {
		return fmt.Errorf("blobber reward failed: %v", err)
	}

	move, err := blobAlloc.challenge(dtu, rdtu)
	if err != nil {
		return err
	}

	// part of tokens goes to related validators
	var validatorsReward currency.Coin
	validatorsReward, err = currency.MultFloat64(move, conf.ValidatorReward)
	if err != nil {
		return err
	}

	move, err = currency.MinusCoin(move, validatorsReward)
	if err != nil {
		return err
	}

	// for a case of a partial verification
	blobberReward, err := currency.MultFloat64(move, partial) // blobber (partial) reward
	if err != nil {
		return err
	}

	back, err := currency.MinusCoin(move, blobberReward) // return back to write pool
	if err != nil {
		return err
	}

	if back > 0 {
		err = alloc.moveFromChallengePool(cp, back)
		if err != nil {
			return fmt.Errorf("moving partial challenge to write pool: %v", err)
		}
		newMoved, err := currency.AddCoin(alloc.MovedBack, back)
		if err != nil {
			return err
		}
		alloc.MovedBack = newMoved

		newReturned, err := currency.AddCoin(blobAlloc.Returned, back)
		if err != nil {
			return err
		}
		blobAlloc.Returned = newReturned

		coin, _ := move.Int64()
		balances.EmitEvent(event.TypeStats, event.TagFromChallengePool, cp.ID, event.ChallengePoolLock{
			Client:       alloc.Owner,
			AllocationId: alloc.ID,
			Amount:       coin,
		})

	}

	var sp *stakePool
	if sp, err = sc.getStakePool(spenum.Blobber, blobAlloc.BlobberID, balances); err != nil {
		return fmt.Errorf("can't get stake pool: %v", err)
	}

	err = cp.moveToBlobbers(sc.ID, blobberReward, blobAlloc.BlobberID, sp, balances, allocationID)
	if err != nil {
		return fmt.Errorf("rewarding blobbers: %v", err)
	}

	newChallengeReward, err := currency.AddCoin(blobAlloc.ChallengeReward, blobberReward)
	if err != nil {
		return err
	}
	blobAlloc.ChallengeReward = newChallengeReward

	// validators' stake pools
	vsps, err := sc.validatorsStakePools(validators, balances)
	if err != nil {
		return err
	}

	err = cp.moveToValidators(validatorsReward, validators, vsps, balances, allocationID)
	if err != nil {
		return fmt.Errorf("rewarding validators: %v", err)
	}

	moveToValidators, err := currency.AddCoin(alloc.MovedToValidators, validatorsReward)
	if err != nil {
		return err
	}
	alloc.MovedToValidators = moveToValidators

	// Save validators' stake pools
	if err = sc.saveStakePools(validators, vsps, balances); err != nil {
		return err
	}

	// Save the pools
	if err = sp.Save(spenum.Blobber, blobAlloc.BlobberID, balances); err != nil {
		return fmt.Errorf("can't save sake pool: %v", err)
	}

	if err = cp.save(sc.ID, alloc, balances); err != nil {
		return fmt.Errorf("can't save allocation's challenge pool: %v", err)
	}

	if err = alloc.saveUpdatedStakes(balances); err != nil {
		return fmt.Errorf("can't save allocation: %v", err)
	}

	return nil
}

// obtain stake pools of given validators
func (ssc *StorageSmartContract) validatorsStakePools(
	validators []datastore.Key, balances cstate.StateContextI) (
	sps []*stakePool, err error) {

	sps = make([]*stakePool, 0, len(validators))
	for _, id := range validators {
		var sp *stakePool
		if sp, err = ssc.getStakePool(spenum.Validator, id, balances); err != nil {
			return nil, fmt.Errorf("can't get validator %s stake pool: %v",
				id, err)
		}
		sps = append(sps, sp)
	}

	return
}

func (ssc *StorageSmartContract) saveStakePools(validators []datastore.Key,
	sps []*stakePool, balances cstate.StateContextI) (err error) {

	for i, sp := range sps {
		if err = sp.Save(spenum.Validator, validators[i], balances); err != nil {
			return fmt.Errorf("saving stake pool: %v", err)
		}

		staked, err := sp.stake()
		if err != nil {
			return fmt.Errorf("can't get stake: %v", err)
		}
		vid := validators[i]
		tag, data := event.NewUpdateBlobberTotalStakeEvent(vid, staked)
		balances.EmitEvent(event.TypeStats, tag, vid, data)
	}
	return
}

// move tokens from challenge pool back to write pool
func (sc *StorageSmartContract) blobberPenalty(
	alloc *StorageAllocation,
	prev common.Timestamp,
	blobAlloc *BlobberAllocation,
	validators []string,
	balances cstate.StateContextI,
	allocationID string,
) (err error) {
	var conf *Config
	if conf, err = sc.getConfig(balances, true); err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	// time of this challenge
	challengeCompleteTime := blobAlloc.LatestCompletedChallenge.Created
	if challengeCompleteTime > alloc.Expiration {
		return errors.New("late challenge response")
	}

	if challengeCompleteTime < prev {
		logging.Logger.Debug("old challenge response - blobber penalty",
			zap.Int64("latestCompletedChallTime", int64(prev)),
			zap.Int64("challenge time", int64(challengeCompleteTime)))
		return errors.New("old challenge response on blobber penalty")
	}

	if challengeCompleteTime > alloc.Expiration {
		challengeCompleteTime = alloc.Expiration // last challenge
	}

	// pools
	var cp *challengePool
	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("can't get allocation's challenge pool: %v", err)
	}

	rdtu, err := alloc.restDurationInTimeUnits(prev, conf.TimeUnit)
	if err != nil {
		return fmt.Errorf("blobber penalty failed: %v", err)
	}

	dtu, err := alloc.durationInTimeUnits(challengeCompleteTime-prev, conf.TimeUnit)
	if err != nil {
		return fmt.Errorf("blobber penalty failed: %v", err)
	}

	move, err := blobAlloc.challenge(dtu, rdtu)
	if err != nil {
		return err
	}

	// part of the tokens goes to related validators
	validatorsReward, err := currency.MultFloat64(move, conf.ValidatorReward)
	if err != nil {
		return err
	}
	move, err = currency.MinusCoin(move, validatorsReward)
	if err != nil {
		return err
	}

	// validators' stake pools
	var vSPs []*stakePool
	if vSPs, err = sc.validatorsStakePools(validators, balances); err != nil {
		return
	}

	// validators reward
	err = cp.moveToValidators(validatorsReward, validators, vSPs, balances, allocationID)
	if err != nil {
		return fmt.Errorf("rewarding validators: %v", err)
	}

	moveToValidators, err := currency.AddCoin(alloc.MovedToValidators, validatorsReward)
	if err != nil {
		return err
	}
	alloc.MovedToValidators = moveToValidators

	// Save validators' stake pools
	if err = sc.saveStakePools(validators, vSPs, balances); err != nil {
		return err
	}

	err = alloc.moveFromChallengePool(cp, move)
	coin, _ := move.Int64()
	balances.EmitEvent(event.TypeStats, event.TagFromChallengePool, cp.ID, event.ChallengePoolLock{
		Client:       alloc.Owner,
		AllocationId: alloc.ID,
		Amount:       coin,
	})

	if err != nil {
		return fmt.Errorf("moving challenge pool rest back to write pool: %v", err)
	}

	moveBack, err := currency.AddCoin(alloc.MovedBack, move)
	if err != nil {
		return err
	}
	alloc.MovedBack = moveBack

	blobReturned, err := currency.AddCoin(blobAlloc.Returned, move)
	if err != nil {
		return err
	}
	blobAlloc.Returned = blobReturned

	slash, err := currency.MultFloat64(move, conf.BlobberSlash)
	if err != nil {
		return err
	}

	// blobber stake penalty
	if conf.BlobberSlash > 0 && move > 0 &&
		slash > 0 {

		// load stake pool
		var sp *stakePool
		if sp, err = sc.getStakePool(spenum.Blobber, blobAlloc.BlobberID, balances); err != nil {
			return fmt.Errorf("can't get blobber's stake pool: %v", err)
		}

		var move currency.Coin
		move, err = sp.slash(blobAlloc.BlobberID, blobAlloc.Offer(), slash, balances, allocationID)
		if err != nil {
			return fmt.Errorf("can't move tokens to write pool: %v", err)
		}

		if err := sp.reduceOffer(move); err != nil {
			return err
		}

		penalty, err := currency.AddCoin(blobAlloc.Penalty, move) // penalty statistic
		if err != nil {
			return err
		}
		blobAlloc.Penalty = penalty
		// Save stake pool
		if err = sp.Save(spenum.Blobber, blobAlloc.BlobberID, balances); err != nil {
			return fmt.Errorf("can't Save blobber's stake pool: %v", err)
		}
	}

	if err = alloc.saveUpdatedStakes(balances); err != nil {
		return common.NewError("fini_alloc_failed",
			"saving allocation pools: "+err.Error())
	}

	if err = cp.save(sc.ID, alloc, balances); err != nil {
		return fmt.Errorf("can't Save allocation's challenge pool: %v", err)
	}

	return
}

func (sc *StorageSmartContract) verifyChallenge(t *transaction.Transaction,
	input []byte, balances cstate.StateContextI) (resp string, err error) {
	var (
		challResp ChallengeResponse
		errCode   = "verify_challenge"
	)

	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewErrorf(errCode,
			"cannot get smart contract configurations: %v", err)
	}

	if err := json.Unmarshal(input, &challResp); err != nil {
		return "", common.NewErrorf(errCode, "failed to decode txn input: %v", err)
	}

	if len(challResp.ID) == 0 || len(challResp.ValidationTickets) == 0 {
		return "", common.NewError(errCode, "invalid parameters to challenge response")
	}

	// get challenge node
	challenge, err := sc.getStorageChallenge(challResp.ID, balances)
	if err != nil {
		return "", common.NewErrorf(errCode, "could not find challenge, %v", err)
	}
	if challenge.Responded != int64(ChallengeNotResponded) {
		return "", common.NewError(errCode, "challenge already processed")
	}

	currentRound := balances.GetBlock().Round
	if challenge.RoundCreatedAt+conf.MaxChallengeCompletionRounds <= currentRound {
		return "", common.NewError(errCode, "challenge expired")
	}

	if challenge.BlobberID != t.ClientID {
		return "", errors.New("challenge blobber id does not match")
	}

	logging.Logger.Info("time_taken: receive challenge response",
		zap.String("challenge_id", challenge.ID),
		zap.Duration("delay", time.Since(common.ToTime(challenge.Created))))

	result, err := verifyChallengeTickets(balances, challenge, &challResp, conf.MaxChallengeCompletionRounds)
	if err != nil {
		return "", common.NewError(errCode, err.Error())
	}

	allocChallenges, err := sc.getAllocationChallenges(challenge.AllocationID, balances)
	if err != nil {
		return "", common.NewErrorf(errCode, "could not find allocation challenges, %v", err)
	}

	alloc, err := sc.getAllocation(challenge.AllocationID, balances)
	if err != nil {
		return "", common.NewErrorf(errCode,
			"can't get related allocation: %v", err)
	}

	if t.CreationDate >= alloc.Expiration {
		return "", common.NewError(errCode, "allocation is finalized")
	}

	blobAlloc, ok := alloc.BlobberAllocsMap[t.ClientID]
	if !ok {
		return "", common.NewError(errCode, "blobber is not part of the allocation")
	}

	lcc := blobAlloc.LatestCompletedChallenge
	_, ok = allocChallenges.ChallengeMap[challResp.ID]
	if !ok {
		// TODO: remove this challenge already redeemed response. This response will be returned only when the
		// challenge is the last completed challenge, which means if we have more challenges completed after it, we
		// will see different result, even the challenge's state is the same as 'it has been redeemed'.
		if lcc != nil && challResp.ID == lcc.ID && lcc.Responded == int64(ChallengeResponded) {
			return "challenge already redeemed", nil
		}

		return "", common.NewErrorf(errCode,
			"could not find the challenge with ID %s", challResp.ID)
	}

	// time of previous complete challenge (not the current one)
	// or allocation start time if no challenges
	latestCompletedChallTime := alloc.StartTime
	if lcc != nil {
		latestCompletedChallTime = lcc.Created
	}

	challenge.Responded = int64(ChallengeResponded)
	cab := &challengeAllocBlobberPassResult{
		verifyTicketsResult:      result,
		alloc:                    alloc,
		allocChallenges:          allocChallenges,
		challenge:                challenge,
		blobAlloc:                blobAlloc,
		latestCompletedChallTime: latestCompletedChallTime,
	}

	if !(result.pass) {
		return sc.challengeFailed(balances, conf.NumValidatorsRewarded, cab)
	}

	return sc.challengePassed(balances, t, conf.BlockReward.TriggerPeriod, conf.NumValidatorsRewarded, cab)
}

type verifyTicketsResult struct {
	pass       bool
	threshold  int
	success    int
	validators []string
}

// challengeAllocBlobberPassResult wraps all the data structs for processing a challenge
type challengeAllocBlobberPassResult struct {
	*verifyTicketsResult
	alloc                    *StorageAllocation
	allocChallenges          *AllocationChallenges
	challenge                *StorageChallenge
	blobAlloc                *BlobberAllocation
	latestCompletedChallTime common.Timestamp
}

func verifyChallengeTickets(balances cstate.StateContextI,
	challenge *StorageChallenge,
	cr *ChallengeResponse,
	maxChallengeCompletionRounds int64,
) (*verifyTicketsResult, error) {
	// get unique validation tickets map
	vtsMap := make(map[string]struct{}, len(cr.ValidationTickets))
	for _, vt := range cr.ValidationTickets {
		if vt == nil {
			return nil, errors.New("found nil validation tickets")
		}

		if _, ok := challenge.ValidatorIDMap[vt.ValidatorID]; !ok {
			return nil, errors.New("found invalid validator id in validation ticket")
		}

		_, ok := vtsMap[vt.ValidatorID]
		if ok {
			return nil, errors.New("found duplicate validation tickets")
		}
		vtsMap[vt.ValidatorID] = struct{}{}
	}

	tksNum := len(cr.ValidationTickets)
	threshold := challenge.TotalValidators / 2
	if tksNum < threshold {
		return nil, fmt.Errorf("validation tickets less than threshold: %d, tickets: %d", threshold, tksNum)
	}

	var (
		success, failure int
		validators       []string // validators for rewards
	)

	for _, vt := range cr.ValidationTickets {
		if err := vt.Validate(challenge.ID, challenge.BlobberID); err != nil {
			return nil, fmt.Errorf("invalid validation ticket: %v", err)
		}

		if ok, err := vt.VerifySign(balances); !ok || err != nil {
			return nil, fmt.Errorf("invalid validation ticket: %v", err)
		}

		validators = append(validators, vt.ValidatorID)
		if !vt.Result {
			failure++
			continue
		}
		success++
	}

	var (
		pass = success > threshold
	)

	if !pass {
		logging.Logger.Info("JAYASH challenge failed",
			zap.Int("success", success),
			zap.Int("threshold", threshold),
			zap.Bool("pass", pass),
			zap.Int64("round_created_at", challenge.RoundCreatedAt),
			zap.Int64("max_challenge_completion_rounds", maxChallengeCompletionRounds),
			zap.Int64("current_round", balances.GetBlock().Round),
		)
	}

	return &verifyTicketsResult{
		pass:       pass,
		threshold:  threshold,
		success:    success,
		validators: validators,
	}, nil
}

func (sc *StorageSmartContract) challengePassed(
	balances cstate.StateContextI,
	t *transaction.Transaction,
	triggerPeriod int64,
	validatorsRewarded int,
	cab *challengeAllocBlobberPassResult,
) (string, error) {
	ongoingParts, err := getOngoingPassedBlobberRewardsPartitions(balances, triggerPeriod)
	if err != nil {
		return "", common.NewError("verify_challenge",
			"cannot get ongoing partition: "+err.Error())
	}

	blobber, err := sc.getBlobber(t.ClientID, balances)
	if err != nil {
		return "", common.NewError("verify_challenge",
			"can't get blobber"+err.Error())
	}

	rewardRound := GetCurrentRewardRound(balances.GetBlock().Round, triggerPeriod)
	// this expiry of blobber needs to be corrected once logic is finalized

	if blobber.RewardRound.StartRound != rewardRound {

		var dataRead float64 = 0
		if blobber.LastRewardDataReadRound >= rewardRound {
			dataRead = blobber.DataReadLastRewardRound
		}

		err := ongoingParts.Add(
			balances,
			&BlobberRewardNode{
				ID:                blobber.ID,
				SuccessChallenges: 0,
				WritePrice:        blobber.Terms.WritePrice,
				ReadPrice:         blobber.Terms.ReadPrice,
				TotalData:         sizeInGB(blobber.SavedData),
				DataRead:          dataRead,
			})
		if err != nil {
			return "", common.NewError("verify_challenge",
				"can't add to ongoing partition list "+err.Error())
		}

		blobber.RewardRound = RewardRound{
			StartRound: rewardRound,
			Timestamp:  t.CreationDate,
		}

		_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
		if err != nil {
			return "", common.NewError("verify_challenge",
				"error inserting blobber to chain"+err.Error())
		}
	}

	var brStats BlobberRewardNode
	if err := ongoingParts.Get(balances, blobber.ID, &brStats); err != nil {
		return "", common.NewError("verify_challenge",
			"can't get blobber reward from partition list: "+err.Error())
	}

	brStats.SuccessChallenges++

	if !sc.completeChallenge(cab) {
		return "", common.NewError("challenge_out_of_order",
			"First challenge on the list is not same as the one"+
				" attempted to redeem")
	}
	cab.alloc.Stats.LastestClosedChallengeTxn = cab.challenge.ID
	cab.alloc.Stats.SuccessChallenges++
	cab.alloc.Stats.OpenChallenges--

	cab.blobAlloc.Stats.LastestClosedChallengeTxn = cab.challenge.ID
	cab.blobAlloc.Stats.SuccessChallenges++
	cab.blobAlloc.Stats.OpenChallenges--

	if err := cab.challenge.Save(balances, sc.ID); err != nil {
		return "", common.NewError("verify_challenge_error", err.Error())
	}

	err = emitUpdateChallenge(cab.challenge, true, ChallengeResponded, balances, cab.alloc.Stats, cab.blobAlloc.Stats)
	if err != nil {
		return "", err
	}

	err = ongoingParts.UpdateItem(balances, &brStats)
	if err != nil {
		return "", common.NewErrorf("verify_challenge",
			"error updating blobber reward item: %v", err)
	}

	err = ongoingParts.Save(balances)
	if err != nil {
		return "", common.NewErrorf("verify_challenge",
			"error saving ongoing blobber reward partition: %v", err)
	}

	if err := cab.allocChallenges.Save(balances, sc.ID); err != nil {
		return "", common.NewError("verify_challenge", err.Error())
	}

	var partial = 1.0
	if cab.success < cab.threshold {
		partial = float64(cab.success) / float64(cab.threshold)
	}
	validators := getRandomSubSlice(cab.validators, validatorsRewarded, balances.GetBlock().GetRoundRandomSeed())

	err = sc.blobberReward(
		cab.alloc, cab.latestCompletedChallTime, cab.blobAlloc,
		validators,
		partial,
		balances,
		cab.challenge.AllocationID,
	)
	if err != nil {
		return "", common.NewError("challenge_reward_error", err.Error())
	}

	// save allocation object
	if err := cab.alloc.save(balances, sc.ID); err != nil {
		return "", common.NewError("challenge_reward_error", err.Error())
	}

	if cab.success < cab.threshold {
		return "challenge passed partially by blobber", nil
	}

	return "challenge passed by blobber", nil
}

func (sc *StorageSmartContract) challengeFailed(
	balances cstate.StateContextI,
	validatorsRewarded int,
	cab *challengeAllocBlobberPassResult,
) (string, error) {
	if !sc.completeChallenge(cab) {
		return "", common.NewError("challenge_out_of_order",
			"First challenge on the list is not same as the one"+
				" attempted to redeem")
	}
	cab.alloc.Stats.LastestClosedChallengeTxn = cab.challenge.ID
	cab.alloc.Stats.FailedChallenges++
	cab.alloc.Stats.OpenChallenges--

	cab.blobAlloc.Stats.LastestClosedChallengeTxn = cab.challenge.ID
	cab.blobAlloc.Stats.FailedChallenges++
	cab.blobAlloc.Stats.OpenChallenges--

	err := emitUpdateChallenge(cab.challenge, false, ChallengeRespondedInvalid, balances, cab.alloc.Stats, cab.blobAlloc.Stats)
	if err != nil {
		return "", err
	}

	if err := cab.allocChallenges.Save(balances, sc.ID); err != nil {
		return "", common.NewError("challenge_penalty_error", err.Error())
	}

	logging.Logger.Info("Challenge failed", zap.String("challenge", cab.challenge.ID))
	validators := getRandomSubSlice(cab.validators, validatorsRewarded, balances.GetBlock().GetRoundRandomSeed())
	err = sc.blobberPenalty(
		cab.alloc, cab.latestCompletedChallTime, cab.blobAlloc, validators,
		balances,
		cab.challenge.AllocationID,
	)
	if err != nil {
		return "", common.NewError("challenge_penalty_error", err.Error())
	}

	// save allocation object
	_, err = balances.InsertTrieNode(cab.alloc.GetKey(sc.ID), cab.alloc)
	if err != nil {
		return "", common.NewError("challenge_reward_error", err.Error())
	}

	return "Challenge Failed by Blobber", nil
}

func getRandomSubSlice(slice []string, size int, seed int64) []string {
	if size > len(slice) {
		size = len(slice)
	}
	sort.Strings(slice)
	indices := rand.New(rand.NewSource(seed)).Perm(len(slice))
	subSlice := make([]string, 0, size)
	for i := 0; i < size; i++ {
		subSlice = append(subSlice, slice[indices[i]])
	}

	return subSlice
}

func (sc *StorageSmartContract) getAllocationForChallenge(
	_ *transaction.Transaction,
	allocID string,
	blobberID string,
	balances cstate.StateContextI) (alloc *StorageAllocation, err error) {

	alloc, err = sc.getAllocation(allocID, balances)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		logging.Logger.Error("client state has invalid allocations",
			zap.String("selected_allocation", allocID),
			zap.Error(err))
		return nil, fmt.Errorf("could not find allocation to challenge: %v", err)
	default:
		return nil, common.NewErrorf("adding_challenge_error",
			"unexpected error getting allocation: %v", err)
	}

	if alloc.Stats == nil {
		return nil, common.NewError("adding_challenge_error",
			"found empty allocation stats")
	}

	// we check that this allocation do have write-commits and can be challenged.
	// We can't check only allocation to be written, because blobbers can commit in different order,
	// so we check particular blobber's allocation to be written
	if alloc.Stats.UsedSize > 0 && alloc.BlobberAllocsMap[blobberID].AllocationRoot != "" {
		return alloc, nil // found
	}
	return nil, nil
}

type challengeOutput struct {
	alloc            *StorageAllocation
	storageChallenge *StorageChallenge
	allocChallenges  *AllocationChallenges
	challInfo        *StorageChallengeResponse
}

type challengeBlobberSelection int

// randomWeightSelection select n blobbers from blobberChallenge partition and then select a blobber with the highest weight
// randomSelection select a blobber randomly from partition
const (
	randomWeightSelection challengeBlobberSelection = iota
	randomSelection
)

// selectBlobberForChallenge select blobber for challenge in random manner
func selectBlobberForChallenge(selection challengeBlobberSelection, challengeBlobbersPartition *partitions.Partitions,
	r *rand.Rand, balances cstate.StateContextI) (string, error) {

	var challengeBlobbers []ChallengeReadyBlobber
	err := challengeBlobbersPartition.GetRandomItems(balances, r, &challengeBlobbers)
	if err != nil {
		return "", fmt.Errorf("error getting random slice from blobber challenge partition: %v", err)
	}

	switch selection {
	case randomWeightSelection:
		const maxBlobbersSelect = 5

		var challengeBlobber ChallengeReadyBlobber
		var maxWeight uint64

		var blobbersSelected = make([]ChallengeReadyBlobber, 0, maxBlobbersSelect)
		if len(challengeBlobbers) <= maxBlobbersSelect {
			blobbersSelected = challengeBlobbers
		} else {
			for i := 0; i < maxBlobbersSelect; i++ {
				randomIndex := r.Intn(len(challengeBlobbers))
				blobbersSelected = append(blobbersSelected, challengeBlobbers[randomIndex])
			}
		}

		for _, bc := range blobbersSelected {
			if bc.Weight > maxWeight {
				maxWeight = bc.Weight
				challengeBlobber = bc
			}
		}

		return challengeBlobber.BlobberID, nil
	case randomSelection:
		randomIndex := r.Intn(len(challengeBlobbers))
		return challengeBlobbers[randomIndex].BlobberID, nil
	default:
		return "", errors.New("invalid blobber selection pattern")
	}
}

func (sc *StorageSmartContract) populateGenerateChallenge(
	challengeBlobbersPartition *partitions.Partitions,
	seed int64,
	validators *partitions.Partitions,
	txn *transaction.Transaction,
	challengeID string,
	balances cstate.StateContextI,
	needValidNum int,
	conf *Config,
) (*challengeOutput, error) {
	r := rand.New(rand.NewSource(seed))
	blobberSelection := challengeBlobberSelection(1) // challengeBlobberSelection(r.Intn(2))
	blobberID, err := selectBlobberForChallenge(blobberSelection, challengeBlobbersPartition, r, balances)
	if err != nil {
		return nil, common.NewError("add_challenge", err.Error())
	}

	if blobberID == "" {
		return nil, common.NewError("add_challenges", "empty blobber id")
	}

	logging.Logger.Debug("generate_challenges", zap.String("blobber id", blobberID))

	// get blobber allocations partitions
	blobberAllocParts, err := partitionsBlobberAllocations(blobberID, balances)
	if err != nil {
		return nil, common.NewErrorf("generate_challenge",
			"error getting blobber_challenge_allocation list: %v", err)
	}

	// get random allocations from the partitions
	var randBlobberAllocs []BlobberAllocationNode
	if err := blobberAllocParts.GetRandomItems(balances, r, &randBlobberAllocs); err != nil {
		return nil, common.NewErrorf("generate_challenge",
			"error getting random slice from blobber challenge allocation partition: %v", err)
	}

	var findValidAllocRetries = 5 // avoid retry for debugging
	var (
		alloc                       *StorageAllocation
		blobberAllocPartitionLength = len(randBlobberAllocs)
		foundAllocation             bool
		randPerm                    = r.Perm(blobberAllocPartitionLength)
	)

	if findValidAllocRetries > blobberAllocPartitionLength {
		findValidAllocRetries = blobberAllocPartitionLength
	}

	if findValidAllocRetries == 0 {
		logging.Logger.Debug("empty blobber")
	}

	for i := 0; i < findValidAllocRetries; i++ {
		// get a random allocation
		allocID := randBlobberAllocs[randPerm[i%blobberAllocPartitionLength]].ID

		// get the storage allocation from MPT
		alloc, err = sc.getAllocationForChallenge(txn, allocID, blobberID, balances)
		if err != nil {
			return nil, err
		}

		if alloc == nil {
			logging.Logger.Debug("allocation not found for blobber", zap.String("blobber_id", blobberID),
				zap.String("alloc_id", allocID))
			continue
		}

		if alloc.Finalized {
			if err := removeAllocationFromBlobberPartitions(balances, blobberID, allocID); err != nil {
				return nil, err
			}
			continue
		}

		if alloc.Expiration >= txn.CreationDate {
			foundAllocation = true
			break
		}
		logging.Logger.Debug("allocation expiry is wrong", zap.String("blobber_id", blobberID),
			zap.String("alloc_id", allocID))

		err = alloc.save(balances, sc.ID)
		if err != nil {
			return nil, common.NewErrorf("populate_challenge",
				"error saving expired allocation: %v", err)
		}
	}

	if err := blobberAllocParts.Save(balances); err != nil {
		return nil, common.NewErrorf("populate_challenge",
			"error saving blobber allocation partitions: %v", err)
	}

	if !foundAllocation {
		logging.Logger.Error("populate_generate_challenge: couldn't find appropriate allocation for a blobber",
			zap.String("blobberId", blobberID))
		return nil, nil
	}

	allocBlobber, ok := alloc.BlobberAllocsMap[blobberID]
	if !ok {
		return nil, errors.New("invalid blobber for allocation")
	}

	var randValidators []ValidationPartitionNode
	if err := validators.GetRandomItems(balances, r, &randValidators); err != nil {
		return nil, common.NewError("add_challenge",
			"error getting validators random slice: "+err.Error())
	}

	if len(randValidators) < needValidNum {
		return nil, errors.New("validators number does not meet minimum challenge requirement")
	}

	var (
		selectedValidators  = make([]*ValidationNode, 0, needValidNum)
		perm                = r.Perm(len(randValidators))
		remainingValidators = len(randValidators)
	)

	now := txn.CreationDate
	filterValidator := filterHealthyValidators(now)

	for i := 0; i < len(randValidators) && len(selectedValidators) < needValidNum; i++ {
		if remainingValidators < needValidNum {
			return nil, errors.New("validators number does not meet minimum challenge requirement after filtering")
		}
		randValidator := randValidators[perm[i]]
		if randValidator.Id == blobberID {
			continue
		}
		validator, err := getValidator(randValidator.Id, balances)
		if err != nil {
			if cstate.ErrInvalidState(err) {
				return nil, common.NewError("add_challenge",
					err.Error())
			}
			continue
		}

		kick, err := filterValidator(validator)
		if err != nil {
			return nil, common.NewError("add_challenge", "failed to filter validator: "+
				err.Error())
		}
		if kick {
			remainingValidators--
			continue
		}

		sp, err := sc.getStakePool(spenum.Validator, validator.ID, balances)
		if err != nil {
			return nil, fmt.Errorf("can't get validator %s stake pool: %v", randValidator.Id, err)
		}
		stake, err := sp.stake()
		if err != nil {
			return nil, err
		}
		if stake < conf.MinStake {
			remainingValidators--
			continue
		}

		selectedValidators = append(selectedValidators,
			&ValidationNode{
				Provider: provider.Provider{
					ID:           randValidator.Id,
					ProviderType: spenum.Validator,
				},
				BaseURL: randValidator.Url,
			})

	}

	if len(selectedValidators) < needValidNum {
		return nil, errors.New("validators number does not meet minimum challenge requirement after filtering")
	}

	validatorIDs := make([]string, len(selectedValidators))
	for i := range selectedValidators {
		validatorIDs[i] = selectedValidators[i].ID
	}

	var storageChallenge = new(StorageChallenge)
	storageChallenge.ID = challengeID
	storageChallenge.TotalValidators = len(selectedValidators)
	storageChallenge.ValidatorIDs = validatorIDs
	storageChallenge.BlobberID = blobberID
	storageChallenge.AllocationID = alloc.ID
	storageChallenge.Created = txn.CreationDate
	storageChallenge.RoundCreatedAt = balances.GetBlock().Round

	challInfo := &StorageChallengeResponse{
		StorageChallenge: storageChallenge,
		Validators:       selectedValidators,
		Seed:             seed,
		AllocationRoot:   allocBlobber.AllocationRoot,
		Timestamp:        allocBlobber.LastWriteMarker.Timestamp,
	}

	allocChallenges, err := sc.getAllocationChallenges(alloc.ID, balances)
	if err != nil {
		if err == util.ErrValueNotPresent {
			allocChallenges = &AllocationChallenges{}
			allocChallenges.AllocationID = alloc.ID
		} else {
			return nil, common.NewError("add_challenge",
				"error fetching allocation challenge: "+err.Error())
		}
	}

	return &challengeOutput{
		alloc:            alloc,
		storageChallenge: storageChallenge,
		allocChallenges:  allocChallenges,
		challInfo:        challInfo,
	}, nil
}

type GenerateChallengeInput struct {
	Round int64 `json:"round,omitempty"`
}

func (sc *StorageSmartContract) generateChallenge(
	t *transaction.Transaction,
	b *block.Block,
	input []byte,
	conf *Config,
	balances cstate.StateContextI,
) (err error) {
	inputRound := GenerateChallengeInput{}
	if err := json.Unmarshal(input, &inputRound); err != nil {
		return err
	}

	if inputRound.Round != b.Round {
		return fmt.Errorf("bad round, block %v but input %v", b.Round, inputRound.Round)
	}

	validators, err := getValidatorsList(balances)
	if err != nil {
		return common.NewErrorf("generate_challenge",
			"error getting the validators list: %v", err)
	}

	// Check if the length of the list of validators is higher than the required number of validators
	needValidNum := conf.ValidatorsPerChallenge
	currentValidatorsCount, err := validators.Size(balances)
	if err != nil {
		return fmt.Errorf("can't get validators partition size: %v", err.Error())
	}

	if currentValidatorsCount < needValidNum {
		err := errors.New("validators number does not meet minimum challenge requirement")
		logging.Logger.Error("generate_challenge", zap.Error(err),
			zap.Int("validator num", currentValidatorsCount),
			zap.Int("minimum required", needValidNum))
		return common.NewError("generate_challenge",
			"validators number does not meet minimum challenge requirement")
	}

	challengeReadyParts, err := partitionsChallengeReadyBlobbers(balances)
	if err != nil {
		return common.NewErrorf("generate_challenge",
			"error getting the blobber challenge list: %v", err)
	}

	bcNum, err := challengeReadyParts.Size(balances)
	if err != nil {
		return common.NewErrorf("generate_challenge", "error getting blobber challenge size: %v", err)
	}

	if bcNum == 0 {
		logging.Logger.Info("skipping generate challenge: empty blobber challenge partition")
		return nil
	}

	hashSeed := encryption.Hash(t.Hash + b.PrevHash)
	// the "1" was the index when generating multiple challenges.
	// keep it in case we need to generate more than 1 challenge at once.
	challengeID := encryption.Hash(hashSeed + "1")

	seedSource, err := strconv.ParseUint(challengeID[0:16], 16, 64)
	if err != nil {
		return common.NewErrorf("generate_challenge",
			"Error in creating challenge seed: %v", err)
	}

	result, err := sc.populateGenerateChallenge(
		challengeReadyParts,
		int64(seedSource),
		validators,
		t,
		challengeID,
		balances,
		needValidNum,
		conf,
	)
	if err != nil {
		return common.NewErrorf("generate_challenge", err.Error())
	}

	if result == nil {
		logging.Logger.Error("received empty data for challenge generation. Skipping challenge generation")
		return nil
	}

	err = sc.addChallenge(result.alloc,
		result.storageChallenge,
		result.allocChallenges,
		result.challInfo,
		conf,
		balances)
	if err != nil {
		return common.NewErrorf("adding_challenge_error",
			"Error in adding challenge: %v", err)
	}

	afterAddChallenge(result.challInfo.ID, result.challInfo.ValidatorIDs)

	return nil
}

func (sc *StorageSmartContract) addChallenge(alloc *StorageAllocation,
	challenge *StorageChallenge,
	allocChallenges *AllocationChallenges,
	challInfo *StorageChallengeResponse,
	conf *Config,
	balances cstate.StateContextI,
) error {
	if challenge.BlobberID == "" {
		return common.NewError("add_challenge",
			"no blobber to add challenge to")
	}

	blobAlloc, ok := alloc.BlobberAllocsMap[challenge.BlobberID]
	if !ok {
		return common.NewError("add_challenge",
			"no blobber Allocation to add challenge to")
	}

	// remove expired challenges
	expiredIDsMap, err := alloc.removeExpiredChallenges(allocChallenges, conf.MaxChallengeCompletionRounds, balances)
	if err != nil {
		return common.NewErrorf("add_challenge", "remove expired challenges: %v", err)
	}

	var expChalIDs []string
	for challengeID := range expiredIDsMap {
		expChalIDs = append(expChalIDs, challengeID)
	}
	sort.Strings(expChalIDs)

	// maps blobberID to count of its expiredIDs.
	expiredCountMap := make(map[string]int)

	// TODO: maybe delete them periodically later instead of remove immediately
	for _, challengeID := range expChalIDs {
		blobberID := expiredIDsMap[challengeID]
		_, err := balances.DeleteTrieNode(storageChallengeKey(sc.ID, challengeID))
		if err != nil {
			return common.NewErrorf("add_challenge", "could not delete challenge node: %v", err)
		}

		if _, ok := expiredCountMap[blobberID]; !ok {
			expiredCountMap[blobberID] = 0
		}
		expiredCountMap[blobberID]++
	}

	// add the generated challenge to the open challenges list in the allocation
	if !allocChallenges.addChallenge(challenge) {
		return common.NewError("add_challenge", "challenge already exist in allocation")
	}

	// Save the allocation challenges to MPT
	if err := allocChallenges.Save(balances, sc.ID); err != nil {
		return common.NewErrorf("add_challenge",
			"error storing alloc challenge: %v", err)
	}

	// Save challenge to MPT
	if err := challenge.Save(balances, sc.ID); err != nil {
		return common.NewErrorf("add_challenge",
			"error storing challenge: %v", err)
	}

	alloc.Stats.OpenChallenges++
	alloc.Stats.TotalChallenges++
	blobAlloc.Stats.OpenChallenges++
	blobAlloc.Stats.TotalChallenges++

	if err := alloc.save(balances, sc.ID); err != nil {
		return common.NewErrorf("add_challenge",
			"error storing allocation: %v", err)
	}

	// balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationChallenges, alloc.ID, alloc.buildUpdateChallengeStat())

	beforeEmitAddChallenge(challInfo)
	return emitAddChallenge(challInfo, expiredCountMap, len(expiredIDsMap), balances, alloc.Stats, blobAlloc.Stats)
}

func isChallengeExpired(currentRound, roundCreatedAt, maxChallengeCompletionRounds int64) bool {
	return roundCreatedAt+maxChallengeCompletionRounds <= currentRound
}
