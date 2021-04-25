package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"fmt"
	"github.com/stretchr/testify/require"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	errLate                = "late challenge response"
	errTokensChallengePool = "not enough tokens in challenge pool"
	errNoStakePools        = "no stake pools to move tokens to"
	errRewardBlobber       = "can't move tokens to blobber"
	errRewardValidator     = "rewarding validators"
)

func TestBlobberReward(t *testing.T) {
	var stakes = []int64{200, 234234, 100000}
	var challengePoolIntegralValue = state.Balance(73000000)
	var challengePoolBalance = state.Balance(700000)
	var partial = 0.9
	var previousChallenge = common.Timestamp(3)
	var thisChallenge = common.Timestamp(5)
	var thisExpires = common.Timestamp(222)
	var now = common.Timestamp(99)
	var validators = []string{
		"vallery", "vincent", "vivian",
	}
	var validatorStakes = [][]int64{{45, 666, 4533}, {999}, {10}}
	var writePoolBalances = []int64{23423, 33333333, 234234234}
	var otherWritePools = 4
	var scYaml = scConfig{
		MaxMint:                    zcnToBalance(4000000.0),
		ValidatorReward:            0.025,
		MaxChallengeCompletionTime: 30 * time.Minute,
		TimeUnit:                   720 * time.Hour,
	}
	var blobberYaml = mockBlobberYaml{
		serviceCharge:           0.30,
		challengeCompletionTime: scYaml.MaxChallengeCompletionTime,
	}
	var validatorYamls = []mockBlobberYaml{
		{serviceCharge: 0.2}, {serviceCharge: 0.25}, {serviceCharge: 0.3},
	}

	t.Run("test blobberReward", func(t *testing.T) {
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run(errLate, func(t *testing.T) {
		var thisChallenge = thisExpires + toSeconds(blobberYaml.challengeCompletionTime) + 1
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errLate)
	})

	t.Run(errTokensChallengePool, func(t *testing.T) {
		var challengePoolBalance = state.Balance(0)
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errTokensChallengePool))
	})

	t.Run(errNoStakePools, func(t *testing.T) {
		var stakes = []int64{}
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errNoStakePools))
		require.True(t, strings.Contains(err.Error(), errRewardBlobber))
	})

	t.Run(errNoStakePools, func(t *testing.T) {
		var validatorStakes = [][]int64{{45, 666, 4533}, {999}, {}}
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errNoStakePools))
		require.True(t, strings.Contains(err.Error(), errRewardValidator))
	})

}

func TestBlobberPenalty(t *testing.T) {
	var stakes = []int64{200, 234234, 100000}
	var challengePoolIntegralValue = state.Balance(73000000)
	var challengePoolBalance = state.Balance(700000)
	var partial = 0.9
	var preiviousChallenge = common.Timestamp(3)
	var thisChallenge = common.Timestamp(5)
	var thisExpires = common.Timestamp(222)
	var now = common.Timestamp(101)
	var validators = []string{
		"vallery", "vincent", "vivian",
	}
	var validatorStakes = [][]int64{{45, 666, 4533}, {999}, {10}}
	var writePoolBalances = []int64{23423, 33333333, 234234234}
	var blobberOffer = int64(123000)
	var otherWritePools = 4
	var scYaml = scConfig{
		MaxMint: zcnToBalance(4000000.0),
		StakePool: &stakePoolConfig{
			InterestRate:     0.0000334,
			InterestInterval: time.Minute,
		},
		BlobberSlash:               0.1,
		ValidatorReward:            0.025,
		MaxChallengeCompletionTime: 30 * time.Minute,
		TimeUnit:                   720 * time.Hour,
	}
	var blobberYaml = mockBlobberYaml{
		serviceCharge:           0.30,
		challengeCompletionTime: scYaml.MaxChallengeCompletionTime,
	}
	var validatorYamls = []mockBlobberYaml{
		{serviceCharge: 0.2}, {serviceCharge: 0.25}, {serviceCharge: 0.3},
	}

	t.Run("test blobberPenalty ", func(t *testing.T) {
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, blobberOffer, preiviousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run("test blobberPenalty ", func(t *testing.T) {
		var blobberOffer = int64(10000)
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, blobberOffer, preiviousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run(errLate, func(t *testing.T) {
		var thisChallenge = thisExpires + toSeconds(blobberYaml.challengeCompletionTime) + 1
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, blobberOffer, preiviousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errLate)
	})

	t.Run(errNoStakePools, func(t *testing.T) {
		var validatorStakes = [][]int64{{45, 666, 4533}, {}, {10}}
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, blobberOffer, preiviousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errNoStakePools))
		require.True(t, strings.Contains(err.Error(), errRewardValidator))
	})

	t.Run(errTokensChallengePool, func(t *testing.T) {
		var challengePoolBalance = state.Balance(0)
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, blobberOffer, preiviousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errTokensChallengePool))
	})
}

