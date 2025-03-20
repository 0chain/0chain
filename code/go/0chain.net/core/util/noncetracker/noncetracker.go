package noncetracker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// BlockInfo represents information about a pending block
type BlockInfo struct {
	BlockHash  string
	StartNonce int64
	EndNonce   int64
	Timestamp  time.Time
}

// NonceTracker tracks and manages nonces for a blockchain client/miner
type NonceTracker struct {
	clientID       string
	confirmedNonce int64

	// Track pending nonces with their sources
	pendingNonces map[int64]string // nonce -> blockHash (empty string means user transaction)

	// Block tracking - height -> block info
	pendingBlocks map[int64]*BlockInfo

	mutex         sync.RWMutex
	lastRefresh   time.Time
	refreshWindow time.Duration

	// Interface for interacting with the blockchain
	chainAPI ChainAPI
}

// ChainAPI defines the interface required to interact with the blockchain
type ChainAPI interface {
	// GetLatestFinalizedBlock returns the latest finalized block
	GetLatestFinalizedBlock() interface{}

	// GetCurrentNonce gets the current nonce for a client from a block state
	GetCurrentNonce(clientID string, blockState interface{}) (int64, error)
}

// NewNonceTracker creates a new nonce tracker for a client
func NewNonceTracker(clientID string, chainAPI ChainAPI) *NonceTracker {
	return &NonceTracker{
		clientID:       clientID,
		confirmedNonce: 0,
		pendingNonces:  make(map[int64]string),
		pendingBlocks:  make(map[int64]*BlockInfo),
		mutex:          sync.RWMutex{},
		lastRefresh:    time.Now(),
		refreshWindow:  30 * time.Second,
		chainAPI:       chainAPI,
	}
}

// ReserveNoncesForBlock reserves nonces for a single block at a specific height
func (nt *NonceTracker) ReserveNoncesForBlock(blockHash string, height int64, count int) []int64 {
	nt.mutex.Lock()
	defer nt.mutex.Unlock()

	// Get highest nonce currently in use
	highestNonce := nt.confirmedNonce
	for nonce := range nt.pendingNonces {
		if nonce > highestNonce {
			highestNonce = nonce
		}
	}

	// Reserve consecutive nonces
	nonces := make([]int64, count)
	startNonce := highestNonce + 1

	for i := 0; i < count; i++ {
		nonce := startNonce + int64(i)
		nonces[i] = nonce
		nt.pendingNonces[nonce] = blockHash // Track which block this nonce belongs to
	}

	// Track this block
	nt.pendingBlocks[height] = &BlockInfo{
		BlockHash:  blockHash,
		StartNonce: startNonce,
		EndNonce:   startNonce + int64(count) - 1,
		Timestamp:  time.Now(),
	}

	logging.Logger.Debug("Reserved nonces for block",
		zap.String("blockHash", blockHash),
		zap.Int64("height", height),
		zap.Int64("startNonce", startNonce),
		zap.Int64("endNonce", startNonce+int64(count)-1))

	return nonces
}

// GetNextNonce provides the next available nonce for a user transaction
func (nt *NonceTracker) GetNextNonce(ctx context.Context) (int64, error) {
	// Check if we should refresh from blockchain
	if time.Since(nt.lastRefresh) > nt.refreshWindow {
		if err := nt.RefreshNonceFromBlockchain(ctx); err != nil {
			return 0, err
		}
	}

	nt.mutex.Lock()
	defer nt.mutex.Unlock()

	// Find highest nonce currently in use
	highestNonce := nt.confirmedNonce
	for nonce := range nt.pendingNonces {
		if nonce > highestNonce {
			highestNonce = nonce
		}
	}

	nextNonce := highestNonce + 1
	nt.pendingNonces[nextNonce] = "" // Empty string means user transaction

	logging.Logger.Debug("Reserved nonce for user transaction",
		zap.Int64("nonce", nextNonce))

	return nextNonce, nil
}

