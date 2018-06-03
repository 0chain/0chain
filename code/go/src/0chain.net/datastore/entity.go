package datastore

import (
	"context"
)

var (
	/*EntityNotFound code should be used to check whether an entity is found or not */
	EntityNotFound = "entity_not_found"
	/*EntityDuplicate codee should be used to check if an entity is already present */
	EntityDuplicate = "duplicate_entity"
)

/*Entity - interface that reads and writes any implementing structure as JSON into the store */
type Entity interface {
	GetEntityName() string
	GetEntityMetadata() EntityMetadata
	SetKey(key Key)
	GetKey() Key
	ComputeProperties()
	Validate(ctx context.Context) error
}

func AllocateEntities(size int, entityMetadata EntityMetadata) []Entity {
	entities := make([]Entity, size)
	for i := 0; i < size; i++ {
		entities[i] = entityMetadata.Instance()
	}
	return entities
}
