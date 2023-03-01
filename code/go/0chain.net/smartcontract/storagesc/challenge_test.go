package storagesc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/currency"

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
		cct          time.Duration
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
		config.SmartContractConfig.SetDefault("smart_contracts.storagesc.max_challenge_completion_time", p.cct)

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

		alloc := &StorageAllocation{
			ID:            allocID,
			BlobberAllocs: make([]*BlobberAllocation, 0, len(p.challengesTS)),

			BlobberAllocsMap: make(map[string]*BlobberAllocation),
			Stats:            &StorageAllocationStats{},
		}

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

			err = ssc.addChallenge(alloc, c, allocChallenges, challInfo, balances)
			require.NoError(t, err)
		}

		return ssc, args{
			alloc:           alloc,
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
				cct: 100 * time.Second,
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
				cct: 100 * time.Second,
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
				cct: 100 * time.Second,
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
				cct: 100 * time.Second,
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

			// add new challenge
			c, challInfo := newChallenge(args.alloc.ID, tt.parameters.add.blobberID, tt.parameters.add.ts)
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

			// assert the open challenge update events are emitted
			es = args.balances.GetEvents()[initESLen:]
			updateOpenChallengeEventMap := make(map[string]int64)
			for _, e := range es {
				if e.Tag == event.TagUpdateBlobberOpenChallenges {
					d, ok := e.Data.(event.ChallengeStatsDeltas)
					require.True(t, ok)
					updateOpenChallengeEventMap[d.Id] = d.OpenDelta
				}
			}

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
		var thisChallenge = thisExpires + toSeconds(scYaml.MaxChallengeCompletionTime) + 1
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
		MaxMint:                    zcnToBalance(4000000.0),
		BlobberSlash:               0.1,
		ValidatorReward:            0.025,
		MaxChallengeCompletionTime: 30 * time.Minute,
		TimeUnit:                   720 * time.Hour,
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

	t.Run(errLate, func(t *testing.T) {
		var thisChallenge = thisExpires + toSeconds(scYaml.MaxChallengeCompletionTime) + 1
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, size, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errLate)
	})

	t.Run("old challenge", func(t *testing.T) {
		var thisChallenge = previousChallenge - 1
		err := testBlobberPenalty(t, scYaml, blobberYaml, validatorYamls, stakes, validators, validatorStakes,
			writePoolBalance, challengePoolIntegralValue,
			challengePoolBalance, partial, size, previousChallenge, thisChallenge, thisExpires, now)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), "old challenge response on blobber penalty")
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
		err                       error
	}{
		{
			name:      "ok",
			ticketNum: 10,
		},
		{
			name:               "duplicate ticket",
			ticketNum:          10,
			hasDuplicateTicket: true,
			err:                common.NewError("verify_challenge", "found duplicate validation tickets"),
		},
		{
			name:      "not enough tickets",
			ticketNum: 4, // threshold is 5
			err:       common.NewError("verify_challenge", "validation tickets less than threshold: 5, tickets: 4"),
		},
		{
			name:                      "ticket signed with unauthorized validator",
			ticketNum:                 5,
			hasNonceSelectedValidator: true,
			err:                       common.NewError("verify_challenge", "found invalid validator id in validation ticket"),
		},
		{
			name:          "wrong txn client id",
			ticketNum:     5,
			wrongClientID: true,
			err:           errors.New("challenge blobber id does not match"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ssc, balances, tp, alloc, b3, valids, validators, blobber, _ := prepareAllocChallenges(t, 10)
			step := (int64(alloc.Expiration) - tp) / 10

			challID := fmt.Sprintf("chall-0")
			genChall(t, ssc, tp, challID, 0, validators, alloc.ID, blobber, balances)

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

			tp += step / 2
			var tx *transaction.Transaction
			if tc.wrongClientID {
				tx = newTransaction(alloc.BlobberAllocs[0].BlobberID, ssc.ID, 0, tp)
			} else {
				tx = newTransaction(b3.id, ssc.ID, 0, tp)
			}
			balances.setTransaction(t, tx)
			var resp string
			resp, err := ssc.verifyChallenge(tx, mustEncode(t, chall), balances)
			require.Equal(t, tc.err, err)
			if err != nil {
				return
			}

			require.Equal(t, resp, "challenge passed by blobber")
		})
	}

}

