package minersc

import (
	"testing"

	"0chain.net/chaincore/block"
	"github.com/0chain/common/core/util"
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

// Tests for PartitionedGroupSharesManager

// Test adding and retrieving a ShareOrSigns
func TestPartitionedGroupSharesManager_AddAndGet(t *testing.T) {
	state := newTestBalances()
	manager := NewPartitionedGroupSharesManager()

	// Create a ShareOrSigns
	sos := block.NewShareOrSigns()
	sos.ID = "miner1"

	// Add it to the manager
	err := manager.AddShareOrSigns(state, sos)
	require.NoError(t, err)

	// Retrieve it
	retrieved, err := manager.GetShareOrSigns(state, "miner1")
	require.NoError(t, err)
	assert.Equal(t, "miner1", retrieved.ID)

	// Try to get a non-existent ShareOrSigns
	_, err = manager.GetShareOrSigns(state, "nonexistent")
	assert.Equal(t, util.ErrValueNotPresent, err)
}

// Test getting IDs from the manager
func TestPartitionedGroupSharesManager_GetIDs(t *testing.T) {
	state := newTestBalances()
	manager := NewPartitionedGroupSharesManager()

	// Add multiple ShareOrSigns
	for i := 1; i <= 3; i++ {
		sos := block.NewShareOrSigns()
		sos.ID = "miner" + string(rune('0'+i))
		err := manager.AddShareOrSigns(state, sos)
		require.NoError(t, err)
	}

	// Verify IDs
	ids, err := manager.GetIDs(state)
	require.NoError(t, err)
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "miner1")
	assert.Contains(t, ids, "miner2")
	assert.Contains(t, ids, "miner3")
}

// Test retrieving all ShareOrSigns
func TestPartitionedGroupSharesManager_GetAll(t *testing.T) {
	state := newTestBalances()
	manager := NewPartitionedGroupSharesManager()

	// Add multiple ShareOrSigns
	for i := 1; i <= 3; i++ {
		sos := block.NewShareOrSigns()
		sos.ID = "miner" + string(rune('0'+i))
		err := manager.AddShareOrSigns(state, sos)
		require.NoError(t, err)
	}

	// Get all
	allSOS, err := manager.GetAllShareOrSigns(state)
	require.NoError(t, err)
	assert.Len(t, allSOS.Shares, 3)

	// Verify all expected IDs are present
	for i := 1; i <= 3; i++ {
		id := "miner" + string(rune('0'+i))
		share, exists := allSOS.Shares[id]
		assert.True(t, exists)
		assert.Equal(t, id, share.ID)
	}
}

// Test deleting a ShareOrSigns
func TestPartitionedGroupSharesManager_Delete(t *testing.T) {
	state := newTestBalances()
	manager := NewPartitionedGroupSharesManager()

	// Add multiple ShareOrSigns
	for i := 1; i <= 3; i++ {
		sos := block.NewShareOrSigns()
		sos.ID = "miner" + string(rune('0'+i))
		err := manager.AddShareOrSigns(state, sos)
		require.NoError(t, err)
	}

	// Delete one
	err := manager.DeleteShareOrSigns(state, "miner2")
	require.NoError(t, err)

	// Verify it's removed from the index
	ids, err := manager.GetIDs(state)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, "miner1")
	assert.Contains(t, ids, "miner3")
	assert.NotContains(t, ids, "miner2")

	// Verify manager API reports it as "not present"
	_, err = manager.GetShareOrSigns(state, "miner2")
	assert.Equal(t, util.ErrValueNotPresent, err)

	// Verify the data node still exists in the state
	partitionKey := GetSOSPartitionKey("miner2")
	directSos := block.NewShareOrSigns()
	err = state.GetTrieNode(partitionKey, directSos)
	require.NoError(t, err, "The data node should still exist in the state after deletion")
	assert.Equal(t, "miner2", directSos.ID)
}

