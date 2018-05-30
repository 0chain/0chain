package datastore

import (
	"context"
	"fmt"
)

/*PrintEntityHandler - handler that prints the received entity */
func PrintEntityHandler(ctx context.Context, object interface{}) (interface{}, error) {
	entity, ok := object.(Entity)
	if ok {
		fmt.Printf("%v: %v\n", entity.GetEntityName(), ToJSON(entity))

	} else {
		fmt.Printf("%T: %v\n", object, object)
	}
	return nil, nil
}
