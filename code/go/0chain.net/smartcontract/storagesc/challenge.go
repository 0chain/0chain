package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

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
//const blobberAllocationPartitionSize = 100

const blobberAllocationPartitionSize = 5

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
	balances cstate.StateContextI, options ...string) error {
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	// time of this challenge
	challengeCompletedTime := blobAlloc.LatestCompletedChallenge.Created
	if challengeCompletedTime > alloc.Expiration+toSeconds(getMaxChallengeCompletionTime()) {
		return errors.New("late challenge response")
	}

	if alloc.Finalized {
		logging.Logger.Info("blobber reward - allocation is finalized",
			zap.String("allocation", alloc.ID),
			zap.Int64("allocation expiry", int64(alloc.Expiration)),
			zap.Int64("challenge time", int64(challengeCompletedTime)))

		return nil
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
	//var cp *challengePool
	//if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
	//	return fmt.Errorf("can't get allocation's challenge pool: %v", err)
	//}
	//cp := alloc.ChallengePool

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

	var challengeID string
	if len(options) > 0 {
		challengeID = options[0]
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
		err = alloc.moveFromChallengePool(back)
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
		balances.EmitEvent(event.TypeStats, event.TagFromChallengePool, alloc.ID, event.ChallengePoolLock{
			Client:       alloc.Owner,
			AllocationId: alloc.ID,
			Amount:       coin,
		})

	}

	var sp *stakePool
	if sp, err = sc.getStakePool(spenum.Blobber, blobAlloc.BlobberID, balances); err != nil {
		return fmt.Errorf("can't get stake pool: %v", err)
	}

	err = sc.moveToBlobbers(alloc, blobberReward, blobAlloc.BlobberID, sp, balances, challengeID)
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

	err = sc.moveToValidators(alloc, validatorsReward, validators, vsps, balances, challengeID)
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

	//if err = cp.save(sc.ID, alloc, balances); err != nil {
	//	return fmt.Errorf("can't save allocation's challenge pool: %v", err)
	//}

	if err = alloc.saveUpdatedStakes(balances); err != nil {
		return fmt.Errorf("can't save allocation: %v", err)
	}

	return nil
}

func (sc *StorageSmartContract) moveToBlobbers(
	alloc *StorageAllocation,
	reward currency.Coin,
	blobberId datastore.Key,
	sp *stakePool,
	balances cstate.StateContextI,
	options ...string,
) error {

	if reward == 0 {
		return nil // nothing to move, or nothing to move to
	}

	if alloc.ChallengePool < reward {
		return fmt.Errorf("not enough tokens in challenge pool: %v < %v", alloc.ChallengePool, reward)
	}

	err := sp.DistributeRewards(reward, blobberId, spenum.Blobber, spenum.ChallengePassReward, balances, options...)
	if err != nil {
		return fmt.Errorf("can't move tokens to blobber: %v", err)
	}

	alloc.ChallengePool -= reward
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
	blobAlloc *BlobberAllocation, validators []string, balances cstate.StateContextI, options ...string) (err error) {
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
	//var cp *challengePool
	//if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
	//	return fmt.Errorf("can't get allocation's challenge pool: %v", err)
	//}
	//cp := alloc.ChallengePool

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

	var challengeID string

	if len(options) > 0 {
		challengeID = options[0]
	}

	// validators reward
	err = sc.moveToValidators(alloc, validatorsReward, validators, vSPs, balances, challengeID)
	if err != nil {
		return fmt.Errorf("rewarding validators: %v", err)
	}

	// Save validators' stake pools
	if err = sc.saveStakePools(validators, vSPs, balances); err != nil {
		return err
	}

	err = alloc.moveFromChallengePool(move)
	coin, err := move.Int64()
	if err != nil {
		return fmt.Errorf("moving challenge pool rest back to write pool: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagFromChallengePool, alloc.ID, event.ChallengePoolLock{
		Client:       alloc.Owner,
		AllocationId: alloc.ID,
		Amount:       coin,
	})

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

		bTerms, ok := alloc.getTerms(blobAlloc.BlobberID)
		if !ok {
			return fmt.Errorf("can't get blobber's terms")
		}

		var move currency.Coin
		move, err = sp.slash(blobAlloc.BlobberID, getOffer(alloc.BSize, bTerms), slash, balances)
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

	//if err = cp.save(sc.ID, alloc, balances); err != nil {
	//	return fmt.Errorf("can't Save allocation's challenge pool: %v", err)
	//}

	return
}

func (sc *StorageSmartContract) moveToValidators(
	alloc *StorageAllocation,
	reward currency.Coin,
	validators []datastore.Key,
	vSPs []*stakePool,
	balances cstate.StateContextI,
	options ...string,
) error {
	if len(validators) == 0 || reward == 0 {
		return nil // nothing to move, or nothing to move to
	}

	if alloc.ChallengePool < reward {
		return fmt.Errorf("not enough tokens in challenge pool: %v < %v", alloc.ChallengePool, reward)
	}

	oneReward, bal, err := currency.DistributeCoin(reward, int64(len(validators)))
	if err != nil {
		return err
	}

	for i, sp := range vSPs {
		err := sp.DistributeRewards(oneReward, validators[i], spenum.Validator, spenum.ValidationReward, balances, options...)
		if err != nil {
			return fmt.Errorf("moving to validator %s: %v",
				validators[i], err)
		}
	}
	if bal > 0 {
		for i := 0; i < int(bal); i++ {
			err := vSPs[i].DistributeRewards(1, validators[i], spenum.Validator, spenum.ValidationReward, balances, options...)
			if err != nil {
				return fmt.Errorf("moving to validator %s: %v",
					validators[i], err)
			}
		}
	}

	alloc.ChallengePool -= reward

	moveToValidators, err := currency.AddCoin(alloc.MovedToValidators, reward)
	if err != nil {
		return err
	}
	alloc.MovedToValidators = moveToValidators

	return nil
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
	if challenge.Responded != 0 {
		return "", common.NewError(errCode, "challenge already processed")
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
	if !allocChallenges.find(challResp.ID) {
		// TODO: remove this challenge already redeemed response. This response will be returned only when the
		// challenge is the last completed challenge, which means if we have more challenges completed after it, we
		// will see different result, even the challenge's state is the same as 'it has been redeemed'.
		if lcc != nil && challResp.ID == lcc.ID && lcc.Responded == 1 {
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

	challenge.Responded = 1
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
	threshold := len(challenge.ValidatorIDs) / 2
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

	if blobber.RewardRound.StartRound != rewardRound {

		var dataRead float64 = 0
		if blobber.LastRewardDataReadRound >= rewardRound {
			dataRead = blobber.DataReadLastRewardRound
		}

		bil, err := getBlobbersInfoList(balances)
		if err != nil {
			return "", common.NewError("verify_challenge", err.Error())
		}

		err = ongoingParts.Add(
			balances,
			&BlobberRewardNode{
				ID:                blobber.ID,
				SuccessChallenges: 0,
				WritePrice:        blobber.Terms.WritePrice,
				ReadPrice:         blobber.Terms.ReadPrice,
				TotalData:         sizeInGB(bil[blobber.Index].SavedData),
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

	emitUpdateChallenge(cab.challenge, true, balances, cab.alloc.Stats, cab.blobAlloc.Stats)

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

	err = sc.blobberReward(cab.alloc, cab.latestCompletedChallTime, cab.blobAlloc, cab.validators, partial, balances, cab.challenge.ID)
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

	emitUpdateChallenge(cab.challenge, false, balances, cab.alloc.Stats, cab.blobAlloc.Stats)

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

	//we check that this allocation do have write-commits and can be challenged.
	//We can't check only allocation to be written, because blobbers can commit in different order,
	//so we check particular blobber's allocation to be written
	if alloc.Stats.NumWrites > 0 && alloc.BlobberAllocsMap[blobberID].AllocationRoot != "" {
		return alloc, nil // found
	}
	return nil, nil
}

type challengeInfo struct {
	*StorageChallenge
	//Validators     []*ValidatorHealthCheck
	Seed           int64
	AllocationRoot string
	Timestamp      common.Timestamp
}

type challengeOutput struct {
	alloc            *allocBlobbers
	storageChallenge *StorageChallenge
	allocChallenges  *AllocationChallenges
	challInfo        *challengeInfo
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

type blobberAllocRootWM struct {
	blobberID       string
	allocRoot       string
	lastWMTimestamp common.Timestamp
}

type challengeAllocBlobber struct {
	alloc                *allocBlobbers
	allocChallenges      *AllocationChallenges
	allocChallengesStats *AllocationChallengeStats
	allocBlobber         *blobberAllocRootWM
	blobberIndex         int8
}

func (sc *StorageSmartContract) selectAllocBlobberForChallenge(
	allocParts *partitions.Partitions,
	r *rand.Rand,
	balances cstate.StateContextI) (*challengeAllocBlobber, error) {
	var allocs []ChallengeReadyAllocNode
	err := allocParts.GetRandomItems(balances, r, &allocs)
	if err != nil {
		return nil, fmt.Errorf("error getting random slice from blobber challenge partition: %v", err)
	}

	randomIndex := r.Intn(len(allocs))
	allocID := allocs[randomIndex].AllocID

	cr := concurrentReader{}
	var alloc *allocBlobbers
	var acs *AllocationChallengeStats
	cr.add(func() error {
		var err error
		alloc, err = getAllocationBlobbers(balances, allocID)
		if err != nil {
			return fmt.Errorf("could not get allocation: %v", err)
		}
		return nil
	})
	cr.add(func() error {
		var err error
		acs, err = getAllocationChallengeStats(balances, allocID)
		if err != nil {
			return fmt.Errorf("could not get allocation stats: %v", err)
		}
		return nil
	})

	var allocChallenges *AllocationChallenges
	cr.add(func() error {
		var err error
		allocChallenges, err = sc.getAllocationChallenges(allocID, balances)
		if err != nil {
			if err == util.ErrValueNotPresent {
				allocChallenges = &AllocationChallenges{}
				allocChallenges.AllocationID = allocID
			} else {
				return common.NewError("add_challenge",
					"error fetching allocation challenge: "+err.Error())
			}
		}
		return nil
	})

	if err := cr.do(); err != nil {
		return nil, err
	}

	// filter out all blobbers that have data written
	//blobbers := make([]string, 0, len(alloc.Blobbers))
	bIdxs := make([]int, 0, len(alloc.Blobbers))
	for i, ba := range acs.GetBlobbersStats() {
		if ba.NumWrites > 0 {
			//blobbers = append(blobbers, alloc.Blobbers[i].BlobberID)
			bIdxs = append(bIdxs, i)
		}
	}

	if len(bIdxs) == 0 {
		// means no available blobbers for challenging
		return nil, nil
	}

	idx := bIdxs[r.Intn(len(bIdxs))]
	// return blobber that has data written
	//b := blobbers[r.Intn(len(blobbers))]
	//idx := alloc.blobberIndex(b)
	//if idx < 0 {
	//	return nil, fmt.Errorf("blobber %s not found in allocation %s", b, allocID)
	//}
	alloc.ID = allocID

	return &challengeAllocBlobber{
		alloc:                alloc,
		allocChallenges:      allocChallenges,
		allocChallengesStats: acs,
		allocBlobber: &blobberAllocRootWM{
			blobberID:       alloc.Blobbers[idx].BlobberID,
			allocRoot:       alloc.BlobberAllocs[idx].AllocationRoot,
			lastWMTimestamp: alloc.BlobberAllocs[idx].LastWriteMarker.Timestamp,
		},
		blobberIndex: int8(idx),
	}, nil
}

func (sc *StorageSmartContract) populateGenerateChallenge(
	txn *transaction.Transaction,
	seed int64,
	cab *challengeAllocBlobber,
	randValidators []*ValidatorHealthCheck,
	challengeID string,
	needValidNum int,
) (*challengeOutput, error) {
	r := rand.New(rand.NewSource(seed))
	var (
		selectedValidators  = make([]*ValidatorHealthCheck, 0, needValidNum)
		perm                = r.Perm(len(randValidators))
		remainingValidators = len(randValidators)
	)

	now := txn.CreationDate
	filterValidator := filterHealthyValidators(now)

	for i := 0; i < len(randValidators) && len(randValidators) < needValidNum; i++ {
		if remainingValidators < needValidNum {
			return nil, errors.New("validators number does not meet minimum challenge requirement after filtering")
		}
		validator := randValidators[perm[i]]
		kick, err := filterValidator(validator)
		if err != nil {
			return nil, common.NewError("add_challenge", "failed to filter validator: "+
				err.Error())
		}
		if kick {
			remainingValidators--
			continue
		}

		//sp, err := sc.getStakePool(spenum.Validator, validator.ID, balances)
		//if err != nil {
		//	return nil, fmt.Errorf("can't get validator %s stake pool: %v", randValidator.Id, err)
		//}
		//stake, err := sp.stake()
		//if err != nil {
		//	return nil, err
		//}
		//if stake < conf.MinStake { // The min stake should be checked when staking rather than here
		//	remainingValidators--
		//	continue
		//}

		selectedValidators = append(selectedValidators, validator)
		//selectedValidators = append(selectedValidators,
		//	&ValidationNode{
		//		Provider: provider.Provider{
		//			ID:           validator.ID,
		//			ProviderType: spenum.Validator,
		//		},
		//		BaseURL: validator.BaseURL,
		//	})
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
	storageChallenge.ValidatorIDs = validatorIDs
	storageChallenge.BlobberID = cab.allocBlobber.blobberID
	storageChallenge.AllocationID = cab.alloc.ID
	storageChallenge.Created = txn.CreationDate
	storageChallenge.BlobberIndex = cab.blobberIndex

	challInfo := &challengeInfo{
		StorageChallenge: storageChallenge,
		Seed:             seed,
		AllocationRoot:   cab.allocBlobber.allocRoot,
		Timestamp:        cab.allocBlobber.lastWMTimestamp,
	}

	return &challengeOutput{
		alloc:           cab.alloc,
		allocChallenges: cab.allocChallenges,
		challInfo:       challInfo,
	}, nil
}

type GenerateChallengeInput struct {
	Round int64 `json:"round,omitempty"`
}

func (sc *StorageSmartContract) generateChallenge(t *transaction.Transaction,
	b *block.Block, input []byte, balances cstate.StateContextI,
	timings map[string]time.Duration,
) (err error) {
	m := Timings{timings: timings, start: common.ToTime(t.CreationDate)}
	inputRound := GenerateChallengeInput{}
	if err := json.Unmarshal(input, &inputRound); err != nil {
		return err
	}

	if inputRound.Round != b.Round {
		return fmt.Errorf("bad round, block %v but input %v", b.Round, inputRound.Round)
	}

	var (
		conf                *Config
		validators          *partitions.Partitions
		challengeReadyParts *partitions.Partitions
		cab                 *challengeAllocBlobber
		bcNum               int
		//acs                 *AllocationChallengeStats
	)

	m.tick("generate challenge start")
	cr := concurrentReader{}
	cr.add(func() error {
		//tm := time.Now()
		if conf, err = sc.getConfig(balances, true); err != nil {
			return fmt.Errorf("can't get SC configurations: %v", err.Error())
		}
		//fmt.Println("get config:", time.Since(tm))
		return nil
	})

	var selectedValidators []*ValidatorHealthCheck
	cr.add(func() error {
		//tm := time.Now()
		var err error
		validators, err = getValidatorsList(balances)
		if err != nil {
			return common.NewErrorf("generate_challenge",
				"error getting the validators list: %v", err)
		}

		var randValidators []ValidationPartitionNode
		r := rand.New(rand.NewSource(b.RoundRandomSeed))
		if err := validators.GetRandomItems(balances, r, &randValidators); err != nil {
			return common.NewErrorf("add_challenge",
				"error getting validators random slice: %v", err)
		}

		//if len(randValidators) < needValidNum {
		//	return nil, errors.New("validators number does not meet minimum challenge requirement")
		//}

		allValidatorIDs := make([]string, len(randValidators))
		for i, v := range randValidators {
			allValidatorIDs[i] = v.Id
		}

		selectedValidators, err = getValidatorsHCsByIDs(allValidatorIDs[:2], balances)
		if err != nil {
			return common.NewErrorf("add_challenge", "could not get validators list: %v", err)
		}

		//fmt.Println("get validators:", time.Since(tm))
		return nil
	})

	cr.add(func() error {
		//tm := time.Now()
		var err error
		challengeReadyParts, err = partitionsChallengeReadyAllocs(balances)
		if err != nil {
			return common.NewErrorf("generate_challenge",
				"error getting the challenge ready list: %v", err)
		}

		bcNum, err = challengeReadyParts.Size(balances)
		if err != nil {
			return common.NewErrorf("generate_challenge", "error getting challenge ready list size: %v", err)
		}

		if bcNum == 0 {
			return nil
		}

		r := rand.New(rand.NewSource(b.RoundRandomSeed))
		cab, err = sc.selectAllocBlobberForChallenge(challengeReadyParts, r, balances)
		if err != nil {
			return common.NewError("add_challenge", err.Error())
		}

		if cab.alloc == nil || cab.allocBlobber == nil {
			return common.NewError("add_challlenge", "no challenge ready alloc or blobber")
		}

		//fmt.Println("get alloc for challenge:", time.Since(tm))
		//logging.Logger.Debug("generate_challenges", zap.String("alloc ID", cab.alloc.ID),
		//	zap.String("blobber ID", cab.allocBlobber.BlobberID))

		return nil
	})

	if err := cr.do(); err != nil {
		return err
	}
	m.tick("concurrent load")
	//fmt.Println("here 1")

	if bcNum == 0 {
		logging.Logger.Info("skipping generate challenge: empty challenge ready list")
		return nil
	}
	// Check if the length of the list of validators is higher than the required number of validators
	needValidNum := conf.ValidatorsPerChallenge
	if len(selectedValidators) < needValidNum {
		return fmt.Errorf("validators number does not meet minimum challenge requirement: %v < %v",
			len(selectedValidators), needValidNum)
	}

	//currentValidatorsCount, err := validators.Size(balances)
	//if err != nil {
	//	return fmt.Errorf("can't get validators partition size: %v", err.Error())
	//}
	//
	//if currentValidatorsCount < needValidNum {
	//	err := errors.New("validators number does not meet minimum challenge requirement")
	//	logging.Logger.Error("generate_challenge", zap.Error(err),
	//		zap.Int("validator num", currentValidatorsCount),
	//		zap.Int("minimum required", needValidNum))
	//	return common.NewError("generate_challenge",
	//		"validators number does not meet minimum challenge requirement")
	//}

	challengeID := encryption.Hash(t.Hash + b.PrevHash)
	// the "1" was the index when generating multiple challenges.
	// keep it in case we need to generate more than 1 challenge at once.
	//challengeID := encryption.Hash(hashSeed + "1")

	//seedSource, err := strconv.ParseUint(challengeID[0:16], 16, 64)
	//if err != nil {
	//	return common.NewErrorf("generate_challenge",
	//		"Error in creating challenge seed: %v", err)
	//}

	result, err := sc.populateGenerateChallenge(
		t,
		b.RoundRandomSeed,
		cab,
		selectedValidators,
		challengeID,
		needValidNum)
	if err != nil {
		return common.NewErrorf("generate_challenge", err.Error())
	}

	if result == nil {
		logging.Logger.Error("received empty data for challenge generation. Skipping challenge generation")
		return nil
	}
	m.tick("populate challenge")
	//fmt.Printf("generate challange, alloc:%s\n blobber: %s\n", result.challInfo.AllocationID, result.challInfo.BlobberID)

	err = sc.addChallenge(
		result.alloc.ID,
		result.alloc,
		result.challInfo,
		result.allocChallenges,
		cab.allocChallengesStats,
		balances)
	if err != nil {
		return common.NewErrorf("adding_challenge_error",
			"Error in adding challenge: %v", err)
	}

	m.tick("save challenges")
	return nil
}

func removeExpiredChallenges(
	allocID string,
	abs *allocBlobbers,
	allocChallenges *AllocationChallenges,
	acs *AllocationChallengeStats,
	now common.Timestamp,
	balances cstate.StateContextI) ([]*AllocOpenChallenge, error) {
	if len(allocChallenges.OpenChallenges) == 0 {
		// no open challenges, nothing to do
		return nil, nil
	}

	var (
		expiredChallengeBlobberMap = make(map[string]struct{})
		cct                        = getMaxChallengeCompletionTime()
	)

	//var nonExpiredChallenges []*AllocOpenChallenge
	var expiredChallenges []*AllocOpenChallenge
	for _, oc := range allocChallenges.OpenChallenges {
		// TODO: The next line writes the id of the challenge to process, in order to find out the duplicate challenge.
		// should be removed when this issue is fixed. See https://github.com/0chain/0chain/pull/2025#discussion_r1080697805
		logging.Logger.Debug("removeExpiredChallenges processing open challenge:", zap.String("challengeID", oc.ChallengeID))
		if _, ok := expiredChallengeBlobberMap[oc.ChallengeID]; ok {
			logging.Logger.Error("removeExpiredChallenges found duplicate expired challenge", zap.String("challengeID", oc.ChallengeID))
			return nil, common.NewError("removeExpiredChallenges", "found duplicates expired challenge")
		}

		if !isChallengeExpired(now, common.Timestamp(oc.CreatedAt), cct) {
			break
		}

		expiredChallenges = append(expiredChallenges, oc)
		expiredChallengeBlobberMap[oc.ChallengeID] = struct{}{}
		acs.FailChallenges(oc.BlobberIndex)
		bsts, err := acs.GetBlobberStatsByIndex(int(oc.BlobberIndex))
		if err != nil {
			return nil, err
		}

		emitUpdateChallenge(&StorageChallenge{
			ID:           oc.ChallengeID,
			AllocationID: allocID,
			BlobberID:    abs.Blobbers[oc.BlobberIndex].BlobberID,
		}, false, balances, acs.GetAllocStats(), bsts)
		// expire 1 challenge at a time
		break
	}

	allocChallenges.OpenChallenges = allocChallenges.OpenChallenges[len(expiredChallenges):]

	return expiredChallenges, nil
}

func (sc *StorageSmartContract) addChallenge(
	//alloc *StorageAllocation,
	allocID string,
	alloc *allocBlobbers,
	challenge *challengeInfo,
	allocChallenges *AllocationChallenges,
	acs *AllocationChallengeStats,
	balances cstate.StateContextI) error {

	if challenge.BlobberID == "" {
		return common.NewError("add_challenge",
			"no blobber to add challenge to")
	}

	//blobAlloc, ok := alloc.BlobberAllocsMap[challenge.BlobberID]
	//if !ok {
	//	return common.NewError("add_challenge",
	//		"no blobber Allocation to add challenge to")
	//}

	// remove expired challenges
	expiredChallenges, err := removeExpiredChallenges(allocID, alloc, allocChallenges, acs, challenge.Created, balances)
	if err != nil {
		return common.NewErrorf("add_challenge", "remove expired challenges: %v", err)
	}

	//fmt.Println("expired challenges:", len(expiredChallenges))

	//var expChalIDs []string
	//for challengeID := range expiredChallenges {
	//	expChalIDs = append(expChalIDs, challengeID)
	//}
	//sort.Strings(expChalIDs)

	// maps blobberID to count of its expiredIDs.
	//expiredCountMap := make(map[string]int)

	// TODO: maybe delete them periodically later instead of remove immediately
	for _, c := range expiredChallenges {
		_, err := balances.DeleteTrieNode(storageChallengeKey(sc.ID, c.ChallengeID))
		if err != nil {
			return common.NewErrorf("add_challenge", "could not delete challenge node: %v", err)
		}
	}

	// add the generated challenge to the open challenges list in the allocation
	//if !allocChallenges.addChallenge(challenge) {
	allocChallenges.addChallenge(challenge.StorageChallenge)
	//	return common.NewError("add_challenge", "challenge already exist in allocation")
	//}

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

	// get blobber index
	bIdx := alloc.blobberIndex(challenge.BlobberID)
	if err := acs.AddAllocOpenChallenge(bIdx); err != nil {
		return common.NewErrorf("add_challenge", "update alloc open challenge stats failed: %v", err)
	}
	if err := acs.Save(balances, alloc.ID); err != nil {
		return common.NewErrorf("add_challenge", "save alloc challenge stats failed: %v", err)
	}

	//alloc.Stats.OpenChallenges++
	//alloc.Stats.TotalChallenges++
	//blobAlloc.Stats.OpenChallenges++
	//blobAlloc.Stats.TotalChallenges++

	//if err := alloc.save(balances, sc.ID); err != nil {
	//	return common.NewErrorf("add_challenge",
	//		"error storing allocation: %v", err)
	//}

	//balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationChallenges, alloc.ID, alloc.buildUpdateChallengeStat())

	beforeEmitAddChallenge(challenge)

	bSts, err := acs.GetBlobberStatsByIndex(bIdx)
	if err != nil {
		return common.NewErrorf("add_challenge", "get blobber stats failed: %v", err)
	}
	emitAddChallenge(challenge, len(expiredChallenges), balances, acs.GetAllocStats(), bSts)
	return nil
}

func isChallengeExpired(now, createdAt common.Timestamp, challengeCompletionTime time.Duration) bool {
	return createdAt+common.ToSeconds(challengeCompletionTime) <= now
}
