package chain

import (
	"context"
	"net/http"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/chain/get", common.ToJSONResponse(datastore.WithConnectionHandler(GetChain)))
	http.HandleFunc("/v1/chain/put", common.ToJSONEntityReqResponse(datastore.WithConnectionEntityJSONHandler(PutChain), Provider))
}

/*GetChain - given an id returns the chain information */
func GetChain(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, Provider, "id")
}

/*PutChain - Given a chain data, it stores it */
func PutChain(ctx context.Context, object interface{}) (interface{}, error) {
	return datastore.PutEntityHandler(ctx, object)
}
