package cases

import (
	"context"
	"errors"
	"sync"
)

type (
	// MissingLFBTickets represents implementation of the TestCase interface.
	//
	//	Flow of this test case:
	//		Missing LFB tickets:
	//		1. On round n sharders stop sending LFB tickets.
	//		2. Check: miners stop working with timeout on round_n+5
	MissingLFBTickets struct {
		results []*RoundInfo

		wg *sync.WaitGroup
	}
)

var (
	// Ensure MissingLFBTickets implements TestCase interface.
	_ TestCase = (*MissingLFBTickets)(nil)
)

// NewMissingLFBTickets creates initialised MissingLFBTickets.
func NewMissingLFBTickets(minersNum int) *MissingLFBTickets {
	wg := new(sync.WaitGroup)
	wg.Add(minersNum) // miners num = number of results
	return &MissingLFBTickets{
		wg: wg,
	}
}

// Check implements TestCase interface.
func (n *MissingLFBTickets) Check(ctx context.Context) (success bool, err error) {
	prepared := make(chan struct{})
	go func() {
		n.wg.Wait()
		prepared <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return false, errors.New("cases state is not prepared, context is done")

	case <-prepared:
		return true, nil // when all soft timeouts are reported, there is no need to make additional checks
	}
}

// Configure implements TestCase interface.
func (n *MissingLFBTickets) Configure(_ []byte) error {
	panic("not implemented")
}

// AddResult implements TestCase interface.
// When miners nodes got soft round timeout, they report round info.
func (n *MissingLFBTickets) AddResult(blob []byte) error {
	defer n.wg.Done()
	res := new(RoundInfo)
	if err := res.Decode(blob); err != nil {
		return err
	}

	n.results = append(n.results, res)

	return nil
}
