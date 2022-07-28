package storagesc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/currency"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
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
		blobberID    string
		allocID      string
		challengesTS []common.Timestamp
		newChallenge *StorageChallenge
		cct          time.Duration
		challInfo    *StorageChallengeResponse
		ct           common.Timestamp
	}

	type args struct {
		balances        cstate.StateContextI
		alloc           *StorageAllocation
		allocChallenges *AllocationChallenges
		newChallenge    func(ts common.Timestamp) *StorageChallenge
	}

	type want struct {
		openChallengeNum int
		error            bool
		errorMsg         string
	}

	var (
		blobberID = "blobber_1"
		allocID   = "alloc_1"
		allocRoot = "alloc_root"
	)

	parepareSSCArgs := func(t *testing.T, p parameters) (*StorageSmartContract, args) {
		ssc := &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}

		balances := &mockStateContext{
			store: make(map[datastore.Key]util.MPTSerializable),
		}

		challengeReadyParts, err := partitions.CreateIfNotExists(
			balances,
			ALL_CHALLENGE_READY_BLOBBERS_KEY,
			allChallengeReadyBlobbersPartitionSize)
		require.NoError(t, err)

		var blobberMap = make(map[string]*BlobberAllocation)

		blobberAllocs := make([]*BlobberAllocation, 1)
		blobberAllocs[0] = &BlobberAllocation{
			BlobberID:      blobberID,
			AllocationRoot: "root " + blobberID,
			Stats:          &StorageAllocationStats{},
			Terms:          Terms{},
		}

		blobberMap[blobberID] = blobberAllocs[0]

		_, err = challengeReadyParts.AddItem(
			balances,
			&ChallengeReadyBlobber{
				BlobberID: blobberID,
			})
		require.NoError(t, err)

		allocChallenges, err := ssc.getAllocationChallenges(allocID, balances)
		if err != nil && errors.Is(err, util.ErrValueNotPresent) {
			allocChallenges = new(AllocationChallenges)
			allocChallenges.AllocationID = allocID
		}

		alloc := &StorageAllocation{
			ID:               allocID,
			BlobberAllocs:    blobberAllocs,
			BlobberAllocsMap: blobberMap,
			Stats:            &StorageAllocationStats{},
		}

		for _, ts := range p.challengesTS {
			c := &StorageChallenge{
				ID:              fmt.Sprintf("%s:%s:%d", allocID, blobberID, ts),
				AllocationID:    allocID,
				BlobberID:       blobberID,
				TotalValidators: 1,
				Created:         ts,
			}

			challInfo := &StorageChallengeResponse{
				StorageChallenge: c,
				AllocationRoot:   alloc.BlobberAllocsMap[blobberID].AllocationRoot,
			}

			err = ssc.addChallenge(alloc, c, allocChallenges, challInfo, balances)
			require.NoError(t, err)
		}

		return ssc, args{
			alloc:           alloc,
			allocChallenges: allocChallenges,
			balances:        balances,
		}
	}

	newChallenge := func(ts common.Timestamp) (*StorageChallenge, *StorageChallengeResponse) {
		if ts == -1 {
			ch := &StorageChallenge{BlobberID: ""}
			return ch, &StorageChallengeResponse{StorageChallenge: ch}
		}
		ch := &StorageChallenge{
			ID:              fmt.Sprintf("%s:%s:%d", allocID, blobberID, ts),
			AllocationID:    allocID,
			BlobberID:       blobberID,
			TotalValidators: 1,
			Created:         ts,
		}
		return ch, &StorageChallengeResponse{
			StorageChallenge: ch,
			AllocationRoot:   allocRoot,
		}
	}

	tests := []struct {
		name string
		parameters
		prepareSSC func(cct common.Timestamp) *StorageSmartContract
		want       want
	}{
		{
			name: "OK",
			parameters: parameters{
				cct: 100 * time.Second,
				ct:  common.Timestamp(10),
			},
			want: want{
				openChallengeNum: 1,
			},
		},
		{
			name: "OK - more than one open challenges",
			parameters: parameters{
				cct:          100 * time.Second,
				challengesTS: []common.Timestamp{10, 20},
				ct:           common.Timestamp(30),
			},
			want: want{
				openChallengeNum: 3,
			},
		},
		{
			name: "OK - one challenge expired",
			parameters: parameters{
				cct:          100 * time.Second,
				challengesTS: []common.Timestamp{10, 20},
				ct:           common.Timestamp(110),
			},
			want: want{
				openChallengeNum: 2,
			},
		},
		{
			name: "OK - two challenge expired",
			parameters: parameters{
				cct:          100 * time.Second,
				challengesTS: []common.Timestamp{10, 20},
				ct:           common.Timestamp(120),
			},
			want: want{
				openChallengeNum: 1,
			},
		},
		{
			name: "Error challenge blobber ID is empty",
			parameters: parameters{
				ct: common.Timestamp(-1),
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
			ssc, args := parepareSSCArgs(t, tt.parameters)

			// add new challenge
			c, challInfo := newChallenge(tt.parameters.ct)
			err := ssc.addChallenge(args.alloc,
				c,
				args.allocChallenges,
				challInfo,
				args.balances)

			if tt.want.error {
				require.EqualValues(t, tt.want.errorMsg, err.Error())
				return
			}

			// assert the challenge is saved to MPT
			var challenge StorageChallenge
			err = args.balances.GetTrieNode(c.GetKey(ssc.ID), &challenge)
			require.NoError(t, err)
			require.EqualValues(t, *c, challenge)

			// assert the allocation is saved to MPT
			var alloc StorageAllocation
			err = args.balances.GetTrieNode(args.alloc.GetKey(ssc.ID), &alloc)
			require.NoError(t, err)

			// assert the open challenge stats is updated
			ba, ok := alloc.BlobberAllocsMap[challenge.BlobberID]
			require.True(t, ok)
			require.Equal(t, int64(tt.want.openChallengeNum), ba.Stats.OpenChallenges)
			require.Equal(t, int64(tt.want.openChallengeNum), alloc.Stats.OpenChallenges)

			// assert the AllocationChallenges that stores open challenges is saved
			var ac AllocationChallenges
			ac.AllocationID = args.alloc.ID
			err = args.balances.GetTrieNode(ac.GetKey(ssc.ID), &ac)
			require.NoError(t, err)

			// assert the open challenge number is correct
			require.Equal(t, tt.want.openChallengeNum, len(ac.OpenChallenges))
		})
	}
}

