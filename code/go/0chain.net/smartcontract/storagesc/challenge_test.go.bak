package storagesc

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/config"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/statecache"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool"
	"github.com/0chain/common/core/util"
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

	type challengeAdd struct {
		blobberID string
		ts        common.Timestamp
	}
	type blobberTS struct {
		blobberID string
		ts        common.Timestamp
	}
	type parameters struct {
		blobberID    string
		allocID      string
		challengesTS []blobberTS
		add          challengeAdd
		cct          int64
		challInfo    *StorageChallengeResponse
	}

	type args struct {
		balances        cstate.StateContextI
		alloc           *StorageAllocation
		allocChallenges *AllocationChallenges
	}

	type want struct {
		openChallengeNum int
		openDelta        map[string]int64
		events           []event.Event
		error            bool
		errorMsg         string
	}

	var (
		allocID   = "alloc_1"
		allocRoot = "alloc_root"
	)

	parepareSSCArgs := func(t *testing.T, p parameters) (*StorageSmartContract, args) {
		ssc := &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		config.SmartContractConfig.SetDefault("smart_contracts.storagesc.max_challenge_completion_rounds", p.cct)

		balances := &mockStateContext{
			store: make(map[datastore.Key]util.MPTSerializable),
		}

		challengeReadyParts, err := partitions.CreateIfNotExists(
			balances,
			ALL_CHALLENGE_READY_BLOBBERS_KEY,
			allChallengeReadyBlobbersPartitionSize)
		require.NoError(t, err)

		allocChallenges, err := ssc.getAllocationChallenges(allocID, balances)
		if err != nil && errors.Is(err, util.ErrValueNotPresent) {
			allocChallenges = new(AllocationChallenges)
			allocChallenges.AllocationID = allocID
		}

		sa := &StorageAllocation{}
		sa.SetEntity(&storageAllocationV2{
			ID:            allocID,
			BlobberAllocs: make([]*BlobberAllocation, 0, len(p.challengesTS)),

			BlobberAllocsMap: make(map[string]*BlobberAllocation),
			Stats:            &StorageAllocationStats{},
		})

		alloc := sa.mustBase()

		for _, bts := range p.challengesTS {
			bid := bts.blobberID
			ts := bts.ts
			ba := &BlobberAllocation{
				BlobberID:      bid,
				AllocationRoot: "root " + bid,
				Stats:          &StorageAllocationStats{},
				Terms:          Terms{},
			}
			alloc.BlobberAllocs = append(alloc.BlobberAllocs, ba)

			alloc.BlobberAllocsMap[bid] = ba

			err = challengeReadyParts.Add(
				balances,
				&ChallengeReadyBlobber{
					BlobberID: bid,
				})

			c := &StorageChallenge{
				ID:              fmt.Sprintf("%s:%s:%d", allocID, bid, ts),
				AllocationID:    allocID,
				BlobberID:       bid,
				TotalValidators: 1,
				Created:         ts,
			}

			challInfo := &StorageChallengeResponse{
				StorageChallenge: c,
				AllocationRoot:   alloc.BlobberAllocsMap[bid].AllocationRoot,
			}
			var conf = &Config{
				MaxChallengeCompletionRounds: p.cct,
			}
			err = ssc.addChallenge(sa, c, allocChallenges, challInfo, conf, balances)
			require.NoError(t, err)
		}

		return ssc, args{
			alloc:           sa,
			allocChallenges: allocChallenges,
			balances:        balances,
		}
	}

	newChallenge := func(allocID, blobberID string, ts common.Timestamp) (*StorageChallenge, *StorageChallengeResponse) {
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
				cct: 720,
				add: challengeAdd{"blobber_1", common.Timestamp(10)},
			},
			want: want{
				openChallengeNum: 1,
				openDelta: map[string]int64{
					"blobber_1": 1,
				},
			},
		},
		{
			name: "OK - more than one open challenges",
			parameters: parameters{
				cct: 720,
				challengesTS: []blobberTS{
					{"blobber_1", 10},
					{"blobber_2", 20},
				},
				add: challengeAdd{"blobber_1", common.Timestamp(30)},
			},
			want: want{
				openChallengeNum: 3,
				openDelta: map[string]int64{
					"blobber_1": 1,
					"blobber_2": 0,
				},
			},
		},
		{
			name: "OK - one challenge expired",
			parameters: parameters{
				cct: 720,
				challengesTS: []blobberTS{
					{"blobber_1", 10},
					{"blobber_2", 20},
				},
				add: challengeAdd{"blobber_1", common.Timestamp(110)},
			},
			want: want{
				openChallengeNum: 2,
				openDelta: map[string]int64{
					"blobber_1": 0,
					"blobber_2": 0,
				},
			},
		},
		{
			name: "OK - more challenges expired, multiple blobbers",
			parameters: parameters{
				cct: 720,
				challengesTS: []blobberTS{
					{"blobber_1", 10},
					{"blobber_2", 20},
					{"blobber_2", 25},
					{"blobber_2", 30},
					{"blobber_3", 30},
				},
				add: challengeAdd{"blobber_1", common.Timestamp(130)},
			},
			want: want{
				openChallengeNum: 1,
				openDelta: map[string]int64{
					"blobber_1": 1,
					"blobber_2": -3,
					"blobber_3": -1,
				},
			},
		},
		{
			name: "Error challenge blobber ID is empty",
			parameters: parameters{
				add: challengeAdd{"blobber_1", common.Timestamp(-1)},
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
			es := args.balances.GetEvents()
			initESLen := len(es)

			alloc := args.alloc.mustBase()

			// add new challenge
			c, challInfo := newChallenge(alloc.ID, tt.parameters.add.blobberID, tt.parameters.add.ts)
			var conf = &Config{
				MaxChallengeCompletionRounds: tt.parameters.cct,
			}
			err := ssc.addChallenge(args.alloc,
				c,
				args.allocChallenges,
				challInfo,
				conf,
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
			var sa StorageAllocation
			err = args.balances.GetTrieNode(sa.GetKey(ssc.ID), &sa)
			require.NoError(t, err)
			alloc = sa.mustBase()

			// assert the open challenge stats is updated
			ba, ok := alloc.BlobberAllocsMap[challenge.BlobberID]
			require.True(t, ok)
			require.Equal(t, int64(tt.want.openChallengeNum), ba.Stats.OpenChallenges)
			require.Equal(t, int64(tt.want.openChallengeNum), alloc.Stats.OpenChallenges)

			// assert the AllocationChallenges that stores open challenges is saved
			var ac AllocationChallenges
			ac.AllocationID = alloc.ID
			err = args.balances.GetTrieNode(ac.GetKey(ssc.ID), &ac)
			require.NoError(t, err)

			// assert the open challenge number is correct
			require.Equal(t, tt.want.openChallengeNum, len(ac.OpenChallenges))

			// assert the open challenge update events are emitted
			es = args.balances.GetEvents()[initESLen:]
			updateOpenChallengeEventMap := make(map[string]int64)

			for bid, od := range tt.want.openDelta {
				if od == 0 {
					// asser there's no event emitted for unchanged open challenges stats
					_, ok := updateOpenChallengeEventMap[bid]
					require.False(t, ok)
					continue
				}
				require.Equal(t, od, updateOpenChallengeEventMap[bid])
			}
		})
	}
}

