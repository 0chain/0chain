package client

import (
	"context"
	"net/http"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"0chain.net/smartcontract/dbs/event"
)

type Chainer interface {
	GetEventDb() *event.EventDb
}

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers(c Chainer) {
	http.HandleFunc("/v1/client/get",
		common.UserRateLimit(
			common.ToJSONResponse(
				memorystore.WithConnectionHandler(GetClientHandler))))
	http.HandleFunc("/v1/client/put",
		common.UserRateLimit(
			datastore.ToJSONEntityReqResponse(
				WithEmitEventHandler(
					c,
					memorystore.WithConnectionEntityJSONHandler(
						PutClient, clientEntityMetadata),
					clientEntityMetadata), clientEntityMetadata)))
}

/*GetClientHandler - given an id returns the client information */
func GetClientHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, clientEntityMetadata, "id")
}

// WithEmitEventHandler emits event on execution of handler
func WithEmitEventHandler(c Chainer, handler datastore.JSONEntityReqResponderF, entityMetadata datastore.EntityMetadata) datastore.JSONEntityReqResponderF {
	return func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		ctx = memorystore.WithEntityConnection(ctx, entityMetadata)
		defer memorystore.Close(ctx)
		if c.GetEventDb() != nil {
			c.GetEventDb().AddEvents(ctx, []event.Event{})
		}
		return handler(ctx, entity)
	}
}