func TestBlobberReward(t *testing.T) {
	var stakes = []int64{200, 234234, 100000}
	var challengePoolIntegralValue = currency.Coin(73000000)
	var challengePoolBalance = currency.Coin(700000)
	var partial = 0.9
	var previousChallenge = common.Timestamp(3)
	var thisChallenge = common.Timestamp(5)
	var thisExpires = common.Timestamp(222)
	var now = common.Timestamp(99)
	var validators = []string{
		"vallery", "vincent", "vivian",
	}
	var validatorStakes = [][]int64{{45, 666, 4533}, {999}, {10}}
	var writePoolBalance currency.Coin = 23423 + 33333333 + 234234234
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
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run(errLate, func(t *testing.T) {
		var thisChallenge = thisExpires + toSeconds(blobberYaml.challengeCompletionTime) + 1
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errLate)
	})

	t.Run(errTokensChallengePool, func(t *testing.T) {
		var challengePoolBalance = currency.Coin(0)
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errTokensChallengePool))
	})

	t.Run(errNoStakePools, func(t *testing.T) {
		var stakes = []int64{}
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run(errNoStakePools, func(t *testing.T) {
		var validatorStakes = [][]int64{{45, 666, 4533}, {999}, {}}
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

}

func TestBlobberPenalty(t *testing.T) {
	var stakes = []int64{200, 234234, 100000}
	var challengePoolIntegralValue = currency.Coin(73000000)
	var challengePoolBalance = currency.Coin(700000)
	var partial = 0.9
	var preiviousChallenge = common.Timestamp(3)
	var thisChallenge = common.Timestamp(5)
	var thisExpires = common.Timestamp(222)
	var now = common.Timestamp(101)
	var validators = []string{
		"vallery", "vincent", "vivian",
	}
	var validatorStakes = [][]int64{{45, 666, 4533}, {999}, {10}}
	var writePoolBalance currency.Coin = 234234234
	var size = int64(123000)
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
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, size, preiviousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run("test blobberPenalty ", func(t *testing.T) {
		var size = int64(10000)
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, size, preiviousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run(errLate, func(t *testing.T) {
		var thisChallenge = thisExpires + toSeconds(blobberYaml.challengeCompletionTime) + 1
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, size, preiviousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errLate)
	})

	t.Run(errNoStakePools, func(t *testing.T) {
		var validatorStakes = [][]int64{{45, 666, 4533}, {}, {10}}
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, size, preiviousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run(errTokensChallengePool, func(t *testing.T) {
		var challengePoolBalance = currency.Coin(0)
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
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
	wpBalance currency.Coin,
	challengePoolIntegralValue, challengePoolBalance currency.Coin,
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
		wpBalance:                  wpBalance,
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
		wpBalance, challengePoolIntegralValue,
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
	wpBalance currency.Coin,
	challengePoolIntegralValue, challengePoolBalance currency.Coin,
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
		wpBalance:                  wpBalance,
		challengePoolIntegralValue: int64(challengePoolIntegralValue),
		challengePoolBalance:       int64(challengePoolBalance),
		partial:                    partial,
		previousChallange:          previous,
		thisChallange:              thisChallange,
		thisExpires:                thisExpires,
		now:                        now,
	}

	var txn, ssc, allocation, challenge, details, ctx = setupChallengeMocks(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
		wpBalance, challengePoolIntegralValue,
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
	wpBalance currency.Coin,
	challengePoolIntegralValue, challengePoolBalance currency.Coin,
	thisChallange, thisExpires, now common.Timestamp,
	size int64,
) (*transaction.Transaction, *StorageSmartContract, *StorageAllocation,
	*AllocationChallenges, *BlobberAllocation, *mockStateContext) {
	require.Len(t, validatorStakes, len(validators))

	var err error
	var allocation = &StorageAllocation{
		ID:         "alice",
		Owner:      "owin",
		Expiration: thisExpires,
		TimeUnit:   scYaml.TimeUnit,
		WritePool:  currency.Coin(wpBalance),
	}
	var allocChallenges = &AllocationChallenges{
		AllocationID: encryption.Hash("alloc_challenges_id"),
		LatestCompletedChallenge: &StorageChallenge{
			Created: thisChallange,
		},
	}
	var details = &BlobberAllocation{
		BlobberID:                  blobberId,
		ChallengePoolIntegralValue: challengePoolIntegralValue,
		Terms: Terms{
			WritePrice: zcnToBalance(blobberYaml.writePrice),
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

	var sp = newStakePool()
	sp.Settings.ServiceChargeRatio = blobberYaml.serviceCharge
	for i, stake := range stakes {
		var id = strconv.Itoa(i)
		sp.Pools["paula"+id] = &stakepool.DelegatePool{}
		sp.Pools["paula"+id].Balance = currency.Coin(stake)
		sp.Pools["paula"+id].DelegateID = "delegate " + id
	}
	sp.Settings.DelegateWallet = blobberId + " wallet"
	require.NoError(t, sp.save(ssc.ID, blobberId, ctx))

	var validatorsSPs []*stakePool
	for i, validator := range validators {
		var sPool = newStakePool()
		sPool.Settings.ServiceChargeRatio = validatorYamls[i].serviceCharge
		for j, stake := range validatorStakes[i] {
			var pool = &stakepool.DelegatePool{}
			pool.Balance = currency.Coin(stake)
			var id = validator + " delegate " + strconv.Itoa(j)
			sPool.Pools[id] = pool
		}
		sPool.Settings.DelegateWallet = validator + " wallet"
		validatorsSPs = append(validatorsSPs, sPool)
	}
	require.NoError(t, ssc.saveStakePools(validators, validatorsSPs, ctx))

	_, err = ctx.InsertTrieNode(scConfigKey(ssc.ID), &scYaml)
	require.NoError(t, err)

	return txn, ssc, allocation, allocChallenges, details, ctx
}

type formulaeBlobberReward struct {
	t                                                  *testing.T
	scYaml                                             Config
	blobberYaml                                        mockBlobberYaml
	validatorYamls                                     []mockBlobberYaml
	stakes                                             []int64
	validators                                         []string
	validatorStakes                                    [][]int64
	wpBalance                                          currency.Coin
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
	if len(f.stakes) == 0 {
		return f.blobberReward()
	}

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
