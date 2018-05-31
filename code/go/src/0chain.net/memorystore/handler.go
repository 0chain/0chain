package memorystore

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/common"
	"0chain.net/datastore"
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

/*GetEntityHandler - default get handler implementation for any Entity */
func GetEntityHandler(ctx context.Context, r *http.Request, entityMetadata datastore.EntityMetadata, idparam string) (interface{}, error) {
	id := r.FormValue(idparam)
	if id == "" {
		return nil, common.InvalidRequest(fmt.Sprintf("%v is required", idparam))
	}
	entity, ok := entityMetadata.Instance().(MemoryEntity)
	if !ok {
		return nil, common.NewError("dev_error", "Invalid entity provider")
	}
	err := entity.Read(ctx, datastore.ToKey(id))
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func DoAsync(ctx context.Context, entity MemoryEntity) bool {
	channel := AsyncChannel(ctx)
	if channel != nil {
		channel <- entity
		return true
	}
	return false
}

/*PutEntityHandler - default put handler implementation for any Entity */
func PutEntityHandler(ctx context.Context, object interface{}) (interface{}, error) {
	entity, ok := object.(MemoryEntity)
	if !ok {
		return nil, fmt.Errorf("invalid request %T", object)
	}
	entity.ComputeProperties()
	if err := entity.Validate(ctx); err != nil {
		return nil, err
	}
	if DoAsync(ctx, entity) {
		return entity, nil
	}
	err := entity.Write(ctx)
	if err != nil {
		return nil, err
	}
	return entity, nil
}
