package memorystore

import (
	"context"
	"net/http"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

/*WithConnectionHandler - a json response handler that adds a memorystore connection to the Context
* It reclaims the connection at the end so there is no connection leak
 */
func WithConnectionHandler(handler common.JSONResponderF) common.JSONResponderF {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		ctx = WithConnection(ctx)
		defer Close(ctx)
		return handler(ctx, r)
	}
}

/*WithConnectionJSONHandler - a json request response handler that adds a memorystore connection to the Context
* It reclaims the connection at the end so there is no connection leak
 */
func WithConnectionJSONHandler(handler common.JSONReqResponderF) common.JSONReqResponderF {
	return func(ctx context.Context, json map[string]interface{}) (interface{}, error) {
		ctx = WithConnection(ctx)
		defer Close(ctx)
		return handler(ctx, json)
	}
}

/*WithConnectionEntityJSONHandler - a json request response handler that adds a memorystore connection to the Context
* Request is deserialized into an entity
* It reclaims the connection at the end so there is no connection leak
 */
func WithConnectionEntityJSONHandler(handler datastore.JSONEntityReqResponderF, entityMetadata datastore.EntityMetadata) datastore.JSONEntityReqResponderF {
	return func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		ctx = WithEntityConnection(ctx, entityMetadata)
		defer Close(ctx)
		return handler(ctx, entity)
	}
}
