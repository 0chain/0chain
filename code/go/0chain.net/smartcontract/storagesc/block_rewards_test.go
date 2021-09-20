package storagesc

import (
	"strconv"
	"testing"

	"0chain.net/chaincore/block"

	"0chain.net/chaincore/state"

	"0chain.net/chaincore/mocks"
	"0chain.net/smartcontract/storagesc/blockrewards"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateBlockRewards(t *testing.T) {
	const (
		mockBlobberId        = "mock blobber id"
		mockDelegateWallet   = "mock delegate wallet"
		carryDelta           = 0.1
		mockNumberStakePools = 10
		mockCapacity         = 1024
		mockUsage            = 2048
	)
	var (
		mockSettings = blockrewards.BlockReward{
			BlockReward:           1 * 1e10,
			QualifyingStake:       1 * 1e10,
			SharderWeight:         0.4,
			MinerWeight:           0.1,
			BlobberCapacityWeight: 0.1,
			BlobberUsageWeight:    0.4,
		}
		mockSettings2 = blockrewards.BlockReward{
			BlockReward:           state.Balance(1.1e10),
			QualifyingStake:       state.Balance(0.9e10),
			SharderWeight:         0.3,
			MinerWeight:           0.2,
			BlobberCapacityWeight: 0.25,
			BlobberUsageWeight:    0.25,
		}
	)
	mockSettings2 = mockSettings2
	type parameters struct {
		deltaCapacity, deltaUsed, lastBlockRewardPaymentRound int64
		capacity, used                                        int64
		round, lastSettingChange                              int64
		stake                                                 float64
		settings                                              map[int64]blockrewards.BlockReward
		inputQtl                                              bool
		spCarries                                             []float64
	}
	type want struct {
		error    bool
		errorMsg string
		blobber  StorageNode
		sp       stakePool
	}

	type args struct {
		deltaCapacity, deltaUsed int64
		blobber                  *StorageNode
		sp                       *stakePool
		conf                     *scConfig
		balances                 *mocks.StateContextI
		qtl                      *blockrewards.QualifyingTotalsList
	}

	var getArgs = func(t *testing.T, p parameters) args {
		_, ok := p.settings[p.lastSettingChange]
		require.True(t, ok)

		var balances = mocks.StateContextI{}
		var conf = &scConfig{
			BlockReward: &mockSettings,
		}
		var blobber = &StorageNode{
			ID:                          mockBlobberId,
			Capacity:                    mockCapacity,
			Used:                        mockUsage,
			LastBlockRewardPaymentRound: p.lastBlockRewardPaymentRound,
		}

		var qtl blockrewards.QualifyingTotalsList
		require.True(t, p.round > p.lastBlockRewardPaymentRound)
		var lastSettingsChange int64 = 0
		for i := int64(0); i < p.round; i++ {
			qt := blockrewards.QualifyingTotals{
				Round:    i,
				Capacity: mockCapacity * i,
				Used:     mockUsage * i,
			}
			if settings, ok := p.settings[i]; ok {
				qt.SettingsChange = &settings
				lastSettingsChange = i
			} else {
				require.NotEqual(t, i, int64(0))
			}
			qt.LastSettingsChange = lastSettingsChange
			qtl.Totals = append(qtl.Totals, qt)
		}

		var sp = newStakePool()
		require.True(t, p.stake > -0)
		require.True(t, len(p.spCarries) >= 1)
		if p.stake > 0 {
			for i := 0; i < len(p.spCarries); i++ {
				id := strconv.Itoa(i)
				var pool delegatePool
				pool.ID = id
				pool.DelegateID = mockDelegateWallet + id
				pool.Balance = state.Balance(p.stake * 1e10 /
					float64(len(p.spCarries)))
				pool.Carry = p.spCarries[i]
				sp.Pools[id] = &pool
			}
		}

		return args{
			deltaCapacity: p.deltaCapacity,
			deltaUsed:     p.deltaUsed,
			blobber:       blobber,
			sp:            sp,
			conf:          conf,
			balances:      &balances,
			qtl:           &qtl,
		}
	}

	setExpectations := func(t *testing.T, p parameters, args args, want want) want {
		var currentBlock block.Block
		currentBlock.Round = p.round
		args.balances.On("GetBlock").Return(&currentBlock)

		if !p.inputQtl {
			args.balances.On(
				"GetTrieNode",
				blockrewards.QualifyingTotalsPerBlockKey,
			).Return(args.qtl, nil).Once()
		}

		if p.deltaCapacity > 0 || p.deltaUsed > 0 {
			args.balances.On(
				"UpdateBlockRewardTotals",
				p.deltaCapacity,
				p.deltaUsed,
			).Return().Once()
		}

		var reward float64
		require.True(t, p.lastBlockRewardPaymentRound < p.round)
		require.True(t, 0 <= p.lastBlockRewardPaymentRound)
		var settings *blockrewards.BlockReward
		for i := p.lastBlockRewardPaymentRound; i >= 0; i-- {
			iSettings, ok := p.settings[i]
			if ok {
				settings = &iSettings
				break
			}
		}
		require.NotNil(t, settings)
		for i := p.lastBlockRewardPaymentRound; i < p.round; i++ {
			newSettings, ok := p.settings[i]
			if ok {
				settings = &newSettings
			}
			var capRatio float64
			if args.qtl.Totals[i].Capacity > 0 {
				capRatio = float64(args.blobber.Capacity) / float64(args.qtl.Totals[i].Capacity)
			}
			capacityReward := float64(settings.BlockReward) * settings.BlobberCapacityWeight * capRatio
			var usedRatio float64
			if args.qtl.Totals[i].Used > 0 {
				usedRatio = float64(args.blobber.Used) / float64(args.qtl.Totals[i].Used)
			}
			usedReward := float64(settings.BlockReward) * settings.BlobberUsageWeight * usedRatio
			reward += capacityReward + usedReward
		}

		var sp stakePool
		require.NoError(t, sp.Decode(args.sp.Encode()))
		require.EqualValues(t, state.Balance(p.stake*1e10), sp.stake())
		if p.stake > 0 {
			for _, pool := range sp.Pools {
				poolReward := pool.Carry + reward/float64(len(p.spCarries))
				toMint := state.Balance(poolReward)
				pool.Carry = poolReward - float64(toMint)

				args.balances.On("AddMint", state.NewMint(
					ADDRESS, pool.DelegateID, toMint,
				)).Return(nil).Once()
			}
		}
		var blobber StorageNode
		require.NoError(t, blobber.Decode(args.blobber.Encode()))
		blobber.LastBlockRewardPaymentRound = p.round

		want.sp = sp
		want.blobber = blobber
		return want
	}

	tests := []struct {
		name       string
		parameters parameters
		want       want
	}{

		{
			name: "ok",
			parameters: parameters{
				deltaCapacity:               3,
				deltaUsed:                   7,
				lastBlockRewardPaymentRound: 5,
				round:                       9,
				stake:                       17,
				inputQtl:                    true,
				settings: map[int64]blockrewards.BlockReward{
					0: mockSettings,
					7: mockSettings2,
				},
				spCarries: []float64{0.1, 0.9},
			},
		},
		{
			name: "ok",
			parameters: parameters{
				deltaCapacity:               3,
				deltaUsed:                   7,
				lastBlockRewardPaymentRound: 5,
				round:                       7,
				stake:                       17,
				inputQtl:                    true,
				settings: map[int64]blockrewards.BlockReward{
					0: mockSettings,
					2: mockSettings2,
				},
				spCarries: []float64{0.0, 0.9},
			},
		},

		{
			name: "ok",
			parameters: parameters{
				deltaCapacity:               3,
				deltaUsed:                   7,
				lastBlockRewardPaymentRound: 5,
				round:                       7,
				stake:                       17,
				inputQtl:                    true,
				settings: map[int64]blockrewards.BlockReward{
					0: mockSettings,
					1: mockSettings,
				},
				spCarries: []float64{0.1, 0.9},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args := getArgs(t, tt.parameters)
			want := setExpectations(t, tt.parameters, args, tt.want)
			if !tt.parameters.inputQtl {
				args.qtl.Totals = nil
			}

			err := updateBlockRewards(
				args.deltaCapacity, args.deltaUsed,
				args.blobber,
				args.sp,
				args.conf,
				args.balances,
				args.qtl,
			)
			require.EqualValues(t, tt.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errorMsg, err.Error())
				return
			}
			require.EqualValues(t, want.blobber.LastBlockRewardPaymentRound, args.blobber.LastBlockRewardPaymentRound)
			for key, pool := range want.sp.Pools {
				require.InDelta(t, pool.Carry, args.sp.Pools[key].Carry, carryDelta)
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}

func TestBlockRewardModifiedStakePool(t *testing.T) {

}
