package memorystore

import (
	"fmt"

	"0chain.net/datastore"
)

/*GetEntityKey = entity name + entity id */
func GetEntityKey(entity datastore.Entity) datastore.Key {
	var key interface{} = entity.GetKey()
	emd := entity.GetEntityMetadata()
	switch v := key.(type) {
	case string:
		return datastore.ToKey(fmt.Sprintf("%s:%v", emd.GetName(), v))
	case []byte:
		return datastore.ToKey(append(append([]byte(emd.GetName()), ':'), v...))
	default:
		return datastore.EmptyKey
	}
}
