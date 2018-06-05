package sharder

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/node"
	"0chain.net/persistencestore"
)

/*SetupM2SReceivers - setup handlers for all the messages received from the miner */
func SetupM2SReceivers() {
	http.HandleFunc("/v1/_m2s/block/finalized", node.ToN2NReceiveEntityHandler(persistencestore.WithConnectionEntityJSONHandler(FinalizedBlockHandler, datastore.GetEntityMetadata("block"))))
}

/*FinalizedBlockHandler - handle the finalized block */
func FinalizedBlockHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	b, ok := entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	//TODO: Store the block in file system and also persistence store
	fmt.Printf("Received Round(%v) Block(%v)\n", b.Round, b.GetKey())
	return true, nil
}
