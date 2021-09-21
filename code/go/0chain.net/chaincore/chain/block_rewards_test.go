package chain_test

import (
	"encoding/json"

	"0chain.net/chaincore/block"
	. "0chain.net/chaincore/chain"
	"0chain.net/chaincore/mocks"
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/storagesc/blockrewards"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"testing"
)

type scConfig struct {
	BlockReward *blockrewards.BlockReward `json:"block_reward"`
}

func (conf *scConfig) Encode() []byte {
	var b, _ = json.Marshal(conf)
	return b
}
func (conf *scConfig) Decode(p []byte) error {
	return json.Unmarshal(p, conf)
}

func TestUpdateRewardTotalList(t *testing.T) {
	const (
		mockCapacity = 1024
		mockUsage    = 2048
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

	type parameters struct {
		round                    int64
		deltaCapacity, deltaUsed int64
		newBlockRewardSettings   *blockrewards.BlockReward
	}
	type want struct {
		error    bool
		errorMsg string
	}

	var setup = func(t *testing.T, p parameters) *mocks.StateContextI {
		var balances = mocks.StateContextI{}

		return &balances
	}

	setExpectations := func(
		t *testing.T,
		p parameters,
		balances *mocks.StateContextI,
		want want,
	) want {
		var currentBlock block.Block
		currentBlock.Round = p.round
		balances.On("GetBlock").Return(&currentBlock)
		balances.On("GetBlockRewardDeltas").Return(p.deltaCapacity, p.deltaUsed).Once()

		var conf scConfig
		if p.newBlockRewardSettings != nil {
			conf.BlockReward = p.newBlockRewardSettings
		} else {
			conf.BlockReward = &mockSettings
		}

		balances.On("GetTrieNode", blockrewards.ConfigKey).Return(&conf, nil).Once()

		var beforeQtl blockrewards.QualifyingTotalsList
		var afterQtl blockrewards.QualifyingTotalsList
		for round := int64(0); round < p.round; round++ {
			var setting *blockrewards.BlockReward
			if round == 0 {
				setting = &mockSettings
			}
			beforeQtl.Totals = append(beforeQtl.Totals, blockrewards.QualifyingTotals{
				Round:              round,
				Capacity:           mockCapacity,
				Used:               mockUsage,
				LastSettingsChange: 0,
				SettingsChange:     setting,
			})
			afterQtl.Totals = append(afterQtl.Totals, blockrewards.QualifyingTotals{
				Round:              round,
				Capacity:           mockCapacity,
				Used:               mockUsage,
				LastSettingsChange: 0,
				SettingsChange:     setting,
			})
		}

		balances.On(
			"GetTrieNode",
			blockrewards.QualifyingTotalsPerBlockKey,
		).Return(&beforeQtl, nil).Once()

		var qt blockrewards.QualifyingTotals
		qt.Capacity += p.deltaCapacity
		qt.Used += p.deltaUsed
		qt.Round = p.round
		if p.newBlockRewardSettings != nil {
			qt.LastSettingsChange = p.round
			qt.SettingsChange = p.newBlockRewardSettings
		} else {
			qt.LastSettingsChange = 0
			qt.SettingsChange = nil
		}

		if len(beforeQtl.Totals) > 0 {
			qt.Capacity += beforeQtl.Totals[len(beforeQtl.Totals)-1].Capacity
			qt.Used += beforeQtl.Totals[len(beforeQtl.Totals)-1].Used
		}
		afterQtl.Totals = append(afterQtl.Totals, qt)
		balances.On(
			"InsertTrieNode",
			blockrewards.QualifyingTotalsPerBlockKey,
			&afterQtl,
		).Return("", nil).Once()

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
				round:                  5,
				deltaCapacity:          1024,
				deltaUsed:              2048,
				newBlockRewardSettings: &mockSettings2,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			balances := setup(t, tt.parameters)
			_ = setExpectations(t, tt.parameters, balances, tt.want)

			err := UpdateRewardTotalList(balances)

			require.EqualValues(t, tt.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errorMsg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, balances))
		})
	}
}
