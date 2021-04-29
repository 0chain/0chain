package datastore

import (
	"sync"
)

type (
	// InstanceProvider describes the instance function signature.
	InstanceProvider func() Entity

	// entityMetadataStore describes the structure of entity metadata storage
	// it is safe for concurrent use by multiple read-write operation access.
	entityMetadataStore struct {
		store map[string]EntityMetadata // entity metadata storage
		mutex sync.Mutex                // read-write mutex
	}
)

var (
	// entityMetadataMap keeps instance of entity metadata storage.
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

// EntityMetadata describes the interface of the metadata entity.
type EntityMetadata interface {
	GetName() string
	GetDB() string
	Instance() Entity
	GetStore() Store
	GetIDColumnName() string
}

// EntityMetadataImpl implements EntityMetadata interface.
type EntityMetadataImpl struct {
	Name         string
	DB           string
	Store        Store
	Provider     InstanceProvider
	IDColumnName string
}

// MetadataProvider constructs entity metadata instance.
func MetadataProvider() *EntityMetadataImpl {
	em := EntityMetadataImpl{IDColumnName: "id"}
	return &em
}

// GetName implements EntityMetadataImpl.GetName method of interface.
func (em *EntityMetadataImpl) GetName() string {
	return em.Name
}

// GetDB implements EntityMetadataImpl.GetDB method of interface.
func (em *EntityMetadataImpl) GetDB() string {
	return em.DB
}

// Instance implements EntityMetadataImpl.Instance method of interface.
func (em *EntityMetadataImpl) Instance() Entity {
	return em.Provider()
}

// GetStore implements EntityMetadataImpl.GetStore method of interface.
func (em *EntityMetadataImpl) GetStore() Store {
	return em.Store
}

// GetIDColumnName implements EntityMetadataImpl.GetIDColumnName method of interface.
func (em *EntityMetadataImpl) GetIDColumnName() string {
	return em.IDColumnName
}
