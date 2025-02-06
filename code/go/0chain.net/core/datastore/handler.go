package datastore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/0chain/common/core/logging"

	"0chain.net/core/common"
)

/*EntityProvider - returns an entity */
type EntityProvider func() Entity

/*JSONEntityReqResponderF - a handler that takes a JSON request and responds with a json response
* Useful for GET operation where the input is coming via url parameters
 */
type JSONEntityReqResponderF func(ctx context.Context, entity Entity) (interface{}, error)

/*ToJSONEntityReqResponse - Similar to ToJSONReqResponse except it takes an EntityProvider
* that returns an interface into which the incoming request json is unmarshalled
* Avoids extra map creation and also wiring it manually from the map to the entity object
 */
func ToJSONEntityReqResponse(handler JSONEntityReqResponderF, entityMetadata EntityMetadata) common.ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			common.SetupCORSResponse(w)
			return
		}
		contentType := r.Header.Get("Content-type")
		if !strings.HasPrefix(contentType, "application/json") {
			http.Error(w, "Header Content-type=application/json not found", 400)
			return
		}
		entity := entityMetadata.Instance()
		if err := json.NewDecoder(r.Body).Decode(entity); err != nil {
			logging.Logger.Error("decode err", zap.Error(err))
			http.Error(w, "Error decoding json", 400)
			return
		}
		if err := entity.ComputeProperties(); err != nil {
			logging.Logger.Error("compute properties err", zap.Error(err))
			http.Error(w, "Error computing properties", 400)
			return
		}

		ctx := r.Context()
		rsp, err := handler(ctx, entity)
		common.Respond(w, r, rsp, err)
	}
}

/*PrintEntityHandler - handler that prints the received entity */
func PrintEntityHandler(ctx context.Context, entity Entity) (interface{}, error) {
	emd := entity.GetEntityMetadata()
	if emd == nil {
		return nil, common.NewError("unknown_entity", "Entity with nil entity metadata")
	}
	fmt.Printf("%v: %v\n", emd.GetName(), ToJSON(entity))
	return nil, nil
}

/*GetEntityHandler - default get handler implementation for any Entity */
func GetEntityHandler(ctx context.Context, r *http.Request, entityMetadata EntityMetadata, idparam string) (interface{}, error) {
	id := r.FormValue(idparam)
	if id == "" {
		return nil, common.InvalidRequest(fmt.Sprintf("%v is required", idparam))
	}
	entity := entityMetadata.Instance()
	err := entity.Read(ctx, ToKey(id))
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func GetEntityByHash(ctx context.Context, entityMetadata EntityMetadata, hash string) (interface{}, error) {
	entity := entityMetadata.Instance()
	err := entity.Read(ctx, ToKey(hash))
	if err != nil {
		return nil, err
	}
	return entity, nil
}

/*PutEntityHandler - default put handler implementation for any Entity */
func PutEntityHandler(ctx context.Context, object interface{}) (interface{}, error) {
	entity, ok := object.(Entity)
	if !ok {
		return nil, fmt.Errorf("invalid request %T", object)
	}
	if err := entity.ComputeProperties(); err != nil {
		return nil, err
	}

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
