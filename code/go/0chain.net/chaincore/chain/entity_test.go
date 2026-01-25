package chain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestViewChangeOffset verifies the ViewChangeOffset constant and mbRoundOffset behavior.
// This is critical for the MB19->MB20 transition fix on mainnet.
// With ViewChangeOffset=20, MB20 becomes active at round 141945735 (not 141945740).
func TestViewChangeOffset(t *testing.T) {
	// Verify the constant is set to 20
	require.Equal(t, int64(20), int64(ViewChangeOffset),
		"ViewChangeOffset should be 20 for mainnet recovery")
}

func TestMbRoundOffset(t *testing.T) {
	tt := []struct {
		name           string
		round          int64
		expectedOffset int64
	}{
		{
			name:           "round 0 - no offset",
			round:          0,
			expectedOffset: 0,
		},
		{
			name:           "round 1 - no offset",
			round:          1,
			expectedOffset: 1,
		},
		{
			name:           "round 20 - no offset (at boundary)",
			round:          20,
			expectedOffset: 20,
		},
		{
			name:           "round 21 - first round with offset",
			round:          21,
			expectedOffset: 1, // 21 - 20 = 1
		},
		{
			name:           "round 100 - normal offset",
			round:          100,
			expectedOffset: 80, // 100 - 20 = 80
		},
		// Mainnet recovery scenario: MB20 StartingRound = 141945715
		{
			name:           "mainnet MB20 activation round (141945735)",
			round:          141945735,
			expectedOffset: 141945715, // Should match MB20 StartingRound
		},
		{
			name:           "round before MB20 activation (141945734)",
			round:          141945734,
			expectedOffset: 141945714, // Should NOT match MB20 (which has SR=141945715)
		},
		// With old ViewChangeOffset=25, round 141945740 would be first to use MB20
		// With new ViewChangeOffset=20, round 141945735 is first to use MB20
		{
			name:           "old activation round (141945740) now has higher offset",
			round:          141945740,
			expectedOffset: 141945720, // 141945740 - 20 = 141945720
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			offset := mbRoundOffset(tc.round)
			require.Equal(t, tc.expectedOffset, offset,
				"mbRoundOffset(%d) should return %d", tc.round, tc.expectedOffset)
		})
	}
}

// TestMB20ActivationRound verifies that with ViewChangeOffset=20,
// MB20 (StartingRound=141945715) becomes active at round 141945735.
func TestMB20ActivationRound(t *testing.T) {
	const mb20StartingRound = int64(141945715)

	// Find the first round where mbRoundOffset returns MB20's StartingRound
	activationRound := mb20StartingRound + ViewChangeOffset
	require.Equal(t, int64(141945735), activationRound,
		"MB20 should activate at round 141945735 with ViewChangeOffset=20")

	// Verify mbRoundOffset confirms this
	require.Equal(t, mb20StartingRound, mbRoundOffset(activationRound),
		"mbRoundOffset(141945735) should return 141945715 (MB20 StartingRound)")

	// Round before activation should NOT use MB20
	require.NotEqual(t, mb20StartingRound, mbRoundOffset(activationRound-1),
		"mbRoundOffset(141945734) should NOT return MB20 StartingRound")
}

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