func TestBlobberReward(t *testing.T) {

	var stakes = []int64{200, 234234, 100000}
	var challengePoolIntegralValue = currency.Coin(73000000)
	var challengePoolBalance = currency.Coin(730000000000)
	var partial = 1.0
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
		ValidatorReward:              0.025,
		MaxChallengeCompletionRounds: 720,
		TimeUnit:                     720 * time.Hour,
	}
	var blobberYaml = mockBlobberYaml{
		serviceCharge: 0.30,
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
		var thisChallenge = thisExpires + 1
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errLate)
	})

	t.Run("test old challenge", func(t *testing.T) {
		var thisChallenge = previousChallenge - 1
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), "old challenge response on blobber rewarding")
	})

	t.Run(errTokensChallengePool, func(t *testing.T) {
		var challengePoolBalance = currency.Coin(0)
		err := testBlobberReward(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errTokensChallengePool))
	})

	t.Run("Setting Validator reward ratio to 100%", func(t *testing.T) {
		newSCYaml := scYaml
		newSCYaml.ValidatorReward = 1
		err := testBlobberReward(t, newSCYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run("uint64 minus overflow", func(t *testing.T) {
		newSCYaml := scYaml
		newSCYaml.ValidatorReward = 2
		err := testBlobberReward(t, newSCYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), "uint64 minus overflow")
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

func TestPopulateGenerateChallenge(t *testing.T) {
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, true)
		err      error
	)

	conf, err := GetConfig(balances)
	require.NoError(t, err)

	blobberAllocationPartitionSize = 3
	blobberRewardsPartitionSize = 3
	allChallengeReadyBlobbersPartitionSize = 3
	allValidatorsPartitionSize = 3

	defer func() {
		blobberAllocationPartitionSize = 10
		blobberRewardsPartitionSize = 5
		allChallengeReadyBlobbersPartitionSize = 50
		allValidatorsPartitionSize = 50
	}()

	preparePopulateGenerateChallenge(t, ssc, balances)
	require.NoError(t, err)

	var challenged map[string]map[string]int64

	var challengedAllocations, challengedBlobbers, challengedValidators map[string]int64
	challenged = make(map[string]map[string]int64)
	challengedAllocations = make(map[string]int64)
	challengedBlobbers = make(map[string]int64)
	challengedValidators = make(map[string]int64)

	numChallenges := 200000
	for i := 0; i < numChallenges; i++ {
		client := newClient(2000*x10, balances)
		txn := newTransaction(client.id, ADDRESS, 100, 1)

		validators, err := getValidatorsList(balances)
		require.NoError(t, err)

		_, partsWeight, err := partitionsChallengeReadyBlobbers(balances)
		require.NoError(t, err)

		hashSeed := encryption.Hash(txn.Hash + "txn.PrevHash")
		// the "1" was the index when generating multiple challenges.
		// keep it in case we need to generate more than 1 challenge at once.
		challengeID := encryption.Hash(hashSeed + "1")

		seedSource, err := strconv.ParseUint(challengeID[0:16], 16, 64)
		require.NoError(t, err)

		result, err := ssc.populateGenerateChallenge(
			partsWeight,
			int64(seedSource),
			validators,
			txn,
			challengeID,
			balances,
			conf.ValidatorsPerChallenge,
			conf,
		)
		require.NoError(t, err)
		require.NotNil(t, result)

		if _, ok := challenged[result.storageChallenge.BlobberID]; !ok {
			challenged[result.storageChallenge.BlobberID] = make(map[string]int64)
		}
		if _, ok := challenged[result.storageChallenge.BlobberID][result.storageChallenge.AllocationID]; !ok {
			challenged[result.storageChallenge.BlobberID][result.storageChallenge.AllocationID] = 0
		}
		challenged[result.storageChallenge.BlobberID][result.storageChallenge.AllocationID]++

		challengedAllocations[result.storageChallenge.AllocationID]++
		challengedBlobbers[result.storageChallenge.BlobberID]++

		for _, v := range result.storageChallenge.ValidatorIDs {
			challengedValidators[v]++
		}
	}

	for _, count := range challengedValidators {
		require.InEpsilon(t, float64(numChallenges*3)/float64(len(challengedValidators)), float64(count), 0.05)
	}

	_, partsWeight, err := partitionsChallengeReadyBlobbers(balances)
	require.NoError(t, err)

	pws := partsWeight.partWeights

	totalWeight := pws.totalWeight()

	for pidx, _ := range pws.Parts {
		if err := partsWeight.iterBlobberWeight(balances, pidx,
			func(id string, bw *ChallengeReadyBlobber) (stop bool) {
				totalChallengesForBlobber := challengedBlobbers[bw.BlobberID]
				require.InEpsilon(t, float64(numChallenges)*(float64(bw.GetWeightV2())/float64(totalWeight)), float64(challengedBlobbers[bw.BlobberID]), 0.05)
				totalAllocationsChallengesForBlobber := int64(len(challenged[bw.BlobberID]))
				for _, count := range challenged[bw.BlobberID] {
					require.InEpsilon(t, totalChallengesForBlobber/totalAllocationsChallengesForBlobber, count, 0.20)
				}
				return false
			}); err != nil {
		}
	}
}

func TestCompleteRewardFlow(t *testing.T) {
	tt := []struct {
		name                      string
		ticketNum                 int
		hasDuplicateTicket        bool
		hasNonceSelectedValidator bool
		wrongClientID             bool
		numChallenges             int64
		ignoreChallengeRange      []int64
		errors                    []error
	}{
		{
			name:          "ok",
			ticketNum:     10,
			numChallenges: 20,
		},
		{
			name:                 "expired middle challenges",
			ticketNum:            10,
			numChallenges:        10,
			ignoreChallengeRange: []int64{2, 8},
		},
		{
			name:                 "never return response",
			ticketNum:            10,
			numChallenges:        10,
			ignoreChallengeRange: []int64{2, 9},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ssc, balances, tp, sa, blobberClients, valids, validators, blobbers, client := prepareAllocChallengesForCompleteRewardFlow(t, 10)

			alloc := sa.mustBase()

			totalExpectedReward := int64(0)
			totalPaidReward := int64(0)
			totalPenalty := int64(0)
			settledChallengePassed := int64(0)

			if tc.name == "never return response" {
				settledChallengePassed = 3
			}

			conf, err := getConfig(balances)
			require.NoError(t, err)

			totalOpenChallengesLeftInAllocation := int64(0)

			for idx := 0; idx < len(blobberClients); idx++ {
				bk := &block.Block{}
				bk.Round = 50000
				balances.setBlock(t, bk)

				blobberClient := blobberClients[idx]
				blobber := blobbers[idx]

				blobberSP, err := ssc.getStakePool(spenum.Blobber, blobber.Id(), balances)
				require.NoError(t, err)
				require.NotNil(t, blobberSP)

				var validatorString []string
				for _, v := range valids {
					validatorString = append(validatorString, v.id)
				}

				step := int64(alloc.Expiration) - tp
				initialTime := tp

				var generatedChallenges []string
				lastFinalizedChallenge := alloc.StartTime
				lastSuccessfulChallenge := alloc.StartTime
				lastVerifiedChallengeRound := balances.GetBlock().Round

				collectedBlobberReward := currency.Coin(0)
				totalExpectedReward += int64(alloc.BlobberAllocs[idx].ChallengePoolIntegralValue)
				lastChallengeIgnored := false

				for i := int64(0); i < tc.numChallenges; i++ {
					// Challenge Timing details
					bk.Round += 200
					balances.setBlock(t, bk)
					currentRound := balances.GetBlock().Round

					challengeGenerationTime := initialTime + (step*(i+1))/tc.numChallenges

					challID := fmt.Sprintf("%s-chall-%d", blobber.Id(), i)

					challengeRoundCreatedAt := currentRound - 200

					// Setting allocations details

					sa, err = ssc.getAllocation(alloc.ID, balances)
					require.NoError(t, err)
					alloc = sa.mustBase()

					blobberAlloc := alloc.BlobberAllocs[idx]

					cp, err := ssc.getChallengePool(alloc.ID, balances)
					require.NoError(t, err)

					cpBalance, _ := cp.Balance.Int64()

					var f = formulaeBlobberReward{
						t:           t,
						scYaml:      *conf,
						blobberYaml: blobberYaml,
						validatorYamls: []mockBlobberYaml{
							{serviceCharge: 0.2}, {serviceCharge: 0.25}, {serviceCharge: 0.3}, {serviceCharge: 0.35}, {serviceCharge: 0.4}, {serviceCharge: 0.45}, {serviceCharge: 0.5}, {serviceCharge: 0.55}, {serviceCharge: 0.6}, {serviceCharge: 0.65},
						},
						stakes:     []int64{10 * x10},
						validators: validatorString,
						validatorStakes: [][]int64{
							{1},
							{1},
							{1},
							{1},
							{1},
							{1},
							{1},
							{1},
							{1},
							{1},
						},
						wpBalance:                  alloc.WritePool,
						challengePoolIntegralValue: int64(blobberAlloc.ChallengePoolIntegralValue),
						challengePoolBalance:       cpBalance,
						partial:                    1,
						lastFinalizedChallenge:     lastFinalizedChallenge,
						lastSuccessfulChallenge:    lastSuccessfulChallenge,
						thisChallange:              common.Timestamp(challengeGenerationTime),
						thisExpires:                alloc.Expiration,
						now:                        common.Timestamp(challengeGenerationTime + 10),
						collectedReward:            collectedBlobberReward,
						size:                       blobberAlloc.Size,
					}

					allocChallenges, err := ssc.getAllocationChallenges(alloc.ID, balances)
					if err != nil {
						require.Equal(t, util.ErrValueNotPresent, err)
						allocChallenges = &AllocationChallenges{}
						allocChallenges.AllocationID = alloc.ID
					}

					countExpiredChallenges, err := alloc.removeExpiredChallenges(allocChallenges, conf.MaxChallengeCompletionRounds, balances, ssc)
					require.NoError(t, err)
					if currentRound-lastVerifiedChallengeRound > conf.MaxChallengeCompletionRounds {
						roundDiff := currentRound - lastVerifiedChallengeRound - conf.MaxChallengeCompletionRounds
						expectedOpenChallenges := roundDiff / 200
						if expectedOpenChallenges > 1 {
							expectedOpenChallenges = 1
						}
						require.Equal(t, int(expectedOpenChallenges), countExpiredChallenges)
					} else {
						require.Equal(t, 0, countExpiredChallenges)
					}
					// Save the allocation challenges to MPT
					err = allocChallenges.Save(balances, ssc.ID)
					require.NoError(t, err)

					sa.mustUpdateBase(func(base *storageAllocationBase) error {
						alloc.deepCopy(base)
						return nil
					})
					err = sa.save(balances, ssc.ID)
					require.NoError(t, err)

					genChall(t, ssc, challengeGenerationTime, challengeRoundCreatedAt, challID, 0, validators, alloc.ID, blobber, balances)
					generatedChallenges = append(generatedChallenges, challID)
					lastFinalizedChallenge = common.Timestamp(challengeGenerationTime)

					allocChallenges, err = ssc.getAllocationChallenges(alloc.ID, balances)
					require.NoError(t, err)

					if tc.ignoreChallengeRange != nil && (i >= tc.ignoreChallengeRange[0] && i <= tc.ignoreChallengeRange[1]+1) {
						expectedOpenChallenges := int64(math.Min(float64(i-tc.ignoreChallengeRange[0]+1), float64(3)))
						expectedOpenChallenges += totalOpenChallengesLeftInAllocation
						require.Equal(t, expectedOpenChallenges, int64(len(allocChallenges.OpenChallenges)))
					} else {
						require.Equal(t, totalOpenChallengesLeftInAllocation+1, int64(len(allocChallenges.OpenChallenges)))
					}

					chall := &ChallengeResponse{
						ID: challID,
					}

					for i := 0; i < tc.ticketNum; i++ {
						chall.ValidationTickets = append(chall.ValidationTickets,
							valids[i].validTicket(t, chall.ID, blobberClient.id, true, challengeGenerationTime))
					}

					if tc.hasDuplicateTicket {
						chall.ValidationTickets[0] = chall.ValidationTickets[1]
					}

					if tc.hasNonceSelectedValidator {
						tp += 10
						var newValids []*Client
						newValids, tp = testAddValidators(t, balances, ssc, 1, challengeGenerationTime)
						// replace the last ticket with the new none selected validator
						chall.ValidationTickets[len(chall.ValidationTickets)-1] = newValids[0].validTicket(t, chall.ID, blobberClient.id, true, tp)
					}

					var tx *transaction.Transaction
					if tc.wrongClientID {
						tx = newTransaction(blobberAlloc.BlobberID, ssc.ID, 0, challengeGenerationTime)
					} else {
						tx = newTransaction(blobberClient.id, ssc.ID, 0, challengeGenerationTime)
					}

					balances.setTransaction(t, tx)

					if tc.ignoreChallengeRange != nil {
						if i >= tc.ignoreChallengeRange[0] && i <= tc.ignoreChallengeRange[1] {
							lastChallengeIgnored = true
							continue
						} else if i == tc.ignoreChallengeRange[1]+2 {
							lastChallengeIgnored = false
						}
					}

					var resp string
					resp, err = ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
					if tc.errors != nil {
						require.Equal(t, tc.errors[i], err)
						continue
					}
					lastSuccessfulChallenge = common.Timestamp(challengeGenerationTime)
					lastVerifiedChallengeRound = challengeRoundCreatedAt

					require.NoError(t, err)
					require.Equal(t, "challenge passed by blobber", resp)

					cp, err = ssc.getChallengePool(alloc.ID, balances)
					require.NoError(t, err)

					vsp, err := ssc.validatorsStakePools(validatorString, balances)
					require.NoError(t, err)

					blobberSP, err := ssc.getStakePool(spenum.Blobber, blobber.Id(), balances)
					require.NoError(t, err)

					if lastChallengeIgnored {
						blobberReward, validatorsReward, penaltyPaid := confirmBlobberPenalty(t, f, *cp, vsp, *blobberSP, true)
						totalPaidReward += blobberReward + validatorsReward - penaltyPaid
						totalPenalty += penaltyPaid
					} else {
						totalPaidReward += confirmBlobberReward(t, f, *cp, vsp, *blobberSP)
					}

					collectedBlobberReward = blobberSP.Reward
				}

				sa, err = ssc.getAllocation(alloc.ID, balances)
				require.NoError(t, err)
				alloc = sa.mustBase()

				//bk.Round += 200000
				//balances.setBlock(t, bk)
				//
				allocChallenges, err := ssc.getAllocationChallenges(alloc.ID, balances)
				if err != nil {
					require.Equal(t, util.ErrValueNotPresent, err)
					allocChallenges = &AllocationChallenges{}
					allocChallenges.AllocationID = alloc.ID
				}
				totalOpenChallengesLeftInAllocation = int64(len(allocChallenges.OpenChallenges))
				//lenOpenChallenges := len(allocChallenges.OpenChallenges)
				//countExpiredChallenges, err := alloc.removeExpiredChallenges(allocChallenges, conf.MaxChallengeCompletionRounds, balances, ssc)
				//require.NoError(t, err)
				//require.Equal(t, lenOpenChallenges, countExpiredChallenges)
				//require.Equal(t, 0, len(allocChallenges.OpenChallenges))
				//
				// Save the allocation challenges to MPT
				err = allocChallenges.Save(balances, ssc.ID)
				require.NoError(t, err)

				sa.mustUpdateBase(func(base *storageAllocationBase) error {
					alloc.deepCopy(base)
					return nil
				})
				err = sa.save(balances, ssc.ID)
				require.NoError(t, err)
			}

			beforeBlobberSPs := make(map[string]*stakePool)
			for _, blobber := range blobbers {
				beforeBlobberSPs[blobber.Id()], err = ssc.getStakePool(spenum.Blobber, blobber.Id(), balances)
				require.NoError(t, err)
			}

			beforeBlobberAllocs := alloc.BlobberAllocs

			var req lockRequest
			req.AllocationID = alloc.ID

			allocOwnerBalanceBefore, err := balances.GetClientBalance(alloc.Owner)
			require.NoError(t, err)
			var tx = newTransaction(client.id, ssc.ID, 0, int64(alloc.Expiration)+2)
			balances.setTransaction(t, tx)
			sa, err = ssc.finalizeAllocationInternal(tx, mustEncode(t, &req), balances)
			require.NoError(t, err)
			alloc = sa.mustBase()

			_, err = ssc.getChallengePool(alloc.ID, balances)
			require.Error(t, err, "challenge pool should be deleted")

			totalReturnedReward := int64(0)

			for idx, blobberAlloc := range alloc.BlobberAllocs {
				beforeBlobberSP, ok := beforeBlobberSPs[blobberAlloc.BlobberID]
				require.True(t, ok)

				blobberSP, err := ssc.getStakePool(spenum.Blobber, blobberAlloc.BlobberID, balances)
				require.NoError(t, err)
				require.NotNil(t, blobberSP)

				require.Equal(t, 0, int(blobberSP.TotalOffers))

				require.Equal(t, tc.numChallenges, blobberAlloc.Stats.TotalChallenges)
				if tc.ignoreChallengeRange != nil {
					require.Equal(t, tc.numChallenges-tc.ignoreChallengeRange[1]+tc.ignoreChallengeRange[0]-1+settledChallengePassed, blobberAlloc.Stats.SuccessChallenges)
					require.Equal(t, tc.ignoreChallengeRange[1]-tc.ignoreChallengeRange[0]+1-settledChallengePassed, blobberAlloc.Stats.FailedChallenges)
				} else {
					require.Equal(t, tc.numChallenges, blobberAlloc.Stats.SuccessChallenges)
					require.Equal(t, int64(0), blobberAlloc.Stats.FailedChallenges)
				}
				require.Equal(t, int64(0), blobberAlloc.Stats.OpenChallenges)

				var f = formulaeBlobberReward{
					t:           t,
					scYaml:      *conf,
					blobberYaml: blobberYaml,
					validatorYamls: []mockBlobberYaml{
						{serviceCharge: 0.2}, {serviceCharge: 0.25}, {serviceCharge: 0.3}, {serviceCharge: 0.35}, {serviceCharge: 0.4}, {serviceCharge: 0.45}, {serviceCharge: 0.5}, {serviceCharge: 0.55}, {serviceCharge: 0.6}, {serviceCharge: 0.65},
					},
					stakes:                     []int64{10 * x10},
					validators:                 []string{},
					validatorStakes:            [][]int64{},
					wpBalance:                  alloc.WritePool,
					challengePoolIntegralValue: int64(beforeBlobberAllocs[idx].ChallengePoolIntegralValue),
					challengePoolBalance:       0,
					partial:                    1,
					lastFinalizedChallenge:     blobberAlloc.LatestFinalizedChallCreatedAt,
					lastSuccessfulChallenge:    blobberAlloc.LatestSuccessfulChallCreatedAt,
					thisChallange:              alloc.Expiration,
					thisExpires:                alloc.Expiration,
					now:                        common.Timestamp(tp + 10),
					collectedReward:            beforeBlobberSP.Reward,
					size:                       blobberAlloc.Size,
				}

				finalBlobberReward := int64(0)
				passRate := float64(blobberAlloc.Stats.SuccessChallenges) / float64(blobberAlloc.Stats.TotalChallenges)

				var expectedCancellationCharge int64

				cancellationCharge, _ := alloc.cancellationCharge(conf.CancellationCharge)

				usedWritePool := alloc.MovedToChallenge - alloc.MovedBack

				if usedWritePool < cancellationCharge {
					cancellationCharge = cancellationCharge - usedWritePool

					if alloc.WritePool < cancellationCharge {
						cancellationCharge = alloc.WritePool
					}

					expectedCancellationCharge = int64(float64(cancellationCharge)*passRate) / int64(len(alloc.BlobberAllocs))
				} else {
					expectedCancellationCharge = 0
				}

				if blobberAlloc.LatestSuccessfulChallCreatedAt < blobberAlloc.LatestFinalizedChallCreatedAt {
					blobberReward, penaltyPaid := confirmBlobberPenaltyOnFinalization(t, f, int64(blobberAlloc.ChallengePoolIntegralValue), *blobberSP, true, passRate, expectedCancellationCharge)
					totalPaidReward += blobberReward - penaltyPaid
					totalPenalty += penaltyPaid
					finalBlobberReward = blobberReward
				} else {
					blobberReward := confirmBlobberRewardOnFinalization(t, f, *blobberSP, passRate, int64(blobberAlloc.ChallengePoolIntegralValue), expectedCancellationCharge)
					totalPaidReward += blobberReward
					finalBlobberReward = blobberReward
				}

				totalReturnedReward += int64(blobberAlloc.ChallengePoolIntegralValue)

				require.InDelta(t, finalBlobberReward, int64(blobberAlloc.ChallengePoolIntegralValue), errDelta)
			}

			// wp will be returned to the owner
			allocOwnerBalance, err := balances.GetClientBalance(alloc.Owner)
			require.NoError(t, err)
			refund := allocOwnerBalance - allocOwnerBalanceBefore

			passRate := float64(alloc.Stats.SuccessChallenges) / float64(alloc.Stats.TotalChallenges)

			var expectedTotalCancellationCharge int64

			cancellationCharge, _ := alloc.cancellationCharge(conf.CancellationCharge)

			usedWritePool := alloc.MovedToChallenge - alloc.MovedBack

			if usedWritePool < cancellationCharge {
				cancellationCharge = cancellationCharge - usedWritePool

				if alloc.WritePool < cancellationCharge {
					cancellationCharge = alloc.WritePool
				}

				expectedTotalCancellationCharge = int64(float64(cancellationCharge)*passRate) / int64(len(alloc.BlobberAllocs))
			} else {
				expectedTotalCancellationCharge = 0
			}

			require.InDelta(t, totalExpectedReward, totalPaidReward+2*totalPenalty+totalReturnedReward, 50)
			require.InDelta(t, int64(refund), 1000*x10-int64(alloc.MovedToChallenge)+totalPenalty-expectedTotalCancellationCharge+totalReturnedReward, 50)
		})
	}
}

func TestRollBack(t *testing.T) {
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, true)
		client   = newClient(2000*x10, balances)
		tp       = int64(0)

		// no owner
		err error
	)

	// new allocation
	tp += 1000
	var allocID, blobs = addAllocation(t, ssc, client, tp, 0, 0, 0, 0, 0, balances, false, false, false)

	var sa *StorageAllocation
	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	blobberClient := testGetBlobber(blobs, alloc, 0)
	require.NotNil(t, blobberClient)

	wpBalance := alloc.WritePool

	_, tp = testCommitWrite(t, balances, client, allocID, "root-1", 100*1024*1024, tp, blobberClient.id, ssc, "")

	cp, err := ssc.getChallengePool(allocID, balances)
	require.NoError(t, err)
	require.Equal(t, 4882812500, int(cp.Balance))

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()
	require.Equal(t, wpBalance-4882812500, alloc.WritePool)
	wpBalance = alloc.WritePool

	_, tp = testCommitWrite(t, balances, client, allocID, "", 0, tp, blobberClient.id, ssc, "")

	cp, err = ssc.getChallengePool(allocID, balances)
	require.NoError(t, err)
	require.Equal(t, 0, int(cp.Balance))

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()
	require.Equal(t, wpBalance+4882812500, alloc.WritePool)
	wpBalance = alloc.WritePool

	_, tp = testCommitWrite(t, balances, client, allocID, "root-2", 100*1024*1024, tp, blobberClient.id, ssc, "")

	cp, err = ssc.getChallengePool(allocID, balances)
	require.NoError(t, err)
	require.Equal(t, 4882812500, int(cp.Balance))

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()
	require.Equal(t, wpBalance-4882812500, alloc.WritePool)
	wpBalance = alloc.WritePool

	_, tp = testCommitWrite(t, balances, client, allocID, "root-3", -100*1024*1024, tp, blobberClient.id, ssc, "root-2")

	cp, err = ssc.getChallengePool(allocID, balances)
	require.NoError(t, err)
	require.Equal(t, 0, int(cp.Balance))

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()
	require.Equal(t, wpBalance+4882812500, alloc.WritePool)
	wpBalance = alloc.WritePool

	_, tp = testCommitWrite(t, balances, client, allocID, "root-2", 0, tp, blobberClient.id, ssc, "root-2")

	cp, err = ssc.getChallengePool(allocID, balances)
	require.NoError(t, err)
	require.Equal(t, 4882812500, int(cp.Balance))

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()
	require.Equal(t, wpBalance-4882812500, alloc.WritePool)
	wpBalance = alloc.WritePool
}

