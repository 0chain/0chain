package minersc_test

import (
	"testing"

	"0chain.net/chaincore/state"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"

	. "0chain.net/smartcontract/minersc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSettings(t *testing.T) {
	require.Len(t, SettingName, int(NumberOfSettings))
	require.Len(t, Settings, int(NumberOfSettings))
	require.Len(t, ConfitTypeName, int(NumberOfTypes))

	for _, name := range SettingName {
		require.EqualValues(t, name, SettingName[Settings[name].Setting])
	}
}

func TestUpdateSettigns(t *testing.T) {
	type args struct {
		msc      *MinerSmartContract
		txn      *transaction.Transaction
		input    []byte
		gn       *GlobalNode
		balances chainstate.StateContextI
	}

	type parameters struct {
		client   string
		inputMap map[string]interface{}
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
					if gn.Get(Settings[key].Setting) != value {
						return false
					}
				}
				return true
			}),
		).Return("", nil).Once()

		return args{
			msc:      msc,
			txn:      txn,
			input:    (&InputMap{p.inputMap}).Encode(),
			gn:       &GlobalNode{},
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
				client: "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802",
				inputMap: map[string]interface{}{
					"min_stake":              zcnToBalance(0.0),
					"max_stake":              zcnToBalance(100),
					"max_n":                  int(7),
					"min_n":                  int(3),
					"t_percent":              float64(0.66),
					"k_percent":              float64(0.75),
					"x_percent":              float64(0.70),
					"max_s":                  int(2),
					"min_s":                  int(1),
					"max_delegates":          int(200),
					"reward_round_frequency": int64(250),
					"interest_rate":          float64(0.0),
					"reward_rate":            float64(1.0),
					"share_ratio":            float64(50),
					"block_reward":           zcnToBalance(0.21),
					"max_charge":             float64(0.5),
					"epoch":                  int64(15000000),
					"reward_decline_rate":    float64(0.1),
					"interest_decline_rate":  float64(0.1),
					"max_mint":               zcnToBalance(1500000.0),
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

func zcnToBalance(token float64) state.Balance {
	const x10 float64 = 10 * 1000 * 1000 * 1000
	return state.Balance(token * x10)
}
