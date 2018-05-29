package client

import (
	"context"
	"net/http"

	"0chain.net/common"
	"0chain.net/memorystore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/client/get", common.ToJSONResponse(memorystore.WithConnectionHandler(GetClient)))
	http.HandleFunc("/v1/client/put", common.ToJSONEntityReqResponse(memorystore.DoAsyncEntityJSONHandler(PutClient, ClientEntityChannel), Provider))
}

/*GetClient - given an id returns the client information */
func GetClient(ctx context.Context, r *http.Request) (interface{}, error) {
	return memorystore.GetEntityHandler(ctx, r, Provider, "id")
}

/*PutClient - Given a client data, it stores it */
func PutClient(ctx context.Context, object interface{}) (interface{}, error) {
	return memorystore.PutEntityHandler(ctx, object)
}
