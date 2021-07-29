package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/core/util"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestPayBlobberBlockRewards(t *testing.T) {
	const mockMaxRewardsTotal = 1500000.0
	type parameters struct {
		blobberStakes        []float64
		blockReward          float64
		qualifyingStake      float64
		blobberCapacityRatio float64
		blobberUsageRatio    float64
		minerRatio           float64
		sharderRatio         float64
	}

	type blockRewards struct {
		total                 state.Balance
		serviceChargeUsage    state.Balance
		serviceChargeCapacity state.Balance
		delegateUsage         []state.Balance
		delegateCapacity      []state.Balance
	}

	var blobberRewards = func(p parameters) (map[string]float64, float64) {
		var totalQStake float64
		var rewards = make(map[string]float64)
		var blobberStakes []float64
		for _, stake := range p.blobberStakes {
			blobberStakes = append(blobberStakes, stake)
			if stake >= p.qualifyingStake {
				totalQStake += stake
			}
		}

		var capacityRewardTotal = p.blockReward * p.blobberCapacityRatio /
			(p.blobberCapacityRatio + p.blobberUsageRatio + p.minerRatio + p.sharderRatio)
		var usageRewardTotal = p.blockReward * p.blobberUsageRatio /
			(p.blobberCapacityRatio + p.blobberUsageRatio + p.minerRatio + p.sharderRatio)

		for i, bStake := range blobberStakes {
			var capacityReward float64
			var usageReward float64
			if bStake >= p.qualifyingStake {
				capacityReward = capacityRewardTotal * bStake / totalQStake
				usageReward = usageRewardTotal * bStake / totalQStake
				rewards[strconv.Itoa(i)] = (capacityReward + usageReward) * 1e10
			}
		}

		require.EqualValues(t, len(p.blobberStakes), len(rewards))
		totalReward := (capacityRewardTotal + usageRewardTotal) * 1e10
		return rewards, totalReward
	}

	var setExpectations = func(t *testing.T, p parameters) (*StorageSmartContract, cstate.StateContextI) {
		var balances = &mocks.StateContextI{}
		var ssc = &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		var conf = &scConfig{
			BlockReward: &blockReward{
				BlockReward:     zcnToBalance(p.blockReward),
				QualifyingStake: zcnToBalance(p.qualifyingStake),
				MaxRewardsTotal: zcnToBalance(mockMaxRewardsTotal),
			},
		}
		conf.BlockReward.setWeightsFromRatio(p.sharderRatio, p.minerRatio, p.blobberCapacityRatio, p.blobberUsageRatio)
		balances.On("GetTrieNode", scConfigKey(ssc.ID)).Return(conf, nil)
		if conf.BlockReward.BlobberCapacityWeight+conf.BlockReward.BlobberUsageWeight == 0 ||
			conf.BlockReward.BlockReward == 0 {
			return ssc, balances
		}

		balances.On(
			"GetTrieNode", BLOCK_REWARD_MINTS,
		).Return(nil, util.ErrValueNotPresent).Once()

		rewards, mintTotal := blobberRewards(p)
		balances.On(
			"InsertTrieNode",
			BLOCK_REWARD_MINTS,
			mock.MatchedBy(func(brm *blockRewardMints) bool {
				for id, amount := range brm.UnProcessedMints {
					if amount != rewards[id] {
						return false
					}
				}
				return brm.MintedRewards == mintTotal && brm.MaxRewardsTotal == mockMaxRewardsTotal*1e10
			}),
		).Return("", nil).Once()

		var blobberStakes = newBlobberStakeTotals()
		for id, stake := range p.blobberStakes {
			blobberStakes.Totals[strconv.Itoa(id)] = zcnToBalance(stake)
		}
		balances.On(
			"GetTrieNode", ALL_BLOBBER_STAKES_KEY,
		).Return(blobberStakes, nil).Once()

		balances.On(
			"InsertTrieNode",
			scConfigKey(ssc.ID),
			mock.MatchedBy(func(c *scConfig) bool {
				return c.Minted == conf.Minted+state.Balance(mintTotal)
			}),
		).Return("", nil)
		return ssc, balances
	}

	type want struct {
		error    bool
		errorMsg string
	}
	tests := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "1 blobbers",
			parameters: parameters{
				blobberStakes:        []float64{11},
				blockReward:          100,
				qualifyingStake:      10.0,
				blobberCapacityRatio: 5,
				blobberUsageRatio:    10,
				minerRatio:           2,
				sharderRatio:         3,
			},
		}, /*
			{
				name: "3 blobbers",
				parameters: parameters{
					blobberStakes:        []float64{10, 20, 4},
					blockReward:          100,
					qualifyingStake:      10.0,
					blobberCapacityRatio: 5,
					blobberUsageRatio:    10,
					minerRatio:           2,
					sharderRatio:         3,
				},
			},
			{
				name: "no blobbers",
				parameters: parameters{
					blobberStakes:        []float64{},
					blockReward:          100,
					qualifyingStake:      10.0,
					blobberCapacityRatio: 5,
					blobberUsageRatio:    10,
					minerRatio:           2,
					sharderRatio:         3,
				},
			},
			{
				name: "3 blobbers no qualifiers",
				parameters: parameters{
					blobberStakes:        []float64{9, 2, 4},
					blockReward:          100,
					qualifyingStake:      1000.0,
					blobberCapacityRatio: 5,
					blobberUsageRatio:    10,
					minerRatio:           2,
					sharderRatio:         3,
				},
			},
			{
				name: "zero block reward",
				parameters: parameters{
					blobberStakes:        []float64{10},
					blockReward:          0,
					qualifyingStake:      10.0,
					blobberCapacityRatio: 5,
					blobberUsageRatio:    10,
					minerRatio:           2,
					sharderRatio:         3,
				},
			},*/
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ssc, balances := setExpectations(t, tt.parameters)

			err := ssc.payBlobberBlockRewards(balances)

			require.EqualValues(t, tt.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errorMsg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, balances))
		})
	}
}
