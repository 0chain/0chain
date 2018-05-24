package chain

import (
	"context"

	"0chain.net/datastore"
	"0chain.net/round"
)

/*BlockWorker - a job that does all the work related to blocks in each round */
func BlockWorker(rootCtx context.Context, chain *Chain) {
	rounds := chain.GetRoundsChannel()
	var cancel context.CancelFunc
	ctx := rootCtx
	for r := range rounds {
		if cancel != nil {
			cancel()
			datastore.GetCon(ctx).Close()
		}
		var ccancel context.CancelFunc
		ctx, ccancel = context.WithCancel(rootCtx)
		ctx = datastore.WithConnection(ctx)
		cancel = ccancel
		switch r.Role {
		case round.RoleGenerator:
			go generateBlock(ctx, r)
		case round.RoleVerifier:
			//TODO
		default:
			//TODO
		}
	}
}

func generateBlock(ctx context.Context, r *round.Round) {
	r.Block.GenerateBlock(ctx)
}

func verifyBlock(ctx context.Context, r *round.Round) {
	r.Block.VerifyBlock(ctx)
}

/*SetupBlockWorker - setup a blockworker for a chain */
func SetupBlockWorker(ctx context.Context, c *Chain) {
	go BlockWorker(ctx, c)
}
