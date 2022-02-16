package cases

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type (
	// NotNotarisedBlockExtension represents implementation of the TestCase interface.
	//
	// 	Flow of this test case:
	//		Leader extends not notarized prev_block
	//		(T0) Leader_0: send Proposal0_0, extends not notarized prev_block
	//		(T0) Leader_1: send Proposal1_0
	//		(T0 + δ) Replica_0: reject Proposal0_, verify Proposal1_0
	NotNotarisedBlockExtension struct {
		mockedBlockHashToExtend string

		result *RoundInfo

		wg *sync.WaitGroup
	}
)

var (
	// Ensure NotNotarisedBlockExtension implements TestCase interface.
	_ TestCase = (*NotNotarisedBlockExtension)(nil)
)

// NewNotNotarisedBlockExtension creates initialised NotNotarisedBlockExtension.
func NewNotNotarisedBlockExtension() *NotNotarisedBlockExtension {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	return &NotNotarisedBlockExtension{
		wg: wg,
	}
}

// Check implements TestCase interface.
func (n *NotNotarisedBlockExtension) Check(ctx context.Context) (success bool, err error) {
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

func (n *NotNotarisedBlockExtension) check() (success bool, err error) {
	if n.mockedBlockHashToExtend == "" {
		return false, errors.New("mocked block is nil")
	}

	for _, b := range n.result.blocks() {
		prevBlockHash, status := b.PrevHash, b.VerificationStatus
		switch {
		case prevBlockHash == n.mockedBlockHashToExtend && status == BlocksVerificationSuccessful:
			return false, fmt.Errorf("block with %s previous block hash has unexpected status: %d", prevBlockHash, status)

		case prevBlockHash == n.mockedBlockHashToExtend && status == BlocksVerificationFailed:
			return true, nil

		case prevBlockHash == n.mockedBlockHashToExtend && status == BlocksVerificationPending:
			return false, errors.New("checked block has verification pending status")
		}
	}
	return true, nil
}

// Configure implements TestCase interface.
func (n *NotNotarisedBlockExtension) Configure(blob []byte) error {
	defer n.wg.Done()
	n.mockedBlockHashToExtend = string(blob)
	return nil
}

// AddResult implements TestCase interface.
func (n *NotNotarisedBlockExtension) AddResult(blob []byte) error {
	defer n.wg.Done()
	n.result = new(RoundInfo)
	return n.result.Decode(blob)
}
