package sharder

import (
	"context"

	"0chain.net/chaincore/node"
)

/*SetupWorkers - setup the background workers */
func SetupWorkers(ctx context.Context) {
	sc := GetSharderChain()
	go sc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go sc.FinalizeRoundWorker(ctx, sc)  // 2) sequentially finalize the rounds
	go sc.FinalizedBlockWorker(ctx, sc) // 3) sequentially processes finalized blocks
	go sc.NodeStatusWorker(ctx)
}

/*BlockWorker - stores the blocks */
func (sc *Chain) BlockWorker(ctx context.Context) {
	for true {
		select {
		case <-ctx.Done():
			return
		case b := <-sc.GetBlockChannel():
			sc.processBlock(ctx, b)
		}
	}
}

func (sc *Chain) NodeStatusWorker(ctx context.Context) {
	for true {
		select {
		case <-ctx.Done():
			return
		case n := <-node.Self.NodeStatusChannel:
			node.Self.ActiveNodes[n.ID] = n
		}
	}
}
