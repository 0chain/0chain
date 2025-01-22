package minersc_test

import (
	"0chain.net/chaincore/block"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
	"time"

	"0chain.net/core/config"
	"github.com/0chain/common/core/currency"

	chainstate "0chain.net/chaincore/chain/state"

	"0chain.net/chaincore/chain/state/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"

	. "0chain.net/smartcontract/minersc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const x10 float64 = 10 * 1000 * 1000 * 1000

const owner = "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

func TestSettings(t *testing.T) {
	require.Len(t, SettingName, int(NumberOfSettings))
	require.Len(t, Settings, int(NumberOfSettings))

	for _, name := range SettingName {
		require.EqualValues(t, name, SettingName[Settings[name].Setting])
	}
}

func enableHardForks(t *testing.T, tb *mocks.StateContextI) {
	hardForks := []string{"apollo", "ares", "artemis", "athena", "demeter", "electra", "hercules", "hermes"}

	for _, name := range hardForks {
		h := chainstate.NewHardFork(name, 0)
		tb.On("InsertTrieNode", h.GetKey(), h).Return("", nil).Once()
		if _, err := tb.InsertTrieNode(h.GetKey(), h); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUpdateSettings(t *testing.T) {
	type args struct {
		msc      *MinerSmartContract
		txn      *transaction.Transaction
		input    []byte
		gn       *GlobalNode
		balances chainstate.StateContextI
	}

	type parameters struct {
		client   string
		inputMap map[string]string
	}

	setExpectations := func(t *testing.T, p parameters) args {
		var balances = &mocks.StateContextI{}
		var msc = &MinerSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		var txn = &transaction.Transaction{
			ClientID: p.client,
		}

		balances.On("GetBlock", mock.Anything, mock.Anything).Return(&block.Block{}, nil)

		balances.On(
			"InsertTrieNode",
			GlobalNodeKey,
			mock.MatchedBy(func(gn *GlobalNode) bool {
				for key, value := range p.inputMap {
					setting, _ := gn.Get(Settings[key].Setting)
					switch Settings[key].ConfigType {
					case config.Int:
						{
							expected, err := strconv.Atoi(value)
							require.NoError(t, err)
							actual, ok := setting.(int)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.Int64:
						{
							expected, err := strconv.ParseInt(value, 10, 64)
							require.NoError(t, err)
							actual, ok := setting.(int64)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.Float64:
						{
							expected, err := strconv.ParseFloat(value, 64)
							require.NoError(t, err)
							actual, ok := setting.(float64)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.Boolean:
						{
							expected, err := strconv.ParseBool(value)
							require.NoError(t, err)
							actual, ok := setting.(bool)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.Duration:
						{
							expected, err := time.ParseDuration(value)
							require.NoError(t, err)
							actual, ok := setting.(time.Duration)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.CurrencyCoin:
						{
							expected, err := strconv.ParseFloat(value, 64)
							expected = x10 * expected
							require.NoError(t, err)
							actual, ok := setting.(currency.Coin)
							require.True(t, ok)
							if currency.Coin(expected) != actual {
								return false
							}
						}
					case config.Cost:
						{
							expected, err := strconv.Atoi(value)
							require.NoError(t, err)
							actual, ok := setting.(int)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case config.Key:
						{
							_, err := hex.DecodeString(value)
							require.NoError(t, err)
							actual, ok := setting.(string)
							require.True(t, ok)
							if value != actual {
								return false
							}
						}
					default:
						return false
					}

				}
				return true
			}),
		).Return("", nil).Once()

		balances.On(
			"GetTrieNode",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("*state.HardFork"),
		).Return(nil, nil).Maybe()

		enableHardForks(t, balances)

		return args{
			msc:      msc,
			txn:      txn,
			input:    (&config.StringMap{p.inputMap}).Encode(),
			gn:       NewGlobalNode(owner, make(map[string]int)),
			balances: balances,
		}
	}

	type want struct {
		error bool
		msg   string
	}

	testCases := []struct {
		title      string
		parameters parameters
		want       want
	}{
		{
			title: "all_settigns",
			parameters: parameters{
				client: owner,
				inputMap: map[string]string{
					"min_stake":                           "0.0",
					"max_stake":                           "100",
					"max_n":                               "7",
					"min_n":                               "3",
					"t_percent":                           "0.66",
					"k_percent":                           "0.75",
					"x_percent":                           "0.70",
					"max_s":                               "2",
					"min_s":                               "1",
					"max_delegates":                       "200",
					"reward_round_frequency":              "64250",
					"reward_rate":                         "1.0",
					"share_ratio":                         "50",
					"block_reward":                        "021",
					"max_charge":                          "0.5",
					"epoch":                               "6415000000",
					"reward_decline_rate":                 "0.1",
					"owner_id":                            owner,
					"cost.add_miner":                      "111",
					"cost.add_sharder":                    "111",
					"cost.delete_miner":                   "111",
					"cost.miner_health_check":             "111",
					"cost.sharder_health_check":           "111",
					strings.ToLower("cost.contributeMpk"): "111",
					strings.ToLower("cost.shareSignsOrShares"): "111",
					"cost.wait":                                    "111",
					"cost.update_globals":                          "111",
					"cost.update_settings":                         "111",
					"cost.update_miner_settings":                   "111",
					"cost.update_sharder_settings":                 "111",
					strings.ToLower("cost.payFees"):                "111",
					strings.ToLower("cost.feesPaid"):               "111",
					strings.ToLower("cost.mintedTokens"):           "111",
					strings.ToLower("cost.addToDelegatePool"):      "111",
					strings.ToLower("cost.deleteFromDelegatePool"): "111",
					"cost.sharder_keep":                            "111",
					"cost.kill_miner":                              "111",
					"cost.kill_sharder":                            "111",
				},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			test := test
			t.Parallel()
			args := setExpectations(t, test.parameters)

			_, err := args.msc.UpdateSettings(args.txn, args.input, args.gn, args.balances)
			require.EqualValues(t, test.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.msg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}
