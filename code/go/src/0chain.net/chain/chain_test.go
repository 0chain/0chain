package chain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/round"
)

func TestChainSetupWorker(t *testing.T) {
	block.BLOCK_SIZE = 10 // Just for testing
	c := Provider().(*Chain)
	c.ID = GetServerChainID()
	c.SetupWorkers(common.GetRootContext())
	timer := time.NewTimer(10 * time.Second)
	startTime := time.Now()
	go RoundLogic(common.GetRootContext(), c)
	ts := <-timer.C
	fmt.Printf("reached timeout: %v %v\n", time.Since(startTime), ts)
	common.Done()
}

func RoundLogic(ctx context.Context, c *Chain) {
	ticker := time.NewTicker(100 * time.Millisecond)
	r := &round.Round{}
	r.Number = 0
	r.Role = round.RoleVerifier
	roundsChannel := c.GetRoundsChannel()
	for true {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			fmt.Printf("round: %v\n", r)
			if r.Block != nil {
				for idx, txn := range r.Block.Txns {
					fmt.Printf("txn(%v): %v\n", idx, txn)
				}
			}
			r.Number++
			b := block.Provider().(*block.Block)
			r.Block = b
			if r.Role == round.RoleVerifier {
				r.Role = round.RoleGenerator
			} else {
				r.Role = round.RoleVerifier
				b.Txns = make([]interface{}, 0)
			}
			roundsChannel <- r
		}
	}
}
