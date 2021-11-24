package blockstore

import (
	"context"
	"testing"
)

func Benchmark_newTiering(b *testing.B) {
	if err := InitializeStore(mockHotAndColdConfig(b), context.Background()); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = newTiering(HotTier, HotTier, Store.ColdTier.DeleteLocal)
	}
}

func Test_newTiering(t *testing.T) {
	t.Parallel()

	if err := InitializeStore(mockHotAndColdConfig(t), context.Background()); err != nil {
		t.Fatal(err)
	}

	tests := [2]struct {
		name        string
		prev        WhichTier
		skip        WhichTier
		deleteLocal bool
		wantNt      WhichTier
	}{
		{
			name:        "newTiering_true_DeleteLocal",
			prev:        HotTier,
			skip:        ColdTier,
			deleteLocal: true,
			wantNt:      HotTier, // HotTier + ColdTier - ColdTier
		},
		{
			name:        "newTiering_false_DeleteLocal",
			prev:        HotTier,
			skip:        0, // no skip for deleteLocal equals false
			deleteLocal: false,
			wantNt:      HotTier + ColdTier,
		},
	}
	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := newTiering(test.prev, test.skip, test.deleteLocal); got != test.wantNt {
				t.Errorf("newTiering() got: %v | want: %v", got, test.wantNt)
			}
		})
	}
}
