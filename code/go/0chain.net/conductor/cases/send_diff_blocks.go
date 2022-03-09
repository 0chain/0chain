package cases

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type (
	// SendDifferentBlocksFromAllGenerators represents implementation of the TestCase interface.
	//
	// 	Flow of this test case:
	//		check no depletion, all first j leaders are adversarial, they send different blocks to different replicas but not enough for notarization
	//		(T0) Leader_0(Adv): send Proposalj_i to Replica_i, j= f , 0 < i < notarization_threshold
	//		(T0 + δ + Δ) Replica_i: send VerificationTicket0_i
	SendDifferentBlocksFromAllGenerators struct {
		*sendDiffBlocksBase
	}

	// SendDifferentBlocksFromFirstGenerator represents implementation of the TestCase interface.
	//
	// 	Flow of this test case:
	//		Protocol tells to verify all messages on the same rank, alas we don’t follow protocol and vote for only one top ranked block.
	//		Propose different block for every replica, this round should achieve notarization (with different block equality proof, not implemented yet)
	//		(T0) Leader_0(Adv): send Proposal0_i to Replica_i i= notarization_threshold
	//		(T0 + δ + Δ) Replica_i: send VerificationTicket0_i
	SendDifferentBlocksFromFirstGenerator struct {
		*sendDiffBlocksBase
	}

	// sendDiffBlocksBase implements base functional for testing by sending different blocks from generators.
	sendDiffBlocksBase struct {
		minersNum int

		wg *sync.WaitGroup

		resultsMu sync.Mutex
		results   []*RoundInfo
	}
)

var (
	// Ensure SendDifferentBlocksFromAllGenerators implements TestCase interface.
	_ TestCase = (*SendDifferentBlocksFromAllGenerators)(nil)

	// Ensure SendDifferentBlocksFromFirstGenerator implements TestCase interface.
	_ TestCase = (*SendDifferentBlocksFromFirstGenerator)(nil)
)

// NewSendDifferentBlocksFromAllGenerators creates initialised SendDifferentBlocksFromAllGenerators.
func NewSendDifferentBlocksFromAllGenerators(minersNum int) *SendDifferentBlocksFromAllGenerators {
	return &SendDifferentBlocksFromAllGenerators{
		sendDiffBlocksBase: newSendDiffBlocksBase(minersNum),
	}
}

// NewSendDifferentBlocksFromFirstGenerator creates initialised SendDifferentBlocksFromFirstGenerator.
func NewSendDifferentBlocksFromFirstGenerator(minersNum int) *SendDifferentBlocksFromFirstGenerator {
	return &SendDifferentBlocksFromFirstGenerator{
		sendDiffBlocksBase: newSendDiffBlocksBase(minersNum),
	}
}

func newSendDiffBlocksBase(minersNum int) *sendDiffBlocksBase {
	wg := new(sync.WaitGroup)
	wg.Add(minersNum)
	return &sendDiffBlocksBase{
		minersNum: minersNum,
		wg:        wg,
	}
}

// Configure implements TestCase interface.
func (s *sendDiffBlocksBase) Configure(_ []byte) error {
	return errors.New("configuration for this test is not allowed")
}

// AddResult implements TestCase interface.
func (s *sendDiffBlocksBase) AddResult(blob []byte) error {
	defer s.wg.Done()
	res := new(RoundInfo)
	if err := res.Decode(blob); err != nil {
		return err
	}
	s.resultsMu.Lock()
	s.results = append(s.results, res)
	s.resultsMu.Unlock()
	return nil
}

// Check implements TestCase interface.
func (s *sendDiffBlocksBase) Check(ctx context.Context) (success bool, err error) {
	prepared := make(chan struct{})
	go func() {
		s.wg.Wait()
		prepared <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		// miners should send only first round restart reports, so if there no reports
		// it means that round has not restarted.
		if len(s.results) != s.minersNum {
			return false, fmt.Errorf("unexpected number of reports: %d, expected %d", len(s.results), s.minersNum)
		}

		return false, ctx.Err()

	case <-prepared:
		return s.check()
	}
}

func (s *sendDiffBlocksBase) check() (success bool, err error) {
	if len(s.results) != s.minersNum {
		return false, errors.New("unexpected reports count")
	}

	for _, roundInfo := range s.results {
		if roundInfo.TimeoutCount != 1 {
			return false, fmt.Errorf("found unexpected timeout count: %d", roundInfo.TimeoutCount)
		}
	}
	return true, nil
}
