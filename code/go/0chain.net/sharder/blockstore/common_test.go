package blockstore

import (
	"testing"

	"github.com/0chain/common/core/logging"

	"github.com/stretchr/testify/require"
)

func init() {
	logging.InitLogging("debug", ".")
}

func TestGetUint64ValueFromYamlConfig(t *testing.T) {
	m := map[uint64]interface{}{
		12:  12,
		120: "12*10",
		130: "10*13",
		8:   "2^3",
		1:   "1^10",
	}

	for k, v := range m {
		convertedValue, err := getUint64ValueFromYamlConfig(v)
		require.Nil(t, err)
		require.Equal(t, k, convertedValue)
	}

	values := []interface{}{
		"12a",
		12.23,
		nil,
		make(map[int]int),
	}

	for _, v := range values {
		_, err := getUint64ValueFromYamlConfig(v)
		require.NotNil(t, err)
	}
}

func TestGetIntValueFromYamlConfig(t *testing.T) {
	m := map[int]interface{}{
		12:  12,
		120: "12*10",
		130: "10*13",
		8:   "2^3",
		1:   "1^10",
	}

	for k, v := range m {
		convertedValue, err := getintValueFromYamlConfig(v)
		require.Nil(t, err)
		require.Equal(t, k, convertedValue)
	}

	values := []interface{}{
		"12a",
		12.23,
		nil,
		make(map[int]int),
	}

	for _, v := range values {
		_, err := getintValueFromYamlConfig(v)
		require.NotNil(t, err)
	}
}

func TestLock(t *testing.T) {
	mu := make(Mutex, 1)
	t.Log("Locking")
	mu.Lock()

	select {
	case mu <- struct{}{}:
		t.Fail()
	default:
	}

	require.NotPanics(t, func() {
		t.Log("Unlocking")
		mu.Unlock()
		t.Log("Unlocked")
	})

	require.Panics(t, func() {
		t.Log("Unlocking unlocked lock")
		mu.Unlock()
	})

	t.Log("Locking")
	mu.Lock()

	require.Equal(t, false, mu.TryLock())
	mu.Unlock()
	require.Equal(t, true, mu.TryLock())
	require.Equal(t, false, mu.TryLock())

	require.NotPanics(t, func() {
		mu.Unlock()
	})
}
