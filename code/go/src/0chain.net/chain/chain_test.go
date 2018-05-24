package chain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"0chain.net/block"
	"0chain.net/round"
)

func TestChainSetupWorker(t *testing.T) {
	block.BLOCK_SIZE = 100 // Just for testing
	c := Provider().(*Chain)
	c.ID = GetServerChainID()
	ctx, cancel := context.WithCancel(context.Background())
	SetupBlockWorker(ctx, c)
	timer := time.NewTimer(100 * time.Second)
	ticker := time.NewTicker(time.Second)
	go func() {
		r := &round.Round{}
		r.Number = 0
		r.Role = round.RoleVerifier
		roundsChannel := c.GetRoundsChannel()
		for _ = range ticker.C {
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
	}()
	for _ = range timer.C {
		cancel()
	}
}
