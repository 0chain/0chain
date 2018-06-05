package client

import (
	"context"
	"net/http"

	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/memorystore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/client/get", common.ToJSONResponse(memorystore.WithConnectionHandler(GetClient)))
	http.HandleFunc("/v1/client/put", datastore.ToJSONEntityReqResponse(datastore.DoAsyncEntityJSONHandler(PutClient, ClientEntityChannel), clientEntityMetadata))
}

/*GetClient - given an id returns the client information */
func GetClient(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, clientEntityMetadata, "id")
}

/*PutClient - Given a client data, it stores it */
func PutClient(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	return datastore.PutEntityHandler(ctx, entity)
}
