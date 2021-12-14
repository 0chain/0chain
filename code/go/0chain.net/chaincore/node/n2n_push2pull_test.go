package node

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPullingEntityCache(t *testing.T) {
	tt := []struct {
		name      string
		trueIndex int
		expect    int
	}{
		{
			name:      "return on first pull",
			trueIndex: 0,
			expect:    10,
		},
		{
			name:      "return on second pull",
			trueIndex: 1,
			expect:    11,
		},
		{
			name:      "return on third pull",
			trueIndex: 2,
			expect:    12,
		},
		{
			name:      "no return",
			trueIndex: 5,
			expect:    0,
		},
		{
			name:      "no return more",
			trueIndex: 6,
			expect:    0,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			cc := newPullingCache(10, 5)

			var value int
			for i := 0; i < 10; i++ {
				func(ct int) {
					cc.pullOrCacheRequest(context.Background(), "key1", func(ctx context.Context) bool {
						if ct < tc.trueIndex {
							return false
						}
						time.Sleep(100 * time.Duration(1+ct) * time.Millisecond)

						select {
						case <-ctx.Done():
							return false
						default:
							value = 10 + ct
							return true
						}
					})
				}(i)
			}

			time.Sleep(time.Duration(200+(tc.trueIndex*100)) * time.Millisecond)
			require.Equal(t, tc.expect, value)
		})
	}
}
