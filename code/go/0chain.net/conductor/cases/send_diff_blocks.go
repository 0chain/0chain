package cases

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"

	"0chain.net/conductor/config"
)

type (
	// SendDifferentBlocksFromAllGenerators represents implementation of the config.TestCase interface.
	//
	// 	Flow of this test case:
	//		check no depletion, all first j leaders are adversarial, they send different blocks to different replicas but not enough for notarization
	//		(T0) Leader_0(Adv): send Proposalj_i to Replica_i, j= f , 0 < i < notarization_threshold
	//		(T0 + δ + Δ) Replica_i: send VerificationTicket0_i
	SendDifferentBlocksFromAllGenerators struct {
		*sendDiffBlocksBase
	}

	// SendDifferentBlocksFromFirstGenerator represents implementation of the config.TestCase interface.
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

		timeoutCountMu sync.Mutex
		timeoutCount   map[string]int // key - miner ID; value - round's timeout count
	}

	SendDiffBlocksResult struct {
		MinerID      string `json:"miner_id"`
		TimeoutCount int    `json:"timeout_count"`
	}
)

var (
	// Ensure SendDifferentBlocksFromAllGenerators implements config.TestCase interface.
	_ config.TestCase = (*SendDifferentBlocksFromAllGenerators)(nil)

	// Ensure SendDifferentBlocksFromFirstGenerator implements config.TestCase interface.
	_ config.TestCase = (*SendDifferentBlocksFromFirstGenerator)(nil)
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
		timeoutCount: make(map[string]int),
		minersNum:    minersNum,
		wg:           wg,
	}
}

// Configure implements config.TestCase interface.
func (s *sendDiffBlocksBase) Configure(_ []byte) error {
	return errors.New("configuration for this test is not allowed")
}

// AddResult implements config.TestCase interface.
func (s *sendDiffBlocksBase) AddResult(blob []byte) error {
	defer s.wg.Done()
	res := new(SendDiffBlocksResult)
	if err := res.Decode(blob); err != nil {
		return err
	}
	s.timeoutCountMu.Lock()
	s.timeoutCount[res.MinerID] = res.TimeoutCount
	s.timeoutCountMu.Unlock()
	return nil
}

// Check implements config.TestCase interface.
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
		if len(s.timeoutCount) == 0 {
			return false, errors.New("no reports about first round timeout, test is failed")
		}

		return false, ctx.Err()

	case <-prepared:
		return s.check()
	}
}

func (s *sendDiffBlocksBase) check() (success bool, err error) {
	if len(s.timeoutCount) != s.minersNum {
		return false, errors.New("unexpected reports count")
	}

	for _, tCount := range s.timeoutCount {
		if tCount != 1 {
			return false, errors.New("found unexpected timeout count: " + strconv.Itoa(tCount))
		}
	}
	return true, nil
}

// Encode encodes SendDiffBlocksResult to bytes.
func (r *SendDiffBlocksResult) Encode() ([]byte, error) {
	return json.Marshal(r)
}

// Decode decodes SendDiffBlocksResult from bytes.
func (r *SendDiffBlocksResult) Decode(blob []byte) error {
	return json.Unmarshal(blob, r)
}
