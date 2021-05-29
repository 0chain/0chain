package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/mocks/chaincore/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"strconv"
	"strings"
	"testing"
)

func TestPayBlobberBlockRewards(t *testing.T) {
	type parameters struct {
		blobberStakes        [][]float64
		blobberServiceCharge []float64
		blockReward          float64
		qualifyingStake      float64
		blobberCapacityRato  float64
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

	var blobberRewards = func(p parameters) []blockRewards {
		var totalQStake float64
		var rewards = []blockRewards{}
		var blobberStakes []float64
		for _, blobber := range p.blobberStakes {
			var stakes float64
			for _, dStake := range blobber {
				stakes += dStake
			}
			blobberStakes = append(blobberStakes, stakes)
			if stakes >= p.qualifyingStake {
				totalQStake += stakes
			}
		}

		var capacityRewardTotal = p.blockReward * p.blobberCapacityRato /
			(p.blobberCapacityRato + p.blobberUsageRatio + p.minerRatio + p.sharderRatio)
		var usageRewardTotal = p.blockReward * p.blobberUsageRatio /
			(p.blobberCapacityRato + p.blobberUsageRatio + p.minerRatio + p.sharderRatio)

		for i, bStake := range blobberStakes {
			var reward blockRewards
			var capacityReward float64
			var usageReward float64
			var sc = p.blobberServiceCharge[i]
			if bStake >= p.qualifyingStake {
				capacityReward = capacityRewardTotal * bStake / totalQStake
				usageReward = usageRewardTotal * bStake / totalQStake

				reward.total = zcnToBalance(capacityReward + usageReward)
				reward.serviceChargeCapacity = zcnToBalance(capacityReward * sc)
				reward.serviceChargeUsage = zcnToBalance(usageReward * sc)
			}
			for _, dStake := range p.blobberStakes[i] {
				if bStake < p.qualifyingStake {
					reward.delegateUsage = append(reward.delegateUsage, 0)
					reward.delegateCapacity = append(reward.delegateCapacity, 0)
				} else {
					reward.delegateUsage = append(reward.delegateUsage, zcnToBalance(usageReward*(1-sc)*dStake/bStake))
					reward.delegateCapacity = append(reward.delegateCapacity, zcnToBalance(capacityReward*(1-sc)*dStake/bStake))
				}
			}
			rewards = append(rewards, reward)
		}
		return rewards
	}

	var setExpectations = func(t *testing.T, p parameters) (*StorageSmartContract, cstate.StateContextI) {
		var balances = &mocks.StateContextI{}
		var blobbers = &StorageNodes{
			Nodes: []*StorageNode{},
		}
		var ssc = &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}

		var conf = &scConfig{
			BlockReward: &blockReward{
				BlockReward:     zcnToBalance(p.blockReward),
				QualifyingStake: zcnToBalance(p.qualifyingStake),
			},
		}
		conf.BlockReward.setWeightsFromRatio(p.sharderRatio, p.minerRatio, p.blobberCapacityRato, p.blobberUsageRatio)
		balances.On("GetTrieNode", scConfigKey(ssc.ID)).Return(conf, nil).Once()
		if conf.BlockReward.BlobberCapacityWeight+conf.BlockReward.BlobberUsageWeight == 0 {
			return ssc, balances
		}

		require.EqualValues(t, len(p.blobberStakes), len(p.blobberServiceCharge))
		rewards := blobberRewards(p)
		require.EqualValues(t, len(p.blobberStakes), len(rewards))

		var sPools []stakePool
		for i, reward := range rewards {
			require.EqualValues(t, len(reward.delegateCapacity), len(p.blobberStakes[i]))
			require.EqualValues(t, len(reward.delegateUsage), len(p.blobberStakes[i]))

			id := "bob " + strconv.Itoa(i)
			var sPool = stakePool{
				Pools: make(map[string]*delegatePool),
			}
			sPool.Settings.ServiceCharge = p.blobberServiceCharge[i]
			sPool.Settings.DelegateWallet = id
			sPools = append(sPools, sPool)
			require.True(t, blobbers.Nodes.add(&StorageNode{
				ID:                id,
				StakePoolSettings: stakePoolSettings{},
			}))

			for j := 0; j < len(reward.delegateUsage); j++ {
				require.EqualValues(t, len(reward.delegateUsage), len(reward.delegateCapacity))
				did := "delroy " + strconv.Itoa(j) + " " + id
				var dPool = &delegatePool{}
				dPool.ID = did
				dPool.Balance = zcnToBalance(p.blobberStakes[i][j])
				dPool.DelegateID = did
				sPool.Pools["paula "+did] = dPool
				if reward.delegateUsage[j] > 0 {
					balances.On("AddMint", &state.Mint{
						Minter: ADDRESS, ToClientID: did, Amount: reward.delegateUsage[j],
					}).Return(nil)
				}
				if reward.delegateCapacity[j] > 0 {
					balances.On("AddMint", &state.Mint{
						Minter: ADDRESS, ToClientID: did, Amount: reward.delegateCapacity[j],
					}).Return(nil)
				}
			}
			if reward.serviceChargeCapacity > 0 {
				balances.On("AddMint", &state.Mint{
					Minter: ADDRESS, ToClientID: id, Amount: reward.serviceChargeCapacity,
				}).Return(nil)
			}
			if reward.serviceChargeUsage > 0 {
				balances.On("AddMint", &state.Mint{
					Minter: ADDRESS, ToClientID: id, Amount: reward.serviceChargeUsage,
				}).Return(nil)
			}
			balances.On("GetTrieNode", stakePoolKey(ssc.ID, id)).Return(&sPool, nil).Once()
		}
		balances.On("GetTrieNode", ALL_BLOBBERS_KEY).Return(blobbers, nil).Once()

		for i, sPool := range sPools {
			i := i
			sPool := sPool
			if rewards[i].total == 0 {
				continue
			}
			balances.On(
				"InsertTrieNode",
				stakePoolKey(ssc.ID, sPool.Settings.DelegateWallet),
				mock.MatchedBy(func(sp *stakePool) bool {
					for key, dPool := range sp.Pools {
						var wSplit = strings.Split(key, " ")
						dIndex, err := strconv.Atoi(wSplit[2])
						require.NoError(t, err)
						value, ok := sPools[i].Pools[key]

						if !ok ||
							value.DelegateID != dPool.DelegateID ||
							dPool.Rewards != rewards[i].delegateUsage[dIndex]+rewards[i].delegateCapacity[dIndex] {
							return false
						}
					}
					return sp.Rewards.Charge == rewards[i].serviceChargeCapacity+rewards[i].serviceChargeUsage &&
						sp.Rewards.Blobber == rewards[i].total &&
						sp.Settings.DelegateWallet == sPool.Settings.DelegateWallet

				}),
			).Return("", nil).Once()
		}
		conf.Minted += zcnToBalance(p.blockReward)
		balances.On("InsertTrieNode", scConfigKey(ssc.ID), conf).Return("", nil).Once()
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
				blobberStakes:        [][]float64{{10}},
				blobberServiceCharge: []float64{0.1},
				blockReward:          100,
				qualifyingStake:      10.0,
				blobberCapacityRato:  5,
				blobberUsageRatio:    10,
				minerRatio:           2,
				sharderRatio:         3,
			},
		},
		{
			name: "3 blobbers",
			parameters: parameters{
				blobberStakes:        [][]float64{{10}, {5, 5, 10}, {2, 2}},
				blobberServiceCharge: []float64{0.1, 0.2, 0.3},
				blockReward:          100,
				qualifyingStake:      10.0,
				blobberCapacityRato:  5,
				blobberUsageRatio:    10,
				minerRatio:           2,
				sharderRatio:         3,
			},
		},
		{
			name: "no blobbers",
			parameters: parameters{
				blobberStakes:        [][]float64{},
				blobberServiceCharge: []float64{},
				blockReward:          100,
				qualifyingStake:      10.0,
				blobberCapacityRato:  5,
				blobberUsageRatio:    10,
				minerRatio:           2,
				sharderRatio:         3,
			},
		},
		{
			name: "3 blobbers no qualifiers",
			parameters: parameters{
				blobberStakes:        [][]float64{{10}, {5, 5, 10}, {2, 2}},
				blobberServiceCharge: []float64{0.1, 0.2, 0.3},
				blockReward:          100,
				qualifyingStake:      1000.0,
				blobberCapacityRato:  5,
				blobberUsageRatio:    10,
				minerRatio:           2,
				sharderRatio:         3,
			},
		},
		{
			name: "zero block reward",
			parameters: parameters{
				blobberStakes:        [][]float64{{10}},
				blobberServiceCharge: []float64{0.1},
				blockReward:          0,
				qualifyingStake:      10.0,
				blobberCapacityRato:  5,
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
