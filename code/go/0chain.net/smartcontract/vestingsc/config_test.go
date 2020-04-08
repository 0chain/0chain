package vestingsc

import (
	"strings"
	"testing"

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

func Test_config_validate(t *testing.T) {
	for _, tt := range []struct {
		config config
		err    string
	}{
		// duration
		{{-1, 0, 0, 0, 0, 0},
			"invalid min_duration (< 1s)"},
		{{0, 0, 0, 0, 0, 0},
			"invalid min_duration (< 1s)"},
		{{1, 0, 0, 0, 0, 0},
			"invalid max_duration: less or equal to min_duration"},
		{{1, 1, 0, 0, 0, 0},
			"invalid max_duration: less or equal to min_duration"},
		// friquency
		{{1, 2, -1, 0, 0, 0},
			"invalid min_friquency (< 1s)"},
		{{1, 2, 0, 0, 0, 0},
			"invalid min_friquency (< 1s)"},
		{{1, 2, 1, 0, 0, 0},
			"invalid max_friquency: less or equal to min_friquency"},
		{{1, 2, 0, 0, 0, 0},
			"invalid max_friquency: less or equal to min_friquency"},
	} {
		assertErrMsg(t, tt.config.validate(), tt.err)
	}

	// MaxDestinations < 1: "invalid max_destinations (< 1)"
	// MaxNameLength < 1: "invalid max_name_length (< 1)"

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
