package storagesc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetBlockPayments(t *testing.T) {
	type want struct {
		sharder, miner, blobbeCapacity, blobberUsage int64
	}

	tests := []struct {
		name        string
		blockReward blockReward
		want        want
	}{
		{
			name: "zeros",
		},
		{
			name: "equal",
			blockReward: blockReward{
				BlockReward:           100,
				QualifyingStake:       50.0,
				SharderWeight:         5.0,
				MinerWeight:           10.0,
				BlobberCapacityWeight: 15.0,
				BlobberUsageWeight:    20.0,
			},
			want: want{
				sharder:        10,
				miner:          20,
				blobbeCapacity: 30,
				blobberUsage:   40,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, m, bc, bu := tt.blockReward.getBlockPayments()
			require.EqualValues(t, tt.want.sharder, s)
			require.EqualValues(t, tt.want.miner, m)
			require.EqualValues(t, tt.want.blobbeCapacity, bc)
			require.EqualValues(t, tt.want.blobberUsage, bu)
		})
	}
}
