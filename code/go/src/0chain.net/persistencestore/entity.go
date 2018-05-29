package persistencestore

import (
	"context"

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

/*PersistenceEntity - Persistence Entity */
type PersistenceEntity interface {
	datastore.Entity
	PRead(ctx context.Context, key datastore.Key) error
	PWrite(ctx context.Context) error
	PDelete(ctx context.Context) error
}
