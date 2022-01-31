package cases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
)

type (
	// CollectVerificationTicket represents implementation of the config.TestCase interface.
	//
	// 	Flow of this test case:
	//		Leader extends not notarized prev_block
	//		(T0) Leader_0: send Proposal5_0
	//		(T0 + δ) Replica_0: ignore Proposal5_i
	CollectVerificationTicket struct {
		cfg *CollectVerificationTicketCfg

		result  *RoundInfo // key - previous block's hash; value - verification status
		enabled bool

		wg    *sync.WaitGroup
		mutex *sync.Mutex
	}

	CollectVerificationTicketCfg struct {
		BlockHash string
	}
)

var (
	// Ensure CollectVerificationTicket implements TestCase interface.
	_ TestCase = (*CollectVerificationTicket)(nil)
)

// NewCollectVerificationTicket creates initialised CollectVerificationTicket.
func NewCollectVerificationTicket() *CollectVerificationTicket {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	return &CollectVerificationTicket{
		result:  &RoundInfo{},
		mutex:   &sync.Mutex{},
		enabled: true,
		wg:      wg,
	}
}

// Check implements config.TestCase interface.
func (n *CollectVerificationTicket) Check(ctx context.Context) (success bool, err error) {
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

func (n *CollectVerificationTicket) check() (success bool, err error) {
	msg := fmt.Sprintf("blockID: %+v info %+v", n.cfg.BlockHash, func() bool {
		for _, elem := range n.result.NotarisedBlocks {
			if elem.Hash == n.cfg.BlockHash {
				return true
			}
		}
		return false
	}())
	log.Println(msg)

	return true, nil
}

// Configure implements TestCase interface.
func (n *CollectVerificationTicket) Configure(blob []byte) error {
	defer func() {
		if n.enabled {
			n.enabled = false
			n.wg.Done()
		}
		n.mutex.Unlock()
	}()
	n.mutex.Lock()
	if !n.enabled {
		return nil
	}
	n.cfg = new(CollectVerificationTicketCfg)
	return n.cfg.Decode(blob)
}

// AddResult implements TestCase interface.
func (n *CollectVerificationTicket) AddResult(blob []byte) error {
	defer n.wg.Done()
	n.result = new(RoundInfo)
	return n.result.Decode(blob)
}

// Encode encodes CollectVerificationTicketCfg to bytes.
func (r *CollectVerificationTicketCfg) Encode() ([]byte, error) {
	return json.Marshal(r)
}

// Decode decodes CollectVerificationTicketCfg from bytes.
func (r *CollectVerificationTicketCfg) Decode(blob []byte) error {
	return json.Unmarshal(blob, r)
}
