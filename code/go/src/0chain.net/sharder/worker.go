package sharder

import (
	"context"

	"0chain.net/common"
)

/*SetupWorkers - setup the background workers */
func SetupWorkers() {
	ctx := common.GetRootContext()
	go GetSharderChain().BlockWorker(ctx)
}

/*BlockWorker - stores the blocks */
func (sc *Chain) BlockWorker(ctx context.Context) {
	for true {
		select {
		case <-ctx.Done():
			return
		case b := <-sc.GetBlockChannel():
			StoreBlock(ctx, b)
		}
	}
}
