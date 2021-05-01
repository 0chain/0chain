package vestingsc

import (
	"context"
	"testing"
	"time"

	configpkg "0chain.net/chaincore/config"
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
		tx             = newTransaction(owner, vsc.ID, 0, common.Now())
		originalConfig = configureConfig()
		err            error
		currentConfig  *config
		update         = &config{}
	)

	// 1. Malformed update
	t.Run("malformed update", func(t *testing.T) {
		_, err = vsc.updateConfig(tx, []byte("} malformed {"), balances)
		assertErrMsg(t, err, "update_config: invalid character '}' looking for beginning of value")
	})

	// 2. Non owner account tries to update
	t.Run("non owner account", func(t *testing.T) {
		tx.ClientID = randString(32)
		_, err = vsc.updateConfig(tx, []byte("} malformed {"), balances)
		assertErrMsg(t, err, "update_config: unauthorized access - only the owner can update the variables")
	})

	// 3. All variables requested shall be denied
	t.Run("all variables denied", func(t *testing.T) {
		tx.ClientID = owner
		_, err = vsc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		currentConfig, err = vsc.getConfig(balances)
		require.NoError(t, err)
		assert.EqualValues(t, currentConfig, originalConfig)
	})

	// 4. Min lock will update
	t.Run("min lock update", func(t *testing.T) {
		update.MinLock = 987654321
		_, err = vsc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		currentConfig, err = vsc.getConfig(balances)
		require.NoError(t, err)
		assert.EqualValues(t, currentConfig.MinLock, update.MinLock)
	})

	// 5. Min duration will update
	t.Run("min duration update", func(t *testing.T) {
		update.MinDuration = time.Hour * 10
		_, err = vsc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		currentConfig, err = vsc.getConfig(balances)
		require.NoError(t, err)
		assert.EqualValues(t, currentConfig.MinDuration, update.MinDuration)
	})

	// 6. Max duration will update
	t.Run("max duration update", func(t *testing.T) {
		update.MaxDuration = time.Hour * 87600
		_, err = vsc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		currentConfig, err = vsc.getConfig(balances)
		require.NoError(t, err)
		assert.EqualValues(t, currentConfig.MaxDuration, update.MaxDuration)
	})

	// 7. Max destinations will update
	t.Run("max destinations update", func(t *testing.T) {
		update.MaxDestinations = 57
		_, err = vsc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		currentConfig, err = vsc.getConfig(balances)
		require.NoError(t, err)
		assert.EqualValues(t, currentConfig.MaxDestinations, update.MaxDestinations)
	})

	// 8. Max description length will update
	t.Run("max description length update", func(t *testing.T) {
		update.MaxDescriptionLength = 32
		_, err = vsc.updateConfig(tx, mustEncode(t, update), balances)
		require.NoError(t, err)
		currentConfig, err = vsc.getConfig(balances)
		require.NoError(t, err)
		assert.EqualValues(t, currentConfig.MaxDescriptionLength, update.MaxDescriptionLength)
	})
}
