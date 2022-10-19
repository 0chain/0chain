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
	BlockStateChangeRequestor struct {
		notInfo *NotarisationInfo

		roundInfo *RoundInfo

		clientStats *stats.NodesClientStats

		caseType BlockStateChangeRequestorCaseType

		wg *sync.WaitGroup

		resultLocker sync.RWMutex
	}

	// BlockStateChangeRequestorCaseType represents type that determines test behavior.
	BlockStateChangeRequestorCaseType int
)

const (
	// BSCRNoReplies determines BlockStateChangeRequestorCaseType in which all nodes ignore Replica0.
	//
	//	Flow of this test case:
	//		Replica0: Ignore all VerifyBlock messages on round_n
	//		Requested nodes: ignore all block state change requests from Replica0
	//		Check: Replica0 must retry requesting.
	BSCRNoReplies BlockStateChangeRequestorCaseType = iota

	// BSCROnlyOneRepliesCorrectly determines BlockStateChangeRequestorCaseType in which all nodes ignore Replica0,
	// but only one node replies correctly.
	//
	//	Flow of this test case:
	//		Replica0: Ignore all VerifyBlock messages on round_n
	//		Requested nodes: ignore all block state change requests from Replica0
	//		Check: Replica0 must finalize round_n.
	BSCROnlyOneRepliesCorrectly

	// BSCRChangeNode determines BlockStateChangeRequestorCaseType in which all nodes ignore Replica0,
	// but only one node replies correctly.
	//
	//	Flow of this test case:
	//		Replica0: Ignore all VerifyBlock messages on round_n
	//		Requested nodes: ignore all block state change requests from Replica0, but only one
	//		node sends incorrect state change with changed MPT node.
	//		Check: Replica0 must retry requesting.
	BSCRChangeNode

	// BSCRDeleteNode determines BlockStateChangeRequestorCaseType in which all nodes ignore Replica0,
	// but only one node replies correctly.
	//
	//	Flow of this test case:
	//		Replica0: Ignore all VerifyBlock messages on round_n
	//		Requested nodes: ignore all block state change requests from Replica0, but only one
	//		node sends incorrect state change with deleted MPT node.
	//		Check: Replica0 must retry requesting.
	BSCRDeleteNode

	// BSCRAddNode determines BlockStateChangeRequestorCaseType in which all nodes ignore Replica0,
	// but only one node replies correctly.
	//
	//	Flow of this test case:
	//		Replica0: Ignore all VerifyBlock messages on round_n
	//		Requested nodes: ignore all block state change requests from Replica0, but only one
	//		node sends incorrect state change with added MPT node.
	//		Check: Replica0 must retry requesting.
	BSCRAddNode

	// BSCRAnotherPartialState determines BlockStateChangeRequestorCaseType in which all nodes ignore Replica0,
	// but only one node replies correctly.
	//
	//	Flow of this test case:
	//		Replica0: Ignore all VerifyBlock messages on round_n
	//		Requested nodes: ignore all block state change requests from Replica0, but only one
	//		node sends incorrect state change from another block.
	//		Check: Replica0 must retry requesting.
	BSCRAnotherPartialState
)

var (
	// Ensure BlockStateChangeRequestor implements TestCase interface.
	_ TestCase = (*BlockStateChangeRequestor)(nil)
)

// NewBlockStateChangeRequestor creates initialised BlockStateChangeRequestor.
func NewBlockStateChangeRequestor(clientStatsCollector *stats.NodesClientStats, caseType BlockStateChangeRequestorCaseType) *BlockStateChangeRequestor {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	return &BlockStateChangeRequestor{
		clientStats: clientStatsCollector,
		caseType:    caseType,
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
	switch n.caseType {
	case BSCROnlyOneRepliesCorrectly:
		return n.checkOnlyOneRepliesCorrectly()

	case BSCRNoReplies:
		fallthrough

	case BSCRChangeNode:
		fallthrough

	case BSCRAnotherPartialState:
		return n.checkRetryRequesting(2)

	case BSCRAddNode:
		fallthrough

	case BSCRDeleteNode:
		return n.checkRetryRequesting(1)

	default:
		panic("unknown case type")
	}
}

func (n *BlockStateChangeRequestor) checkRetryRequesting(minRequests int) (success bool, err error) {
	replica0 := n.roundInfo.getNodeID(false, 0)
	replica0Stats, ok := n.clientStats.BlockStateChange[replica0]
	if !ok {
		return false, errors.New("no reports from replica0")
	}

	blockHash := n.notInfo.BlockID
	numReports := replica0Stats.CountWithHash(blockHash)
	if numReports < minRequests {
		return false, fmt.Errorf("insufficient reports count: %d", numReports)
	}
	return true, nil
}

func (n *BlockStateChangeRequestor) checkOnlyOneRepliesCorrectly() (success bool, err error) {
	if _, err := n.checkRetryRequesting(1); err != nil {
		return false, err
	}

	success = n.roundInfo != nil && n.roundInfo.IsFinalised
	if !success {
		err = errors.New("round is not finalised")
	}
	return success, err
}

// Configure implements TestCase interface.
func (n *BlockStateChangeRequestor) Configure(blob []byte) error {
	defer n.wg.Done()
	n.notInfo = new(NotarisationInfo)
	return n.notInfo.Decode(blob)
}

// AddResult implements TestCase interface.
func (n *BlockStateChangeRequestor) AddResult(blob []byte) error {
	n.resultLocker.Lock()
	defer n.resultLocker.Unlock()

	if n.roundInfo != nil {
		return nil
	}

	defer n.wg.Done()
	n.roundInfo = new(RoundInfo)
	return n.roundInfo.Decode(blob)
}
