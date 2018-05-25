package chain

import (
	"context"

	"0chain.net/datastore"
	"0chain.net/round"
)

/*SetupWorkers - setup a blockworker for a chain */
func (c *Chain) SetupWorkers(ctx context.Context) {
	go c.Miners.StatusMonitor(ctx)
	go c.Sharders.StatusMonitor(ctx)
	go c.Blobbers.StatusMonitor(ctx)
	go c.BlockWorker(ctx)
}

/*BlockWorker - a job that does all the work related to blocks in each round */
func (c *Chain) BlockWorker(ctx context.Context) {
	rounds := c.GetRoundsChannel()
	var cancel context.CancelFunc
	lctx := ctx
	for true {
		select {
		case <-ctx.Done():
			return
		case r := <-rounds:
			if cancel != nil {
				cancel()
			}
			var ccancel context.CancelFunc
			lctx, ccancel = context.WithCancel(ctx)
			lctx = datastore.WithConnection(lctx)
			cancel = ccancel
			switch r.Role {
			case round.RoleGenerator:
				go generateBlock(lctx, r)
			case round.RoleVerifier:
				go verifyBlock(lctx, r)
			default:
				//TODO
			}
		}
	}
}

func generateBlock(ctx context.Context, r *round.Round) {
	r.Block.GenerateBlock(ctx)
}

func verifyBlock(ctx context.Context, r *round.Round) {
	r.Block.VerifyBlock(ctx)
}
