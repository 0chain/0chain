package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync/atomic"
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
	maxChallengeCompletionTime time.Duration,
	balances cstate.StateContextI,
	allocationID string) error {
	conf, err := sc.getConfig(balances, true)
	if err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	// time of this challenge
	challengeCompletedTime := blobAlloc.LatestCompletedChallenge.Created
	if challengeCompletedTime > alloc.Expiration+toSeconds(maxChallengeCompletionTime) {
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

	err = sc.moveToBlobbers(alloc, blobberReward, blobAlloc.BlobberID, sp, balances, allocationID)
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

	err = sc.moveToValidators(alloc, validatorsReward, validators, vsps, balances, allocationID)
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

	if err = alloc.saveUpdatedStakes(balances); err != nil {
		return fmt.Errorf("can't save allocation: %v", err)
	}

	emitChallengePoolEvent(alloc, balances)

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
func (sc *StorageSmartContract) blobberPenalty(
	alloc *StorageAllocation,
	prev common.Timestamp,
	blobAlloc *BlobberAllocation,
	validators []string,
	maxChallengeCompletionTime time.Duration,
	balances cstate.StateContextI,
	allocationID string,
) (err error) {
	var conf *Config
	if conf, err = sc.getConfig(balances, true); err != nil {
		return fmt.Errorf("can't get SC configurations: %v", err.Error())
	}

	// time of this challenge
	challengeCompleteTime := blobAlloc.LatestCompletedChallenge.Created
	if challengeCompleteTime > alloc.Expiration+toSeconds(maxChallengeCompletionTime) {
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
	err = sc.moveToValidators(alloc, validatorsReward, validators, vSPs, balances, allocationID)
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
		move, err = sp.slash(blobAlloc.BlobberID, getOffer(alloc.BSize, bTerms), slash, balances, allocationID)
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

	emitChallengePoolEvent(alloc, balances)
	return
}

func (sc *StorageSmartContract) moveToValidators(
	alloc *StorageAllocation,
	reward currency.Coin,
	validators []datastore.Key,
	vSPs []*stakePool,
	balances cstate.StateContextI,
	allocationID string,
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
		err := sp.DistributeRewards(oneReward, validators[i], spenum.Validator, spenum.ValidationReward, balances, allocationID)
		if err != nil {
			return fmt.Errorf("moving to validator %s: %v",
				validators[i], err)
		}
	}
	if bal > 0 {
		for i := 0; i < int(bal); i++ {
			err := vSPs[i].DistributeRewards(1, validators[i], spenum.Validator, spenum.ValidationReward, balances, allocationID)
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

	var (
		challenge *StorageChallenge
		conf      *Config
	)
	cr := concurrentReader{}
	cr.add(func() error {
		// get challenge node
		challenge, err = sc.getStorageChallenge(challResp.ID, balances)
		if err != nil {
			return common.NewErrorf(errCode, "could not find challenge, %v", err)
		}
		if challenge.Responded != 0 {
			return common.NewError(errCode, "challenge already processed")
		}

		if challenge.BlobberID != t.ClientID {
			return errors.New("challenge blobber id does not match")
		}

		logging.Logger.Info("time_taken: receive challenge response",
			zap.String("challenge_id", challenge.ID),
			zap.Duration("delay", time.Since(common.ToTime(challenge.Created))))
		return nil
	})

	cr.add(func() error {
		conf, err = sc.getConfig(balances, true)
		if err != nil {
			return common.NewErrorf(errCode,
				"cannot get smart contract configurations: %v", err)
		}

		return nil
	})

	if err := cr.do(); err != nil {
		return "", err
	}

	var (
		result          *verifyTicketsResult
		allocChallenges *AllocationChallenges
		alloc           *StorageAllocation
		blobber         *StorageNode
		bil             BlobberOfferStakeList
		ongoingParts    *partitions.Partitions
	)
	cr = concurrentReader{}
	cr.add(func() error {
		result, err = verifyChallengeTickets(balances, t, challenge, &challResp, conf.MaxChallengeCompletionTime)
		if err != nil {
			return common.NewError(errCode, err.Error())
		}

		return nil
	})

	cr.add(func() error {
		allocChallenges, err = sc.getAllocationChallenges(challenge.AllocationID, balances)
		if err != nil {
			return common.NewErrorf(errCode, "could not find allocation challenges, %v", err)
		}

		return nil
	})
	cr.add(func() error {
		alloc, err = sc.getAllocation(challenge.AllocationID, balances)
		if err != nil {
			return common.NewErrorf(errCode,
				"can't get related allocation: %v", err)
		}

		if alloc.Finalized {
			return common.NewError(errCode, "allocation is finalized")
		}
		return nil
	})
	cr.add(func() error {
		blobber, err = getBlobber(t.ClientID, balances)
		if err != nil {
			return common.NewErrorf(errCode, "could not get blobber: %v", err)
		}
		return nil
	})

	cr.add(func() error {
		bil, err = getBlobbersInfoList(balances)
		if err != nil {
			return common.NewErrorf(errCode, "could not get blobbers info list: %v", err)
		}
		return nil
	})
	cr.add(func() error {
		ongoingParts, err = getOngoingPassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
		if err != nil {
			return common.NewErrorf(errCode, "could not get ongoing partition: %v", err)
		}

		return nil
	})

	if err := cr.do(); err != nil {
		return "", err
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
		blobber:                  blobber,
		blobAlloc:                blobAlloc,
		bil:                      bil,
		ongoingParts:             ongoingParts,
		latestCompletedChallTime: latestCompletedChallTime,
	}

	if !(result.pass && result.fresh) {
		return sc.challengeFailed(balances, conf.NumValidatorsRewarded, cab, conf.MaxChallengeCompletionTime)
	}

	return sc.challengePassed(balances, t, conf.BlockReward.TriggerPeriod, conf.NumValidatorsRewarded, cab, conf.MaxChallengeCompletionTime)
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
	blobber                  *StorageNode
	blobAlloc                *BlobberAllocation
	bil                      BlobberOfferStakeList
	ongoingParts             *partitions.Partitions
	latestCompletedChallTime common.Timestamp
}

func verifyChallengeTickets(balances cstate.StateContextI,
	t *transaction.Transaction,
	challenge *StorageChallenge,
	cr *ChallengeResponse,
	maxChallengeCompletionTime time.Duration,
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
		success, failure int32
		validators       = make([]string, len(cr.ValidationTickets)) // validators for rewards
		ccr              = concurrentReader{}
	)

	for i, vt := range cr.ValidationTickets {
		func(idx int, v *ValidationTicket) {
			ccr.add(func() error {
				if err := v.Validate(challenge.ID, challenge.BlobberID); err != nil {
					return fmt.Errorf("invalid validation ticket: %v", err)
				}

				if ok, err := v.VerifySign(balances); !ok || err != nil {
					return fmt.Errorf("invalid validation ticket: %v", err)
				}
				validators[idx] = v.ValidatorID
				if !v.Result {
					atomic.AddInt32(&failure, 1)
				} else {
					atomic.AddInt32(&success, 1)
				}
				return nil
			})
		}(i, vt)
	}
	if err := ccr.do(); err != nil {
		return nil, err
	}

	var (
		pass  = success > int32(threshold)
		fresh = challenge.Created+toSeconds(maxChallengeCompletionTime) >= t.CreationDate
	)

	return &verifyTicketsResult{
		pass:       pass,
		fresh:      fresh,
		threshold:  threshold,
		success:    int(success),
		validators: validators,
	}, nil
}

func (sc *StorageSmartContract) challengePassed(
	balances cstate.StateContextI,
	t *transaction.Transaction,
	triggerPeriod int64,
	validatorsRewarded int,
	cab *challengeAllocBlobberPassResult,
	maxChallengeCompletionTime time.Duration,
) (string, error) {
	rewardRound := GetCurrentRewardRound(balances.GetBlock().Round, triggerPeriod)
	// this expiry of blobber needs to be corrected once logic is finalized

	if cab.blobber.RewardRound.StartRound != rewardRound {
		var dataRead float64 = 0
		if cab.blobber.LastRewardDataReadRound >= rewardRound {
			dataRead = cab.blobber.DataReadLastRewardRound
		}

		err := cab.ongoingParts.Add(
			balances,
			&BlobberRewardNode{
				ID:                cab.blobber.ID,
				SuccessChallenges: 0,
				WritePrice:        cab.blobber.Terms.WritePrice,
				ReadPrice:         cab.blobber.Terms.ReadPrice,
				TotalData:         sizeInGB(cab.bil[cab.blobber.Index].SavedData),
				DataRead:          dataRead,
			})
		if err != nil {
			return "", common.NewError("verify_challenge",
				"can't add to ongoing partition list "+err.Error())
		}

		cab.blobber.RewardRound = RewardRound{
			StartRound: rewardRound,
			Timestamp:  t.CreationDate,
		}

		_, err = balances.InsertTrieNode(cab.blobber.GetKey(), cab.blobber)
		if err != nil {
			return "", common.NewError("verify_challenge",
				"error inserting blobber to chain"+err.Error())
		}
	}

	var brStats BlobberRewardNode
	if err := cab.ongoingParts.Get(balances, cab.blobber.ID, &brStats); err != nil {
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

	err := cab.ongoingParts.UpdateItem(balances, &brStats)
	if err != nil {
		return "", common.NewErrorf("verify_challenge",
			"error updating blobber reward item: %v", err)
	}

	err = cab.ongoingParts.Save(balances)
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
		maxChallengeCompletionTime,
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
	maxChallengeCompletionTime time.Duration,
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

	emitUpdateChallenge(cab.challenge, false, balances, cab.alloc.Stats, cab.blobAlloc.Stats)

	if err := cab.allocChallenges.Save(balances, sc.ID); err != nil {
		return "", common.NewError("challenge_penalty_error", err.Error())
	}

	logging.Logger.Info("Challenge failed", zap.String("challenge", cab.challenge.ID))
	validators := getRandomSubSlice(cab.validators, validatorsRewarded, balances.GetBlock().GetRoundRandomSeed())
	err := sc.blobberPenalty(
		cab.alloc, cab.latestCompletedChallTime, cab.blobAlloc, validators,
		maxChallengeCompletionTime,
		balances,
		cab.challenge.AllocationID,
	)
	if err != nil {
		return "", common.NewError("challenge_penalty_error", err.Error())
	}

	// save allocation object
	if err := cab.alloc.save(balances, sc.ID); err != nil {
		return "", common.NewError("challenge_reward_error", err.Error())
	}

	//balances.EmitEvent(event.TypeStats, event.TagUpdateAllocation, alloc.ID, alloc.buildDbUpdates())
	if cab.pass && !cab.fresh {
		return "late challenge (failed)", nil
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

func (sc *StorageSmartContract) populateGenerateChallenge(
	challengeReadyAllocsParts *partitions.Partitions,
	seed int64,
	validators *partitions.Partitions,
	txn *transaction.Transaction,
	challengeID string,
	balances cstate.StateContextI,
	needValidNum int,
	conf *Config,
) (*challengeOutput, error) {
	r := rand.New(rand.NewSource(seed))
	// get random allocations from the partitions
	var randAllocs []ChallengeReadyAllocNode
	if err := challengeReadyAllocsParts.GetRandomItems(balances, r, &randAllocs); err != nil {
		return nil, common.NewErrorf("generate_challenge",
			"error getting random items from challenge ready partition: %v", err)
	}

	allocID := randAllocs[r.Intn(len(randAllocs))].AllocID
	// get the allocation
	alloc, err := sc.getAllocation(allocID, balances)
	if err != nil {
		return nil, common.NewErrorf("generate_challenge", "could not get allocation: %v", err)
	}

	if alloc.Stats.UsedSize == 0 {
		logging.Logger.Warn("generate_challenge: allocation in challenge ready partitions has no used space",
			zap.String("allocation", alloc.ID))
		return nil, common.NewErrorf("generate_challenge", "allocation has no used space")
	}

	var (
		allocBlobbersNum = len(alloc.Blobbers)
		bIDs             = make([]string, 0, allocBlobbersNum)
	)

	// get all blobbers that have data written
	for _, ba := range alloc.BlobberAllocs {
		if ba.Stats.UsedSize > 0 {
			bIDs = append(bIDs, ba.BlobberID)
		}
	}

	// select a blobber randomly
	pm := r.Perm(len(bIDs))
	var blobber *StorageNode
	for i := 0; i < len(bIDs); i++ {
		randBID := bIDs[pm[i]]
		var err error
		blobber, err = getBlobber(randBID, balances)
		if err != nil {
			return nil, common.NewErrorf("generate_challenge", "could not get blobber: %v", err)
		}

		if !blobber.NotAvailable {
			break
		}
	}

	if blobber == nil {
		// means no available blobbers, they were either killed or shutdown
		return nil, common.NewErrorf("generate_challenge", "no blobber to challenge")
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
		if randValidator.Id == blobber.ID {
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
	storageChallenge.BlobberID = blobber.ID
	storageChallenge.AllocationID = alloc.ID
	storageChallenge.Created = txn.CreationDate

	allocBlobber := alloc.BlobberAllocsMap[blobber.ID]
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
	currentValidatorsCount := validators.Size()
	if currentValidatorsCount < needValidNum {
		err := errors.New("validators number does not meet minimum challenge requirement")
		logging.Logger.Error("generate_challenge", zap.Error(err),
			zap.Int("validator num", currentValidatorsCount),
			zap.Int("minimum required", needValidNum))
		return common.NewError("generate_challenge",
			"validators number does not meet minimum challenge requirement")
	}

	// get blobber allocations partitions
	challengeReadyAllocsParts, err := partitionsChallengeReadyAllocs(balances)
	if err != nil {
		return common.NewErrorf("generate_challenge",
			"error getting challenge ready allocation list: %v", err)
	}

	if challengeReadyAllocsParts.Size() == 0 {
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
		challengeReadyAllocsParts,
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
	expiredIDsMap, err := alloc.removeExpiredChallenges(allocChallenges, challenge.Created, conf.MaxChallengeCompletionTime, balances)
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

	//balances.EmitEvent(event.TypeStats, event.TagUpdateAllocationChallenges, alloc.ID, alloc.buildUpdateChallengeStat())

	beforeEmitAddChallenge(challInfo)

	emitAddChallenge(challInfo, expiredCountMap, len(expiredIDsMap), balances, alloc.Stats, blobAlloc.Stats)
	return nil
}

func isChallengeExpired(now, createdAt common.Timestamp, challengeCompletionTime time.Duration) bool {
	return createdAt+common.ToSeconds(challengeCompletionTime) <= now
}
