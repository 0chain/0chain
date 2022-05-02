package cases

import (
	"context"
	"errors"
	"sync"
)

type (
	// ResendProposedBlock represents implementation of the TestCase interface.
	//
	// 	Flow of this test case:
	//		Miner sends previous proposed block used in notarized round with one/several blocks notarized in it to propose in new round
	//		(T0) Leader_0: send Proposal0_0 (after getting first Notarisation)
	//		(T0) Leader_1: send Proposal0_1
	//		(T0 + δ + Δ) Replica_j: send VerificationTicket1_j
	//		(T0 + 2δ + Δ) Replica_j: send Notarization0_1
	//		(T0 + 3δ + Δ) Leader_1: resend Proposal0_0 (after getting Proposal0_0)
	//		(T0 + 4δ + Δ) Replica_j: reject Proposal0_0
	ResendProposedBlock struct {
		resentBlockHash string

		result *RoundInfo

		wg *sync.WaitGroup
	}
)

var (
	// Ensure ResendProposedBlock implements TestCase interface.
	_ TestCase = (*ResendProposedBlock)(nil)
)

// NewResendProposedBlock creates initialised ResendProposedBlock.
func NewResendProposedBlock() *ResendProposedBlock {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	return &ResendProposedBlock{
		wg: wg,
	}
}

// Check implements TestCase interface.
func (n *ResendProposedBlock) Check(ctx context.Context) (success bool, err error) {
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

func (n *ResendProposedBlock) check() (success bool, err error) {
	if n.resentBlockHash == "" {
		return false, errors.New("resent block hash is empty")
	}

	blocks := make(map[string]*BlockInfo)
	for _, b := range n.result.ProposedBlocks {
		_, ok := blocks[b.Hash]
		if ok && b.Hash == n.resentBlockHash {
			return false, errors.New("resent block is duplicated in stored round blocks")
		}
		blocks[b.Hash] = b
	}

	resentBlock, ok := blocks[n.resentBlockHash]
	if ok && len(resentBlock.VerificationTickets) != 0 {
		return false, errors.New("resent block has verification tickets")
	}

	return true, nil
}

// Configure implements TestCase interface.
func (n *ResendProposedBlock) Configure(blob []byte) error {
	defer n.wg.Done()
	n.resentBlockHash = string(blob)
	return nil
}

// AddResult implements TestCase interface.
func (n *ResendProposedBlock) AddResult(blob []byte) error {
	defer n.wg.Done()
	n.result = new(RoundInfo)
	return n.result.Decode(blob)
}