func TestVerifyChallengeOldChallenge(t *testing.T) {
	ssc, balances, tp, alloc, b3, valids, validators, blobber, blobbers := prepareAllocChallenges(t, 10)
	step := (int64(alloc.Expiration) - tp) / 10

	t.Run("verify challenge first time", func(t *testing.T) {
		challID := fmt.Sprintf("chall-0")
		genChall(t, ssc, tp, challID, 0, validators, alloc.ID, blobber, balances)

		chall := &ChallengeResponse{
			ID: challID,
		}

		for i := 0; i < 10; i++ {
			chall.ValidationTickets = append(chall.ValidationTickets,
				valids[i].validTicket(t, chall.ID, b3.id, true, tp))
		}

		tp += step / 2
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

		blobber1, err := ssc.getBlobber(b1.id, balances)
		// reduce timestamp to generate challenge with older create time
		tp := tp - 10
		genChall(t, ssc, tp, challID, 0, validators, alloc.ID, blobber1, balances)

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

		challID := fmt.Sprintf("chall-1")
		var blobber1 *StorageNode
		blobber1, err := ssc.getBlobber(b1.id, balances)
		// reduce timestamp to generate challenge with older create time
		tp := tp - 20
		genChall(t, ssc, tp, challID, 0, validators, alloc.ID, blobber1, balances)

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
		bk := &block.Block{}
		bk.Round = 500
		balances.setBlock(t, bk)
		_, err = ssc.verifyChallenge(tx, mustEncode(t, chall1), balances)
		require.EqualError(t, err, "challenge_reward_error: old challenge response on blobber rewarding")
	})
}

func createTxnMPT(mpt util.MerklePatriciaTrieI) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), mpt.GetNodeDB(), false)
	tmpt := util.NewMerklePatriciaTrie(tdb, mpt.GetVersion(), mpt.GetRoot())
	return tmpt
}

