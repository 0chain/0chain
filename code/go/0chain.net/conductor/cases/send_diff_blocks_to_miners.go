package cases

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"0chain.net/conductor/config"
)

type (
	// SendDifferentBlocksToMiners represents implementation of the config.TestCase interface.
	//
	// 	Flow of this test case:
	//		Protocol tells to verify all messages on the same rank, alas we don’t follow protocol and vote for only one top ranked block.
	//		Propose different block for every replica, this round should achieve notarization (with different block equality proof, not implemented yet)
	//		(T0) Leader_0(Adv): send Proposal0_i to Replica_i i= notarization_threshold
	//		(T0 + δ + Δ) Replica_i: send VerificationTicket0_i
	SendDifferentBlocksToMiners struct {
		cfg *SendDifferentBlocksToMinersConfig
		res *SendDifferentBlocksToMinersResult
		wg  *sync.WaitGroup
	}

	// SendDifferentBlocksToMinersConfig represents configuration for SendDifferentBlocksToMiners test.
	// When Leader_0 starts sending different blocks to miners, SendDifferentBlocksToMiners.Configure
	// will be called to save SendDifferentBlocksToMinersConfig to SendDifferentBlocksToMiners.
	SendDifferentBlocksToMinersConfig struct {
		MinersRoundRank int `json:"round_rank"`
	}

	// SendDifferentBlocksToMinersResult represents test result for SendDifferentBlocksToMiners test.
	// When Replica_0 updates expected finalised block, SendDifferentBlocksToMiners.AddResult
	// will be called to save SendDifferentBlocksToMinersResult to SendDifferentBlocksToMiners.
	SendDifferentBlocksToMinersResult struct {
		BlocksRoundRank int `json:"blocks_round_rank"`
	}
)

var (
	// Ensure SendDifferentBlocksToMiners implements config.TestCase interface.
	_ config.TestCase = (*SendDifferentBlocksToMiners)(nil)
)

// NewSendDifferentBlocksToMiners creates initialised SendDifferentBlocksToMiners.
func NewSendDifferentBlocksToMiners() *SendDifferentBlocksToMiners {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	return &SendDifferentBlocksToMiners{
		wg: wg,
	}
}

// Configure implements config.TestCase interface.
func (s *SendDifferentBlocksToMiners) Configure(blob []byte) error {
	defer s.wg.Done()
	s.cfg = new(SendDifferentBlocksToMinersConfig)
	return s.cfg.Decode(blob)
}

// AddResult implements config.TestCase interface.
func (s *SendDifferentBlocksToMiners) AddResult(blob []byte) error {
	defer s.wg.Done()
	s.res = new(SendDifferentBlocksToMinersResult)
	return s.res.Decode(blob)
}

// Check implements config.TestCase interface.
func (s *SendDifferentBlocksToMiners) Check(ctx context.Context) (success bool, err error) {
	prepared := make(chan struct{})
	go func() {
		s.wg.Wait()
		prepared <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return false, errors.New("cases state is not prepared, context is done")

	case <-prepared:
		return s.check()
	}
}

func (s *SendDifferentBlocksToMiners) check() (bool, error) {
	switch {
	case s.cfg == nil || s.res == nil:
		return false, errors.New("cases state is not prepared")

	case s.cfg.MinersRoundRank != 0:
		return false, errors.New("sending blocks was started not from the first ranked generator")

	case s.res.BlocksRoundRank != 1:
		return false, errors.New("finalised block was not from the second ranked generator")

	default:
		return true, nil
	}
}

// Encode encodes SendDifferentBlocksToMinersConfig to bytes.
func (s *SendDifferentBlocksToMinersConfig) Encode() ([]byte, error) {
	return json.Marshal(s)
}

// Decode decodes SendDifferentBlocksToMinersConfig from bytes.
func (s *SendDifferentBlocksToMinersConfig) Decode(blob []byte) error {
	return json.Unmarshal(blob, s)
}

// Encode encodes SendDifferentBlocksToMinersResult to bytes.
func (s *SendDifferentBlocksToMinersResult) Encode() ([]byte, error) {
	return json.Marshal(s)
}

// Decode decodes SendDifferentBlocksToMinersResult from bytes.
func (s *SendDifferentBlocksToMinersResult) Decode(blob []byte) error {
	return json.Unmarshal(blob, s)
}
