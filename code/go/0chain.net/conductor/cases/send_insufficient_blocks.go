package cases

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"0chain.net/conductor/config"
)

type (
	// SendInsufficientProposals represents implementation of the config.TestCase interface.
	//
	//	Flow of this test case:
	//		Check make progress for adversarial leader
	//		(T0) Leader_0 (ad):  send Proposal0_0 for replica j , 0 <= j <1/3f
	//		(T0) Leader_1:  send Proposal0_1
	//		(T0 + δ + Δ) Replica_i: send Verification0_1
	SendInsufficientProposals struct {
		firstGenBlockHash string

		res *RoundInfo

		wg *sync.WaitGroup
	}

	SendInsufficientProposalsResult []*BlockInfo
)

var (
	// Ensure SendInsufficientProposals implements config.TestCase interface.
	_ config.TestCase = (*SendInsufficientProposals)(nil)
)

// NewSendInsufficientProposals creates initialised SendInsufficientProposals.
func NewSendInsufficientProposals() *SendInsufficientProposals {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	return &SendInsufficientProposals{
		wg: wg,
	}
}

// Check implements config.TestCase interface.
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
	for _, bl := range n.res.blocks() {
		if bl.Hash == n.firstGenBlockHash {
			if !bl.Notarised {
				err = errors.New("first generator's block was not notarised")
			}
			return bl.Notarised, err
		}
	}
	return false, errors.New("first generator's block is not found in reports")
}

// Configure implements config.TestCase interface.
func (n *SendInsufficientProposals) Configure(blob []byte) error {
	defer n.wg.Done()
	n.firstGenBlockHash = string(blob)
	return nil
}

// AddResult implements config.TestCase interface.
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
