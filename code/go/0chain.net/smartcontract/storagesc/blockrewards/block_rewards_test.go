package blockrewards

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetBlockPayments(t *testing.T) {
	type want struct {
		reward BlockReward
	}

	tests := []struct {
		name                 string
		SharderRatio         float64
		MinerRatio           float64
		BlobberCapacityRatio float64
		BlobberUsageRatio    float64
		want                 want
	}{
		{
			name: "zeros",
		},
		{
			name:                 "equal",
			SharderRatio:         5.0,
			MinerRatio:           10.0,
			BlobberCapacityRatio: 15.0,
			BlobberUsageRatio:    20.0,

			want: want{
				BlockReward{
					SharderWeight:         0.1,
					MinerWeight:           0.2,
					BlobberCapacityWeight: 0.3,
					BlobberUsageWeight:    0.4,
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var br = BlockReward{}
			br.SetWeightsFromRatio(tt.SharderRatio, tt.MinerRatio, tt.BlobberCapacityRatio, tt.BlobberUsageRatio)
			require.EqualValues(t, br, tt.want.reward)
		})
	}
}
