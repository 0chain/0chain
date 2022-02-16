package cases

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"0chain.net/conductor/conductrpc/stats"
)

type (
	// MinerNotarisedBlockRequestor represents implementation of the TestCase interface.
	MinerNotarisedBlockRequestor struct {
		roundInfo *RoundInfo

		clientStats *stats.NodesClientStats

		caseType MinerNotarisedBlockRequestorType

		wg *sync.WaitGroup
	}

	// MinerNotarisedBlockRequestorType represents type that determines test behavior.
	MinerNotarisedBlockRequestorType int
)

const (
	// MSBRNoReplies determines MinerNotarisedBlockRequestorType in which all nodes ignore Replica0.
	//
	//	Flow of this test case:
	//		Replica0: Ignore proposal
	//		Replica0: Gets Notarisation and starts requests.
	//		Requested nodes: ignore requests.
	//		Check: Replica0 must retry requesting.
	MSBRNoReplies MinerNotarisedBlockRequestorType = iota

	// MSBROnlyOneRepliesCorrectly determines MinerNotarisedBlockRequestorType in which all nodes ignore Replica0,
	// but only one node replies correctly.
	//
	//	Flow of this test case:
	//		Replica0: Ignore proposal
	//		Replica0: Gets Notarisation and starts requests.
	//		Requested nodes: ignore requests, but only one node replies correctly.
	//		Check: round must be finalized.
	MSBROnlyOneRepliesCorrectly

	// MSBRValidBlockWithChangedHash determines MinerNotarisedBlockRequestorType in which all nodes ignore Replica0,
	// but only one node replies with valid block (changed hash).
	//
	//	Flow of this test case:
	//		Replica0: Ignore proposal
	//		Replica0: Gets Notarisation and starts requests.
	//		Requested nodes: ignore requests, but only one node sends valid block (with changed hash).
	//		Check: Replica0 must retry requesting.
	MSBRValidBlockWithChangedHash

	// MSBRInvalidBlockWithChangedHash determines MinerNotarisedBlockRequestorType in which all nodes ignore Replica0,
	// but only one node replies with invalid block (changed hash).
	//
	//	Flow of this test case:
	//		Replica0: Ignore proposal
	//		Replica0: Gets Notarisation and starts requests.
	//		Requested nodes: ignore requests, but only one node sends invalid block (with changed hash).
	//		Check: Replica0 must retry requesting.
	MSBRInvalidBlockWithChangedHash

	// MSBRBlockWithoutVerTickets determines MinerNotarisedBlockRequestorType in which all nodes ignore Replica0,
	// but only one node replies with block without verification tickets.
	//
	//	Flow of this test case:
	//		Replica0: Ignore proposal
	//		Replica0: Gets Notarisation and starts requests.
	//		Requested nodes: ignore requests, but only one node sends block without verification tickets.
	//		Check: round must be finalized.
	MSBRBlockWithoutVerTickets
)

var (
	// Ensure MinerNotarisedBlockRequestor implements TestCase interface.
	_ TestCase = (*MinerNotarisedBlockRequestor)(nil)
)

// NewMinerNotarisedBlockRequestor creates initialised MinerNotarisedBlockRequestor.
func NewMinerNotarisedBlockRequestor(clientStatsCollector *stats.NodesClientStats, caseType MinerNotarisedBlockRequestorType) *MinerNotarisedBlockRequestor {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	return &MinerNotarisedBlockRequestor{
		clientStats: clientStatsCollector,
		caseType:    caseType,
		wg:          wg,
	}
}

// Check implements TestCase interface.
func (n *MinerNotarisedBlockRequestor) Check(ctx context.Context) (success bool, err error) {
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

func (n *MinerNotarisedBlockRequestor) check() (success bool, err error) {
	switch n.caseType {
	case MSBROnlyOneRepliesCorrectly:
		return n.checkRetryRequesting(1, true)

	case MSBRBlockWithoutVerTickets:
		fallthrough

	case MSBRInvalidBlockWithChangedHash:
		fallthrough

	case MSBRValidBlockWithChangedHash:
		fallthrough

	case MSBRNoReplies:
		return n.checkRetryRequesting(2, false)

	default:
		panic("unknown case type")
	}
}

func (n *MinerNotarisedBlockRequestor) checkRetryRequesting(minRequests int, checkFinalisation bool) (success bool, err error) {
	replica0 := n.roundInfo.getNodeID(false, 0)
	replica0Stats, ok := n.clientStats.MinerNotarisedBlock[replica0]
	if !ok {
		return false, errors.New("no reports from replica0")
	}

	if len(n.roundInfo.NotarisedBlocks) != 1 {
		return false, errors.New("expected 1 notarised block")
	}

	notBlock := n.roundInfo.NotarisedBlocks[0]
	numReports := replica0Stats.CountWithHash(notBlock.Hash)
	if numReports < minRequests {
		return false, fmt.Errorf("wrong reports count: %d; min %d", numReports, minRequests)
	}

	if checkFinalisation && !n.roundInfo.IsFinalised {
		return false, errors.New("round is not finalised")
	}

	return true, nil
}

// Configure implements TestCase interface.
func (n *MinerNotarisedBlockRequestor) Configure(_ []byte) error {
	panic("configuring is not allowed for this test case")
}

// AddResult implements TestCase interface.
func (n *MinerNotarisedBlockRequestor) AddResult(blob []byte) error {
	defer n.wg.Done()
	n.roundInfo = new(RoundInfo)
	return n.roundInfo.Decode(blob)
}
