package datastore

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/common"
	"github.com/gomodule/redigo/redis"
)

/*WithConnectionHandler - a json response handler that adds a datastore connection to the Context
* It reclaims the connection at the end so there is no connection leak
 */
func WithConnectionHandler(handler common.JSONResponderF) common.JSONResponderF {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		ctx = WithConnection(ctx)
		con := ctx.Value(CONNECTION).(redis.Conn)
		defer con.Close()
		return handler(ctx, r)
	}
}

/*WithConnectionJSONHandler - a json request response handler that adds a datastore connection to the Context
* It reclaims the connection at the end so there is no connection leak
 */
func WithConnectionJSONHandler(handler common.JSONReqResponderF) common.JSONReqResponderF {
	return func(ctx context.Context, json map[string]interface{}) (interface{}, error) {
		ctx = WithConnection(ctx)
		con := ctx.Value(CONNECTION).(redis.Conn)
		defer con.Close()
		return handler(ctx, json)
	}
}

/*WithConnectionEntityJSONHandler - a json request response handler that adds a datastore connection to the Context
* Request is deserialized into an entity
* It reclaims the connection at the end so there is no connection leak
 */
func WithConnectionEntityJSONHandler(handler common.JSONEntityReqResponderF) common.JSONEntityReqResponderF {
	return func(ctx context.Context, object interface{}) (interface{}, error) {
		ctx = WithConnection(ctx)
		con := ctx.Value(CONNECTION).(redis.Conn)
		defer con.Close()
		return handler(ctx, object)
	}
}

/*GetEntityHandler - default get handler implementation for any Entity */
func GetEntityHandler(ctx context.Context, r *http.Request, entityProvider common.EntityProvider, idparam string) (interface{}, error) {
	id := r.FormValue(idparam)
	if id == "" {
		return nil, common.InvalidRequest(fmt.Sprintf("%v is required", idparam))
	}
	entity, ok := entityProvider().(Entity)
	if !ok {
		return nil, common.NewError("dev_error", "Invalid entity provider")
	}
	err := entity.Read(ctx, id)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func DoAsync(ctx context.Context, entity Entity) bool {
	channel := AsyncChannel(ctx)
	if channel != nil {
		channel <- entity
		return true
	}
	return false
}

/*PutEntityHandler - default put handler implementation for any Entity */
func PutEntityHandler(ctx context.Context, object interface{}) (interface{}, error) {
	entity, ok := object.(Entity)
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
