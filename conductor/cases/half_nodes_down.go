package cases

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
)

type (
	// HalfNodesDown represents implementation of the TestCase interface.
	//
	//	Flow of this test case:
	//		Half of nodes are down, generator proposes but can't get enough verification tickets, restarts.
	//		Soon half of nodes get up and restarts round.
	//
	//		(T0) Replica_i: down, 0<=i<1/2f
	//		(T0) Leader0_0: send Proposal0_0
	//		(T0 + δ + Δ) Replica_0: send VerificationTicket0_0
	//		(T0 + timeout) Replica_j: send VRFShare(timeout), 1/2f<j<=f
	//		(T0 + 4*timeout) Replica_i: get up, send VRFShare(timeout), 0<=i<1/2f
	//		(T0 + 4*timeout + δ): Leader0_1: send Proposal0_1
	HalfNodesDown struct {
		resultsMu sync.Mutex
		results   []*RoundInfo

		roundRandomSeedFromStartMu sync.Mutex
		roundRandomSeedFromStart   int

		minersNum int

		wg *sync.WaitGroup
	}
)

var (
	// Ensure HalfNodesDown implements TestCase interface.
	_ TestCase = (*HalfNodesDown)(nil)
)

// NewHalfNodesDown creates initialised HalfNodesDown.
func NewHalfNodesDown(minersNum int) *HalfNodesDown {
	wg := new(sync.WaitGroup)
	wg.Add(minersNum + 1)
	return &HalfNodesDown{
		minersNum: minersNum,
		wg:        wg,
	}
}

// Check implements TestCase interface.
func (n *HalfNodesDown) Check(ctx context.Context) (success bool, err error) {
	prepared := make(chan struct{})
	go func() {
		n.wg.Wait()
		prepared <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		// miners should send only first round restart reports, so if there is insufficient reports number
		// it means that round has not restarted.
		if expectedReportsNum := n.minersNum - n.minersNum/2; len(n.results) != expectedReportsNum {
			return false, fmt.Errorf("unexpected number of result reports: %d, expected %d", len(n.results), expectedReportsNum)
		}

		return false, errors.New("cases state is not prepared, context is done")

	case <-prepared:
		return n.check()
	}
}

func (n *HalfNodesDown) check() (bool, error) {
	for _, res := range n.results {
		switch {
		case res.TimeoutCount != 1:
			return false, fmt.Errorf("unexpected round timeout count: %d", res.TimeoutCount)

		case res.RoundRandomSeed == int64(n.roundRandomSeedFromStart):
			return false, errors.New("round random seed after timeout is unchanged")
		}
	}
	return true, nil
}

// Configure implements TestCase interface.
func (n *HalfNodesDown) Configure(blob []byte) (err error) {
	n.roundRandomSeedFromStartMu.Lock()
	defer n.roundRandomSeedFromStartMu.Unlock()

	if n.roundRandomSeedFromStart != 0 {
		return nil
	}

	defer n.wg.Done()
	n.roundRandomSeedFromStart, err = strconv.Atoi(string(blob))
	return err
}

// AddResult implements TestCase interface.
func (n *HalfNodesDown) AddResult(blob []byte) error {
	n.resultsMu.Lock()
	defer n.resultsMu.Unlock()

	if len(n.results) == n.minersNum {
		return nil
	}

	defer n.wg.Done()
	res := new(RoundInfo)
	if err := res.Decode(blob); err != nil {
		return err
	}
	n.results = append(n.results, res)
	return nil
}
