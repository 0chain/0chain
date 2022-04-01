package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool"
	"github.com/stretchr/testify/require"
)

const (
	errLate                = "late challenge response"
	errTokensChallengePool = "not enough tokens in challenge pool"
	errNoStakePools        = "no stake pools to move tokens to"
	errRewardBlobber       = "can't move tokens to blobber"
	errRewardValidator     = "rewarding validators"
)

func TestAddChallenge(t *testing.T) {
	type parameters struct {
		numBlobbers            int
		numValidators          int
		validatorsPerChallenge int
		randomSeed             int
	}

	type args struct {
		alloc               *StorageAllocation
		storageChallenge    *StorageChallenge
		blobberChallengeObj *BlobberChallenge
		allocChallengeObj   *AllocationChallenge
		blobberAllocation   *BlobberAllocation
		validators          partitions.RandPartition
		r                   *rand.Rand
		blobberID           string
		balances            cstate.StateContextI
	}

	type want struct {
		validators []int
		error      bool
		errorMsg   string
	}

	parametersToArgs := func(p parameters, ssc *StorageSmartContract) args {

		blobberChallenge := partitions.NewRandomSelector(
			ALL_BLOBBERS_CHALLENGE_KEY,
			allBlobbersChallengePartitionSize,
			nil,
			partitions.ItemBlobberChallenge,
		)

		balances := &mockStateContext{
			store: make(map[datastore.Key]util.MPTSerializable),
		}

		var blobbers []*StorageNode
		var blobberMap = make(map[string]*BlobberAllocation)
		for i := 0; i < p.numBlobbers; i++ {
			var sn = StorageNode{
				ID: strconv.Itoa(i),
			}
			blobbers = append(blobbers, &sn)
			blobberMap[sn.ID] = &BlobberAllocation{
				AllocationRoot: "root " + sn.ID,
				Stats:          &StorageAllocationStats{},
			}

			_, err := blobberChallenge.Add(
				&partitions.BlobberChallengeNode{
					BlobberID: sn.ID,
				}, balances)
			require.NoError(t, err)
		}

		validators := partitions.NewRandomSelector(
			ALL_VALIDATORS_KEY,
			allValidatorsPartitionSize,
			nil,
			partitions.ItemValidator,
		)

		for i := 0; i < p.numValidators; i++ {
			_, err := validators.Add(
				&partitions.ValidationNode{
					Id:  strconv.Itoa(i),
					Url: strconv.Itoa(i) + ".com",
				}, balances,
			)
			require.NoError(t, err)
		}

		var bID string
		r := rand.New(rand.NewSource(int64(p.randomSeed)))
		if p.numBlobbers > 0 {
			bcList, err := blobberChallenge.GetRandomSlice(r, balances)
			require.NoError(t, err)
			i := rand.Intn(len(bcList))
			bcItem := bcList[i]
			bID = bcItem.Name()
		}

		selectedValidators := make([]*ValidationNode, 0)
		randSlice, err := validators.GetRandomSlice(r, balances)
		require.NoError(t, err)

		perm := r.Perm(len(randSlice))
		for i := 0; i < minInt(len(randSlice), p.validatorsPerChallenge+1); i++ {
			if randSlice[perm[i]].Name() != bID {
				selectedValidators = append(selectedValidators,
					&ValidationNode{
						ID:      randSlice[perm[i]].Name(),
						BaseURL: randSlice[perm[i]].Data(),
					})
			}
			if len(selectedValidators) >= p.validatorsPerChallenge {
				break
			}
		}

		allocChall, err := ssc.getAllocationChallenge("", balances)
		if err != nil && errors.Is(err, util.ErrValueNotPresent) {
			allocChall = new(AllocationChallenge)
		}
		var storageChallenge = new(StorageChallenge)
		storageChallenge.TotalValidators = len(selectedValidators)
		storageChallenge.BlobberID = bID
		storageChallenge.Created = creationDate

		blobberChall, err := ssc.getBlobberChallenge(bID, balances)
		if err != nil && errors.Is(err, util.ErrValueNotPresent) {
			blobberChall = new(BlobberChallenge)
		}

		return args{
			alloc: &StorageAllocation{
				BlobberMap: blobberMap,
				Stats:      &StorageAllocationStats{},
			},
			allocChallengeObj:   allocChall,
			storageChallenge:    storageChallenge,
			blobberAllocation:   blobberMap[bID],
			blobberChallengeObj: blobberChall,
			validators:          validators,
			r:                   r,
			blobberID:           bID,

			balances: &mockStateContext{
				store: make(map[datastore.Key]util.MPTSerializable),
			},
		}
	}

	validate := func(t *testing.T, resp string, err error, p parameters, want want) {
		if want.error {
			require.Error(t, err)
			require.EqualValues(t, want.errorMsg, err.Error())
			return
		}

		challenge := &StorageChallenge{}
		require.NoError(t, json.Unmarshal([]byte(resp), challenge))

		if p.numValidators > p.validatorsPerChallenge {
			require.EqualValues(t, challenge.TotalValidators, p.validatorsPerChallenge)

		} else {
			require.EqualValues(t, challenge.TotalValidators, p.numValidators-1)
		}
		require.EqualValues(t, len(want.validators), challenge.TotalValidators)
	}

	tests := []struct {
		name string

		parameters
		want want
	}{
		{
			name: "OK validators > validatorsPerChallenge",
			parameters: parameters{
				numBlobbers:            10,
				numValidators:          10,
				validatorsPerChallenge: 4,
				randomSeed:             1,
			},
			want: want{
				validators: []int{6, 3, 8, 4},
			},
		},
		{
			name: "OK validatorsPerChallenge > validators",
			parameters: parameters{
				numBlobbers:            6,
				numValidators:          6,
				validatorsPerChallenge: 10,
				randomSeed:             1,
			},
			want: want{
				validators: []int{3, 0, 1, 4, 2},
			},
		},
		{
			name: "Error no blobbers",
			parameters: parameters{
				numBlobbers:            0,
				numValidators:          6,
				validatorsPerChallenge: 10,
				randomSeed:             1,
			},
			want: want{
				error:    true,
				errorMsg: "add_challenge: no blobber to add challenge to",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var ssc = &StorageSmartContract{
				SmartContract: sci.NewSC(ADDRESS),
			}

			args := parametersToArgs(tt.parameters, ssc)

			resp, err := ssc.addChallenge(args.alloc,
				args.storageChallenge,
				args.blobberChallengeObj,
				args.allocChallengeObj,
				args.blobberAllocation,
				args.balances)

			validate(t, resp, err, tt.parameters, tt.want)
		})
	}
}

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
	var scYaml = Config{
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
	var size = int64(123000)
	var otherWritePools = 4
	var scYaml = Config{
		MaxMint:                    zcnToBalance(4000000.0),
		BlobberSlash:               0.1,
		ValidatorReward:            0.025,
		MaxChallengeCompletionTime: 30 * time.Minute,
		TimeUnit:                   720 * time.Hour,
	}
	var blobberYaml = mockBlobberYaml{
		serviceCharge:           0.30,
		challengeCompletionTime: scYaml.MaxChallengeCompletionTime,
		writePrice:              1,
	}
	var validatorYamls = []mockBlobberYaml{
		{serviceCharge: 0.2}, {serviceCharge: 0.25}, {serviceCharge: 0.3},
	}

	t.Run("test blobberPenalty ", func(t *testing.T) {
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, size, preiviousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run("test blobberPenalty ", func(t *testing.T) {
		var size = int64(10000)
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, size, preiviousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run(errLate, func(t *testing.T) {
		var thisChallenge = thisExpires + toSeconds(blobberYaml.challengeCompletionTime) + 1
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, size, preiviousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errLate)
	})

	t.Run(errNoStakePools, func(t *testing.T) {
		var validatorStakes = [][]int64{{45, 666, 4533}, {}, {10}}
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, size, preiviousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errNoStakePools))
		require.True(t, strings.Contains(err.Error(), errRewardValidator))
	})

	t.Run(errTokensChallengePool, func(t *testing.T) {
		var challengePoolBalance = state.Balance(0)
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalances, otherWritePools, challengePoolIntegralValue,
			challengePoolBalance, partial, size, preiviousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errTokensChallengePool))
	})
}