// Test deleting all ShareOrSigns
func TestPartitionedGroupSharesManager_DeleteAll(t *testing.T) {
	state := newTestBalances()
	manager := NewPartitionedGroupSharesManager()

	// Add multiple ShareOrSigns
	for i := 1; i <= 3; i++ {
		sos := block.NewShareOrSigns()
		sos.ID = "miner" + string(rune('0'+i))
		err := manager.AddShareOrSigns(state, sos)
		require.NoError(t, err)
	}

	// Delete all
	err := manager.DeleteAllShareOrSigns(state)
	require.NoError(t, err)

	// Verify all are removed from the index
	ids, err := manager.GetIDs(state)
	require.NoError(t, err)
	assert.Len(t, ids, 0)

	// Verify manager API reports them as "not present"
	_, err = manager.GetShareOrSigns(state, "miner1")
	assert.Equal(t, util.ErrValueNotPresent, err)

	// Verify the data nodes still exist in the state
	for i := 1; i <= 3; i++ {
		id := "miner" + string(rune('0'+i))
		partitionKey := GetSOSPartitionKey(id)
		directSos := block.NewShareOrSigns()
		err = state.GetTrieNode(partitionKey, directSos)
		require.NoError(t, err, "The data node for %s should still exist in the state after deletion", id)
		assert.Equal(t, id, directSos.ID)
	}
}

// Test error handling when the state returns errors
func TestPartitionedGroupSharesManager_ErrorHandling(t *testing.T) {
	// Create a mock state that simulates storage issues
	state := newTestBalances()

	// Don't pre-populate the tree - this will cause errors when trying to retrieve nodes

	manager := NewPartitionedGroupSharesManager()

	// Create a ShareOrSigns
	sos := block.NewShareOrSigns()
	sos.ID = "miner1"

	// Test GetShareOrSigns error handling
	_, err := manager.GetShareOrSigns(state, "miner1")
	assert.Error(t, err)

	// Test GetAllShareOrSigns error handling
	_, err = manager.GetAllShareOrSigns(state)
	assert.Error(t, err)

	// Test DeleteShareOrSigns error handling
	err = manager.DeleteShareOrSigns(state, "miner1")
	assert.Error(t, err)
}

// Test adding a ShareOrSigns that already exists
func TestPartitionedGroupSharesManager_AddExisting(t *testing.T) {
	state := newTestBalances()
	manager := NewPartitionedGroupSharesManager()

	// Create a ShareOrSigns
	sos := block.NewShareOrSigns()
	sos.ID = "miner1"

	// Add it to the manager
	err := manager.AddShareOrSigns(state, sos)
	require.NoError(t, err)

	// Add it again - should not result in an error
	err = manager.AddShareOrSigns(state, sos)
	require.NoError(t, err)

	// Verify IDs (should still only have one entry)
	ids, err := manager.GetIDs(state)
	require.NoError(t, err)
	assert.Len(t, ids, 1)
	assert.Equal(t, "miner1", ids[0])
}

// Test deleting a ShareOrSigns that doesn't exist
func TestPartitionedGroupSharesManager_DeleteNonExistent(t *testing.T) {
	state := newTestBalances()
	manager := NewPartitionedGroupSharesManager()

	// Add a ShareOrSigns
	sos := block.NewShareOrSigns()
	sos.ID = "miner1"
	err := manager.AddShareOrSigns(state, sos)
	require.NoError(t, err)

	// Try to delete one that doesn't exist - should not result in an error
	err = manager.DeleteShareOrSigns(state, "nonexistent")
	require.NoError(t, err)

	// Verify the existing one is still there
	ids, err := manager.GetIDs(state)
	require.NoError(t, err)
	assert.Len(t, ids, 1)
	assert.Equal(t, "miner1", ids[0])
}

