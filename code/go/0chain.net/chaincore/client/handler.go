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
	http.HandleFunc("/v1/client/put",
		common.WithCORS(
			common.UserRateLimit(
				datastore.ToJSONEntityReqResponse(
					memorystore.WithConnectionEntityJSONHandler(
						PutClient, clientEntityMetadata),
					clientEntityMetadata))))
}

/*GetClientHandler - given an id returns the client information */
func GetClientHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, clientEntityMetadata, "id")
}
