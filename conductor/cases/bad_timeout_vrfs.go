package cases

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"0chain.net/conductor/conductrpc/stats"
)

type (
	// BadTimeoutVRFS represents implementation of the TestCase interface.
	//
	//	Flow of this test case:
	//		Malicious replicas start to send VRFShares for current round as if they get up after restart
	//		(T0) Replica_i(Adv): send VRFShare(timeout), 0<=i<1/3f (on round n)
	BadTimeoutVRFS struct {
		res *RoundInfo

		statsCollector *stats.NodesServerStats
		monitorID      string

		wg *sync.WaitGroup
	}
)

var (
	// Ensure BadTimeoutVRFS implements TestCase interface.
	_ TestCase = (*BadTimeoutVRFS)(nil)
)

// NewBadTimeoutVRFS creates initialised BadTimeoutVRFS.
func NewBadTimeoutVRFS(statsCollector *stats.NodesServerStats, monitorID string) *BadTimeoutVRFS {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	return &BadTimeoutVRFS{
		statsCollector: statsCollector,
		monitorID:      monitorID,
		wg:             wg,
	}
}

// Check implements TestCase interface.
func (n *BadTimeoutVRFS) Check(ctx context.Context) (success bool, err error) {
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

func (n *BadTimeoutVRFS) check() (success bool, err error) {
	if n.monitorID == "" {
		return false, errors.New("configured monitor ID is empty")
	}
	if n.res.TimeoutCount != 0 {
		return false, errors.New("expected 0 timeout count")
	}

	for _, requests := range n.statsCollector.VRFS {
		requestsByRound := requests.GetByRound(n.res.Num)
		for _, req := range requestsByRound {
			if req.SenderID != n.monitorID && req.TimeoutCount == 1 {
				return false, fmt.Errorf("unexpected vrf share sending from node: %s", req.SenderID)
			}
		}
	}
	return true, nil
}

// Configure implements TestCase interface.
func (n *BadTimeoutVRFS) Configure(_ []byte) error {
	panic("configuring test case is not allowed")
}

// AddResult implements TestCase interface.
func (n *BadTimeoutVRFS) AddResult(blob []byte) error {
	defer n.wg.Done()
	n.res = new(RoundInfo)
	return n.res.Decode(blob)
}
