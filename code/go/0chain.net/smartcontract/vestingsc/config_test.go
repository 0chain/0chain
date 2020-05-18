package vestingsc

import (
	"context"
	"testing"
	"time"

	configpkg "0chain.net/chaincore/config"

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

	for i, tt := range []struct {
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
		t.Log(i)
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
		configured = configureConfig()
		conf, err  = getConfig()
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
