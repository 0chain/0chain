# NonceTracker

The NonceTracker provides robust nonce management for blockchain clients and miners in the 0chain network. It solves the challenging problem of managing nonces when a client is both generating transactions and creating blocks, especially in scenarios where blocks may not be finalized.

## Problem Solved

In blockchain systems, each transaction from a client must have a unique, sequential nonce. When a client is also a miner (generating blocks), several challenges arise:

1. **Concurrent Transactions**: Transactions created during block generation may have nonce conflicts
2. **Block Finalization**: Nonces from blocks that aren't finalized need to be reset
3. **Non-Consecutive Block Generation**: Miners may generate blocks at non-consecutive heights
4. **Race Conditions**: Multiple goroutines may try to obtain nonces simultaneously

This package addresses these challenges by providing a thread-safe nonce management system that tracks both user transactions and block-generated transactions.

## Features

- Thread-safe nonce allocation for both user transactions and block generation
- Proper handling of block finalization and rejection
- Automatic cleanup of confirmed nonces
- Periodic synchronization with blockchain state
- Efficient handling of concurrent nonce requests
- Support for non-consecutive block generation

## Usage

### Setup

```go
// Initialize the nonceTracker
chainAPI := YourChainAPIAdapter{...} // Implement the ChainAPI interface
nonceTracker := noncetracker.NewNonceTracker(clientID, chainAPI)
```

### Getting a Nonce for User Transaction

```go
// Get a nonce for a user transaction
ctx := context.Background()
nonce, err := nonceTracker.GetNextNonce(ctx)
if err != nil {
    // Handle error
}
```

### Reserving Nonces for Block Generation

```go
// Reserve nonces for a block with N transactions
blockHash := "your-block-hash"
blockHeight := int64(1050)
txnCount := 5
nonces := nonceTracker.ReserveNoncesForBlock(blockHash, blockHeight, txnCount)

// Use these nonces for your block's transactions
for i, txn := range blockTxns {
    txn.Nonce = nonces[i]
}
```

### Handling Block Finalization/Rejection

```go
// When a block is finalized
nonceTracker.BlockWasFinalized(blockHeight)

// When a block is rejected
nonceTracker.BlockWasRejected(blockHeight)
```

### Periodic Synchronization

```go
// Start a background process to periodically sync with blockchain
func startPeriodicRefresh(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if err := nonceTracker.RefreshNonceFromBlockchain(ctx); err != nil {
                // Handle error
            }
        case <-ctx.Done():
            return
        }
    }
}

go startPeriodicRefresh(context.Background())
```

## Integration Guide

See `example.go` for detailed integration examples with the 0chain codebase.

## Testing

The NonceTracker package includes comprehensive unit tests covering:

- Basic nonce allocation
- Block handling
- Concurrent access
- Error handling
- Blockchain state synchronization

Run the tests with:

```
go test -v ./code/go/0chain.net/core/util/noncetracker
```

Before integrating into production, thorough testing is recommended with various scenarios including concurrent transaction submission, block generation, finalization, and rejection.

## Contributing

Contributions are welcome! Please ensure all tests pass and add tests for any new functionality. 