func testBlobberPenalty(
	t *testing.T,
	scYaml scConfig,
	blobberYaml mockBlobberYaml,
	validatorYamls []mockBlobberYaml,
	stakes []int64,
	validators []string,
	validatorStakes [][]int64,
	wpBalances []int64,
	otherWritePools int,
	challengePoolIntegralValue, challengePoolBalance state.Balance,
	partial float64,
	blobberOffer int64,
	previous, thisChallange, thisExpires, now common.Timestamp,
) (err error) {
	var f = formulaeBlobberReward{
		t:                          t,
		scYaml:                     scYaml,
		blobberYaml:                blobberYaml,
		validatorYamls:             validatorYamls,
		stakes:                     stakes,
		validators:                 validators,
		validatorStakes:            validatorStakes,
		wpBalances:                 wpBalances,
		otherWritePools:            otherWritePools,
		challengePoolIntegralValue: int64(challengePoolIntegralValue),
		challengePoolBalance:       int64(challengePoolBalance),
		partial:                    partial,
		previousChallange:          previous,
		blobberOffer:               blobberOffer,
		thisChallange:              thisChallange,
		thisExpires:                thisExpires,
		now:                        now,
	}

	var txn, ssc, allocation, challenge, details, ctx = setupChallengeMocks(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
		wpBalances, otherWritePools, challengePoolIntegralValue,
		challengePoolBalance, thisChallange, thisExpires, now, blobberOffer)

	err = ssc.blobberPenalty(txn, allocation, previous, challenge, details, validators, ctx)
	if err != nil {
		return err
	}

	newCP, err := ssc.getChallengePool(allocation.ID, ctx)
	require.NoError(t, err)

	newVSp, err := ssc.validatorsStakePools(validators, ctx)
	require.NoError(t, err)

	afterBlobber, err := ssc.getStakePool(blobberId, ctx)
	require.NoError(t, err)

	confirmBlobberPenalty(t, f, *newCP, newVSp, *afterBlobber, ctx)
	return nil
}

func testBlobberReward(
	t *testing.T,
	scYaml scConfig,
	blobberYaml mockBlobberYaml,
	validatorYamls []mockBlobberYaml,
	stakes []int64,
	validators []string,
	validatorStakes [][]int64,
	wpBalances []int64,
	otherWritePools int,
	challengePoolIntegralValue, challengePoolBalance state.Balance,
	partial float64,
	previous, thisChallange, thisExpires, now common.Timestamp,
) (err error) {
	require.Len(t, validatorStakes, len(validators))

	var f = formulaeBlobberReward{
		t:                          t,
		scYaml:                     scYaml,
		blobberYaml:                blobberYaml,
		validatorYamls:             validatorYamls,
		stakes:                     stakes,
		validators:                 validators,
		validatorStakes:            validatorStakes,
		wpBalances:                 wpBalances,
		otherWritePools:            otherWritePools,
		challengePoolIntegralValue: int64(challengePoolIntegralValue),
		challengePoolBalance:       int64(challengePoolBalance),
		partial:                    partial,
		previousChallange:          previous,
		thisChallange:              thisChallange,
		thisExpires:                thisExpires,
		now:                        now,
	}

	var txn, ssc, allocation, challenge, details, ctx = setupChallengeMocks(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
		wpBalances, otherWritePools, challengePoolIntegralValue,
		challengePoolBalance, thisChallange, thisExpires, now, 0)

	blobber, err := ssc.getStakePool(blobberId, ctx)
	require.NoError(t, err)
	blobber = blobber

	err = ssc.blobberReward(txn, allocation, previous, challenge, details, validators, partial, ctx)
	if err != nil {
		return err
	}

	newCP, err := ssc.getChallengePool(allocation.ID, ctx)
	require.NoError(t, err)

	newVSp, err := ssc.validatorsStakePools(validators, ctx)
	require.NoError(t, err)

	afterBlobber, err := ssc.getStakePool(blobberId, ctx)
	require.NoError(t, err)

	confirmBlobberReward(t, f, *newCP, newVSp, *afterBlobber, ctx)
	return nil
}

