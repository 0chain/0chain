package minersc

import (
	"testing"

	"0chain.net/chaincore/block"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the GroupSharesManager serialization
func TestGroupSharesManager_Serialization(t *testing.T) {
	gsm := NewGroupSharesOrSignsV2()

	// Directly set IDs for testing
	gsm.IDs = append(gsm.IDs, "id1", "id2", "id3")

	data, err := gsm.MarshalMsg(nil)
	require.NoError(t, err)

	gsm2 := NewGroupSharesOrSignsV2()
	_, err = gsm2.UnmarshalMsg(data)
	require.NoError(t, err)

	assert.Len(t, gsm2.IDs, 3)
	assert.Contains(t, gsm2.IDs, "id1")
	assert.Contains(t, gsm2.IDs, "id2")
	assert.Contains(t, gsm2.IDs, "id3")

	// Test ContainsID
	assert.True(t, gsm2.ContainsID("id1"))
	assert.False(t, gsm2.ContainsID("nonexistent"))

	// Test GetIDs
	ids := gsm2.GetIDs()
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "id1")
	assert.Contains(t, ids, "id2")
	assert.Contains(t, ids, "id3")
}

// Test that GetSOSPartitionKey generates unique keys
func TestGetSOSPartitionKey(t *testing.T) {
	key1 := GetSOSPartitionKey("id1")
	key2 := GetSOSPartitionKey("id2")

	assert.NotEqual(t, key1, key2)
	assert.Contains(t, key1, "id1")
	assert.Contains(t, key2, "id2")
}

// Test adding and retrieving a ShareOrSigns
func TestGroupSharesManager_AddAndGet(t *testing.T) {
	state := newTestBalances()
	manager := NewGroupSharesOrSignsV2()

	// Create a ShareOrSigns
	sos := block.NewShareOrSigns()
	sos.ID = "miner1"

	// Add it to the manager
	err := manager.AddShareOrSigns(state, sos)
	require.NoError(t, err)

	// Load manager again to simulate fresh start
	retrieved := NewGroupSharesOrSignsV2()
	err = retrieved.Load(state)
	require.NoError(t, err)

	// Retrieve the ShareOrSigns
	retrievedSOS, err := retrieved.GetShareOrSigns(state, "miner1")
	require.NoError(t, err)
	assert.Equal(t, "miner1", retrievedSOS.ID)

	// Try to get a non-existent ShareOrSigns
	_, err = retrieved.GetShareOrSigns(state, "nonexistent")
	assert.Equal(t, util.ErrValueNotPresent, err)
}

// Test getting IDs from the manager
func TestGroupSharesManager_GetIDs(t *testing.T) {
	state := newTestBalances()
	manager := NewGroupSharesOrSignsV2()

	// Add multiple ShareOrSigns
	for i := 1; i <= 3; i++ {
		sos := block.NewShareOrSigns()
		sos.ID = "miner" + string(rune('0'+i))
		err := manager.AddShareOrSigns(state, sos)
		require.NoError(t, err)
	}

	// Load manager again to simulate fresh start
	retrieved := NewGroupSharesOrSignsV2()
	err := retrieved.Load(state)
	require.NoError(t, err)

	// Verify IDs
	ids := retrieved.GetIDs()
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "miner1")
	assert.Contains(t, ids, "miner2")
	assert.Contains(t, ids, "miner3")
}

