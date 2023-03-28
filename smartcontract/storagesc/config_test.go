package storagesc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetBlockPayments(t *testing.T) {
	type want struct {
		reward blockReward
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
			BlobberCapacityRatio: 35.0,

			want: want{
				blockReward{
					SharderWeight: 0.1,
					MinerWeight:   0.2,
					BlobberWeight: 0.7,
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var br = blockReward{}
			err := br.setWeightsFromRatio(tt.SharderRatio, tt.MinerRatio, tt.BlobberCapacityRatio)
			require.NoError(t, err)
			require.EqualValues(t, br, tt.want.reward)
		})
	}
}