func setupChallengeMocks(
	t *testing.T,
	scYaml scConfig,
	blobberYaml mockBlobberYaml,
	validatorYamls []mockBlobberYaml,
	stakes []int64,
	validators []string,
	validatorStakes [][]int64,
	wpBalances []int64,
	otherWritePools int,
	challengePoolIntegralValue, challengePoolBalance state.Balance,
	thisChallange, thisExpires, now common.Timestamp,
	blobberOffer int64,
) (*transaction.Transaction, *StorageSmartContract, *StorageAllocation,
	*BlobberChallenge, *BlobberAllocation, *mockStateContext) {
	require.Len(t, validatorStakes, len(validators))

	var err error
	var allocation = &StorageAllocation{
		ID:         "alice",
		Owner:      "owin",
		Expiration: thisExpires,
		TimeUnit:   scYaml.TimeUnit,
	}
	var challenge = &BlobberChallenge{
		BlobberID: blobberId,
		LatestCompletedChallenge: &StorageChallenge{
			Created: thisChallange,
		},
	}
	var details = &BlobberAllocation{
		BlobberID:                  blobberId,
		ChallengePoolIntegralValue: challengePoolIntegralValue,
		Terms: Terms{
			ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
		},
	}

	var txn = &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: datastore.Key(transactionHash),
		},
		ClientID:     clientId,
		ToClientID:   storageScId,
		CreationDate: now,
	}
	var ctx = &mockStateContext{
		ctx: *cstate.NewStateContext(
			nil,
			&util.MerklePatriciaTrie{},
			&state.Deserializer{},
			txn,
			nil,
			nil,
			nil,
			nil,
		),
		clientBalance: zcnToBalance(3),
		store:         make(map[datastore.Key]util.Serializable),
	}
	var ssc = &StorageSmartContract{
		&sci.SmartContract{
			ID: storageScId,
		},
	}

	var cPool = challengePool{
		ZcnPool: &tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      allocation.ID,
				Balance: challengePoolBalance,
			},
		},
	}
	require.NoError(t, cPool.save(ssc.ID, allocation.ID, ctx))

	var wPool = writePool{
		Pools: allocationPools{},
	}
	for _, balance := range wpBalances {
		var newPool = &allocationPool{}
		newPool.Balance = state.Balance(balance)
		newPool.AllocationID = allocation.ID
		newPool.Blobbers = blobberPools{}
		newPool.Blobbers.add(&blobberPool{BlobberID: blobberId})
		wPool.Pools.add(newPool)
	}
	for i := 0; i < otherWritePools; i++ {
		var id = strconv.Itoa(i)
		var newPool = &allocationPool{}
		newPool.AllocationID = "alice" + id
		wPool.Pools.add(newPool)
	}
	require.NoError(t, wPool.save(ssc.ID, allocation.Owner, ctx))

	var sp = newStakePool()
	sp.Settings.ServiceCharge = blobberYaml.serviceCharge
	for i, stake := range stakes {
		var id = strconv.Itoa(i)
		sp.Pools["paula"+id] = &delegatePool{}
		sp.Pools["paula"+id].Balance = state.Balance(stake)
		sp.Pools["paula"+id].DelegateID = "delegate " + id
	}
	sp.Offers[allocation.ID] = &offerPool{
		Expire: thisExpires,
		Lock:   state.Balance(blobberOffer),
	}
	sp.Settings.DelegateWallet = blobberId + " wallet"
	require.NoError(t, sp.save(ssc.ID, challenge.BlobberID, ctx))

	var validatorsSPs = []*stakePool{}
	for i, validator := range validators {
		var sPool = newStakePool()
		sPool.Settings.ServiceCharge = validatorYamls[i].serviceCharge
		for j, stake := range validatorStakes[i] {
			var pool = &delegatePool{}
			pool.Balance = state.Balance(stake)
			var id = validator + " delegate " + strconv.Itoa(j)
			pool.DelegateID = id
			sPool.Pools[id] = pool
		}
		sPool.Settings.DelegateWallet = validator + " wallet"
		validatorsSPs = append(validatorsSPs, sPool)
	}
	require.NoError(t, ssc.saveStakePools(validators, validatorsSPs, ctx))

	_, err = ctx.InsertTrieNode(scConfigKey(ssc.ID), &scYaml)
	require.NoError(t, err)

	return txn, ssc, allocation, challenge, details, ctx
}

