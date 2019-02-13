package block

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
}

/*GetBlock - given an id returns the block information */
func GetBlock(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, blockEntityMetadata, "hash")
}

/*BLOCK_TIME_TOLERANCE - the block creation date should be within these many seconds before/after of current time */
const BLOCK_TIME_TOLERANCE = 5

/*PutBlock - Given a block data, it stores it */
func PutBlock(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	txn, ok := entity.(*Block)
	if !ok {
		return nil, fmt.Errorf("invalid request %T", entity)
	}
	if !common.Within(int64(txn.CreationDate), BLOCK_TIME_TOLERANCE) {
		return nil, common.InvalidRequest("Block creation time not within tolerance")
	}
	err := txn.Write(ctx)
	if err != nil {
		return nil, err
	}
	return txn, nil
}
