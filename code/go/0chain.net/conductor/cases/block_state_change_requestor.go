package cases

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"0chain.net/conductor/conductrpc/stats"
)

type (
	// BlockStateChangeRequestor represents implementation of the TestCase interface.
	//
	//	Flow of this test case:
	//		Attack BlockStateChangeRequestor: no replies
	//		Replica0: Ignore all VerifyBlock messages on round_n
	//		Requested nodes: ignore all block state change requests from Replica0
	//		Check: Replica0 must retry requesting.
	BlockStateChangeRequestor struct {
		notInfo *NotarisationInfo

		roundInfo *RoundInfo

		clientStats *stats.NodesClientStats

		wg *sync.WaitGroup
	}
)

var (
	// Ensure BlockStateChangeRequestor implements TestCase interface.
	_ TestCase = (*BlockStateChangeRequestor)(nil)
)

// NewBlockStateChangeRequestor creates initialised BlockStateChangeRequestor.
func NewBlockStateChangeRequestor(clientStatsCollector *stats.NodesClientStats) *BlockStateChangeRequestor {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	return &BlockStateChangeRequestor{
		clientStats: clientStatsCollector,
		wg:          wg,
	}
}

// Check implements TestCase interface.
func (n *BlockStateChangeRequestor) Check(ctx context.Context) (success bool, err error) {
	prepared := make(chan struct{})
	go func() {
		n.wg.Wait()
		prepared <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return false, errors.New("cases state is not prepared, context is done")

	case <-prepared:
		return n.check()
	}
}

func (n *BlockStateChangeRequestor) check() (success bool, err error) {
	replica0 := n.roundInfo.getNodeID(false, 0)
	replica0Stats, ok := n.clientStats.BlockStateChange[replica0]
	if !ok {
		return false, errors.New("no reports from replica0")
	}

	blockHash := n.notInfo.BlockID
	numReports := replica0Stats.CountWithHash(blockHash)
	if numReports < 2 {
		return false, fmt.Errorf("insufficient reports count: %d", numReports)
	}
	return true, nil
}

// Configure implements TestCase interface.
func (n *BlockStateChangeRequestor) Configure(blob []byte) error {
	defer n.wg.Done()
	n.notInfo = new(NotarisationInfo)
	return n.notInfo.Decode(blob)
}

// AddResult implements TestCase interface.
func (n *BlockStateChangeRequestor) AddResult(blob []byte) error {
	defer n.wg.Done()
	n.roundInfo = new(RoundInfo)
	return n.roundInfo.Decode(blob)
}
