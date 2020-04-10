package vestingsc

import (
	"context"
	"testing"
	"time"

	configpkg "0chain.net/chaincore/config"
	"0chain.net/core/datastore"
	"0chain.net/core/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_configKey(t *testing.T) {
	const vscKey = "vsc-key"
	assert.Equal(t, vscKey+":configurations", configKey(vscKey))
}

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

func Test_config_Encode_Decode(t *testing.T) {
	var confe, confd = avgConfig(), new(config)
	require.NoError(t, confd.Decode(confe.Encode()))
	assert.EqualValues(t, confe, confd)
}

func Test_config_validate(t *testing.T) {

	var (
		invcs = []datastore.Key{"", "c"}
		invts = []datastore.Key{"", "t"}

		cs = []datastore.Key{"one-c", "two-c"}
		tr = []datastore.Key{"one-t", "two-t"}
	)

	for i, tt := range []struct {
		config config
		err    string
	}{
		// configurators
		{config{nil, nil, 0, 0, 0, 0, 0, 0, 0}, "empty configurators list"},
		// triggers
		{config{cs, nil, 0, 0, 0, 0, 0, 0, 0}, "empty triggers list"},
		// min lock
		{config{cs, tr, -1, 0, 0, 0, 0, 0, 0}, "invalid min_lock (<= 0)"},
		{config{cs, tr, 0, 0, 0, 0, 0, 0, 0}, "invalid min_lock (<= 0)"},
		// min duration
		{config{cs, tr, 1, s(-1), 0, 0, 0, 0, 0},
			"invalid min_duration (< 1s)"},
		{config{cs, tr, 1, s(0), 0, 0, 0, 0, 0},
			"invalid min_duration (< 1s)"},
		// max duration
		{config{cs, tr, 1, s(1), s(0), 0, 0, 0, 0},
			"invalid max_duration: less or equal to min_duration"},
		{config{cs, tr, 1, s(1), s(1), 0, 0, 0, 0},
			"invalid max_duration: less or equal to min_duration"},
		// min friquency
		{config{cs, tr, 1, s(1), s(2), s(-1), 0, 0, 0},
			"invalid min_friquency (< 1s)"},
		{config{cs, tr, 1, s(1), s(2), s(0), 0, 0, 0},
			"invalid min_friquency (< 1s)"},
		// max friquency
		{config{cs, tr, 1, s(1), s(2), s(1), s(0), 0, 0},
			"invalid max_friquency: less or equal to min_friquency"},
		{config{cs, tr, 1, s(1), s(2), s(1), s(1), 0, 0},
			"invalid max_friquency: less or equal to min_friquency"},
		// max_destinations
		{config{cs, tr, 1, s(1), s(2), s(1), s(2), 0, 0},
			"invalid max_destinations (< 1)"},
		// max_description_length
		{config{cs, tr, 1, s(1), s(2), s(1), s(2), 1, 0},
			"invalid max_description_length (< 1)"},
		//
		{config{invcs, tr, 1, s(1), s(2), s(1), s(2), 1, 1},
			"empty configurator ID in list"},
		{config{cs, invts, 1, s(1), s(2), s(1), s(2), 1, 1},
			"empty trigger ID in list"},
	} {
		t.Log(i)
		assertErrMsg(t, tt.config.validate(), tt.err)
	}
}

func TestVestingSmartContract_getConfigBytes(t *testing.T) {
	var (
		balances = newTestBalances()
		vsc      = newTestVestingSC()
		confb    []byte
		set      *config
		err      error
	)
	_, err = vsc.getConfigBytes(balances)
	assert.Equal(t, util.ErrValueNotPresent, err)
	set = setConfig(t, balances)
	confb, err = vsc.getConfigBytes(balances)
	require.NoError(t, err)
	assert.Equal(t, string(set.Encode()), string(confb))
}

func configureConfig() (configured *config) {
	const pfx = "smart_contracts.vestingsc."

	configpkg.SmartContractConfig.Set(pfx+"configurators",
		[]string{"one-c", "two-c"})
	configpkg.SmartContractConfig.Set(pfx+"triggers",
		[]string{"one-t", "two-t"})
	configpkg.SmartContractConfig.Set(pfx+"min_lock", 10)
	configpkg.SmartContractConfig.Set(pfx+"min_duration", 1*time.Minute)
	configpkg.SmartContractConfig.Set(pfx+"max_duration", 10*time.Minute)
	configpkg.SmartContractConfig.Set(pfx+"min_friquency", 2*time.Minute)
	configpkg.SmartContractConfig.Set(pfx+"max_friquency", 20*time.Minute)
	configpkg.SmartContractConfig.Set(pfx+"max_destinations", 50)
	configpkg.SmartContractConfig.Set(pfx+"max_description_length", 255)

	return &config{
		[]datastore.Key{"one-c", "two-c"},
		[]datastore.Key{"one-t", "two-t"},
		10,
		1 * time.Minute, 10 * time.Minute,
		2 * time.Minute, 20 * time.Minute,
		50, 255,
	}
}

func Test_getConfiguredConfig_setupConfig(t *testing.T) {
	var (
		expected  = configureConfig()
		conf, err = getConfiguredConfig()
	)
	require.NoError(t, err)
	assert.EqualValues(t, expected, conf)

	var (
		vsc      = newTestVestingSC()
		balances = newTestBalances()
	)
	conf, err = vsc.setupConfig(balances)
	require.NoError(t, err)
	assert.EqualValues(t, expected, conf)
}

func TestVestingSmartContract_getConfig(t *testing.T) {
	var (
		vsc       = newTestVestingSC()
		balances  = newTestBalances()
		conf, err = vsc.getConfig(balances, false)

		configured, set *config
	)
	assert.Equal(t, util.ErrValueNotPresent, err)
	configured = configureConfig()
	conf, err = vsc.getConfig(balances, true)
	require.NoError(t, err)
	assert.EqualValues(t, configured, conf)
	set = setConfig(t, balances)
	conf, err = vsc.getConfig(balances, false)
	require.NoError(t, err)
	assert.EqualValues(t, set, conf)
}

func TestVestingSmartContract_updateConfig(t *testing.T) {
	var (
		vsc      = newTestVestingSC()
		balances = newTestBalances()
		txn      = newTransaction("not-owner", ADDRESS, 0, 10)
		set      = avgConfig()

		resp string
		err  error
	)
	balances.txn = txn

	setConfig(t, balances)

	_, err = vsc.updateConfig(txn, mustEncode(t, set), balances)
	require.Error(t, err)

	txn.ClientID = "one-c"
	set.MinLock = -1
	_, err = vsc.updateConfig(txn, mustEncode(t, set), balances)
	require.Error(t, err)

	set.MinLock = 144
	resp, err = vsc.updateConfig(txn, mustEncode(t, set), balances)
	require.NoError(t, err)

	assert.Equal(t, resp, string(set.Encode()))
	var get *config
	get, err = vsc.getConfig(balances, false)
	require.NoError(t, err)
	assert.EqualValues(t, set, get)
}

func TestVestingSmartContract_getConfigHandler(t *testing.T) {

	var (
		vsc        = newTestVestingSC()
		balances   = newTestBalances()
		ctx        = context.Background()
		configured = configureConfig()
		resp, err  = vsc.getConfigHandler(ctx, nil, balances)
		set        *config
	)

	require.NoError(t, err)
	assert.EqualValues(t, configured, resp)

	set = setConfig(t, balances)
	resp, err = vsc.getConfigHandler(ctx, nil, balances)
	require.NoError(t, err)
	assert.EqualValues(t, set, resp)
}
