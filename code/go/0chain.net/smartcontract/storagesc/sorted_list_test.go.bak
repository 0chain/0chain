package storagesc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_sortedList(t *testing.T) {

	// getIndex(id string) (i int, ok bool)
	// removeByIndex(i int)
	// remove(id string) (ok bool)
	// add(id string) (ok bool)

	var (
		sl    SortedList
		order = []string{"b1", "b2", "b3"}
		not   = []string{"b0", "b4"}
		i     int
		ok    bool
	)

	for _, id := range order {
		_, ok = sl.getIndex(id)
		require.False(t, ok)
	}

	for _, id := range order {
		require.True(t, sl.add(id))
	}
	require.Len(t, sl, 3)

	for _, id := range order {
		require.False(t, sl.add(id))
	}
	require.Len(t, sl, 3)

	for k, id := range order {
		i, ok = sl.getIndex(id)
		require.True(t, ok)
		require.Equal(t, k, i)
	}

	for _, id := range not {
		_, ok = sl.getIndex(id)
		require.False(t, ok)
	}

	var cp = make(SortedList, len(sl))
	copy(cp, sl)

	for range order {
		sl.removeByIndex(0)
	}
	require.Len(t, sl, 0)

	sl = cp
	for k, id := range order {
		require.True(t, sl.remove(id))
		assert.Len(t, sl, 3-k-1)
	}
	assert.Len(t, sl, 0)

	for _, id := range order {
		require.False(t, sl.remove(id))
	}
	assert.Len(t, sl, 0)

}
