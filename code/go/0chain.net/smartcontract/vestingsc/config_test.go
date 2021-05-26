package vestingsc

import (
	"context"
	"testing"
	"time"

	configpkg "0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertErrMsg(t *testing.T, err error, msg string) {
	t.Helper()

	if msg == "" {
		assert.Nil(t, err)
		return
	}

	if assert.NotNil(t, err) {
		assert.Equal(t, msg, err.Error())
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
		assertErrMsg(t, tt.config.validate(), tt.err)
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
	assert.EqualValues(t, configured, conf)
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
	assert.EqualValues(t, configured, resp)
}

func TestVestingSmartContractUpdate(t *testing.T) {

	var (
		vsc            = newTestVestingSC()
		balances       = newTestBalances()
		ownerTxn       = newTransaction(owner, vsc.ID, 0, common.Now())
		nonOwnerTxn    = newTransaction(randString(32), vsc.ID, 0, common.Now())
		originalConfig = configureConfig()
		err            error
		currentConfig  *config
	)

	//test cases that produce errors
	errorTestCases := []struct {
		title string
		txn   *transaction.Transaction
		bytes []byte
		err   string
	}{
		{"malformed update", ownerTxn, []byte("} malformed {"), "update_config: invalid character '}' looking for beginning of value"},
		{"non owner account", nonOwnerTxn, []byte("} malformed {"), "update_config: unauthorized access - only the owner can update the variables"},
	}
	for _, tc := range errorTestCases {
		t.Run(tc.title, func(t *testing.T) {
			balances.txn = tc.txn
			_, err = vsc.updateConfig(tc.txn, tc.bytes, balances)
			require.Error(t, err)
			require.EqualError(t, err, tc.err)
		})
	}

	//test cases that will be denied
	deniedTestCases := []struct {
		title       string
		request     *config
		requireFunc func(config, request *config)
	}{
		{"all variables denied",
			&config{},
			func(config, request *config) {
				require.Equal(t, config, request)
			},
		},
	}
	for _, tc := range deniedTestCases {
		t.Run(tc.title, func(t *testing.T) {
			_, err = vsc.updateConfig(ownerTxn, tc.request.Encode(), balances)
			require.NoError(t, err)
			currentConfig, err = vsc.getConfig(balances)
			require.NoError(t, err)
			tc.requireFunc(currentConfig, originalConfig)
		})
	}

	//test cases that will be updated
	updateTestCases := []struct {
		title       string
		request     *config
		requireFunc func(config, request *config)
	}{
		{
			"min lock update",
			&config{
				MinLock: 987654321,
			},
			func(config, request *config) {
				require.Equal(t, config.MinLock, request.MinLock)
			},
		},
		{
			"min duration update",
			&config{
				MinDuration: time.Hour * 10,
			},
			func(config, request *config) {
				require.Equal(t, config.MinDuration, request.MinDuration)
			},
		},
		{
			"max duration update",
			&config{
				MaxDuration: time.Hour * 87600,
			},
			func(config, request *config) {
				require.Equal(t, config.MaxDuration, request.MaxDuration)
			},
		},
		{
			"max destinations update",
			&config{
				MaxDestinations: 57,
			},
			func(config, request *config) {
				require.Equal(t, config.MaxDestinations, request.MaxDestinations)
			},
		},
		{
			"max description length update",
			&config{
				MaxDescriptionLength: 32,
			},
			func(config, request *config) {
				require.Equal(t, config.MaxDescriptionLength, request.MaxDescriptionLength)
			},
		},
	}
	for _, tc := range updateTestCases {
		t.Run(tc.title, func(t *testing.T) {
			balances.txn = ownerTxn
			_, err = vsc.updateConfig(ownerTxn, tc.request.Encode(), balances)
			require.NoError(t, err)
			currentConfig, err = vsc.getConfig(balances)
			require.NoError(t, err)
			tc.requireFunc(currentConfig, tc.request)
		})
	}
}