func TestBlobberPenalty(t *testing.T) {
	var stakes = []int64{200, 234234, 100000}
	var challengePoolIntegralValue = currency.Coin(73000000)
	var challengePoolBalance = currency.Coin(7000000000)
	var partial = 0.9
	var previousChallenge = common.Timestamp(3)
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
		BlobberSlash:                 0.1,
		ValidatorReward:              0.025,
		MaxChallengeCompletionRounds: 720,
		TimeUnit:                     720 * time.Hour,
	}
	var blobberYaml = mockBlobberYaml{
		serviceCharge: 0.30,
		writePrice:    1,
	}
	var validatorYamls = []mockBlobberYaml{
		{serviceCharge: 0.2}, {serviceCharge: 0.25}, {serviceCharge: 0.3},
	}

	t.Run("test blobberPenalty ", func(t *testing.T) {
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, size, previousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run("test blobberPenalty ", func(t *testing.T) {
		var size = int64(10000)
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, size, previousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run(errNoStakePools, func(t *testing.T) {
		var validatorStakes = [][]int64{{45, 666, 4533}, {}, {10}}
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, size, previousChallenge, thisChallenge, thisExpires, now)
		require.NoError(t, err)
	})

	t.Run(errTokensChallengePool, func(t *testing.T) {
		var challengePoolBalance = currency.Coin(0)
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, size, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errTokensChallengePool))
	})
}

