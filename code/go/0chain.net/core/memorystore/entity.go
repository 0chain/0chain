package memorystore

import (
	"0chain.net/core/datastore"
)

/*GetEntityKey = entity name + entity id */
func GetEntityKey(entity datastore.Entity) datastore.Key {
	var key interface{} = entity.GetKey()
	emd := entity.GetEntityMetadata()
	switch v := key.(type) {
	case string:
		return datastore.ToKey(emd.GetName() + ":" + v)
	case []byte:
		return datastore.ToKey(append(append([]byte(emd.GetName()), ':'), v...))
	default:
		return datastore.EmptyKey
	}
}
