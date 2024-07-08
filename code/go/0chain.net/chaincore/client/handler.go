package client

import (
	"context"
	"net/http"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/client/get",
		common.WithCORS(
			common.UserRateLimit(
				common.ToJSONResponse(
					memorystore.WithConnectionHandler(GetClientHandler)))))
}

/*GetClientHandler - given an id returns the client information */
// swagger:route GET /v1/client/get miner GetClient
// Get client.
// Retrieves the client information.
//
// parameters:
//    +name: id
//      in: query
//      required: true
//      type: string
//      description: "Client ID"
//
// responses:
//
//	200: Client
//  400:
func GetClientHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, clientEntityMetadata, "id")
}