func TestVerifyChallenge(t *testing.T) {
	tt := []struct {
		name                      string
		ticketNum                 int
		hasDuplicateTicket        bool
		hasNonceSelectedValidator bool
		wrongClientID             bool
		numChallenges             int
		ignoreChallengeRange      []int
		errors                    []error
	}{
		{
			name:          "ok",
			ticketNum:     10,
			numChallenges: 10,
		},
		{
			name:          "should return expired challenge error",
			ticketNum:     10,
			numChallenges: 1,
			errors:        []error{common.NewError("verify_challenge", "challenge expired")},
		},
		{
			name:                 "expired middle challenges",
			ticketNum:            10,
			numChallenges:        10,
			ignoreChallengeRange: []int{2, 8},
		},
		{
			name:                 "never return response",
			ticketNum:            10,
			numChallenges:        10,
			ignoreChallengeRange: []int{2, 80},
		},
		{
			name:          "old challenge",
			ticketNum:     10,
			numChallenges: 2,
			errors:        []error{nil, common.NewError("verify_challenge", "old challenge response")},
		},
		{
			name:               "duplicate ticket",
			ticketNum:          10,
			hasDuplicateTicket: true,
			numChallenges:      1,
			errors:             []error{common.NewError("verify_challenge", "found duplicate validation tickets")},
		},
		{
			name:          "not enough tickets",
			ticketNum:     4, // threshold is 5
			numChallenges: 1,
			errors:        []error{common.NewError("verify_challenge", "validation tickets less than threshold: 5, tickets: 4")},
		},
		{
			name:                      "ticket signed with unauthorized validator",
			ticketNum:                 5,
			hasNonceSelectedValidator: true,
			numChallenges:             1,
			errors:                    []error{common.NewError("verify_challenge", "found invalid validator id in validation ticket")},
		},
		{
			name:          "wrong txn client id",
			ticketNum:     5,
			wrongClientID: true,
			numChallenges: 1,
			errors:        []error{errors.New("challenge blobber id does not match")},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ssc, balances, tp, sa, b3, valids, validators, blobber, _, _ := prepareAllocChallenges(t, 10)
			alloc := sa.mustBase()
			step := (int64(alloc.Expiration) - tp) / 10
			tp += step / 2

			bk := &block.Block{}
			bk.Round = 50000
			balances.setBlock(t, bk)

			currentRound := balances.GetBlock().Round

			var generatedChallenges []string

			now := tp + 10

			for i := 0; i < tc.numChallenges; i++ {
				challID := fmt.Sprintf("chall-%d", i)

				challengeRoundCreatedAt := currentRound - 10*(20-int64(i))

				if tc.name == "should return expired challenge error" {
					challengeRoundCreatedAt = currentRound - 1000*(20-int64(i))
				} else if tc.name == "old challenge" {
					challengeRoundCreatedAt = currentRound - 100*(int64(i)+1)
					now--
				}

				genChall(t, ssc, now, challengeRoundCreatedAt, challID, 0, validators, alloc.ID, blobber, balances)
				generatedChallenges = append(generatedChallenges, challID)

				allocChallenges, err := ssc.getAllocationChallenges(alloc.ID, balances)
				require.NoError(t, err)

				if tc.ignoreChallengeRange != nil && (i >= tc.ignoreChallengeRange[0] && i <= tc.ignoreChallengeRange[1]+1) {
					require.Equal(t, i-tc.ignoreChallengeRange[0]+1, len(allocChallenges.OpenChallenges))
				} else {
					require.Equal(t, 1, len(allocChallenges.OpenChallenges))
				}

				chall := &ChallengeResponse{
					ID: challID,
				}

				for i := 0; i < tc.ticketNum; i++ {
					chall.ValidationTickets = append(chall.ValidationTickets,
						valids[i].validTicket(t, chall.ID, b3.id, true, tp))
				}

				if tc.hasDuplicateTicket {
					chall.ValidationTickets[0] = chall.ValidationTickets[1]
				}

				if tc.hasNonceSelectedValidator {
					tp += 10
					var newValids []*Client
					newValids, tp = testAddValidators(t, balances, ssc, 1, tp)
					// replace the last ticket with the new none selected validator
					chall.ValidationTickets[len(chall.ValidationTickets)-1] = newValids[0].validTicket(t, chall.ID, b3.id, true, tp)
				}

				var tx *transaction.Transaction
				if tc.wrongClientID {
					tx = newTransaction(alloc.BlobberAllocs[0].BlobberID, ssc.ID, 0, tp)
				} else {
					tx = newTransaction(b3.id, ssc.ID, 0, tp)
				}

				balances.setTransaction(t, tx)

				if tc.ignoreChallengeRange != nil {
					if i >= tc.ignoreChallengeRange[0] && i <= tc.ignoreChallengeRange[1] {
						continue
					}
				}

				var resp string
				resp, err = ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
				if tc.errors != nil {
					require.Equal(t, tc.errors[i], err)
					continue
				}

				require.NoError(t, err)
				require.Equal(t, "challenge passed by blobber", resp)
			}

			if tc.ignoreChallengeRange != nil {
				for i := tc.ignoreChallengeRange[0]; i <= tc.ignoreChallengeRange[1] && i < tc.numChallenges; i++ {
					challID := generatedChallenges[i]

					_, err := ssc.getStorageChallenge(challID, balances)

					if tc.ignoreChallengeRange[1] >= tc.numChallenges-1 {
						require.NoError(t, err)
					} else {
						require.Error(t, err)
					}
				}

				allocChallenges, err := ssc.getAllocationChallenges(alloc.ID, balances)
				require.NoError(t, err)

				if tc.ignoreChallengeRange[1] >= tc.numChallenges-1 {
					require.Equal(t, tc.numChallenges-tc.ignoreChallengeRange[0], len(allocChallenges.OpenChallenges))
				} else {
					require.Equal(t, 0, len(allocChallenges.OpenChallenges))
				}
			}
		})
	}

}

