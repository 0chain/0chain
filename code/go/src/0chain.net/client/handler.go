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
	http.HandleFunc("/v1/client/get", common.UserRateLimit(common.ToJSONResponse(memorystore.WithConnectionHandler(GetClientHandler))))
	http.HandleFunc("/v1/client/put", common.UserRateLimit(datastore.ToJSONEntityReqResponse(datastore.DoAsyncEntityJSONHandler(PutClient, ClientEntityChannel), clientEntityMetadata)))
}

/*GetClientHandler - given an id returns the client information */
func GetClientHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, clientEntityMetadata, "id")
}