type formulaeBlobberReward struct {
	t                                                  *testing.T
	scYaml                                             scConfig
	blobberYaml                                        mockBlobberYaml
	validatorYamls                                     []mockBlobberYaml
	stakes                                             []int64
	validators                                         []string
	validatorStakes                                    [][]int64
	wpBalances                                         []int64
	otherWritePools                                    int
	challengePoolIntegralValue, challengePoolBalance   int64
	partial                                            float64
	previousChallange, thisChallange, thisExpires, now common.Timestamp
	blobberOffer                                       int64
}

func (f formulaeBlobberReward) reward() int64 {
	var challengePool = float64(f.challengePoolIntegralValue)
	var passedPrevious = float64(f.previousChallange)
	var passedCurrent = float64(f.thisChallange)
	var currentExpires = float64(f.thisExpires)
	var interpolationFraction = (passedCurrent - passedPrevious) / (currentExpires - passedPrevious)

	return int64(challengePool * interpolationFraction)
}

func (f formulaeBlobberReward) validatorsReward() int64 {
	var validatorCut = f.scYaml.ValidatorReward
	var totalReward = float64(f.reward())

	return int64(totalReward * validatorCut)
}

func (f formulaeBlobberReward) validatorReward() int64 {
	var total = float64(f.validatorsReward())
	var numberValidators = float64(len(f.validators))

	return int64(total / numberValidators)
}

func (f formulaeBlobberReward) blobberReward() int64 {
	var totalReward = float64(f.reward())
	var validatorReward = float64(f.validatorsReward())
	var blobberTotal = totalReward - validatorReward

	return int64(blobberTotal * f.partial)
}

func (f formulaeBlobberReward) rewardReturned() int64 {
	var blobberTotal = float64(f.reward() - f.validatorsReward())

	return int64(blobberTotal * (1 - f.partial))
}

func (f formulaeBlobberReward) blobberServiceCharge() int64 {
	var serviceCharge = blobberYaml.serviceCharge
	var blobberRewards = float64(f.blobberReward())

	return int64(blobberRewards * serviceCharge)
}

func (f formulaeBlobberReward) validatorServiceCharge(validator string) int64 {
	var serviceCharge = f.validatorYamls[f.indexFromValidator(validator)].serviceCharge
	var rewardPerValidator = float64(f.validatorsReward()) / float64(len(f.validators))

	return int64(rewardPerValidator * serviceCharge)
}

func (f formulaeBlobberReward) blobberDelegateReward(index int) int64 {
	require.True(f.t, index < len(f.stakes))
	var totalStake = 0.0
	for _, stake := range f.stakes {
		totalStake += float64(stake)
	}
	var delegateStake = float64(f.stakes[index])
	var totalDelegateReward = float64(f.blobberReward() - f.blobberServiceCharge())

	return int64(totalDelegateReward * delegateStake / totalStake)
}

func (f formulaeBlobberReward) indexFromValidator(validator string) int {
	for i, v := range f.validators {
		if v == validator {
			return i
		}
	}
	panic(fmt.Sprintf("cannot find validator %s", validator))
}

func (f formulaeBlobberReward) validatorDelegateReward(validator string, delegate int) int64 {
	var vIndex = f.indexFromValidator(validator)

	var totalStake = 0.0
	for _, stake := range f.validatorStakes[vIndex] {
		totalStake += float64(stake)
	}
	var delegateStake = float64(f.validatorStakes[vIndex][delegate])
	var validatorReward = float64(f.validatorsReward()) / float64(len(f.validators))
	var deleatesReward = validatorReward - float64(f.validatorServiceCharge(validator))
	return int64(deleatesReward * delegateStake / totalStake)
}

func (f formulaeBlobberReward) totalMoved() int64 {
	var reward = float64(f.reward())
	var validators = float64((f.validatorsReward()))
	var partial = f.partial
	var movedBack = (reward - validators) * (1 - partial)

	return int64(reward - movedBack)
}