func TestVerifyChallengeOldChallenge(t *testing.T) {
	ssc, balances, tp, sa, b3, valids, validators, blobber, blobbers, _ := prepareAllocChallenges(t, 10)
	alloc := sa.mustBase()
	step := (int64(alloc.Expiration) - tp) / 10

	t.Run("verify challenge first time", func(t *testing.T) {
		challID := fmt.Sprintf("chall-0")
		tp += step / 2

		bk := &block.Block{}
		bk.Round = 500
		balances.setBlock(t, bk)

		currentRound := balances.GetBlock().Round

		genChall(t, ssc, tp, currentRound-100, challID, 0, validators, alloc.ID, blobber, balances)

		chall := &ChallengeResponse{
			ID: challID,
		}

		for i := 0; i < 10; i++ {
			chall.ValidationTickets = append(chall.ValidationTickets,
				valids[i].validTicket(t, chall.ID, b3.id, true, tp))
		}

		tx := newTransaction(b3.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)

		var resp string
		resp, err := ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
		require.NoError(t, err)

		require.Equal(t, resp, "challenge passed by blobber")
	})

	t.Run("same alloc, different blobber, older timestamp, should be ok", func(t *testing.T) {
		var (
			b1       = testGetBlobber(blobbers, alloc, 0)
			challID  = fmt.Sprintf("chall-1")
			blobber1 *StorageNode
		)

		bk := &block.Block{}
		bk.Round = 500
		balances.setBlock(t, bk)

		blobber1, err := ssc.getBlobber(b1.id, balances)
		// reduce timestamp to generate challenge with older create time
		tp := tp - 10
		currentRound := balances.GetBlock().Round

		genChall(t, ssc, tp, currentRound-200, challID, 0, validators, alloc.ID, blobber1, balances)

		chall1 := &ChallengeResponse{
			ID: challID,
		}

		for i := 0; i < 10; i++ {
			chall1.ValidationTickets = append(chall1.ValidationTickets,
				valids[i].validTicket(t, chall1.ID, b1.id, true, tp))
		}

		tx := newTransaction(b1.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		resp, err := ssc.verifyChallenge(tx, mustEncode(t, chall1), balances)
		require.NoError(t, err)
		require.Equal(t, resp, "challenge passed by blobber")
	})

	t.Run("same alloc, same blobber, older timestamp, should fail", func(t *testing.T) {
		b1 := testGetBlobber(blobbers, alloc, 0)

		bk := &block.Block{}
		bk.Round = 500
		balances.setBlock(t, bk)

		challID := fmt.Sprintf("chall-1")
		var blobber1 *StorageNode
		blobber1, err := ssc.getBlobber(b1.id, balances)
		// reduce timestamp to generate challenge with older create time
		tp := tp - 20
		currentRound := balances.GetBlock().Round

		genChall(t, ssc, tp, currentRound-300, challID, 0, validators, alloc.ID, blobber1, balances)

		chall1 := &ChallengeResponse{
			ID: challID,
		}

		for i := 0; i < 10; i++ {
			chall1.ValidationTickets = append(chall1.ValidationTickets,
				valids[i].validTicket(t, chall1.ID, b1.id, true, tp))
		}

		tx := newTransaction(b1.id, ssc.ID, 0, tp)
		balances.setTransaction(t, tx)
		// update block round to ignore the ongoing blobber reward checking

		_, err = ssc.verifyChallenge(tx, mustEncode(t, chall1), balances)
		require.EqualError(t, err, "verify_challenge: old challenge response")
	})
}

func createTxnMPT(mpt util.MerklePatriciaTrieI) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), mpt.GetNodeDB(), false)
	tmpt := util.NewMerklePatriciaTrie(tdb, mpt.GetVersion(), mpt.GetRoot(), statecache.NewEmpty())
	return tmpt
}

func TestVerifyChallengeRunMultipleTimes(t *testing.T) {
	ssc, balances, tp, sa, b3, valids, validators, blobber, _, _ := prepareAllocChallenges(t, 10)
	alloc := sa.mustBase()
	step := (int64(alloc.Expiration) - tp) / 10
	tp += step / 2

	bk := &block.Block{}
	bk.Round = 500
	balances.setBlock(t, bk)

	currentRound := balances.GetBlock().Round

	challID := fmt.Sprintf("chall-0")
	genChall(t, ssc, tp, currentRound-100, challID, 0, validators, alloc.ID, blobber, balances)

	chall := &ChallengeResponse{
		ID: challID,
	}

	for i := 0; i < 10; i++ {
		chall.ValidationTickets = append(chall.ValidationTickets,
			valids[i].validTicket(t, chall.ID, b3.id, true, tp))
	}

	tx := newTransaction(b3.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)

	round := 100

	stateRoots := make(map[string]struct{}, 10)
	for i := 0; i < 20; i++ {
		clientState := createTxnMPT(balances.GetState())

		bk := &block.Block{}
		bk.Round = int64(round)
		balances.setBlock(t, bk)

		cs := cstate.NewStateContext(balances.block, clientState,
			balances.txn, nil, nil, nil, balances.GetSignatureScheme, nil, nil, nil, nil)

		var resp string
		resp, err := ssc.verifyChallenge(tx, mustEncode(t, chall), cs)
		require.NoError(t, err)

		require.Equal(t, resp, "challenge passed by blobber")
		stateRoots[util.ToHex(cs.GetState().GetRoot())] = struct{}{}
	}

	// Assert muultiple verify challenges running would all result in the same state root, i.e. there's only one
	// record in the stateRoots map.
	require.Equal(t, len(stateRoots), 1)
}

func TestGetRandomSubSlice(t *testing.T) {
	const seed = 29
	t.Run("length greater than size", func(t *testing.T) {
		size := 3
		slice := []string{"2", "4", "3", "1"}
		result := getRandomSubSlice(slice, size, seed)
		require.Len(t, result, 3)
	})

	t.Run("length length than size", func(t *testing.T) {
		size := 6
		slice := []string{"2", "4", "3", "1"}
		result := getRandomSubSlice(slice, size, seed)
		require.Len(t, result, 4)
	})

	t.Run("size zero", func(t *testing.T) {
		size := 0
		slice := []string{"2", "4", "3", "1"}
		result := getRandomSubSlice(slice, size, seed)
		require.Len(t, result, 0)
	})

	t.Run("length zero", func(t *testing.T) {
		size := 6
		slice := []string{}
		result := getRandomSubSlice(slice, size, seed)
		require.Len(t, result, 0)
	})

	t.Run("slice nil", func(t *testing.T) {
		size := 6
		var slice []string
		slice = nil
		result := getRandomSubSlice(slice, size, seed)
		require.Len(t, result, 0)
	})
}

