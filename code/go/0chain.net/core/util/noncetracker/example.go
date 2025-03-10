package noncetracker

// This file provides examples of how to integrate the NonceTracker with the 0chain codebase.
// It is not meant to be used directly but serves as documentation.

/*
INTEGRATION GUIDE

To integrate the NonceTracker with your 0chain blockchain implementation, follow these steps:

1. Create a Chain API adapter that implements the ChainAPI interface:

```go
type ChainAPIAdapter struct {
    chain *chain.Chain
}

func (ca *ChainAPIAdapter) GetLatestFinalizedBlock() interface{} {
    return ca.chain.GetLatestFinalizedBlock()
}

func (ca *ChainAPIAdapter) GetCurrentNonce(clientID string, blockState interface{}) (int64, error) {
    lfb := blockState.(*block.Block)
    state, err := chain.GetStateById(lfb.ClientState, clientID)
    if err != nil {
        if err != util.ErrValueNotPresent {
            return 0, err
        }
        return 1, nil
    }
    return state.Nonce, nil
}
```

2. Initialize the NonceTracker in your Chain or use it as a singleton:

```go
var nonceTracker *noncetracker.NonceTracker

func initNonceTracker(chain *chain.Chain) {
    clientID := node.Self.Underlying().GetKey()
    chainAPI := &ChainAPIAdapter{chain: chain}
    nonceTracker = noncetracker.NewNonceTracker(clientID, chainAPI)
}
```

3. Replace direct nonce calls in the SendSmartContractTxn or similar functions:

Original code:
```go
nextNonce := node.Self.GetNextNonce()
if nextNonce == 0 {
    // try get nonce from LFB
    lfb := c.GetLatestFinalizedBlock()
    if lfb != nil {
        var err error
        nextNonce, err = c.GetCurrentSelfNonce(node.Self.Underlying().GetKey(), lfb.ClientState)
        if err != nil && state.ErrInvalidState(err) {
            return err
        }
    }
}
txn.Nonce = nextNonce
```

Replaced with:
```go
nextNonce, err := nonceTracker.GetNextNonce(ctx)
if err != nil {
    return err
}
txn.Nonce = nextNonce
```

4. Add event handlers for block creation, finalization, and rejection:

```go
// When creating a block with transactions
func onBlockCreation(block *block.Block) {
    blockHash := block.Hash
    height := block.Round
    txnCount := len(block.Transactions)
    nonces := nonceTracker.ReserveNoncesForBlock(blockHash, height, txnCount)

    // Use these nonces for your block transactions
    for i, txn := range block.Transactions {
        txn.Nonce = nonces[i]
    }
}

// When a block is finalized
func onBlockFinalization(height int64) {
    nonceTracker.BlockWasFinalized(height)
}

// When a block is rejected/invalidated
func onBlockRejection(height int64) {
    nonceTracker.BlockWasRejected(height)
}
```

5. Set up a periodic refresh to maintain synchronization:

```go
func startPeriodicRefresh(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if err := nonceTracker.RefreshNonceFromBlockchain(ctx); err != nil {
                logging.Logger.Error("Failed to refresh nonce", zap.Error(err))
            }
        case <-ctx.Done():
            return
        }
    }
}

// Start the refresh goroutine
go startPeriodicRefresh(context.Background(), 30*time.Second)
```

USAGE SCENARIOS:

1. Sending a transaction:
```go
ctx := context.Background()
txn := &httpclientutil.Transaction{...}

// Get nonce
nonce, err := nonceTracker.GetNextNonce(ctx)
if err != nil {
    return err
}
txn.Nonce = nonce

// Send the transaction
err = httpclientutil.SendSmartContractTxn(txn, minerUrls, sharderUrls)
```

2. Block generation:
```go
// When generating a block with N transactions
txnCount := len(blockTxns)
nonces := nonceTracker.ReserveNoncesForBlock(blockHash, blockHeight, txnCount)

// Use these nonces for your block's transactions
for i, txn := range blockTxns {
    txn.Nonce = nonces[i]
}
```

3. Block finalization:
```go
// When a block is finalized
nonceTracker.BlockWasFinalized(blockHeight)
```

4. Block rejection:
```go
// When a block is rejected/invalidated
nonceTracker.BlockWasRejected(blockHeight)
```

TESTING CONSIDERATIONS:

Before using in production, test thoroughly with:
1. Concurrent transaction submission
2. Block generation during transaction submission
3. Block finalization scenarios
4. Block rejection/invalidation scenarios
5. Chain reorganization cases
*/
