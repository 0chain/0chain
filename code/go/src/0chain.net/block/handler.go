package block

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/common"
	"0chain.net/memorystore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/block/get", common.ToJSONResponse(memorystore.WithConnectionHandler(GetBlock)))
	http.HandleFunc("/v1/block/put", common.ToJSONEntityReqResponse(memorystore.WithConnectionEntityJSONHandler(PutBlock), Provider))
}

/*GetBlock - given an id returns the block information */
func GetBlock(ctx context.Context, r *http.Request) (interface{}, error) {
	return memorystore.GetEntityHandler(ctx, r, Provider, "hash")
}

/*BLOCK_TIME_TOLERANCE - the txn creation date should be within 5 seconds before/after of current time */
const BLOCK_TIME_TOLERANCE = 5

/*PutBlock - Given a block data, it stores it */
func PutBlock(ctx context.Context, object interface{}) (interface{}, error) {
	txn, ok := object.(*Block)
	if !ok {
		return nil, fmt.Errorf("invalid request %T", object)
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
