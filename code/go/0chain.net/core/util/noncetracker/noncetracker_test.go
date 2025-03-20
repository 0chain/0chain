package noncetracker

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/0chain/common/core/logging"
)

// init logging
func init() {
	logging.InitLogging("testing", "")
}

// MockChainAPI implements ChainAPI for testing
type MockChainAPI struct {
	latestFinalizedBlock MockBlock
	clientNonces         map[string]int64
	mu                   sync.RWMutex
}

type MockBlock struct {
	Height int64
}

func NewMockChainAPI() *MockChainAPI {
	return &MockChainAPI{
		latestFinalizedBlock: MockBlock{Height: 1},
		clientNonces:         make(map[string]int64),
	}
}

func (m *MockChainAPI) GetLatestFinalizedBlock() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latestFinalizedBlock
}

func (m *MockChainAPI) GetCurrentNonce(clientID string, blockState interface{}) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if nonce, ok := m.clientNonces[clientID]; ok {
		return nonce, nil
	}
	return 0, nil
}

func (m *MockChainAPI) UpdateClientNonce(clientID string, nonce int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clientNonces[clientID] = nonce
}

func (m *MockChainAPI) UpdateBlock(height int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latestFinalizedBlock = MockBlock{Height: height}
}

// Create a variant of the mock API that returns nil for GetLatestFinalizedBlock
type FailingMockChainAPI struct {
	clientNonces map[string]int64
	mu           sync.RWMutex
}

func NewFailingMockChainAPI() *FailingMockChainAPI {
	return &FailingMockChainAPI{
		clientNonces: make(map[string]int64),
	}
}

func (m *FailingMockChainAPI) GetLatestFinalizedBlock() interface{} {
	return nil
}

func (m *FailingMockChainAPI) GetCurrentNonce(clientID string, blockState interface{}) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if nonce, ok := m.clientNonces[clientID]; ok {
		return nonce, nil
	}
	return 0, nil
}

func TestNewNonceTracker(t *testing.T) {
	mockChain := NewMockChainAPI()
	clientID := "test_client"

	tracker := NewNonceTracker(clientID, mockChain)

	if tracker.clientID != clientID {
		t.Errorf("Expected clientID %s, got %s", clientID, tracker.clientID)
	}

	if tracker.confirmedNonce != 0 {
		t.Errorf("Expected initial confirmedNonce 0, got %d", tracker.confirmedNonce)
	}

	if len(tracker.pendingNonces) != 0 {
		t.Errorf("Expected empty pendingNonces, got %d items", len(tracker.pendingNonces))
	}

	if len(tracker.pendingBlocks) != 0 {
		t.Errorf("Expected empty pendingBlocks, got %d items", len(tracker.pendingBlocks))
	}
}

func TestReserveNoncesForBlock(t *testing.T) {
	mockChain := NewMockChainAPI()
	clientID := "test_client"

	tracker := NewNonceTracker(clientID, mockChain)

	// Set initial confirmed nonce
	tracker.confirmedNonce = 100

	blockHash := "block123"
	height := int64(105)
	count := 5

	nonces := tracker.ReserveNoncesForBlock(blockHash, height, count)

	// Check returned nonces
	if len(nonces) != count {
		t.Errorf("Expected %d nonces, got %d", count, len(nonces))
	}

	for i, nonce := range nonces {
		expected := int64(101 + i)
		if nonce != expected {
			t.Errorf("Expected nonce %d, got %d at position %d", expected, nonce, i)
		}
	}

	// Check internal state
	if len(tracker.pendingNonces) != count {
		t.Errorf("Expected %d pending nonces, got %d", count, len(tracker.pendingNonces))
	}

	for i := 0; i < count; i++ {
		nonce := int64(101 + i)
		if source, exists := tracker.pendingNonces[nonce]; !exists {
			t.Errorf("Nonce %d not found in pendingNonces", nonce)
		} else if source != blockHash {
			t.Errorf("Expected nonce %d to be associated with block %s, got %s", nonce, blockHash, source)
		}
	}

	// Check block info
	if len(tracker.pendingBlocks) != 1 {
		t.Errorf("Expected 1 pending block, got %d", len(tracker.pendingBlocks))
	}

	if blockInfo, exists := tracker.pendingBlocks[height]; !exists {
		t.Errorf("Block at height %d not found in pendingBlocks", height)
	} else {
		if blockInfo.BlockHash != blockHash {
			t.Errorf("Expected block hash %s, got %s", blockHash, blockInfo.BlockHash)
		}
		if blockInfo.StartNonce != 101 {
			t.Errorf("Expected start nonce 101, got %d", blockInfo.StartNonce)
		}
		if blockInfo.EndNonce != 105 {
			t.Errorf("Expected end nonce 105, got %d", blockInfo.EndNonce)
		}
	}
}

