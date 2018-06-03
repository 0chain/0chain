package memorystore

import (
	"context"
	"fmt"

	"0chain.net/datastore"
)

type MemoryEntity interface {
	datastore.Entity
	Read(ctx context.Context, key datastore.Key) error
	Write(ctx context.Context) error
	Delete(ctx context.Context) error
}

/*GetEntityKey = entity name + entity id */
func GetEntityKey(entity datastore.Entity) datastore.Key {
	var key interface{} = entity.GetKey()
	switch v := key.(type) {
	case string:
		return datastore.ToKey(fmt.Sprintf("%s:%v", entity.GetEntityName(), v))
	case []byte:
		return datastore.ToKey(append(append([]byte(entity.GetEntityName()), ':'), v...))
	default:
		return datastore.EmptyKey
	}
}
