package minersc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Just test the GroupSharesIndex serialization
func TestGroupSharesIndex_Serialization(t *testing.T) {
	// Create an index with some IDs
	gsi := NewGroupSharesIndex()
	gsi.AddID("id1")
	gsi.AddID("id2")
	gsi.AddID("id3")

	// Serialize
	data, err := gsi.MarshalMsg(nil)
	require.NoError(t, err)

	// Deserialize into a new instance
	gsi2 := NewGroupSharesIndex()
	_, err = gsi2.UnmarshalMsg(data)
	require.NoError(t, err)

	// Verify the data was preserved
	assert.Len(t, gsi2.IDs, 3)
	assert.Contains(t, gsi2.IDs, "id1")
	assert.Contains(t, gsi2.IDs, "id2")
	assert.Contains(t, gsi2.IDs, "id3")

	// Test adding and removing
	added := gsi2.AddID("id4")
	assert.True(t, added)

	added = gsi2.AddID("id1") // Already exists
	assert.False(t, added)

	removed := gsi2.RemoveID("id2")
	assert.True(t, removed)

	removed = gsi2.RemoveID("nonexistent")
	assert.False(t, removed)

	// Verify final state
	assert.Len(t, gsi2.IDs, 3)
	assert.Contains(t, gsi2.IDs, "id1")
	assert.NotContains(t, gsi2.IDs, "id2")
	assert.Contains(t, gsi2.IDs, "id3")
	assert.Contains(t, gsi2.IDs, "id4")
}
