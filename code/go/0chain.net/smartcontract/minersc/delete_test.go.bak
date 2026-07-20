package minersc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveIDs(t *testing.T) {
	tt := []struct {
		name   string
		init   NodeIDs
		remove NodeIDs
		expect NodeIDs
	}{
		{
			name:   "remove first 1",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"1"},
			expect: NodeIDs{"2", "3", "4", "5", "6", "7", "8"},
		},
		{
			name:   "remove first 2",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"1", "2"},
			expect: NodeIDs{"3", "4", "5", "6", "7", "8"},
		},
		{
			name:   "remove random 2",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"1", "4"},
			expect: NodeIDs{"2", "3", "5", "6", "7", "8"},
		},
		{
			name:   "remove middle 3",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"3", "4", "5"},
			expect: NodeIDs{"1", "2", "6", "7", "8"},
		},
		{
			name:   "remove last 1",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"8"},
			expect: NodeIDs{"1", "2", "3", "4", "5", "6", "7"},
		},
		{
			name:   "remove last 3",
			init:   NodeIDs{"1", "2", "3", "4", "5", "6", "7", "8"},
			remove: NodeIDs{"6", "7", "8"},
			expect: NodeIDs{"1", "2", "3", "4", "5"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expect, removeIDs(tc.init, tc.remove))
		})
	}
}