func testBlobberPenalty(
	t *testing.T,
	scYaml Config,
	blobberYaml mockBlobberYaml,
	validatorYamls []mockBlobberYaml,
	stakes []int64,
	validators []string,
	validatorStakes [][]int64,
	wpBalances []int64,
	otherWritePools int,
	challengePoolIntegralValue, challengePoolBalance state.Balance,
	partial float64,
	size int64,
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
		size:                       size,
		thisChallange:              thisChallange,
		thisExpires:                thisExpires,
		now:                        now,
	}

	var txn, ssc, allocation, challenge, details, ctx = setupChallengeMocks(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
		wpBalances, otherWritePools, challengePoolIntegralValue,
		challengePoolBalance, thisChallange, thisExpires, now, size)

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
	scYaml Config,
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
	scYaml Config,
	blobberYaml mockBlobberYaml,
	validatorYamls []mockBlobberYaml,
	stakes []int64,
	validators []string,
	validatorStakes [][]int64,
	wpBalances []int64,
	otherWritePools int,
	challengePoolIntegralValue, challengePoolBalance state.Balance,
	thisChallange, thisExpires, now common.Timestamp,
	size int64,
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
			WritePrice:              zcnToBalance(blobberYaml.writePrice),
			ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
		},
		Size: size,
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
			txn,
			nil,
			nil,
			nil,
			nil,
			nil,
		),
		clientBalance: zcnToBalance(3),
		store:         make(map[datastore.Key]util.MPTSerializable),
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
		sp.Pools["paula"+id] = &stakepool.DelegatePool{}
		sp.Pools["paula"+id].Balance = state.Balance(stake)
		sp.Pools["paula"+id].DelegateID = "delegate " + id
	}
	sp.Settings.DelegateWallet = blobberId + " wallet"
	require.NoError(t, sp.save(ssc.ID, challenge.BlobberID, ctx))

	var validatorsSPs = []*stakePool{}
	for i, validator := range validators {
		var sPool = newStakePool()
		sPool.Settings.ServiceCharge = validatorYamls[i].serviceCharge
		for j, stake := range validatorStakes[i] {
			var pool = &stakepool.DelegatePool{}
			pool.Balance = state.Balance(stake)
			var id = validator + " delegate " + strconv.Itoa(j)
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
	scYaml                                             Config
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
	size                                               int64
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
	var offer = int64(sizeInGB(f.size) * float64(zcnToInt64(f.blobberYaml.writePrice)))

	if offer <= slashedAmount {
		return int64(float64(offer) * delegateStake / totalStake)
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

	require.EqualValues(t, 0, int64(blobber.Reward))
	require.EqualValues(t, 0, int64(blobber.Reward))

	for _, sp := range validatorsSPs {
		for wallet, pool := range sp.Pools {
			var wSplit = strings.Split(wallet, " ")
			require.InDelta(t, f.validatorServiceCharge(wSplit[0]), int64(sp.Reward), errDelta)
			index, err := strconv.Atoi(wSplit[2])
			require.NoError(t, err)
			require.InDelta(t, f.validatorDelegateReward(wSplit[0], index), int64(pool.Reward), errDelta)
		}
	}

	if f.scYaml.BlobberSlash > 0.0 {
		for _, pool := range blobber.Pools {
			var delegate = strings.Split(pool.DelegateID, " ")
			index, err := strconv.Atoi(delegate[1])
			require.NoError(t, err)

			require.InDelta(t, f.stakes[index]-f.delegatePenalty(index), int64(pool.Balance), errDelta)
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
	require.InDelta(t, f.challengePoolBalance-f.rewardReturned()-f.validatorsReward(), int64(challengePool.Balance), errDelta)
	require.InDelta(t, f.blobberServiceCharge(), int64(blobber.Reward), errDelta)
	require.InDelta(t, f.blobberServiceCharge(), int64(blobber.Reward), errDelta)

	for _, sp := range validatorsSPs {
		for wallet, pool := range sp.Pools {
			var wSplit = strings.Split(wallet, " ")
			require.InDelta(t, f.validatorServiceCharge(wSplit[0]), int64(sp.Reward), errDelta)
			index, err := strconv.Atoi(wSplit[2])
			require.NoError(t, err)
			require.InDelta(t, f.validatorDelegateReward(wSplit[0], index), int64(pool.Reward), errDelta)
		}
	}
}
