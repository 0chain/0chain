package cases

import (
	"context"
	"errors"
	"sync"
)

type (
	// ResendNotarisation represents implementation of the TestCase interface.
	//
	//	Flow of this test case:
	//		Miner sends previous notarization used in notarized/finalized round in new round
	//		(T0) Replica_j: obtain Notarization0_0
	//		(T1) Leader_1: send Proposal1_0
	//		(T1 + δ + Δ) Replica_j: send VerificationTicket1_j
	//		(T1 + 2δ + Δ) Replica_j: send Notarization1_1
	//		(T1 + 3δ + Δ Replica_j: send Notarisation0_0
	//		(T1 + 4δ + Δ) Replica_j+1: reject Notarization0_0
	ResendNotarisation struct {
		result *RoundInfo

		wg *sync.WaitGroup
	}
)

var (
	// Ensure ResendNotarisation implements TestCase interface.
	_ TestCase = (*ResendNotarisation)(nil)
)

// NewResendNotarisation creates initialised ResendNotarisation.
func NewResendNotarisation() *ResendNotarisation {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	return &ResendNotarisation{
		wg: wg,
	}
}

// Check implements TestCase interface.
func (n *ResendNotarisation) Check(ctx context.Context) (success bool, err error) {
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

func (n *ResendNotarisation) check() (success bool, err error) {
	if !n.result.IsFinalised {
		err = errors.New("round is not finalised")
	}
	return n.result.IsFinalised, err
}

// Configure implements TestCase interface.
func (n *ResendNotarisation) Configure(_ []byte) error {
	panic("configuration for this test case is not allowed")
}

// AddResult implements TestCase interface.
func (n *ResendNotarisation) AddResult(blob []byte) error {
	defer n.wg.Done()
	n.result = new(RoundInfo)
	return n.result.Decode(blob)
}
