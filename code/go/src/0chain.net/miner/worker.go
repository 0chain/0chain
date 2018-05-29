package miner

import (
	"context"

	"0chain.net/common"
	"0chain.net/memorystore"
	"0chain.net/round"
)

/*SetupWorkers - Setup the miner's workers */
func SetupWorkers() {
	ctx := common.GetRootContext()
	go GetMinerChain().BlockWorker(ctx)
}

/*BlockWorker - a job that does all the work related to blocks in each round */
func (c *Chain) BlockWorker(ctx context.Context) {
	rounds := c.GetRoundsChannel()
	for true {
		select {
		case <-ctx.Done():
			return
		case r := <-rounds:
			switch r.Role {
			case round.RoleGenerator:
				go generateBlock(ctx, r)
			case round.RoleVerifier:
				go verifyBlock(ctx, r)
			}
		}
	}
}

func generateBlock(ctx context.Context, r *round.Round) {
	lctx := memorystore.WithConnection(ctx)
	defer memorystore.GetCon(lctx).Close()
	r.Block.GenerateBlock(ctx)
}

func verifyBlock(ctx context.Context, r *round.Round) {
	lctx := memorystore.WithConnection(ctx)
	defer memorystore.GetCon(lctx).Close()
	r.Block.VerifyBlock(ctx)
}