func (f formulaeBlobberReward) blobberPenalty() int64 {
	var totalAction = float64(f.reward())
	var validatorReward = float64(f.validatorsReward())
	var blobberRisk = totalAction - validatorReward
	var slash = f.scYaml.BlobberSlash
	var slashedAmount = int64(blobberRisk * slash)

	if f.blobberOffer <= slashedAmount {
		return f.blobberOffer
	} else {
		return slashedAmount
	}
}

func (f formulaeBlobberReward) delegatePenalty(index int) int64 {
	require.True(f.t, index < len(f.stakes))
	var totalStake = 0.0
	for _, stake := range f.stakes {
		totalStake += float64(stake)
	}
	var delegateStake = float64(f.stakes[index])
	var slash = f.scYaml.BlobberSlash

	var totalAction = float64(f.reward())
	var validatorReward = float64(f.validatorsReward())
	var blobberRisk = totalAction - validatorReward
	var slashedAmount = int64(blobberRisk * slash)

	if f.blobberOffer <= slashedAmount {
		return int64(float64(f.blobberOffer) * delegateStake / totalStake)
	} else {
		return int64(float64(slashedAmount) * delegateStake / totalStake)
	}
}

func confirmBlobberPenalty(
	t *testing.T,
	f formulaeBlobberReward,
	challengePool challengePool,
	validatorsSPs []*stakePool,
	blobber stakePool,
	ctx cstate.StateContextI,
) {
	require.InDelta(t, f.challengePoolBalance-f.reward(), int64(challengePool.Balance), errDelta)

	require.EqualValues(t, 0, int64(blobber.Rewards.Charge))
	require.EqualValues(t, 0, int64(blobber.Rewards.Blobber))

	for _, sp := range validatorsSPs {
		for wallet, pool := range sp.Pools {
			var wSplit = strings.Split(wallet, " ")
			require.InDelta(t, f.validatorServiceCharge(wSplit[0]), int64(sp.Rewards.Charge), errDelta)
			require.InDelta(t, f.validatorReward()-f.validatorServiceCharge(wSplit[0]), int64(sp.Rewards.Validator), errDelta)
			index, err := strconv.Atoi(wSplit[2])
			require.NoError(t, err)
			require.InDelta(t, f.validatorDelegateReward(wSplit[0], index), int64(pool.Rewards), errDelta)
		}
	}

	if f.scYaml.BlobberSlash > 0.0 {
		require.InDelta(t, f.blobberOffer-f.blobberPenalty(), int64(blobber.Offers[challengePool.ID].Lock), errDelta)
		for _, pool := range blobber.Pools {
			var delegate = strings.Split(pool.DelegateID, " ")
			index, err := strconv.Atoi(delegate[1])
			require.NoError(t, err)
			require.InDelta(t, f.delegatePenalty(index), int64(pool.Penalty), errDelta)
			require.InDelta(t, f.stakes[index]-f.delegatePenalty(index), int64(pool.Balance), errDelta)
		}
	}

	validators := make(map[string]bool)
	for _, v := range f.validators {
		validators[v] = false
	}
	var validatorDelegates = make(map[string][]bool)
	for i, v := range f.validators {
		validatorDelegates[v] = []bool{}
		for range f.validatorStakes[i] {
			validatorDelegates[v] = append(validatorDelegates[v], false)
		}
	}

	var totalAmount = int64(0)
	for _, transfer := range ctx.GetTransfers() {
		require.EqualValues(t, storageScId, transfer.ClientID)
		var amount = int64(transfer.Amount)
		totalAmount += amount
		var wallet = strings.Split(transfer.ToClientID, " ")
		var validator = wallet[0]
		if wallet[1] == "wallet" { // validator service charge
			done, ok := validators[validator]
			require.True(t, ok)
			require.False(t, done)
			require.InDelta(t, f.validatorServiceCharge(validator), amount, errDelta)
			validators[validator] = true
			continue
		}
		require.Len(t, wallet, 3)
		index, err := strconv.Atoi(wallet[2])
		delegates, ok := validatorDelegates[validator]
		require.True(t, ok)
		require.False(t, delegates[index])
		require.NoError(t, err)
		require.InDelta(t, f.validatorDelegateReward(validator, index), amount, errDelta)
		validatorDelegates[validator][index] = true
	}
	require.InDelta(t, f.validatorsReward(), totalAmount, errDelta)

	for v, done := range validators {
		if !done {
			require.InDelta(t, f.validatorServiceCharge(v), 0, errDelta)
		}
	}
	for v, delegates := range validatorDelegates {
		for index, done := range delegates {
			if !done {
				require.InDelta(t, f.validatorDelegateReward(v, index), 0, errDelta)
			}
		}
	}
}

