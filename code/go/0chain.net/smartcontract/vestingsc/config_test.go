package vestingsc

import (
	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"

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
		{config{-1, 0, 0, 0, 0}, "invalid min_lock (<= 0)"},
		{config{0, 0, 0, 0, 0}, "invalid min_lock (<= 0)"},
		// min duration
		{config{1, s(-1), 0, 0, 0}, "invalid min_duration (< 1s)"},
		{config{1, s(0), 0, 0, 0}, "invalid min_duration (< 1s)"},
		// max duration
		{config{1, s(1), s(0), 0, 0},
			"invalid max_duration: less or equal to min_duration"},
		{config{1, s(1), s(1), 0, 0},
			"invalid max_duration: less or equal to min_duration"},
		// max_destinations
		{config{1, s(1), s(2), 0, 0}, "invalid max_destinations (< 1)"},
		// max_description_length
		{config{1, s(1), s(2), 1, 0}, "invalid max_description_length (< 1)"},
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

	return &config{
		100e10,
		1 * time.Second, 10 * time.Hour,
		2, 20,
	}
}

func Test_getConfig(t *testing.T) {
	var (
		vsc        = newTestVestingSC()
		balances   = newTestBalances()
		configured = configureConfig()
		conf, err  = vsc.getConfig(balances)
	)
	require.NoError(t, err)
	require.EqualValues(t, configured, conf)
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
	require.EqualValues(t, configured, resp)
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
		input  map[string]interface{}
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
		var inputObj = inputMap{
			Fields: p.input,
		}
		input, err := json.Marshal(&inputObj)
		require.NoError(t, err)

		balances.On("GetTrieNode", scConfigKey(vsc.ID)).Return(&config{}, nil).Once()
		var conf config
		if value, ok := p.input[Settings[MinLock]]; ok {
			conf.MinLock, ok = value.(state.Balance)
		}
		if value, ok := p.input[Settings[MinDuration]]; ok {
			conf.MinDuration = value.(time.Duration)
		}
		if value, ok := p.input[Settings[MaxDuration]]; ok {
			conf.MaxDuration = value.(time.Duration)
		}
		if value, ok := p.input[Settings[MaxDestinations]]; ok {
			conf.MaxDestinations = value.(int)
		}
		if value, ok := p.input[Settings[MaxDescriptionLength]]; ok {
			conf.MaxDescriptionLength = value.(int)
		}

		balances.On(
			"InsertTrieNode",
			scConfigKey(vsc.ID),
			&conf).Return("", nil).Once()

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
				input: map[string]interface{}{
					Settings[MinLock]:              state.Balance(5),
					Settings[MinDuration]:          time.Second,
					Settings[MaxDuration]:          time.Hour,
					Settings[MaxDestinations]:      int(0),
					Settings[MaxDescriptionLength]: int(17),
				},
			},
		},
		{
			title: "ok_1",
			parameters: parameters{
				client: owner,
				input: map[string]interface{}{
					Settings[MaxDuration]: time.Hour,
				},
			},
		},
		{
			title: "not_owner",
			parameters: parameters{
				client: mockNotOwner,
				input: map[string]interface{}{
					Settings[MaxDuration]: time.Hour,
				},
			},
			want: want{
				error: true,
				msg:   "update_config: unauthorized access - only the owner can update the variables",
			},
		},
		{
			title: "bad_data",
			parameters: parameters{
				client: owner,
				input: map[string]interface{}{
					mockBadKey: mockBadData,
				},
			},
			want: want{
				error: true,
				msg:   "update_config: value mock bad data is not of numeric type, failing to set config key mock bad key",
			},
		},
		{
			title: "bad_key",
			parameters: parameters{
				client: owner,
				input: map[string]interface{}{
					mockBadKey: 1,
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
				require.EqualValues(t, test.want.msg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}
