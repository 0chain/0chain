package client

import (
	"context"
	"github.com/0chain/common/constants/endpoint/v1_endpoint/miner_endpoint"
	"net/http"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc(miner_endpoint.GetClient.Path(),
		common.WithCORS(
			common.UserRateLimit(
				common.ToJSONResponse(
					memorystore.WithConnectionHandler(GetClientHandler)))))
	http.HandleFunc(miner_endpoint.PutClient.Path(),
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
