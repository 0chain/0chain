package persistencestore

import (
	"context"
	"fmt"

	"0chain.net/common"
	"0chain.net/datastore"
)

var providers = make(map[string]common.EntityProvider)

/*RegisterEntityProvider - keep track of a list of entity providers. An entity can be registered with multiple names
* as long as two entities don't use the same name
 */
func RegisterEntityProvider(entityName string, provider common.EntityProvider) {
	providers[entityName] = provider
}

/*GetProvider - return the provider registered for the given entity */
func GetProvider(entityName string) common.EntityProvider {
	return providers[entityName]
}

type PersistenceEntity interface {
	datastore.Entity
	Read(ctx context.Context, key datastore.Key) error
	Write(ctx context.Context) error
	Delete(ctx context.Context) error
}

func GetEntityKey(entity PersistenceEntity) datastore.Key {
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
