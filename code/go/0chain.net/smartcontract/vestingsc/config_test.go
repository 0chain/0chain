package vestingsc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract"
	"github.com/stretchr/testify/mock"

	configpkg "0chain.net/chaincore/config"

	"github.com/stretchr/testify/require"
)

func requireErrMsg(t *testing.T, err error, msg string) {
	t.Helper()
	if msg == "" {
		require.Nil(t, err)
	} else {
		require.NotNil(t, err)
		require.Equal(t, msg, err.Error())
	}
}

func s(n time.Duration) time.Duration {
	return n * time.Second
}

func Test_config_validate(t *testing.T) {

	for _, tt := range []struct {
		config config
		err    string
	}{
		// min lock
		{config{-1, 0, 0, 0, 0, ""}, "invalid min_lock (<= 0)"},
		{config{0, 0, 0, 0, 0, ""}, "invalid min_lock (<= 0)"},
		// min duration
		{config{1, s(-1), 0, 0, 0, ""}, "invalid min_duration (< 1s)"},
		{config{1, s(0), 0, 0, 0, ""}, "invalid min_duration (< 1s)"},
		// max duration
		{config{1, s(1), s(0), 0, 0, ""},
			"invalid max_duration: less or equal to min_duration"},
		{config{1, s(1), s(1), 0, 0, ""},
			"invalid max_duration: less or equal to min_duration"},
		// max_destinations
		{config{1, s(1), s(2), 0, 0, ""}, "invalid max_destinations (< 1)"},
		// max_description_length
		{config{1, s(1), s(2), 1, 0, ""}, "invalid max_description_length (< 1)"},
		{config{1, s(1), s(2), 1, 1, ""}, "owner_id is not set or empty"},
	} {
		requireErrMsg(t, tt.config.validate(), tt.err)
	}
}

func configureConfig() (configured *config) {
	const pfx = "smart_contracts.vestingsc."

	configpkg.SmartContractConfig.Set(pfx+"min_lock", 100)
	configpkg.SmartContractConfig.Set(pfx+"min_duration", 1*time.Second)
	configpkg.SmartContractConfig.Set(pfx+"max_duration", 10*time.Hour)
	configpkg.SmartContractConfig.Set(pfx+"max_destinations", 2)
	configpkg.SmartContractConfig.Set(pfx+"max_description_length", 20)
	configpkg.SmartContractConfig.Set(pfx+"owner_id", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")

	return &config{
		100e10,
		1 * time.Second, 10 * time.Hour,
		2, 20, "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802",
	}
}

func TestVestingSmartContract_getConfigHandler(t *testing.T) {
	var (
		vsc        = newTestVestingSC()
		balances   = newTestBalances()
		ctx        = context.Background()
		configured = configureConfig()
		resp, err  = vsc.getConfigHandler(ctx, nil, balances)
	)
	require.NoError(t, err)
	require.EqualValues(t, configured.getConfigMap(), resp)
}

func TestUpdateConfig(t *testing.T) {
	const (
		mockNotOwner = "mock not the owner"
		mockBadData  = "mock bad data"
		mockBadKey   = "mock bad key"
	)
	type args struct {
		vsc      *VestingSmartContract
		txn      *transaction.Transaction
		input    []byte
		balances chainstate.StateContextI
	}

	type parameters struct {
		client string
		input  map[string]string
	}

	type want struct {
		error bool
		msg   string
	}

	setExpectations := func(t *testing.T, p parameters) args {
		var balances = &mocks.StateContextI{}
		var vsc = &VestingSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		var txn = &transaction.Transaction{
			ClientID: p.client,
		}
		var inputObj = smartcontract.StringMap{
			Fields: p.input,
		}
		input, err := json.Marshal(&inputObj)
		require.NoError(t, err)
		prevConf := configureConfig()
		balances.On("GetTrieNode", scConfigKey(vsc.ID)).Return(prevConf, nil).Once()
		var conf config
		// not testing for error here to allow entering bad data
		if value, ok := p.input[Settings[MinLock]]; ok {
			fValue, _ := strconv.ParseFloat(value, 64)
			conf.MinLock = state.Balance(fValue * 1e10)
		}
		if value, ok := p.input[Settings[MinDuration]]; ok {
			conf.MinDuration, err = time.ParseDuration(value)
		}
		if value, ok := p.input[Settings[MaxDuration]]; ok {
			conf.MaxDuration, err = time.ParseDuration(value)
		}
		if value, ok := p.input[Settings[MaxDestinations]]; ok {
			conf.MaxDestinations, err = strconv.Atoi(value)
		}
		if value, ok := p.input[Settings[MaxDescriptionLength]]; ok {
			conf.MaxDescriptionLength, err = strconv.Atoi(value)
		}
		if value, ok := p.input[Settings[OwnerId]]; ok {
			conf.OwnerId = value
		}
		fmt.Println("setExpectations conf", conf)
		balances.On(
			"InsertTrieNode",
			scConfigKey(vsc.ID),
			&conf,
		).Return("", nil).Once()

		return args{
			vsc:      vsc,
			txn:      txn,
			input:    input,
			balances: balances,
		}
	}

	testCases := []struct {
		title      string
		parameters parameters
		want       want
	}{
		{
			title: "ok_all",
			parameters: parameters{
				client: owner,
				input: map[string]string{
					Settings[MinLock]:              "5",
					Settings[MinDuration]:          "1s",
					Settings[MaxDuration]:          "1h",
					Settings[MaxDestinations]:      "0",
					Settings[MaxDescriptionLength]: "17",
					Settings[OwnerId]:              "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802",
				},
			},
		},
		{
			title: "not_owner",
			parameters: parameters{
				client: mockNotOwner,
				input: map[string]string{
					Settings[MaxDuration]: "1h",
					Settings[OwnerId]:     "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802",
				},
			},
			want: want{
				error: true,
				msg:   "update_config: unauthorized access - only the owner can access",
			},
		},
		{
			title: "bad_data",
			parameters: parameters{
				client: owner,
				input: map[string]string{
					Settings[MinDuration]: mockBadData,
					Settings[OwnerId]:     "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802",
				},
			},
			want: want{
				error: true,
				msg:   "update_config: value mock bad data cannot be converted to time.Duration, failing to set config key min_duration",
			},
		},
		{
			title: "bad_key",
			parameters: parameters{
				client: owner,
				input: map[string]string{
					mockBadKey:        "1",
					Settings[OwnerId]: "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802",
				},
			},
			want: want{
				error: true,
				msg:   "update_config: config setting mock bad key not found",
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			test := test
			t.Parallel()
			args := setExpectations(t, test.parameters)

			_, err := args.vsc.updateConfig(args.txn, args.input, args.balances)
			require.EqualValues(t, test.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.msg, err.Error(), test)
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}
