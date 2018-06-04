package sharder

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/datastore"
	"0chain.net/persistencestore"
)

func SetupHandlers() {
	http.HandleFunc("/sharder/put", datastore.ToJSONEntityReqResponse(persistencestore.WithConnectionEntityJSONHandler(PostSharder, blockEntityMetadata), blockEntityMetadata))
}

func PostSharder(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	fmt.Println("At Post Sharder")
	txn, ok := entity.(*Sharder)
	if !ok {
		return nil, fmt.Errorf("invalid request %T", entity)
	}
	err := txn.PWrite(ctx)
	if err != nil {
		return nil, err
	}
	return txn, nil

}
