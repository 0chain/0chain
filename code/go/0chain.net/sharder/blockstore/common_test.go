package blockstore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetUint64ValueFromYamlConfig(t *testing.T) {
	m := map[uint64]interface{}{
		12:  12,
		120: "12*10",
		130: "10*13",
		8:   "2^8",
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
		8:   "2^8",
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
