package cases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"0chain.net/chaincore/block"
	"0chain.net/conductor/config"
)

type (
	// NotNotarisedBlockExtension represents implementation of the config.TestCase interface.
	//
	// 	Flow of this test case:
	//		Leader extends not notarized prev_block
	//		(T0) Leader_0: send Proposal0_0, extends not notarized prev_block
	//		(T0) Leader_1: send Proposal1_0
	//		(T0 + δ) Replica_0: reject Proposal0_, verify Proposal1_0
	NotNotarisedBlockExtension struct {
		mockedBlockHashToExtend string

		result map[string]int // key - previous block's hash; value - verification status

		wg *sync.WaitGroup
	}
)

var (
	// Ensure NotNotarisedBlockExtension implements config.TestCase interface.
	_ config.TestCase = (*NotNotarisedBlockExtension)(nil)
)

// NewNotNotarisedBlockExtension creates initialised NotNotarisedBlockExtension.
func NewNotNotarisedBlockExtension() *NotNotarisedBlockExtension {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	return &NotNotarisedBlockExtension{
		result: make(map[string]int),
		wg:     wg,
	}
}

// Check implements config.TestCase interface.
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

	for hash, status := range n.result {
		switch {
		case hash == n.mockedBlockHashToExtend && status == block.VerificationSuccessful:
			msg := fmt.Sprintf("block with %s previous block hash have unexpected status: %d", hash, status)
			return false, errors.New(msg)

		case hash == n.mockedBlockHashToExtend && status == block.VerificationFailed:
			return true, nil

		case hash == n.mockedBlockHashToExtend && status == block.VerificationPending:
			return false, errors.New("checked block has verification pending status")
		}
	}
	return true, nil
}

// Configure implements config.TestCase interface.
func (n *NotNotarisedBlockExtension) Configure(blob []byte) error {
	defer n.wg.Done()
	n.mockedBlockHashToExtend = string(blob)
	return nil
}

// AddResult implements config.TestCase interface.
func (n *NotNotarisedBlockExtension) AddResult(blob []byte) error {
	defer n.wg.Done()
	return json.Unmarshal(blob, &n.result)
}
