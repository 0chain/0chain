package cases

import (
	"context"
	"fmt"
	"sync"

	"0chain.net/conductor/conductrpc/stats"
)

type (
	// VerifyingNonExistentBlock represents implementation of the TestCase interface.
	//
	//	Flow of this test case:
	//		Verify not existent block, this should work fine, since we only formally verify verification_tickets
	//		(T0) Replica_0: send Verification0_X
	//		(T0 + δ) Replica_1: check and forget
	VerifyingNonExistentBlock struct {
		nonExistentBlockHash string
		round                int

		serverStatsMu sync.Mutex
		serverStats   *stats.NodesServerStats

		wg *sync.WaitGroup
	}
)

var (
	// Ensure VerifyingNonExistentBlock implements TestCase interface.
	_ TestCase = (*VerifyingNonExistentBlock)(nil)
)

// NewVerifyingNonExistentBlock creates initialised VerifyingNonExistentBlock.
func NewVerifyingNonExistentBlock(nonExistentBlockHash string, round int, serverStats *stats.NodesServerStats) *VerifyingNonExistentBlock {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	return &VerifyingNonExistentBlock{
		nonExistentBlockHash: nonExistentBlockHash,
		round:                round,
		serverStats:          serverStats,
		wg:                   wg,
	}
}

// Check implements TestCase interface.
func (n *VerifyingNonExistentBlock) Check(ctx context.Context) (success bool, err error) {
	prepared := make(chan struct{})
	go func() {
		n.wg.Wait()
		prepared <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return false, ctx.Err()

	case <-prepared:
		return n.check()
	}
}

func (n *VerifyingNonExistentBlock) check() (success bool, err error) {
	n.serverStatsMu.Lock()
	defer n.serverStatsMu.Unlock()

	for _, requests := range n.serverStats.Block {
		if br := requests.GetByHashOrRound(n.nonExistentBlockHash, n.round); br != nil {
			return false, fmt.Errorf("non existent block was fetched from the network; block info: %#v", br)
		}
	}

	return true, nil
}

// Configure implements TestCase interface.
func (n *VerifyingNonExistentBlock) Configure(_ []byte) error {
	n.wg.Done()
	// configuring should be called after sending bad verification ticket, so just checking that it was happened.
	return nil
}

// AddResult implements TestCase interface.
func (n *VerifyingNonExistentBlock) AddResult(_ []byte) error {
	panic("adding result for test case is not allowed")
}
