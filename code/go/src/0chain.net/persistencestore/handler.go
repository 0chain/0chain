package persistencestore

import (
	"context"

	"0chain.net/datastore"
)

/*WithConnectionEntityJSONHandler - a json request response handler that adds a memorystore connection to the Context
* Request is deserialized into an entity
* It reclaims the connection at the end so there is no connection leak
 */
func WithConnectionEntityJSONHandler(handler datastore.JSONEntityReqResponderF, entityMetadata datastore.EntityMetadata) datastore.JSONEntityReqResponderF {
	return func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		ctx = WithConnection(ctx, entityMetadata)
		// defer Close(ctx)
		return handler(ctx, entity)
	}
}
