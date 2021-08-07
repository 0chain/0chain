package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/util"
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"math"
	"strconv"
	"testing"
)

func TestPayBlobberBlockRewards(t *testing.T) {
	const mockMaxRewardsTotal = 1500000.0
	const mockMaxMint = 1500000.0 * 5
	type parameters struct {
		blobberStakes        []float64
		blobberCapacities    []int64
		blobberUsage         []int64
		blockReward          float64
		qualifyingStake      float64
		blobberCapacityRatio float64
		blobberUsageRatio    float64
		minerRatio           float64
		sharderRatio         float64
	}

	var blobberRewards = func(t *testing.T, p parameters) (map[string]float64, float64) {
		var totalCapacity int64
		var totalUsage int64
		var rewards = make(map[string]float64)
		require.EqualValues(t, len(p.blobberStakes), len(p.blobberCapacities))
		require.EqualValues(t, len(p.blobberCapacities), len(p.blobberUsage))
		var qualifier = false

		for i, stake := range p.blobberStakes {
			if stake >= p.qualifyingStake {
				qualifier = true
				totalCapacity += p.blobberCapacities[i]
				totalUsage += p.blobberUsage[i]
			}
		}

		var capacityRewardTotal = p.blockReward * p.blobberCapacityRatio /
			(p.blobberCapacityRatio + p.blobberUsageRatio + p.minerRatio + p.sharderRatio)
		var usageRewardTotal = p.blockReward * p.blobberUsageRatio /
			(p.blobberCapacityRatio + p.blobberUsageRatio + p.minerRatio + p.sharderRatio)

		for i, stake := range p.blobberStakes {
			if stake >= p.qualifyingStake {
				var capacityReward float64
				var usageReward float64
				capacityReward = capacityRewardTotal * float64(p.blobberCapacities[i]) / float64(totalCapacity)
				usageReward = usageRewardTotal * float64(p.blobberUsage[i]) / float64(totalUsage)
				rewards[strconv.Itoa(i)] = (capacityReward + usageReward) * 1e10
			}
		}

		var totalReward = 0.0
		if qualifier {
			totalReward = (capacityRewardTotal + usageRewardTotal) * 1e10
		}

		return rewards, totalReward
	}

	var setExpectations = func(t *testing.T, p parameters) (*StorageSmartContract, cstate.StateContextI) {
		var balances = &mocks.StateContextI{}
		var ssc = &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		var conf = &scConfig{
			MaxMint: zcnToBalance(mockMaxMint),
			BlockReward: &blockReward{
				BlockReward:     zcnToBalance(p.blockReward),
				QualifyingStake: zcnToBalance(p.qualifyingStake),
				MaxMintRewards:  zcnToBalance(mockMaxRewardsTotal),
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

		rewards, mintTotal := blobberRewards(t, p)
		balances.On(
			"InsertTrieNode",
			BLOCK_REWARD_MINTS,
			mock.MatchedBy(func(brm *blockRewardMints) bool {
				for id, amount := range brm.UnProcessedMints {
					if int64(amount) != int64(rewards[id]) {
						return false
					}
				}
				a := brm.MintedRewards == mintTotal
				b := brm.MaxMintRewards == mockMaxRewardsTotal*1e10
				c := len(rewards) == len(brm.UnProcessedMints)
				fmt.Println(a, b, c)
				return brm.MintedRewards == mintTotal &&
					brm.MaxMintRewards == mockMaxRewardsTotal*1e10 &&
					len(rewards) == len(brm.UnProcessedMints)

			}),
		).Return("", nil).Once()

		var blobberStakes = newBlobberStakeTotals()
		for id, stake := range p.blobberStakes {
			blobberStakes.add(strconv.Itoa(id), zcnToInt64(stake), BsStakeTotals)
			blobberStakes.add(strconv.Itoa(id), p.blobberCapacities[id], BsCapacities)
			blobberStakes.add(strconv.Itoa(id), p.blobberUsage[id], BsUsed)
		}
		balances.On(
			"GetTrieNode", ALL_BLOBBER_STAKES_KEY,
		).Return(blobberStakes, nil).Once()

		balances.On(
			"InsertTrieNode",
			scConfigKey(ssc.ID),
			mock.MatchedBy(func(c *scConfig) bool {
				return approxEqual(float64(c.Minted), float64(conf.Minted)+mintTotal)
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
				blobberCapacities:    []int64{20},
				blobberUsage:         []int64{5},
				blockReward:          100,
				qualifyingStake:      10.0,
				blobberCapacityRatio: 5,
				blobberUsageRatio:    10,
				minerRatio:           2,
				sharderRatio:         3,
			},
		},
		{
			name: "3 blobbers",
			parameters: parameters{
				blobberStakes:        []float64{10, 20, 4},
				blobberCapacities:    []int64{20, 10, 5},
				blobberUsage:         []int64{5, 10, 0},
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
				blobberCapacities:    []int64{},
				blobberUsage:         []int64{},
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
				blobberCapacities:    []int64{20, 10, 5},
				blobberUsage:         []int64{5, 10, 0},
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
				blobberCapacities:    []int64{200},
				blobberUsage:         []int64{5},
				blockReward:          0,
				qualifyingStake:      10.0,
				blobberCapacityRatio: 5,
				blobberUsageRatio:    10,
				minerRatio:           2,
				sharderRatio:         3,
			},
		},
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

// remove type casting errors
func approxEqual(a, b float64) bool {
	roundA := math.Round(a / 10)
	roundB := math.Round(b / 10)
	return roundA == roundB
}