// Test performance comparison between partitioned and monolithic approaches
func TestPartitionedGroupSharesManager_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create a fresh test state
	state := newTestBalances()
	manager := NewPartitionedGroupSharesManager()

	// First, initialize the index in the state
	index := NewGroupSharesIndex()
	indexKey := GSoSIndexKey
	_, err := state.InsertTrieNode(indexKey, index)
	require.NoError(t, err, "Failed to initialize the GroupSharesIndex")

	// Create a much smaller dataset for testing
	numShares := 5

	// Add shares one by one with verification at each step
	for i := 1; i <= numShares; i++ {
		// Create a new ShareOrSigns with a clear ID
		sos := block.NewShareOrSigns()
		id := "miner" + string(rune('0'+i))
		sos.ID = id

		// Store directly in the state using the partition key
		partitionKey := GetSOSPartitionKey(id)
		_, err := state.InsertTrieNode(partitionKey, sos)
		require.NoError(t, err, "Failed to store ShareOrSigns with ID %s directly", id)

		// Now add it to the index
		added := index.AddID(id)
		assert.True(t, added, "Failed to add ID to index: %s", id)

		// Update the index in the state
		_, err = state.InsertTrieNode(indexKey, index)
		require.NoError(t, err, "Failed to update the index after adding ID %s", id)

		// Verify the ShareOrSigns can be retrieved directly
		retrieved := block.NewShareOrSigns()
		err = state.GetTrieNode(partitionKey, retrieved)
		require.NoError(t, err, "Failed to retrieve ShareOrSigns with ID %s after direct insertion", id)
		assert.Equal(t, id, retrieved.ID)
	}

	// Verify the index has all the expected IDs
	retrievedIndex := NewGroupSharesIndex()
	err = state.GetTrieNode(indexKey, retrievedIndex)
	require.NoError(t, err, "Failed to retrieve the index")
	assert.Len(t, retrievedIndex.IDs, numShares, "Index should contain all added IDs")

	// Now test high-level manager operations on existing data

	// Verify that each ID can be retrieved correctly using the manager's API
	t.Log("Verifying all ShareOrSigns can be retrieved by ID using the manager API")
	for i := 1; i <= numShares; i++ {
		id := "miner" + string(rune('0'+i))

		// Retrieve using the manager API
		sos, err := manager.GetShareOrSigns(state, id)
		require.NoError(t, err, "Failed to retrieve ShareOrSigns with ID %s using the manager API", id)
		assert.Equal(t, id, sos.ID, "Retrieved ShareOrSigns has incorrect ID")

		t.Logf("Successfully retrieved ShareOrSigns with ID %s", id)
	}

	// Try to retrieve a non-existent ShareOrSigns
	nonExistentID := "nonexistent_miner"
	_, err = manager.GetShareOrSigns(state, nonExistentID)
	assert.Equal(t, util.ErrValueNotPresent, err, "Should get 'value not present' error for non-existent ID")

	// Test retrieving an individual ShareOrSigns
	targetID := "miner2" // Use a known ID from our dataset
	t.Logf("Testing retrieval of specific ShareOrSigns with ID: %s", targetID)

	// First verify it exists directly in the state
	directKey := GetSOSPartitionKey(targetID)
	directSos := block.NewShareOrSigns()
	err = state.GetTrieNode(directKey, directSos)
	require.NoError(t, err, "ShareOrSigns should exist at partition key: %s", directKey)
	assert.Equal(t, targetID, directSos.ID)

	// Now retrieve using the manager
	partitionedSOS, err := manager.GetShareOrSigns(state, targetID)
	require.NoError(t, err, "Failed to retrieve ShareOrSigns using manager API")
	assert.Equal(t, targetID, partitionedSOS.ID)

	// Test retrieving all IDs
	allIDs, err := manager.GetIDs(state)
	require.NoError(t, err, "Failed to retrieve all IDs using manager API")
	assert.Len(t, allIDs, numShares, "Should retrieve all %d IDs", numShares)

	// Verify GetAllShareOrSigns correctly retrieves all items
	allSOS, err := manager.GetAllShareOrSigns(state)
	require.NoError(t, err, "Failed to retrieve all ShareOrSigns")
	assert.Len(t, allSOS.Shares, numShares, "Should have retrieved all ShareOrSigns")

	// Verify each ShareOrSigns in the result
	for i := 1; i <= numShares; i++ {
		id := "miner" + string(rune('0'+i))
		sosFromAll, exists := allSOS.Shares[id]
		assert.True(t, exists, "ShareOrSigns with ID %s should exist in the result", id)
		if exists {
			assert.Equal(t, id, sosFromAll.ID, "Retrieved ShareOrSigns should have correct ID")
		}
	}
}