// BlockWasFinalized handles finalizing any block at any height
func (nt *NonceTracker) BlockWasFinalized(height int64) {
	// Force refresh from blockchain to get the latest confirmed nonce
	ctx := context.Background()
	if err := nt.RefreshNonceFromBlockchain(ctx); err != nil {
		logging.Logger.Error("Failed to refresh nonce after block finalization", zap.Error(err))
		return
	}

	nt.mutex.Lock()
	defer nt.mutex.Unlock()

	// Since we've updated the confirmed nonce from blockchain,
	// we just need to clean up our tracking for this height
	delete(nt.pendingBlocks, height)

	// Clean up any nonces that are now confirmed
	for nonce := range nt.pendingNonces {
		if nonce <= nt.confirmedNonce {
			delete(nt.pendingNonces, nonce)
		}
	}

	logging.Logger.Debug("Block finalized, updated nonce state",
		zap.Int64("height", height),
		zap.Int64("confirmedNonce", nt.confirmedNonce))
}

// BlockWasRejected handles rejection of a block
func (nt *NonceTracker) BlockWasRejected(height int64) {
	nt.mutex.Lock()
	defer nt.mutex.Unlock()

	// Get the block information
	blockInfo, exists := nt.pendingBlocks[height]
	if !exists {
		logging.Logger.Debug("Block rejection handler called for unknown block",
			zap.Int64("height", height))
		return
	}

	blockHash := blockInfo.BlockHash

	// Remove block from tracking
	delete(nt.pendingBlocks, height)

	// Release nonces associated with this block
	for nonce, hash := range nt.pendingNonces {
		if hash == blockHash {
			delete(nt.pendingNonces, nonce)
		}
	}

	logging.Logger.Debug("Released nonces for rejected block",
		zap.String("blockHash", blockHash),
		zap.Int64("height", height),
		zap.Int64("startNonce", blockInfo.StartNonce),
		zap.Int64("endNonce", blockInfo.EndNonce))
}

// RefreshNonceFromBlockchain updates the confirmed nonce from the blockchain
func (nt *NonceTracker) RefreshNonceFromBlockchain(ctx context.Context) error {
	// Get latest finalized block
	lfb := nt.chainAPI.GetLatestFinalizedBlock()
	if lfb == nil {
		return errors.New("no finalized block available")
	}

	// Get confirmed nonce for our client ID
	nonce, err := nt.chainAPI.GetCurrentNonce(nt.clientID, lfb)
	if err != nil {
		return err
	}

	nt.mutex.Lock()
	defer nt.mutex.Unlock()

	prevNonce := nt.confirmedNonce
	nt.confirmedNonce = nonce
	nt.lastRefresh = time.Now()

	if prevNonce != nonce {
		logging.Logger.Debug("Updated confirmed nonce from blockchain",
			zap.Int64("oldNonce", prevNonce),
			zap.Int64("newNonce", nonce))
	}

	// Clean up any nonces that are now confirmed
	for pendingNonce := range nt.pendingNonces {
		if pendingNonce <= nonce {
			delete(nt.pendingNonces, pendingNonce)
		}
	}

	return nil
}

// GetConfirmedNonce returns the current confirmed nonce
func (nt *NonceTracker) GetConfirmedNonce() int64 {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()
	return nt.confirmedNonce
}

// SetRefreshWindow sets the duration between automatic nonce refreshes
func (nt *NonceTracker) SetRefreshWindow(d time.Duration) {
	nt.mutex.Lock()
	defer nt.mutex.Unlock()
	nt.refreshWindow = d
}

// GetPendingNonceCount returns the number of pending nonces
func (nt *NonceTracker) GetPendingNonceCount() int {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()
	return len(nt.pendingNonces)
}

// GetPendingBlockCount returns the number of pending blocks
func (nt *NonceTracker) GetPendingBlockCount() int {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()
	return len(nt.pendingBlocks)
}

// HasPendingBlock checks if a block at the given height is being tracked
func (nt *NonceTracker) HasPendingBlock(height int64) bool {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()
	_, exists := nt.pendingBlocks[height]
	return exists
}