func generateRandomNumbers(n, m int) []int {
	if n > m {
		panic("Cannot generate unique numbers if " + strconv.Itoa(n) + " is greater than " + strconv.Itoa(m))
	}

	if n == m {
		numbers := make([]int, 0, n)
		for i := 0; i <= n; i++ {
			numbers = append(numbers, i)
		}
		return numbers
	}

	rand.Seed(time.Now().UnixNano())
	numbers := make([]int, 0, n)
	numMap := make(map[int]bool)

	for len(numbers) <= n {
		num := rand.Intn(m)
		if !numMap[num] {
			numMap[num] = true
			numbers = append(numbers, num)
		}
	}

	return numbers
}

func preparePopulateGenerateChallenge(t *testing.T, ssc *StorageSmartContract, balances *testBalances) {
	// Initialise blobbers and add them to challenge ready partitions
	stake := currency.Coin(1e12)
	used := uint64(1e6)

	var blobbers []*Client

	for i := 0; i < 10; i++ {
		b := addBlobber(t, ssc, 100*GB, 1, avgTerms, 500000*x10, balances, false, false)
		blobbers = append(blobbers, b)

		err := PartitionsChallengeReadyBlobberAddOrUpdate(balances, b.id, stake, used)
		require.NoError(t, err)

		if i%2 == 1 {
			stake *= 2
		}
	}

	// Initialise validators and add them to validator partitions
	testAddValidators(t, balances, ssc, 10, 1)

	// Add 24 allocations for different data and parity shards
	allocDataParities := [][]int{
		{1, 1}, {7, 2}, {6, 4}, {5, 5}, {9, 1}, {1, 9},
	}
	var data, parity int
	for i := 0; i < 24; i++ {
		if i%4 == 0 {
			data = allocDataParities[i/4][0]
			parity = allocDataParities[i/4][1]
		}

		client := newClient(2000*x10, balances)
		var nar = new(newAllocationRequest)
		nar.DataShards = data
		nar.ParityShards = parity
		nar.Owner = client.id
		nar.OwnerPublicKey = client.pk
		nar.ReadPriceRange = PriceRange{1 * x10, 10 * x10}
		nar.WritePriceRange = PriceRange{2 * x10, 20 * x10}
		nar.Size = 1 * GB // 2 GB

		blobberIndexes := generateRandomNumbers(data+parity-1, len(blobbers)-1)
		for _, i := range blobberIndexes {
			nar.Blobbers = append(nar.Blobbers, blobbers[i].id)
			nar.BlobberAuthTickets = append(nar.BlobberAuthTickets, "")
		}

		var resp, err = nar.callNewAllocReq(t, client.id, 1000*x10, ssc, 1,
			balances)
		require.NoError(t, err)

		var deco StorageAllocation
		require.NoError(t, deco.Decode([]byte(resp)))

		sa, err := ssc.getAllocation(deco.mustBase().ID, balances)
		require.NoError(t, err)
		alloc := sa.mustBase()

		for _, b := range alloc.BlobberAllocs {
			b.Stats.UsedSize = int64(1 * GB / data)
			b.AllocationRoot = "root-" + strconv.Itoa(i)

			lastWM := &WriteMarker{}
			lastWm2 := &writeMarkerV2{}
			lastWm2.AllocationRoot = b.AllocationRoot
			lastWm2.ClientID = alloc.Owner
			lastWm2.Timestamp = common.Timestamp(time.Now().Unix())
			lastWm2.Size = b.Stats.UsedSize
			lastWM.SetEntity(lastWm2)

			b.LastWriteMarker = lastWM

			err = partitionsBlobberAllocationsAdd(balances, b.BlobberID, alloc.ID)
			require.NoError(t, err)

			r := rand.New(rand.NewSource(int64(time.Now().Second())))

			// get blobber allocations partitions
			blobberAllocParts, err := partitionsBlobberAllocations(b.BlobberID, balances)
			require.NoError(t, err)

			// get random allocations from the partitions
			var randBlobberAllocs []BlobberAllocationNode
			blobberAllocParts.GetRandomItems(balances, r, &randBlobberAllocs)
		}

		alloc.Stats.UsedSize = 1 * GB

		sa.mustUpdateBase(func(base *storageAllocationBase) error {
			alloc.deepCopy(base)
			return nil
		})
		err = sa.save(balances, ADDRESS)
		require.NoError(t, err)
	}
}

func prepareAllocChallengesForCompleteRewardFlow(t *testing.T, validatorsNum int) (*StorageSmartContract, *testBalances, int64,
	*StorageAllocation, []*Client, []*Client, *partitions.Partitions, []*StorageNode, *Client) {
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, true)
		client   = newClient(2000*x10, balances)
		tp       = int64(0)

		// no owner
		err error
	)

	// new allocation
	tp += 1000
	var allocID, blobs = addAllocation(t, ssc, client, tp, 0, 0, 0, 0, 0, balances, false, false, false)

	// blobbers: stake 10k, balance 40k

	var sa *StorageAllocation
	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	var blobbers []*StorageNode
	var blobberClients []*Client
	for i := 0; i < len(alloc.BlobberAllocs); i++ {
		blobberClient := testGetBlobber(blobs, alloc, i)
		require.NotNil(t, blobberClient)

		_, tp = testCommitWrite(t, balances, client, allocID, "root-1", 100*1024*1024, tp, blobberClient.id, ssc, "")

		blobber, err := ssc.getBlobber(blobberClient.id, balances)
		require.NoError(t, err)

		blobbers = append(blobbers, blobber)
		blobberClients = append(blobberClients, blobberClient)
	}

	// add 10 validators
	valids, tp := testAddValidators(t, balances, ssc, validatorsNum, tp)

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	// load validators
	validators, err := getValidatorsList(balances)
	require.NoError(t, err)

	require.NoError(t, err)
	return ssc, balances, tp, sa, blobberClients, valids, validators, blobbers, client
}

func prepareAllocChallenges(t *testing.T, validatorsNum int) (*StorageSmartContract, *testBalances, int64,
	*StorageAllocation, *Client, []*Client, *partitions.Partitions, *StorageNode, []*Client, *Client) {
	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, true)
		client   = newClient(2000*x10, balances)
		tp       = int64(0)

		// no owner
		reader = newClient(100*x10, balances)
		err    error
	)

	// new allocation
	tp += 1000
	var allocID, blobs = addAllocation(t, ssc, client, tp, 0, 0, 0, 0, 0, balances, false, false, false)

	// blobbers: stake 10k, balance 40k

	var sa *StorageAllocation
	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	b1 := testGetBlobber(blobs, alloc, 0)
	require.NotNil(t, b1)

	// read as owner
	tp = testCommitRead(t, balances, client, client, alloc, b1.id, ssc, tp)

	//read as unauthorized separate user
	tp = testCommitRead(t, balances, client, reader, alloc, b1.id, ssc, tp)

	b2 := testGetBlobber(blobs, alloc, 1)
	require.NotNil(t, b2)

	_, tp = testCommitWrite(t, balances, client, allocID, "root-1", 100*1024*1024, tp, b2.id, ssc, "")

	b3 := testGetBlobber(blobs, alloc, 2)
	require.NotNil(t, b3)

	// add 10 validators
	valids, tp := testAddValidators(t, balances, ssc, validatorsNum, tp)

	//_, err := ssc.getChallengePool(allocID, balances)
	//require.NoError(t, err)

	const allocRoot = "alloc-root-1"

	// write 100MB
	_, tp = testCommitWrite(t, balances, client, allocID, allocRoot, 100*1024*1024, tp, b3.id, ssc, "")

	sa, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc = sa.mustBase()

	// load validators
	validators, err := getValidatorsList(balances)
	require.NoError(t, err)

	// load blobber
	var blobber *StorageNode
	blobber, err = ssc.getBlobber(b3.id, balances)
	require.NoError(t, err)
	return ssc, balances, tp, sa, b3, valids, validators, blobber, blobs, client
}

func testAddValidators(t *testing.T, balances *testBalances, ssc *StorageSmartContract, num int, tp int64) ([]*Client, int64) {
	var valids []*Client
	tp += 1000
	for i := 0; i < num; i++ {
		valids = append(valids, addValidator(t, ssc, tp, balances))
	}
	return valids, tp
}

func testGetBlobber(blobs []*Client, alloc *storageAllocationBase, i int) *Client {
	var bc *Client
	for _, b := range blobs {
		if b.id == alloc.BlobberAllocs[i].BlobberID {
			bc = b
			break
		}
	}
	return bc
}

