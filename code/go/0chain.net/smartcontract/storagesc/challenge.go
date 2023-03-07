package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
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

const blobberAllocationPartitionSize = 100

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
func (sc *StorageSmartContract) blobberReward(alloc *StorageAllocation, latestCompletedChallTime common.Timestamp,
	blobAlloc *BlobberAllocation, validators []string, partial float64,
	balances cstate.StateContextI) error {
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	// time of this challenge
	challengeCompletedTime := blobAlloc.LatestCompletedChallenge.Created
	if challengeCompletedTime > alloc.Expiration+toSeconds(getMaxChallengeCompletionTime()) {
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

	before, err := sp.stake()
	if err != nil {
		return err
	}

	err = sp.DistributeRewards(blobberReward, blobAlloc.BlobberID, spenum.Blobber, spenum.ChallengePassReward, balances)
	if err != nil {
		return fmt.Errorf("can't move tokens to blobber: %v", err)
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

	err = cp.moveToValidators(sc.ID, validatorsReward, validators, vsps, balances)
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

	if blobAlloc.Terms.WritePrice > 0 {
		stake, err := sp.stake()
		if err != nil {
			return err
		}
		balances.EmitEvent(event.TypeStats, event.TagAllocBlobberValueChange, blobAlloc.BlobberID, event.AllocationBlobberValueChanged{
			FieldType:    event.Staked,
			AllocationId: "",
			BlobberId:    blobAlloc.BlobberID,
			Delta:        int64((stake - before) / blobAlloc.Terms.WritePrice),
		})
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

		// TODO: add code below back after validators staking are supported
		//staked, err := sp.stake()
		//if err != nil {
		//	return fmt.Errorf("can't get stake: %v", err)
		//}
		//vid := validators[i]
		//tag, data := event.NewUpdateBlobberTotalStakeEvent(vid, staked)
		//balances.EmitEvent(event.TypeStats, tag, vid, data)
	}
	return
}

// move tokens from challenge pool back to write pool
func (sc *StorageSmartContract) blobberPenalty(alloc *StorageAllocation, prev common.Timestamp,
	blobAlloc *BlobberAllocation, validators []string, balances cstate.StateContextI) (err error) {
	var conf *Config
	if conf, err = sc.getConfig(balances, true); err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	// time of this challenge
	challengeCompleteTime := blobAlloc.LatestCompletedChallenge.Created
	if challengeCompleteTime > alloc.Expiration+toSeconds(getMaxChallengeCompletionTime()) {
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
	err = cp.moveToValidators(sc.ID, validatorsReward, validators, vSPs, balances)
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
		move, err = sp.slash(blobAlloc.BlobberID, blobAlloc.Offer(), slash, balances)
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
		if blobAlloc.Terms.WritePrice > 0 {
			balances.EmitEvent(event.TypeStats, event.TagAllocBlobberValueChange, blobAlloc.BlobberID, event.AllocationBlobberValueChanged{
				FieldType:    event.Staked,
				AllocationId: "",
				BlobberId:    blobAlloc.BlobberID,
				Delta:        -int64(move / blobAlloc.Terms.WritePrice),
			})
		}
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

	if challenge.BlobberID != t.ClientID {
		return "", errors.New("challenge blobber id does not match")
	}

	logging.Logger.Info("time_taken: receive challenge response",
		zap.String("challenge_id", challenge.ID),
		zap.Duration("delay", time.Since(common.ToTime(challenge.Created))))

	result, err := verifyChallengeTickets(balances, t, challenge, &challResp)
	if err != nil {
		return "", common.NewError(errCode, err.Error())
	}

	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return "", common.NewErrorf(errCode,
			"cannot get smart contract configurations: %v", err)
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
		if lcc != nil && challResp.ID == lcc.ID && lcc.Responded {
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

	challenge.Responded = true
	cab := &challengeAllocBlobberPassResult{
		verifyTicketsResult:      result,
		alloc:                    alloc,
		allocChallenges:          allocChallenges,
		challenge:                challenge,
		blobAlloc:                blobAlloc,
		latestCompletedChallTime: latestCompletedChallTime,
	}

	if !(result.pass && result.fresh) {
		return sc.challengeFailed(balances, t, cab)
	}

	return sc.challengePassed(balances, t, conf.BlockReward.TriggerPeriod, cab)
}

type verifyTicketsResult struct {
	pass       bool
	fresh      bool
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
	t *transaction.Transaction,
	challenge *StorageChallenge,
	cr *ChallengeResponse) (*verifyTicketsResult, error) {
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
		pass  = success > threshold
		cct   = toSeconds(getMaxChallengeCompletionTime())
		fresh = challenge.Created+cct >= t.CreationDate
	)

	return &verifyTicketsResult{
		pass:       pass,
		fresh:      fresh,
		threshold:  threshold,
		success:    success,
		validators: validators,
	}, nil
}

func (sc *StorageSmartContract) challengePassed(
	balances cstate.StateContextI,
	t *transaction.Transaction,
	triggerPeriod int64,
	cab *challengeAllocBlobberPassResult) (string, error) {
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
	if blobber.RewardRound.StartRound != rewardRound ||
		balances.GetBlock().Round == 0 {

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

	emitUpdateChallenge(cab.challenge, true, balances)

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

	err = sc.blobberReward(cab.alloc, cab.latestCompletedChallTime, cab.blobAlloc, cab.validators, partial, balances)
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
	t *transaction.Transaction,
	cab *challengeAllocBlobberPassResult) (string, error) {
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

	emitUpdateChallenge(cab.challenge, false, balances)

	if err := cab.allocChallenges.Save(balances, sc.ID); err != nil {
		return "", common.NewError("challenge_penalty_error", err.Error())
	}

	logging.Logger.Info("Challenge failed", zap.String("challenge", cab.challenge.ID))

	err := sc.blobberPenalty(cab.alloc, cab.latestCompletedChallTime, cab.blobAlloc,
		cab.validators, balances)
	if err != nil {
		return "", common.NewError("challenge_penalty_error", err.Error())
	}

	// save allocation object
	_, err = balances.InsertTrieNode(cab.alloc.GetKey(sc.ID), cab.alloc)
	if err != nil {
		return "", common.NewError("challenge_reward_error", err.Error())
	}

	//balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, alloc.ID, alloc.buildDbUpdates())
	if cab.pass && !cab.fresh {
		return "late challenge (failed)", nil
	}

	return "Challenge Failed by Blobber", nil
}

func (sc *StorageSmartContract) getAllocationForChallenge(
	t *transaction.Transaction,
	allocID string,
	blobberID string,
	balances cstate.StateContextI) (alloc *StorageAllocation, err error) {

	alloc, err = sc.getAllocation(allocID, balances)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		logging.Logger.Error("client state has invalid allocations",
			zap.String("selected_allocation", allocID))
		return nil, common.NewErrorf("invalid_allocation",
			"client state has invalid allocations")
	default:
		return nil, common.NewErrorf("adding_challenge_error",
			"unexpected error getting allocation: %v", err)
	}

	if alloc.Stats == nil {
		return nil, common.NewError("adding_challenge_error",
			"found empty allocation stats")
	}

	//we check that this allocation do have write-commits and can be challenged.
	//We can't check only allocation to be written, because blobbers can commit in different order,
	//so we check particular blobber's allocation to be written
	if alloc.Stats.NumWrites > 0 && alloc.BlobberAllocsMap[blobberID].AllocationRoot != "" {
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
) (*challengeOutput, error) {
	r := rand.New(rand.NewSource(seed))
	blobberSelection := challengeBlobberSelection(r.Intn(2))
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
		return nil, common.NewErrorf("generate_challenges",
			"error getting blobber_challenge_allocation list: %v", err)
	}

	// get random allocations from the partitions
	var randBlobberAllocs []BlobberAllocationNode
	if err := blobberAllocParts.GetRandomItems(balances, r, &randBlobberAllocs); err != nil {
		return nil, common.NewErrorf("generate_challenges",
			"error getting random slice from blobber challenge allocation partition: %v", err)
	}

	const findValidAllocRetries = 5
	var (
		alloc                       *StorageAllocation
		blobberAllocPartitionLength = len(randBlobberAllocs)
		foundAllocation             bool
	)

	for i := 0; i < findValidAllocRetries; i++ {
		// get a random allocation
		randomIndex := r.Intn(blobberAllocPartitionLength)
		allocID := randBlobberAllocs[randomIndex].ID

		// get the storage allocation from MPT
		alloc, err = sc.getAllocationForChallenge(txn, allocID, blobberID, balances)
		if err != nil {
			return nil, err
		}

		if alloc == nil {
			continue
		}

		if alloc.Expiration >= txn.CreationDate {
			foundAllocation = true
			break
		}

		err = alloc.save(balances, sc.ID)
		if err != nil {
			return nil, common.NewErrorf("populate_challenge",
				"error saving expired allocation: %v", err)
		}
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

	var (
		needValidNum       = minInt(len(randValidators), alloc.DataShards+1)
		selectedValidators = make([]*ValidationNode, 0, needValidNum)
		perm               = r.Perm(len(randValidators))
	)

	for i := 0; i < needValidNum; i++ {
		randValidator := randValidators[perm[i]]
		if randValidator.Id != blobberID {
			selectedValidators = append(selectedValidators,
				&ValidationNode{
					Provider: provider.Provider{
						ID:           randValidator.Id,
						ProviderType: spenum.Validator,
					},
					BaseURL: randValidator.Url,
				})
		}
		if len(selectedValidators) > alloc.DataShards {
			break
		}
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

	challInfo := &StorageChallengeResponse{
		StorageChallenge: storageChallenge,
		Validators:       selectedValidators,
		Seed:             seed,
		AllocationRoot:   allocBlobber.AllocationRoot,
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

func (sc *StorageSmartContract) generateChallenge(t *transaction.Transaction,
	b *block.Block, _ []byte, balances cstate.StateContextI) (err error) {

	var conf *Config
	if conf, err = sc.getConfig(balances, true); err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	validators, err := getValidatorsList(balances)
	if err != nil {
		return common.NewErrorf("generate_challenge",
			"error getting the validators list: %v", err)
	}

	// Check if the length of the list of validators is higher than the lower bound of validators
	minValidators := conf.ValidatorsPerChallenge
	currentValidatorsCount, err := validators.Size(balances)
	if err != nil {
		return fmt.Errorf("can't get validators partition size: %v", err.Error())
	}

	if currentValidatorsCount < minValidators {
		err := errors.New("validators number does not meet minimum challenge requirement")
		logging.Logger.Error("generate_challenge", zap.Error(err),
			zap.Int("validator num", currentValidatorsCount),
			zap.Int("minimum required", minValidators))
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
		balances)
	if err != nil {
		return common.NewErrorf("adding_challenge_error", err.Error())
	}

	if result == nil {
		logging.Logger.Error("received empty data for challenge generation. Skipping challenge generation")
		return nil
	}

	err = sc.addChallenge(result.alloc,
		result.storageChallenge,
		result.allocChallenges,
		result.challInfo,
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
	balances cstate.StateContextI) error {

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
	expiredIDsMap, err := alloc.removeExpiredChallenges(allocChallenges, challenge.Created)
	if err != nil {
		return common.NewErrorf("add_challenge", "remove expired challenges: %v", err)
	}

	// maps blobberID to count of its expiredIDs.
	expiredCountMap := make(map[string]int)

	// TODO: maybe delete them periodically later instead of remove immediately
	for challengeID, blobberID := range expiredIDsMap {
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

	//balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationChallenges, alloc.ID, alloc.buildUpdateChallengeStat())

	beforeEmitAddChallenge(challInfo)

	emitAddChallenge(challInfo, expiredCountMap, len(expiredIDsMap), balances)
	return nil
}

func isChallengeExpired(now, createdAt common.Timestamp, challengeCompletionTime time.Duration) bool {
	return createdAt+common.ToSeconds(challengeCompletionTime) <= now
}