// Test retrieving all ShareOrSigns
func TestGroupSharesManager_GetAll(t *testing.T) {
	state := newTestBalances()
	manager := NewGroupSharesOrSignsV2()

	// Add multiple ShareOrSigns
	for i := 1; i <= 3; i++ {
		sos := block.NewShareOrSigns()
		sos.ID = "miner" + string(rune('0'+i))
		err := manager.AddShareOrSigns(state, sos)
		require.NoError(t, err)
	}

	// Load manager again to simulate fresh start
	retrieved := NewGroupSharesOrSignsV2()
	err := retrieved.Load(state)
	require.NoError(t, err)

	// Get all
	allSOS, err := retrieved.GetAllShareOrSigns(state)
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
func TestGroupSharesManager_Delete(t *testing.T) {
	state := newTestBalances()
	manager := NewGroupSharesOrSignsV2()

	// Add multiple ShareOrSigns
	for i := 1; i <= 3; i++ {
		sos := block.NewShareOrSigns()
		sos.ID = "miner" + string(rune('0'+i))
		err := manager.AddShareOrSigns(state, sos)
		require.NoError(t, err)
	}

	// Load manager again to simulate fresh start
	retrieved := NewGroupSharesOrSignsV2()
	err := retrieved.Load(state)
	require.NoError(t, err)

	// Delete one
	err = retrieved.DeleteShareOrSigns(state, "miner2")
	require.NoError(t, err)

	// Load again to verify changes were saved
	verifier := NewGroupSharesOrSignsV2()
	err = verifier.Load(state)
	require.NoError(t, err)

	// Verify it's removed from the index
	ids := verifier.GetIDs()
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, "miner1")
	assert.Contains(t, ids, "miner3")
	assert.NotContains(t, ids, "miner2")

	// Verify manager API reports it as "not present"
	_, err = verifier.GetShareOrSigns(state, "miner2")
	assert.Equal(t, util.ErrValueNotPresent, err)

	// Verify the data node still exists in the state
	partitionKey := GetSOSPartitionKey("miner2")
	directSos := block.NewShareOrSigns()
	err = state.GetTrieNode(partitionKey, directSos)
	require.NoError(t, err, "The data node should still exist in the state after deletion")
	assert.Equal(t, "miner2", directSos.ID)
}

// Test deleting all ShareOrSigns
func TestGroupSharesManager_DeleteAll(t *testing.T) {
	state := newTestBalances()
	manager := NewGroupSharesOrSignsV2()

	// Add multiple ShareOrSigns
	for i := 1; i <= 3; i++ {
		sos := block.NewShareOrSigns()
		sos.ID = "miner" + string(rune('0'+i))
		err := manager.AddShareOrSigns(state, sos)
		require.NoError(t, err)
	}

	// Load manager again to simulate fresh start
	retrieved := NewGroupSharesOrSignsV2()
	err := retrieved.Load(state)
	require.NoError(t, err)

	// Delete all
	err = retrieved.DeleteAllShareOrSigns(state)
	require.NoError(t, err)

	// Load again to verify changes were saved
	verifier := NewGroupSharesOrSignsV2()
	err = verifier.Load(state)
	require.NoError(t, err)

	// Verify all are removed from the index
	ids := verifier.GetIDs()
	assert.Len(t, ids, 0)

	// Verify manager API reports them as "not present"
	_, err = verifier.GetShareOrSigns(state, "miner1")
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
func TestGroupSharesManager_ErrorHandling(t *testing.T) {
	// Create a mock state that simulates storage issues
	state := newTestBalances()

	// Don't pre-populate the tree - this will cause errors when trying to retrieve nodes
	manager := NewGroupSharesOrSignsV2()

	// Create a ShareOrSigns
	sos := block.NewShareOrSigns()
	sos.ID = "miner1"

	// Test GetShareOrSigns error handling
	_, err := manager.GetShareOrSigns(state, "miner1")
	assert.Equal(t, util.ErrValueNotPresent, err)

	// Test GetAllShareOrSigns with an empty index
	allSOS, err := manager.GetAllShareOrSigns(state)
	require.NoError(t, err)
	assert.Len(t, allSOS.Shares, 0)

	// Test DeleteShareOrSigns when ID doesn't exist
	err = manager.DeleteShareOrSigns(state, "miner1")
	assert.NoError(t, err)
}

// Test adding a ShareOrSigns that already exists
func TestGroupSharesManager_AddExisting(t *testing.T) {
	state := newTestBalances()
	manager := NewGroupSharesOrSignsV2()

	// Create a ShareOrSigns
	sos := block.NewShareOrSigns()
	sos.ID = "miner1"

	// Add it to the manager
	err := manager.AddShareOrSigns(state, sos)
	require.NoError(t, err)

	// Add it again - should not result in an error
	err = manager.AddShareOrSigns(state, sos)
	require.NoError(t, err)

	// Load manager again to simulate fresh start
	retrieved := NewGroupSharesOrSignsV2()
	err = retrieved.Load(state)
	require.NoError(t, err)

	// Verify IDs (should still only have one entry)
	ids := retrieved.GetIDs()
	assert.Len(t, ids, 1)
	assert.Equal(t, "miner1", ids[0])
}
