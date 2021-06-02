package client

import (
	"context"
	"net/http"

	"github.com/0chain/0chain/code/go/0chain.net/core/common"
	"github.com/0chain/0chain/code/go/0chain.net/core/datastore"
	"github.com/0chain/0chain/code/go/0chain.net/core/memorystore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/client/get",
		common.UserRateLimit(
			common.ToJSONResponse(
				memorystore.WithConnectionHandler(GetClientHandler))))
	http.HandleFunc("/v1/client/put",
		common.UserRateLimit(
			datastore.ToJSONEntityReqResponse(
				memorystore.WithConnectionEntityJSONHandler(
					PutClient, clientEntityMetadata),
				clientEntityMetadata)))
}

/*GetClientHandler - given an id returns the client information */
func GetClientHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, clientEntityMetadata, "id")
}
