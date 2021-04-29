package datastore

import (
	"sync"
)

type (
	// entityMetadataStore describes the structure of entity metadata storage
	// it is safe for concurrent use by multiple read-write operation access.
	entityMetadataStore struct {
		store map[string]EntityMetadata // entity metadata storage
		mutex sync.Mutex                // read-write mutex
	}
)

var (
	// entityMetadataMap keeps instance of entity metadata store.
	entityMetadataMap = entityMetadataStore{
		store: make(map[string]EntityMetadata),
	}
)

// RegisterEntityMetadata registers an instance of the entity
// in track of a list of entity providers by given name.
// An entity can be registered with multiple names
// as long as two entities don't use the same name.
func RegisterEntityMetadata(entityName string, entityMetadata EntityMetadata) {
	entityMetadataMap.mutex.Lock()
	entityMetadataMap.store[entityName] = entityMetadata
	entityMetadataMap.mutex.Unlock()
}

// GetEntityMetadata returns an instance of the entity by name.
func GetEntityMetadata(entityName string) EntityMetadata {
	entityMetadataMap.mutex.Lock()
	em := entityMetadataMap.store[entityName]
	entityMetadataMap.mutex.Unlock()

	return em
}

// GetEntity returns an instance of the entity.
func GetEntity(entityName string) Entity {
	return GetEntityMetadata(entityName).Instance()
}