func confirmBlobberReward(
	t *testing.T,
	f formulaeBlobberReward,
	challengePool challengePool,
	validatorsSPs []*stakePool,
	blobber stakePool,
	ctx cstate.StateContextI,
) {
	require.InDelta(t, f.challengePoolBalance-f.reward(), int64(challengePool.Balance), errDelta)

	require.InDelta(t, f.blobberServiceCharge(), int64(blobber.Rewards.Charge), errDelta)
	require.InDelta(t, f.blobberReward()-f.blobberServiceCharge(), int64(blobber.Rewards.Blobber), errDelta)

	for _, sp := range validatorsSPs {
		for wallet, pool := range sp.Pools {
			var wSplit = strings.Split(wallet, " ")
			require.InDelta(t, f.validatorServiceCharge(wSplit[0]), int64(sp.Rewards.Charge), errDelta)
			require.InDelta(t, f.validatorReward()-f.validatorServiceCharge(wSplit[0]), int64(sp.Rewards.Validator), errDelta)
			index, err := strconv.Atoi(wSplit[2])
			require.NoError(t, err)
			require.InDelta(t, f.validatorDelegateReward(wSplit[0], index), int64(pool.Rewards), errDelta)
		}
	}

	var blobberPaid = false
	var blobberDelegaresPaid = []bool{}
	for range f.stakes {
		blobberDelegaresPaid = append(blobberDelegaresPaid, false)
	}
	validators := make(map[string]bool)
	for _, v := range f.validators {
		validators[v] = false
	}
	var validatorDelegates = make(map[string][]bool)
	for i, v := range f.validators {
		validatorDelegates[v] = []bool{}
		for range f.validatorStakes[i] {
			validatorDelegates[v] = append(validatorDelegates[v], false)
		}
	}

	var totalAmount = int64(0)
	for _, transfer := range ctx.GetTransfers() {
		require.EqualValues(t, storageScId, transfer.ClientID)
		var amount = int64(transfer.Amount)
		totalAmount += amount
		var wallet = strings.Split(transfer.ToClientID, " ")
		if wallet[0] == blobberId { // blobber service charge
			require.False(t, blobberPaid)
			require.InDelta(t, f.blobberServiceCharge(), amount, errDelta)
			blobberPaid = true
			continue
		}
		if wallet[0] == "delegate" { // payment  to blobber delegate
			index, err := strconv.Atoi(wallet[1])
			require.NoError(t, err)
			require.False(t, blobberDelegaresPaid[index])
			require.InDelta(t, f.blobberDelegateReward(index), amount, errDelta)
			blobberDelegaresPaid[index] = true
			continue
		}
		var validator = wallet[0]
		if wallet[1] == "wallet" { // validator service charge
			done, ok := validators[validator]
			require.True(t, ok)
			require.False(t, done)
			require.InDelta(t, f.validatorServiceCharge(validator), amount, errDelta)
			validators[validator] = true
			continue
		}
		require.Len(t, wallet, 3)
		index, err := strconv.Atoi(wallet[2])
		delegates, ok := validatorDelegates[validator]
		require.True(t, ok)
		require.False(t, delegates[index])
		require.NoError(t, err)
		require.InDelta(t, f.validatorDelegateReward(validator, index), amount, errDelta)
		validatorDelegates[validator][index] = true
	}
	require.InDelta(t, f.totalMoved(), totalAmount, errDelta)

	if !blobberPaid {
		require.InDelta(t, f.blobberServiceCharge(), 0, errDelta)
	}
	require.True(t, blobberPaid)
	for index, done := range blobberDelegaresPaid {
		if !done {
			require.InDelta(t, f.blobberDelegateReward(index), 0, errDelta)
		}
	}
	for v, done := range validators {
		if !done {
			require.InDelta(t, f.validatorServiceCharge(v), 0, errDelta)
		}
	}
	for v, delegates := range validatorDelegates {
		for index, done := range delegates {
			if !done {
				require.InDelta(t, f.validatorDelegateReward(v, index), 0, errDelta)
			}
		}
	}
}
