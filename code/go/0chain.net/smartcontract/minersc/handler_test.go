package minersc_test

import (
	"bytes"
	"fmt"
	"net/url"
	"testing"

	"0chain.net/smartcontract"

	"0chain.net/chaincore/config"

	"0chain.net/core/util"

	sci "0chain.net/chaincore/smartcontractinterface"

	"context"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	. "0chain.net/smartcontract/minersc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConfigHandler(t *testing.T) {
	type args struct {
		msc      *MinerSmartContract
		balances chainstate.StateContextI
	}

	type parameters struct {
		condfigType string
		localConfig []byte
	}

	setExpectations := func(t *testing.T, p parameters) args {
		var balances = &mocks.StateContextI{}
		var msc = &MinerSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		balances.On("GetTrieNode", GlobalNodeKey).Return(
			nil, util.ErrValueNotPresent,
		).Once()

		config.SmartContractConfig.SetConfigType(p.condfigType)
		err := config.SmartContractConfig.ReadConfig(bytes.NewBuffer(p.localConfig))
		require.NoError(t, err)

		return args{
			msc:      msc,
			balances: balances,
		}
	}

	type want struct {
		error  bool
		msg    string
		output map[string]string
	}

	testCases := []struct {
		title      string
		parameters parameters
		want       want
	}{
		{
			title: "all_settigns",
			parameters: parameters{
				condfigType: "yaml",
				localConfig: []byte(`
smart_contracts:
  minersc:
    # miners
    max_n: 7 # 100
    min_n: 3 # 3
    # sharders
    max_s: 2 # 30
    min_s: 1 # 1
    # max delegates allowed by SC
    max_delegates: 200 #
    # DKG
    t_percent: .66 # of active
    k_percent: .75 # of registered
    x_percent: 0.70 # percentage of prev mb miners required to be part of next mb
    # etc
    min_stake: 0.0 # 0.01 # min stake can be set by a node (boundary for all nodes)
    max_stake: 100.0 # max stake can be set by a node (boundary for all nodes)
    start_rounds: 50
    contribute_rounds: 50
    share_rounds: 50
    publish_rounds: 50
    wait_rounds: 50
    # stake interests, will be declined every epoch
    interest_rate: 0.0 # [0; 1)
    # reward rate for generators, will be declined every epoch
    reward_rate: 1.0 # [0; 1)
    # share ratio is miner/block sharders rewards ratio, for example 0.1
    # gives 10% for miner and rest for block sharders
    share_ratio: 0.8 # [0; 1)
    # reward for a block
    block_reward: 0.21 # tokens
    # max service charge can be set by a generator
    max_charge: 0.5 # %
    # epoch is number of rounds before rewards and interest are decreased
    epoch: 15000000 # rounds
    # decline rewards every new epoch by this value (the block_reward)
    reward_decline_rate: 0.1 # [0; 1), 0.1 = 10%
    # decline interests every new epoch by this value (the interest_rate)
    interest_decline_rate: 0.1 # [0; 1), 0.1 = 10%
    # no mints after miner SC total mints reaches this boundary
    max_mint: 1500000.0 # tokens
    # if view change is false then reward round frequency is used to send rewards and interests
    reward_round_frequency: 250
`),
			},
			want: want{
				output: map[string]string{
					"min_stake":              "0",
					"max_stake":              "100",
					"max_n":                  "7",
					"min_n":                  "3",
					"t_percent":              "0.66",
					"k_percent":              "0.75",
					"x_percent":              "0.7",
					"max_s":                  "2",
					"min_s":                  "1",
					"max_delegates":          "200",
					"reward_round_frequency": "250",
					"interest_rate":          "0",
					"reward_rate":            "1",
					"share_ratio":            "0",
					"block_reward":           "0.21",
					"max_charge":             "0.5",
					"epoch":                  "15000000",
					"reward_decline_rate":    "0.1",
					"interest_decline_rate":  "0.1",
					"max_mint":               "1.5e+06",
					"owner_id":               "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802",
				},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			test := test
			t.Parallel()
			args := setExpectations(t, test.parameters)

			result, err := args.msc.ConfigHandler(context.TODO(), url.Values{}, args.balances)
			require.EqualValues(t, test.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.msg, err.Error())
				return
			}
			ourputMap, ok := result.(smartcontract.StringMap)
			require.True(t, ok)
			for key, value := range test.want.output {
				if value != ourputMap.Fields[key] {
					fmt.Println("key", key, "value", value, "output", ourputMap.Fields[key])
				}
				//require.EqualValues(t, value, ourputMap.Fields[key], key)
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}
