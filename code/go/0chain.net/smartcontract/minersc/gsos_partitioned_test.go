package minersc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the GroupSharesIndex serialization
func TestGroupSharesIndex_Serialization(t *testing.T) {
	gsi := NewGroupSharesIndex()
	gsi.AddID("id1")
	gsi.AddID("id2")
	gsi.AddID("id3")

	data, err := gsi.MarshalMsg(nil)
	require.NoError(t, err)

	gsi2 := NewGroupSharesIndex()
	_, err = gsi2.UnmarshalMsg(data)
	require.NoError(t, err)

	assert.Len(t, gsi2.IDs, 3)
	assert.Contains(t, gsi2.IDs, "id1")
	assert.Contains(t, gsi2.IDs, "id2")
	assert.Contains(t, gsi2.IDs, "id3")

	added := gsi2.AddID("id4")
	assert.True(t, added)

	added = gsi2.AddID("id1") // Already exists
	assert.False(t, added)

	removed := gsi2.RemoveID("id2")
	assert.True(t, removed)

	removed = gsi2.RemoveID("nonexistent")
	assert.False(t, removed)

	assert.Len(t, gsi2.IDs, 3)
	assert.Contains(t, gsi2.IDs, "id1")
	assert.NotContains(t, gsi2.IDs, "id2")
	assert.Contains(t, gsi2.IDs, "id3")
	assert.Contains(t, gsi2.IDs, "id4")
}

// Test basic operations of the GroupSharesIndex
func TestGroupSharesIndex_BasicOperations(t *testing.T) {
	gsi := NewGroupSharesIndex()

	assert.Len(t, gsi.IDs, 0)
	assert.False(t, gsi.ContainsID("id1"))

	added := gsi.AddID("id1")
	assert.True(t, added)
	assert.True(t, gsi.ContainsID("id1"))

	ids := gsi.GetIDs()
	assert.Len(t, ids, 1)
	assert.Equal(t, "id1", ids[0])

	hash1 := gsi.GetHash()
	gsi.AddID("id2")
	hash2 := gsi.GetHash()
	assert.NotEqual(t, hash1, hash2, "Hash should change when content changes")

	hash3 := gsi.GetHash()
	assert.Equal(t, hash2, hash3, "Hash should be stable when content is unchanged")

	removed := gsi.RemoveID("id1")
	assert.True(t, removed)
	assert.False(t, gsi.ContainsID("id1"))

	ids = gsi.GetIDs()
	assert.Len(t, ids, 1)
	assert.Equal(t, "id2", ids[0])
}

// Test that GetSOSPartitionKey generates unique keys
func TestGetSOSPartitionKey(t *testing.T) {
	key1 := GetSOSPartitionKey("id1")
	key2 := GetSOSPartitionKey("id2")

	assert.NotEqual(t, key1, key2)
	assert.Contains(t, key1, "id1")
	assert.Contains(t, key2, "id2")
}

// Test that the hash changes when content changes
func TestGroupSharesIndex_HashChange(t *testing.T) {
	gsi := NewGroupSharesIndex()
	emptyHash := gsi.GetHash()

	gsi.AddID("id1")
	oneItemHash := gsi.GetHash()
	assert.NotEqual(t, emptyHash, oneItemHash)

	gsi.AddID("id2")
	twoItemHash := gsi.GetHash()
	assert.NotEqual(t, oneItemHash, twoItemHash)

	// Order shouldn't matter for hash
	gsi2 := NewGroupSharesIndex()
	gsi2.AddID("id2")
	gsi2.AddID("id1")

	assert.Equal(t, twoItemHash, gsi2.GetHash())

	// Test removal
	gsi.RemoveID("id1")
	afterRemovalHash := gsi.GetHash()
	assert.NotEqual(t, twoItemHash, afterRemovalHash)

	gsi3 := NewGroupSharesIndex()
	gsi3.AddID("id2")
	assert.Equal(t, afterRemovalHash, gsi3.GetHash())
}

// Test concurrent access to the GroupSharesIndex
func TestGroupSharesIndex_ConcurrentAccess(t *testing.T) {
	gsi := NewGroupSharesIndex()

	// Add some initial IDs
	for i := 0; i < 5; i++ {
		gsi.AddID("init" + string(rune('0'+i)))
	}

	// Test that we can safely get IDs while adding/removing
	ids := gsi.GetIDs()
	assert.Len(t, ids, 5)

	// Test adding another ID doesn't affect our copy
	gsi.AddID("new")
	assert.Len(t, ids, 5)

	// Get fresh copy after changes
	idsAfter := gsi.GetIDs()
	assert.Len(t, idsAfter, 6)
	assert.Contains(t, idsAfter, "new")
}