// TODO: test to run the same verify challenge SC multiple times result in the same state
func TestVerifyChallengeRunMultipleTimes(t *testing.T) {
	ssc, balances, tp, alloc, b3, valids, validators, blobber, _ := prepareAllocChallenges(t, 10)
	step := (int64(alloc.Expiration) - tp) / 10

	challID := fmt.Sprintf("chall-0")
	genChall(t, ssc, tp, challID, 0, validators, alloc.ID, blobber, balances)

	chall := &ChallengeResponse{
		ID: challID,
	}

	for i := 0; i < 10; i++ {
		chall.ValidationTickets = append(chall.ValidationTickets,
			valids[i].validTicket(t, chall.ID, b3.id, true, tp))
	}

	tp += step / 2
	tx := newTransaction(b3.id, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)

	stateRoots := make(map[string]struct{}, 10)
	for i := 0; i < 20; i++ {
		clientState := createTxnMPT(balances.GetState())
		signatureScheme := &encryption.BLS0ChainScheme{}
		cs := cstate.NewStateContext(balances.block, clientState,
			balances.txn, nil, nil, nil, func() encryption.SignatureScheme { return signatureScheme }, nil, nil)

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

func prepareAllocChallenges(t *testing.T, validatorsNum int) (*StorageSmartContract, *testBalances, int64,
	*StorageAllocation, *Client, []*Client, *partitions.Partitions, *StorageNode, []*Client) {
	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(t, true)
		client         = newClient(100*x10, balances)
		tp, exp  int64 = 0, int64(toSeconds(time.Hour))

		// no owner
		reader = newClient(100*x10, balances)
		err    error
	)

	// new allocation
	tp += 100
	var allocID, blobs = addAllocation(t, ssc, client, tp, exp, 0, balances)

	// blobbers: stake 10k, balance 40k

	var alloc *StorageAllocation
	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	b1 := testGetBlobber(blobs, alloc, 0)
	require.NotNil(t, b1)

	// read as owner
	tp = testCommitRead(t, balances, client, client, alloc, b1.id, ssc, tp)

	//read as unauthorized separate user
	tp = testCommitRead(t, balances, client, reader, alloc, b1.id, ssc, tp)

	b2 := testGetBlobber(blobs, alloc, 1)
	require.NotNil(t, b2)

	_, tp = testCommitWrite(t, balances, client, allocID, "root-1", 100*1024*1024, tp, b2.id, ssc)

	b3 := testGetBlobber(blobs, alloc, 2)
	require.NotNil(t, b3)

	// add 10 validators
	valids, tp := testAddValidators(t, balances, ssc, validatorsNum, tp)

	//_, err := ssc.getChallengePool(allocID, balances)
	//require.NoError(t, err)

	const allocRoot = "alloc-root-1"

	// write 100MB
	_, tp = testCommitWrite(t, balances, client, allocID, allocRoot, 100*1024*1024, tp, b3.id, ssc)

	alloc, err = ssc.getAllocation(allocID, balances)
	require.NoError(t, err)

	// load validators
	validators, err := getValidatorsList(balances)
	require.NoError(t, err)

	// load blobber
	var blobber *StorageNode
	blobber, err = ssc.getBlobber(b3.id, balances)
	require.NoError(t, err)
	return ssc, balances, tp, alloc, b3, valids, validators, blobber, blobs
}

func testAddValidators(t *testing.T, balances *testBalances, ssc *StorageSmartContract, num int, tp int64) ([]*Client, int64) {
	var valids []*Client
	tp += 100
	for i := 0; i < num; i++ {
		valids = append(valids, addValidator(t, ssc, tp, balances))
	}
	return valids, tp
}

func testGetBlobber(blobs []*Client, alloc *StorageAllocation, i int) *Client {
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
	alloc *StorageAllocation, blobberID string, ssc *StorageSmartContract, tp int64) int64 {
	tp += 100
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

	tp += 100
	var tx = newTransaction(blobberID, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
	require.Error(t, err)

	// read pool lock
	tp += 100

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
	tp += 100
	tx = newTransaction(blobberID, ssc.ID, 0, tp)
	balances.setTransaction(t, tx)
	_, err = ssc.commitBlobberRead(tx, mustEncode(t, &rm), balances)
	require.NoError(t, err)
	return tp
}

func testCommitWrite(t *testing.T, balances *testBalances, client *Client, allocID, allocRoot string, size int64, tp int64, blobberID string, ssc *StorageSmartContract) (*transaction.Transaction, int64) {
	tp += 100
	cc := &BlobberCloseConnection{
		AllocationRoot:     allocRoot,
		PrevAllocationRoot: "",
		WriteMarker: &WriteMarker{
			AllocationRoot:         allocRoot,
			PreviousAllocationRoot: "",
			AllocationID:           allocID,
			//Size:                   100 * 1024 * 1024, // 100 MB
			Size:      size,
			BlobberID: blobberID,
			Timestamp: common.Timestamp(tp),
			ClientID:  client.id,
		},
	}
	var err error
	cc.WriteMarker.Signature, err = client.scheme.Sign(
		encryption.Hash(cc.WriteMarker.GetHashData()))
	require.NoError(t, err)

	// write
	//tp += 100
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
		previousChallange:          previous,
		size:                       size,
		thisChallange:              thisChallange,
		thisExpires:                thisExpires,
		now:                        now,
	}

	var ssc, allocation, details, ctx = setupChallengeMocks(t, scYaml, blobberYaml, validatorYamls, stakes, validators,
		validatorStakes, wpBalance, challengePoolIntegralValue, challengePoolBalance, thisChallange, thisExpires, now, size)

	err = ssc.blobberPenalty(allocation, previous, details, validators, ctx)
	if err != nil {
		return err
	}

	newCP, err := ssc.getChallengePool(allocation.ID, ctx)
	require.NoError(t, err)

	newVSp, err := ssc.validatorsStakePools(validators, ctx)
	require.NoError(t, err)

	afterBlobber, err := ssc.getStakePool(spenum.Blobber, blobberId, ctx)
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

	var ssc, allocation, details, ctx = setupChallengeMocks(t, scYaml, blobberYaml, validatorYamls, stakes, validators,
		validatorStakes, wpBalance, challengePoolIntegralValue, challengePoolBalance, thisChallange, thisExpires, now, 0)

	err = ssc.blobberReward(allocation, previous, details, validators, partial, ctx)
	if err != nil {
		return err
	}

	newCP, err := ssc.getChallengePool(allocation.ID, ctx)
	require.NoError(t, err)

	newVSp, err := ssc.validatorsStakePools(validators, ctx)
	require.NoError(t, err)

	afterBlobber, err := ssc.getStakePool(spenum.Blobber, blobberId, ctx)
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
) (*StorageSmartContract, *StorageAllocation, *BlobberAllocation, *mockStateContext) {
	require.Len(t, validatorStakes, len(validators))

	var err error
	var allocation = &StorageAllocation{
		ID:         "alice",
		Owner:      "owin",
		Expiration: thisExpires,
		TimeUnit:   scYaml.TimeUnit,
		WritePool:  currency.Coin(wpBalance),
	}

	var details = &BlobberAllocation{
		BlobberID:                  blobberId,
		ChallengePoolIntegralValue: challengePoolIntegralValue,
		Terms: Terms{
			WritePrice: zcnToBalance(blobberYaml.writePrice),
		},
		Size: size,
		LatestCompletedChallenge: &StorageChallenge{
			Created: thisChallange,
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
		),
		clientBalance: zcnToBalance(3),
		store:         make(map[datastore.Key]util.MPTSerializable),
	}
	var ssc = &StorageSmartContract{
		&sci.SmartContract{
			ID: storageScId,
		},
	}
	_, err = ctx.InsertTrieNode(allocation.GetKey(ADDRESS), allocation)

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
			sPool.Pools[id] = pool
		}
		sPool.Settings.DelegateWallet = validator + " wallet"
		validatorsSPs = append(validatorsSPs, sPool)
	}
	require.NoError(t, ssc.saveStakePools(validators, validatorsSPs, ctx))

	_, err = ctx.InsertTrieNode(scConfigKey(ADDRESS), &scYaml)
	require.NoError(t, err)

	return ssc, allocation, details, ctx
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