func testCommitRead(t *testing.T, balances *testBalances, client, reader *Client,
	alloc *storageAllocationBase, blobberID string, ssc *StorageSmartContract, tp int64) int64 {
	tp += 1000
	var rm ReadConnection
	rm.ReadMarker = &ReadMarker{
		ClientID:        reader.id,
		ClientPublicKey: reader.pk,
		BlobberID:       blobberID,
		AllocationID:    alloc.ID,
		OwnerID:         client.id,
		Timestamp:       common.Timestamp(tp),
		ReadCounter:     1 * GB / (64 * KB),
	}
	var err error
	rm.ReadMarker.Signature, err = reader.scheme.Sign(
		encryption.Hash(rm.ReadMarker.GetHashData()))
	require.NoError(t, err)

	tp += 1000
	var tx = newTransaction(blobberID, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
	require.Error(t, err)

	// read pool lock
	tp += 1000

	readPoolFund, err := currency.ParseZCN(float64(len(alloc.BlobberAllocs)) * 2)
	require.NoError(t, err)
	tx = newTransaction(reader.id, ssc.ID, readPoolFund, tp)
	balances.setTransaction(t, tx)
	_, err = ssc.readPoolLock(tx, mustEncode(t, &readPoolLockRequest{
		TargetId: reader.id,
	}), balances)
	require.NoError(t, err)

	var rp *readPool
	rp, err = ssc.getReadPool(reader.id, balances)
	require.NoError(t, err)
	require.EqualValues(t, readPoolFund, int64(rp.Balance))

	// read
	tp += 1000
	tx = newTransaction(blobberID, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
	require.NoError(t, err)
	return tp
}

func testCommitWrite(t *testing.T, balances *testBalances, client *Client, allocID, allocRoot string, size int64, tp int64, blobberID string, ssc *StorageSmartContract, prevAllocRoot string) (*transaction.Transaction, int64) {
	cc := &BlobberCloseConnection{
		AllocationRoot:     allocRoot,
		PrevAllocationRoot: prevAllocRoot,
		WriteMarker:        &WriteMarker{},
	}
	wm1 := &writeMarkerV1{
		AllocationRoot:         allocRoot,
		PreviousAllocationRoot: prevAllocRoot,
		AllocationID:           allocID,
		Size:                   size,
		BlobberID:              blobberID,
		Timestamp:              common.Timestamp(tp),
		ClientID:               client.id,
	}

	var err error
	wm1.Signature, err = client.scheme.Sign(
		encryption.Hash(wm1.GetHashData()))
	require.NoError(t, err)
	cc.WriteMarker.SetEntity(wm1)

	// write
	//tp += 1000
	var tx = newTransaction(blobberID, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	var resp string
	resp, err = ssc.commitBlobberConnection(tx, mustEncode(t, &cc),
		balances)
	require.NoError(t, err)
	require.NotZero(t, resp)
	return tx, tp
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
		lastFinalizedChallenge:     previous,
		lastSuccessfulChallenge:    0,
		size:                       size,
		thisChallange:              thisChallange,
		thisExpires:                thisExpires,
		now:                        now,
	}

	var ssc, allocation, details, ctx = setupChallengeMocks(t, scYaml, blobberYaml, validatorYamls, stakes, validators,
		validatorStakes, wpBalance, challengePoolIntegralValue, challengePoolBalance, thisChallange, thisExpires, now, size)

	alloc := allocation.mustBase()
	err = ssc.blobberPenalty(alloc, 0, previous, details, validators, ctx, allocationId)
	if err != nil {
		return err
	}

	newCP, err := ssc.getChallengePool(alloc.ID, ctx)
	require.NoError(t, err)

	newVSp, err := ssc.validatorsStakePools(validators, ctx)
	require.NoError(t, err)

	afterBlobber, err := ssc.getStakePool(spenum.Blobber, blobberId, ctx)
	require.NoError(t, err)

	confirmBlobberPenalty(t, f, *newCP, newVSp, *afterBlobber, false)

	allocation.mustUpdateBase(func(base *storageAllocationBase) error {
		alloc.deepCopy(base)
		return nil
	})
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
		lastFinalizedChallenge:     previous,
		thisChallange:              thisChallange,
		thisExpires:                thisExpires,
		now:                        now,
	}

	var ssc, allocation, details, ctx = setupChallengeMocks(t, scYaml, blobberYaml, validatorYamls, stakes, validators,
		validatorStakes, wpBalance, challengePoolIntegralValue, challengePoolBalance, thisChallange, thisExpires, now, 0)

	alloc := allocation.mustBase()
	err = ssc.blobberReward(alloc, previous, details, validators, ctx, allocationId)
	if err != nil {
		return err
	}

	newCP, err := ssc.getChallengePool(alloc.ID, ctx)
	require.NoError(t, err)

	newVSp, err := ssc.validatorsStakePools(validators, ctx)
	require.NoError(t, err)

	afterBlobber, err := ssc.getStakePool(spenum.Blobber, blobberId, ctx)
	require.NoError(t, err)

	confirmBlobberReward(t, f, *newCP, newVSp, *afterBlobber)

	allocation.mustUpdateBase(func(base *storageAllocationBase) error {
		alloc.deepCopy(base)
		return nil
	})
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
) (*StorageSmartContract, *StorageAllocation, *BlobberAllocation, *mockStateContext) {
	require.Len(t, validatorStakes, len(validators))

	var err error

	var sa = &StorageAllocation{}
	sa.SetEntity(&storageAllocationV2{
		ID:         "alice",
		Owner:      "owin",
		Expiration: thisExpires,
		TimeUnit:   scYaml.TimeUnit,
		WritePool:  currency.Coin(wpBalance),
	})

	allocation := sa.mustBase()

	var details = &BlobberAllocation{
		BlobberID:                  blobberId,
		ChallengePoolIntegralValue: challengePoolIntegralValue,
		Terms: Terms{
			WritePrice: zcnToBalance(blobberYaml.writePrice),
		},
		Size:                          size,
		LatestFinalizedChallCreatedAt: thisChallange,
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
		StateContext: *cstate.NewStateContext(
			nil,
			&util.MerklePatriciaTrie{},
			txn,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil, nil,
		),
		clientBalance: zcnToBalance(3),
		store:         make(map[datastore.Key]util.MPTSerializable),
	}
	var ssc = &StorageSmartContract{
		&sci.SmartContract{
			ID: storageScId,
		},
	}

	sa.mustUpdateBase(func(base *storageAllocationBase) error {
		allocation.deepCopy(base)
		return nil
	})
	_, err = ctx.InsertTrieNode(sa.GetKey(ADDRESS), sa)

	var cPool = challengePool{
		ZcnPool: &tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      allocation.ID,
				Balance: challengePoolBalance,
			},
		},
	}
	require.NoError(t, cPool.save(ssc.ID, allocation, ctx))

	var sp = newStakePool()
	sp.Settings.ServiceChargeRatio = blobberYaml.serviceCharge
	for i, stake := range stakes {
		var id = strconv.Itoa(i)
		sp.Pools["paula"+id] = &stakepool.DelegatePool{}
		sp.Pools["paula"+id].Balance = currency.Coin(stake)
		sp.Pools["paula"+id].DelegateID = "delegate " + id
	}
	sp.TotalOffers = 100e10
	sp.Settings.DelegateWallet = blobberId + " wallet"
	require.NoError(t, sp.Save(spenum.Blobber, blobberId, ctx))

	var validatorsSPs []*stakePool
	for i, validator := range validators {
		var sPool = newStakePool()
		sPool.Settings.ServiceChargeRatio = validatorYamls[i].serviceCharge
		for j, stake := range validatorStakes[i] {
			var pool = &stakepool.DelegatePool{}
			pool.Balance = currency.Coin(stake)
			var id = validator + " delegate " + strconv.Itoa(j)
			pool.DelegateID = id
			sPool.Pools[id] = pool
		}
		sPool.Settings.DelegateWallet = validator + " wallet"
		validatorsSPs = append(validatorsSPs, sPool)
	}
	require.NoError(t, ssc.saveStakePools(validators, validatorsSPs, ctx))

	_, err = ctx.InsertTrieNode(scConfigKey(ADDRESS), &scYaml)
	require.NoError(t, err)

	return ssc, sa, details, ctx
}

type formulaeBlobberReward struct {
	t                                                                                *testing.T
	scYaml                                                                           Config
	blobberYaml                                                                      mockBlobberYaml
	validatorYamls                                                                   []mockBlobberYaml
	stakes                                                                           []int64
	validators                                                                       []string
	validatorStakes                                                                  [][]int64
	wpBalance                                                                        currency.Coin
	challengePoolIntegralValue, challengePoolBalance                                 int64
	partial                                                                          float64
	lastFinalizedChallenge, lastSuccessfulChallenge, thisChallange, thisExpires, now common.Timestamp
	size                                                                             int64
	collectedReward                                                                  currency.Coin
}

func (f formulaeBlobberReward) reward() int64 {
	var challengePool = float64(f.challengePoolIntegralValue)
	var lastFinalizedChallenge = float64(f.lastFinalizedChallenge)
	var passedCurrent = math.Min(float64(f.thisChallange), float64(f.thisExpires))
	var currentExpires = float64(f.thisExpires)
	var interpolationFraction = (passedCurrent - lastFinalizedChallenge) / (currentExpires - lastFinalizedChallenge)

	return int64(challengePool * interpolationFraction)
}

