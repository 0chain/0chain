package cases

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type (
	// SendInsufficientProposals represents implementation of the TestCase interface.
	//
	//	Flow of this test case:
	//		check make progress for an adversarial leader
	//		(T0) Leader_0:  send Proposal0_0 for replica j , 0 <= j < 1/3f
	//		(T0) Leader_1:  send Proposal0_1
	//		(T0 + δ + Δ) Replica_i: send Verification0_0
	SendInsufficientProposals struct {
		firstGenBlockHash string // Generator0 blocks hash

		res *RoundInfo

		wg *sync.WaitGroup
	}

	SendInsufficientProposalsResult []*BlockInfo
)

var (
	// Ensure SendInsufficientProposals implements TestCase interface.
	_ TestCase = (*SendInsufficientProposals)(nil)
)

// NewSendInsufficientProposals creates initialised SendInsufficientProposals.
func NewSendInsufficientProposals() *SendInsufficientProposals {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	return &SendInsufficientProposals{
		wg: wg,
	}
}

// Check implements TestCase interface.
func (n *SendInsufficientProposals) Check(ctx context.Context) (success bool, err error) {
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

func (n *SendInsufficientProposals) check() (success bool, err error) {
	if len(n.res.NotarisedBlocks) != 1 {
		return false, fmt.Errorf("unexpected number of notarised blocks: %d, expected 1", len(n.res.NotarisedBlocks))
	}

	if notBlockRank := n.res.NotarisedBlocks[0].Rank; notBlockRank != 1 {
		return false, fmt.Errorf("unexpected notarised block rank: %d, expected 1", notBlockRank)
	}

	return true, nil
}

// Configure implements TestCase interface.
func (n *SendInsufficientProposals) Configure(blob []byte) error {
	defer n.wg.Done()
	n.firstGenBlockHash = string(blob)
	return nil
}

// AddResult implements TestCase interface.
func (n *SendInsufficientProposals) AddResult(blob []byte) error {
	defer n.wg.Done()
	n.res = new(RoundInfo)
	return n.res.Decode(blob)
}

// Encode encodes SendInsufficientProposalsResult to bytes.
func (r SendInsufficientProposalsResult) Encode() ([]byte, error) {
	return json.Marshal(r)
}

// Decode decodes SendInsufficientProposalsResult from bytes.
func (r SendInsufficientProposalsResult) Decode(blob []byte) error {
	return json.Unmarshal(blob, &r)
}
