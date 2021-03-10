package memorystore

import (
	"0chain.net/core/datastore"
)

/*GetEntityKey = entity name + entity id */
func GetEntityKey(entity datastore.Entity) datastore.Key {
	key := entity.GetKey()
	emd := entity.GetEntityMetadata()
	return datastore.ToKey(emd.GetName() + ":" + key)
}
