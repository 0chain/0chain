package cases

import (
	"context"
	"errors"
	"sync"

	"0chain.net/conductor/conductrpc/stats"
)

type (
	// NotarisingNonExistentBlock represents implementation of the TestCase interface.
	//
	//	Flow of this test case:
	//		Notarize not existent block and fetch it
	//		(T0) Replica_i: send Verification_i_X  0<i<=2f/3 + 1
	//		(T0 + δ) Replica_1: fetch Block_i
	//		(T0 + 2*δ) Replica_1: send Notarization_i
	NotarisingNonExistentBlock struct {
		serverStats *stats.NodesServerStats

		res *RoundInfo

		wg *sync.WaitGroup
	}
)

var (
	// Ensure NotarisingNonExistentBlock implements TestCase interface.
	_ TestCase = (*NotarisingNonExistentBlock)(nil)
)

// NewNotarisingNonExistentBlock creates initialised NotarisingNonExistentBlock.
func NewNotarisingNonExistentBlock(serverStats *stats.NodesServerStats) *NotarisingNonExistentBlock {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	return &NotarisingNonExistentBlock{
		serverStats: serverStats,
		wg:          wg,
	}
}

// Check implements TestCase interface.
func (n *NotarisingNonExistentBlock) Check(ctx context.Context) (success bool, err error) {
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

func (n *NotarisingNonExistentBlock) check() (success bool, err error) {
	notBlocks := n.res.NotarisedBlocks
	if len(notBlocks) != 1 || notBlocks[0].Rank != 0 {
		return false, errors.New("notarised block is unexpected, expected 1 block from the first ranked leader")
	}

	var generator0blockFetched bool
	for _, requests := range n.serverStats.Block {
		var (
			generator0Block = notBlocks[0]
			replica0        = n.res.getNodeID(false, 0)
		)
		if br := requests.GetBySenderIDAndHash(replica0, generator0Block.Hash); br != nil {
			generator0blockFetched = true
		}
	}

	if !generator0blockFetched {
		return false, errors.New("expected fetch block request from replica0 with leader0 block")
	}

	return true, nil
}

func (n *NotarisingNonExistentBlock) Configure(_ []byte) error {
	panic("configuring is not allowed")
}

func (n *NotarisingNonExistentBlock) AddResult(blob []byte) error {
	defer n.wg.Done()
	n.res = new(RoundInfo)
	return n.res.Decode(blob)
}
