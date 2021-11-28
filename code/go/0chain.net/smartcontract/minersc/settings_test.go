package minersc_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	chainstate "0chain.net/chaincore/chain/state"

	"0chain.net/smartcontract"

	"0chain.net/chaincore/state"

	"0chain.net/chaincore/mocks"
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

		balances.On(
			"InsertTrieNode",
			GlobalNodeKey,
			mock.MatchedBy(func(gn *GlobalNode) bool {
				for key, value := range p.inputMap {
					//if gn.Get(Settings[key].Setting) != value {
					//	return false
					//}

					//var setting interface{} = getConfField(*conf, key)
					setting, _ := gn.Get(Settings[key].Setting)
					fmt.Println("setting", setting, "value", value)
					switch Settings[key].ConfigType {
					case smartcontract.Int:
						{
							expected, err := strconv.Atoi(value)
							require.NoError(t, err)
							actual, ok := setting.(int)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case smartcontract.Int64:
						{
							expected, err := strconv.ParseInt(value, 10, 64)
							require.NoError(t, err)
							actual, ok := setting.(int64)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case smartcontract.Float64:
						{
							expected, err := strconv.ParseFloat(value, 64)
							require.NoError(t, err)
							actual, ok := setting.(float64)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case smartcontract.Boolean:
						{
							expected, err := strconv.ParseBool(value)
							require.NoError(t, err)
							actual, ok := setting.(bool)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case smartcontract.Duration:
						{
							expected, err := time.ParseDuration(value)
							require.NoError(t, err)
							actual, ok := setting.(time.Duration)
							require.True(t, ok)
							if expected != actual {
								return false
							}
						}
					case smartcontract.StateBalance:
						{
							expected, err := strconv.ParseFloat(value, 64)
							expected = x10 * expected
							require.NoError(t, err)
							actual, ok := setting.(state.Balance)
							require.True(t, ok)
							if state.Balance(expected) != actual {
								return false
							}
						}
					}
				}
				return true
			}),
		).Return("", nil).Once()

		return args{
			msc:      msc,
			txn:      txn,
			input:    (&smartcontract.StringMap{p.inputMap}).Encode(),
			gn:       &GlobalNode{OwnerId: owner},
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
					"min_stake":              "0.0",
					"max_stake":              "100",
					"max_n":                  "7",
					"min_n":                  "3",
					"t_percent":              "0.66",
					"k_percent":              "0.75",
					"x_percent":              "0.70",
					"max_s":                  "2",
					"min_s":                  "1",
					"max_delegates":          "200",
					"reward_round_frequency": "64250",
					"interest_rate":          "0.0",
					"reward_rate":            "1.0",
					"share_ratio":            "50",
					"block_reward":           "021",
					"max_charge":             "0.5",
					"epoch":                  "6415000000",
					"reward_decline_rate":    "0.1",
					"interest_decline_rate":  "0.1",
					"max_mint":               "1500000.0",
					"owner_id":               owner,
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
