package vestingsc

import (
	// "strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require"
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

func Test_config_validate(t *testing.T) {
	for i, tt := range []struct {
		config config
		err    string
	}{
		// min lock
		{config{-1, 0, 0, 0, 0, 0, 0}, "invalid min_lock (<= 0)"},
		{config{0, 0, 0, 0, 0, 0, 0}, "invalid min_lock (<= 0)"},
		// min duration
		{config{1, s(-1), 0, 0, 0, 0, 0}, "invalid min_duration (< 1s)"},
		{config{1, s(0), 0, 0, 0, 0, 0}, "invalid min_duration (< 1s)"},
		// max duration
		{config{1, s(1), s(0), 0, 0, 0, 0},
			"invalid max_duration: less or equal to min_duration"},
		{config{1, s(1), s(1), 0, 0, 0, 0},
			"invalid max_duration: less or equal to min_duration"},
		// min friquency
		{config{1, s(1), s(2), s(-1), 0, 0, 0}, "invalid min_friquency (< 1s)"},
		{config{1, s(1), s(2), s(0), 0, 0, 0}, "invalid min_friquency (< 1s)"},
		// max friquency
		{config{1, s(1), s(2), s(1), s(0), 0, 0},
			"invalid max_friquency: less or equal to min_friquency"},
		{config{1, s(1), s(2), s(1), s(1), 0, 0},
			"invalid max_friquency: less or equal to min_friquency"},
		// max_destinations
		{config{1, s(1), s(2), s(1), s(2), 0, 0},
			"invalid max_destinations (< 1)"},
		// max_description_length
		{config{1, s(1), s(2), s(1), s(2), 1, 0},
			"invalid max_description_length (< 1)"},
	} {
		t.Log(i)
		assertErrMsg(t, tt.config.validate(), tt.err)
	}
}

func TestVestingSmartContract_getConfigBytes(t *testing.T) {
	//
}

func Test_getConfiguredConfig(t *testing.T) {
	//
}

func TestVestingSmartContract_setupConfig(t *testing.T) {
	//
}

func TestVestingSmartContract_getConfig(t *testing.T) {
	//
}

func TestVestingSmartContract_getConfigHandler(t *testing.T) {
	//
}

func TestVestingSmartContract_updateConfig(t *testing.T) {
	//
}
