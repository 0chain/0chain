package client

import (
	"context"
	"net/http"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/client/get", common.ToJSONResponse(datastore.WithConnectionHandler(GetClient)))
	http.HandleFunc("/v1/client/put", common.ToJSONEntityReqResponse(datastore.DoAsyncEntityJSONHandler(PutClient, ClientEntityChannel), ClientProvider))
}

/*GetClient - given an id returns the client information */
func GetClient(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, ClientProvider, "id")
}

/*PutClient - Given a client data, it stores it */
func PutClient(ctx context.Context, object interface{}) (interface{}, error) {
	return datastore.PutEntityHandler(ctx, object)
}