func TestGetNextNonce(t *testing.T) {
	mockChain := NewMockChainAPI()
	clientID := "test_client"

	tracker := NewNonceTracker(clientID, mockChain)

	// Set initial confirmed nonce
	tracker.confirmedNonce = 100

	ctx := context.Background()

	// Get first nonce
	nonce1, err := tracker.GetNextNonce(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if nonce1 != 101 {
		t.Errorf("Expected nonce 101, got %d", nonce1)
	}

	// Get second nonce
	nonce2, err := tracker.GetNextNonce(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if nonce2 != 102 {
		t.Errorf("Expected nonce 102, got %d", nonce2)
	}

	// Check internal state
	if len(tracker.pendingNonces) != 2 {
		t.Errorf("Expected 2 pending nonces, got %d", len(tracker.pendingNonces))
	}

	if source, exists := tracker.pendingNonces[101]; !exists {
		t.Errorf("Nonce 101 not found in pendingNonces")
	} else if source != "" {
		t.Errorf("Expected nonce 101 to be a user transaction, got block %s", source)
	}

	if source, exists := tracker.pendingNonces[102]; !exists {
		t.Errorf("Nonce 102 not found in pendingNonces")
	} else if source != "" {
		t.Errorf("Expected nonce 102 to be a user transaction, got block %s", source)
	}
}

func TestBlockWasFinalized(t *testing.T) {
	mockChain := NewMockChainAPI()
	clientID := "test_client"

	tracker := NewNonceTracker(clientID, mockChain)

	// Set initial confirmed nonce
	tracker.confirmedNonce = 100

	// Reserve nonces for a block
	blockHash := "block123"
	height := int64(105)
	tracker.ReserveNoncesForBlock(blockHash, height, 5)

	// Add a user transaction
	ctx := context.Background()
	userNonce, _ := tracker.GetNextNonce(ctx)

	// Update the blockchain's view of the nonce
	mockChain.UpdateClientNonce(clientID, 105)

	// Mark the block as finalized
	tracker.BlockWasFinalized(height)

	// Check that the confirmed nonce was updated
	if tracker.confirmedNonce != 105 {
		t.Errorf("Expected confirmed nonce 105, got %d", tracker.confirmedNonce)
	}

	// Check that block tracking was cleared
	if len(tracker.pendingBlocks) != 0 {
		t.Errorf("Expected 0 pending blocks after finalization, got %d", len(tracker.pendingBlocks))
	}

	// Check that only the user transaction nonce remains
	if len(tracker.pendingNonces) != 1 {
		t.Errorf("Expected 1 pending nonce after finalization, got %d", len(tracker.pendingNonces))
	}

	if _, exists := tracker.pendingNonces[userNonce]; !exists {
		t.Errorf("User transaction nonce %d should still be pending", userNonce)
	}
}

func TestBlockWasRejected(t *testing.T) {
	mockChain := NewMockChainAPI()
	clientID := "test_client"

	tracker := NewNonceTracker(clientID, mockChain)

	// Set initial confirmed nonce
	tracker.confirmedNonce = 100

	// Reserve nonces for a block
	blockHash := "block123"
	height := int64(105)
	tracker.ReserveNoncesForBlock(blockHash, height, 5)

	// Add a user transaction
	ctx := context.Background()
	userNonce, _ := tracker.GetNextNonce(ctx)

	// Reject the block
	tracker.BlockWasRejected(height)

	// Check that confirmed nonce was not changed
	if tracker.confirmedNonce != 100 {
		t.Errorf("Expected confirmed nonce to remain 100, got %d", tracker.confirmedNonce)
	}

	// Check that block tracking was cleared
	if len(tracker.pendingBlocks) != 0 {
		t.Errorf("Expected 0 pending blocks after rejection, got %d", len(tracker.pendingBlocks))
	}

	// Check that only the user transaction nonce remains
	if len(tracker.pendingNonces) != 1 {
		t.Errorf("Expected 1 pending nonce after rejection, got %d", len(tracker.pendingNonces))
	}

	if _, exists := tracker.pendingNonces[userNonce]; !exists {
		t.Errorf("User transaction nonce %d should still be pending", userNonce)
	}
}

func TestRefreshNonceFromBlockchain(t *testing.T) {
	mockChain := NewMockChainAPI()
	clientID := "test_client"

	tracker := NewNonceTracker(clientID, mockChain)

	// Set initial confirmed nonce
	tracker.confirmedNonce = 100

	// Add some pending nonces
	tracker.pendingNonces[101] = "" // User transaction
	tracker.pendingNonces[102] = "block123"
	tracker.pendingNonces[103] = "block123"

	// Update the blockchain state
	mockChain.UpdateClientNonce(clientID, 102)

	// Refresh from blockchain
	ctx := context.Background()
	err := tracker.RefreshNonceFromBlockchain(ctx)
	if err != nil {
		t.Errorf("Unexpected error refreshing nonce: %v", err)
	}

	// Check that confirmed nonce was updated
	if tracker.confirmedNonce != 102 {
		t.Errorf("Expected confirmed nonce 102, got %d", tracker.confirmedNonce)
	}

	// Check that nonces <= 102 were removed
	if len(tracker.pendingNonces) != 1 {
		t.Errorf("Expected 1 pending nonce after refresh, got %d", len(tracker.pendingNonces))
	}

	if _, exists := tracker.pendingNonces[103]; !exists {
		t.Errorf("Nonce 103 should still be pending")
	}
}

func TestConcurrentAccess(t *testing.T) {
	mockChain := NewMockChainAPI()
	clientID := "test_client"

	tracker := NewNonceTracker(clientID, mockChain)

	// Set initial confirmed nonce
	tracker.confirmedNonce = 100

	// Start multiple goroutines to simulate concurrent access
	const numGoroutines = 10
	var wg sync.WaitGroup

	ctx := context.Background()

	// Goroutines to reserve nonces for transactions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nonce, err := tracker.GetNextNonce(ctx)
			if err != nil {
				t.Errorf("Unexpected error getting nonce: %v", err)
			}
			if nonce <= 100 {
				t.Errorf("Expected nonce > 100, got %d", nonce)
			}
		}(i)
	}

	// Goroutines to reserve nonces for blocks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			blockHash := "block" + fmt.Sprintf("A%d", id)
			height := int64(105 + id)
			nonces := tracker.ReserveNoncesForBlock(blockHash, height, 3)
			if len(nonces) != 3 {
				t.Errorf("Expected 3 nonces, got %d", len(nonces))
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check that we allocated the expected number of nonces
	// numGoroutines transactions + numGoroutines blocks * 3 nonces each
	expectedNonces := numGoroutines + (numGoroutines * 3)
	if len(tracker.pendingNonces) != expectedNonces {
		t.Errorf("Expected %d pending nonces, got %d", expectedNonces, len(tracker.pendingNonces))
	}

	// Check that we created the expected number of blocks
	if len(tracker.pendingBlocks) != numGoroutines {
		t.Errorf("Expected %d pending blocks, got %d", numGoroutines, len(tracker.pendingBlocks))
	}
}

func TestFailedRefresh(t *testing.T) {
	// Create a chain API that fails by returning nil for the block
	failingAPI := NewFailingMockChainAPI()

	clientID := "test_client"
	tracker := NewNonceTracker(clientID, failingAPI)

	// Try to refresh
	ctx := context.Background()
	err := tracker.RefreshNonceFromBlockchain(ctx)

	// Should fail with "no finalized block available"
	if err == nil {
		t.Error("Expected error refreshing with nil block, got nil")
	}

	// Fix the chain API by using a working one
	workingAPI := NewMockChainAPI()
	tracker.chainAPI = workingAPI

	err = tracker.RefreshNonceFromBlockchain(ctx)

	// Should succeed now
	if err != nil {
		t.Errorf("Unexpected error after fixing API: %v", err)
	}
}
