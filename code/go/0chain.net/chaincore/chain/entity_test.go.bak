package chain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainGetGeneratorsNum(t *testing.T) {
	tt := []struct {
		name        string
		min         int
		poolSize    int
		percent     float64
		expectedNum int
	}{
		{
			name:        "percent num < min num, percent=0.0, min=2, got 2",
			min:         2,
			poolSize:    10,
			percent:     0.0,
			expectedNum: 2,
		},
		{
			name:        "percent num < min, percent=0.2, pool=11 min=4, got 4",
			min:         4,
			poolSize:    11,
			percent:     0.2,
			expectedNum: 4,
		},
		{
			name:        "percent num = min, percent=0.2, min=2, got 2",
			min:         2,
			poolSize:    10,
			percent:     0.2,
			expectedNum: 2,
		},
		{
			name:        "percent num = min, percent=0.2, pool=11 min=3, got 3",
			min:         3,
			poolSize:    11,
			percent:     0.2,
			expectedNum: 3,
		},
		{
			name:        "percent num > min, percent=0.5, min=2, got 5",
			min:         2,
			poolSize:    10,
			percent:     0.2,
			expectedNum: 2,
		},
		{
			name:        "percent num > min, percent=0.2, pool=11 min=2, got 3",
			min:         2,
			poolSize:    11,
			percent:     0.2,
			expectedNum: 3,
		},
		{
			name:        "all miners, percent=1.0, min=2, got 10",
			min:         2,
			poolSize:    10,
			percent:     1.0,
			expectedNum: 10,
		},
		{
			name:        "all miners, percent=0.0, min=0, got 0",
			min:         0,
			poolSize:    10,
			percent:     0.0,
			expectedNum: 0,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			num := getGeneratorsNum(tc.poolSize, tc.min, tc.percent)
			require.Equal(t, tc.expectedNum, num)
		})
	}
}
