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
func (mc *Chain) BlockWorker(ctx context.Context) {
	rounds := mc.GetRoundsChannel()
	for true {
		select {
		case <-ctx.Done():
			return
		case r := <-rounds:
			switch r.Role {
			case round.RoleGenerator:
				go mc.generateBlock(ctx, r)
			case round.RoleVerifier:
				go mc.verifyBlock(ctx, r)
			}
		}
	}
}

func (mc *Chain) generateBlock(ctx context.Context, r *round.Round) {
	lctx := memorystore.WithConnection(ctx)
	defer memorystore.Close(lctx)
	mc.GenerateBlock(ctx, r.Block)
}

func (mc *Chain) verifyBlock(ctx context.Context, r *round.Round) {
	lctx := memorystore.WithConnection(ctx)
	defer memorystore.Close(lctx)
	mc.VerifyBlock(ctx, r.Block)
}