func (f formulaeBlobberReward) rewardOnFinalization(challengePool, passRate float64) int64 {
	if challengePool == 0 {
		return 0
	}

	var lastFinalizedChallenge = float64(f.lastFinalizedChallenge)
	var passedCurrent = math.Min(float64(f.thisChallange), float64(f.thisExpires))
	var currentExpires = float64(f.thisExpires)
	var interpolationFraction = (passedCurrent - lastFinalizedChallenge) / (currentExpires - lastFinalizedChallenge)

	return int64(challengePool * interpolationFraction * passRate)
}

func (f formulaeBlobberReward) penalty() (int64, int64) {
	var challengePool = float64(f.challengePoolIntegralValue)
	var lastFinalizedChallenge = float64(f.lastFinalizedChallenge)
	var lastSuccessfulChallenge = float64(f.lastSuccessfulChallenge)
	var currentExpires = float64(f.thisExpires)
	var interpolationFraction = (lastFinalizedChallenge - lastSuccessfulChallenge) / (currentExpires - lastSuccessfulChallenge)

	move := float64(challengePool * interpolationFraction)

	penaltyPaid := move * (1.0 - f.scYaml.ValidatorReward)
	validatorReward := move * f.scYaml.ValidatorReward

	return int64(penaltyPaid), int64(validatorReward)
}

func (f formulaeBlobberReward) penaltyOnFinalization() int64 {
	var challengePool = float64(f.challengePoolIntegralValue)
	var lastFinalizedChallenge = float64(f.lastFinalizedChallenge)
	var lastSuccessfulChallenge = float64(f.lastSuccessfulChallenge)
	var currentExpires = float64(f.thisExpires)
	var interpolationFraction = (lastFinalizedChallenge - lastSuccessfulChallenge) / (currentExpires - lastSuccessfulChallenge)

	penaltyPaid := challengePool * interpolationFraction

	return int64(penaltyPaid)
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

func (f formulaeBlobberReward) blobberServiceChargeOnFinalization() int64 {
	if len(f.stakes) == 0 {
		return f.reward()
	}

	var serviceCharge = blobberYaml.serviceCharge
	var blobberRewards = float64(f.reward())

	return int64(blobberRewards * serviceCharge)
}

func (f formulaeBlobberReward) validatorServiceCharge(validator string) int64 {
	var serviceCharge = f.validatorYamls[f.indexFromValidator(validator)].serviceCharge
	var rewardPerValidator = float64(f.validatorsReward()) / float64(len(f.validators))

	return int64(rewardPerValidator * serviceCharge)
}

func (f formulaeBlobberReward) validatorServiceChargeForBlobberPenalty(validator string, validatorsReward int64) int64 {
	var serviceCharge = f.validatorYamls[f.indexFromValidator(validator)].serviceCharge
	var rewardPerValidator = float64(validatorsReward) / float64(len(f.validators))

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

func (f formulaeBlobberReward) validatorDelegateRewardForBlobberPenalty(validator string, delegate int, validatorsReward int64) int64 {
	var vIndex = f.indexFromValidator(validator)

	var totalStake = 0.0
	for _, stake := range f.validatorStakes[vIndex] {
		totalStake += float64(stake)
	}
	var delegateStake = float64(f.validatorStakes[vIndex][delegate])
	var validatorReward = float64(validatorsReward) / float64(len(f.validators))
	var deleatesReward = validatorReward - float64(f.validatorServiceChargeForBlobberPenalty(validator, validatorsReward))
	return int64(deleatesReward * delegateStake / totalStake)
}

func (f formulaeBlobberReward) delegatePenalty(index int, penaltyPaid int64) int64 {
	require.True(f.t, index < len(f.stakes))
	var totalStake = 0.0
	for _, stake := range f.stakes {
		totalStake += float64(stake)
	}
	var delegateStake = float64(f.stakes[index])
	var slash = f.scYaml.BlobberSlash

	var slashedAmount = int64(float64(penaltyPaid) * slash)

	return int64(float64(slashedAmount) * delegateStake / totalStake)
}

func confirmBlobberPenalty(
	t *testing.T,
	f formulaeBlobberReward,
	challengePool challengePool,
	validatorsSPs []*stakePool,
	blobber stakePool,
	includeBlobberReward bool,
) (int64, int64, int64) {
	penaltyPaid, validatorsReward := f.penalty()
	f.challengePoolIntegralValue -= penaltyPaid + validatorsReward
	blobberReward := int64(0)
	if includeBlobberReward {
		blobberReward = f.reward()
	}
	require.InDelta(t, f.challengePoolBalance-penaltyPaid-validatorsReward-blobberReward, int64(challengePool.Balance), errDelta)

	for _, sp := range validatorsSPs {
		orderedPoolIds := sp.OrderedPoolIds()
		for _, wallet := range orderedPoolIds {
			pool := sp.Pools[wallet]
			var wSplit = strings.Split(wallet, " ")
			require.InDelta(t, f.validatorServiceChargeForBlobberPenalty(wSplit[0], validatorsReward), int64(sp.Reward), errDelta)
			index, err := strconv.Atoi(wSplit[2])
			require.NoError(t, err)
			require.InDelta(t, f.validatorDelegateRewardForBlobberPenalty(wSplit[0], index, validatorsReward), int64(pool.Reward), errDelta)
		}
	}

	if f.scYaml.BlobberSlash > 0.0 {
		blobberOrderedPoolIds := blobber.OrderedPoolIds()
		for idx, id := range blobberOrderedPoolIds {
			pool := blobber.Pools[id]

			delegatePenalty := f.delegatePenalty(idx, penaltyPaid)
			require.InDelta(t, f.stakes[idx]-delegatePenalty, int64(pool.Balance), errDelta)
		}
	}

	return blobberReward, validatorsReward, penaltyPaid
}

func confirmBlobberPenaltyOnFinalization(
	t *testing.T,
	f formulaeBlobberReward,
	challengePoolIntergalValue int64,
	blobber stakePool,
	includeBlobberReward bool,
	passRate float64,
	expectedCancellationCharge int64,
) (int64, int64) {
	penaltyPaid := f.penaltyOnFinalization()
	blobberReward := int64(0)
	if includeBlobberReward {
		blobberReward = f.rewardOnFinalization(float64(f.challengePoolIntegralValue-penaltyPaid), passRate)
	}
	require.InDelta(t, f.challengePoolIntegralValue-penaltyPaid-blobberReward, challengePoolIntergalValue, errDelta)

	blobberCollectedReward, _ := f.collectedReward.Int64()
	require.InDelta(t, int64(float64(blobberReward)*blobberYaml.serviceCharge+float64(expectedCancellationCharge)*blobberYaml.serviceCharge), int64(blobber.Reward)-blobberCollectedReward, errDelta)

	if f.scYaml.BlobberSlash > 0.0 {
		blobberOrderedPoolIds := blobber.OrderedPoolIds()
		for idx, id := range blobberOrderedPoolIds {
			pool := blobber.Pools[id]

			delegatePenalty := f.delegatePenalty(idx, penaltyPaid)
			require.InDelta(t, f.stakes[idx]-delegatePenalty, int64(pool.Balance), errDelta)
		}
	}

	return blobberReward, penaltyPaid
}

func confirmBlobberReward(
	t *testing.T,
	f formulaeBlobberReward,
	challengePool challengePool,
	validatorsSPs []*stakePool,
	blobber stakePool,
) int64 {

	blobberCollectedReward, _ := f.collectedReward.Int64()

	require.InDelta(t, f.challengePoolBalance-f.blobberReward()-f.rewardReturned()-f.validatorsReward(), int64(challengePool.Balance), errDelta)
	require.InDelta(t, f.blobberServiceCharge(), int64(blobber.Reward)-blobberCollectedReward, errDelta)

	for _, sp := range validatorsSPs {
		orderedPoolIds := sp.OrderedPoolIds()
		for _, wallet := range orderedPoolIds {
			pool := sp.Pools[wallet]
			var wSplit = strings.Split(wallet, " ")
			require.InDelta(t, f.validatorServiceCharge(wSplit[0]), int64(sp.Reward), errDelta)
			index, err := strconv.Atoi(wSplit[2])
			require.NoError(t, err)
			require.InDelta(t, f.validatorDelegateReward(wSplit[0], index), int64(pool.Reward), errDelta)
		}
	}

	return f.blobberReward() + f.rewardReturned() + f.validatorsReward()
}

func confirmBlobberRewardOnFinalization(
	t *testing.T,
	f formulaeBlobberReward,
	blobber stakePool,
	passRate float64,
	challengePoolIntergalValue int64,
	expectedCancellationCharge int64,
) int64 {

	blobberCollectedReward, _ := f.collectedReward.Int64()
	blobberReward := f.rewardOnFinalization(float64(f.challengePoolIntegralValue), passRate)

	require.InDelta(t, f.challengePoolIntegralValue-blobberReward, challengePoolIntergalValue, errDelta)

	require.InDelta(t, int64(float64(blobberReward)*blobberYaml.serviceCharge+float64(expectedCancellationCharge)*blobberYaml.serviceCharge), int64(blobber.Reward)-blobberCollectedReward, errDelta)

	return blobberReward
}
