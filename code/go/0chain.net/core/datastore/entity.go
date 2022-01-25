package datastore

import (
	"context"

	"0chain.net/core/common"
)

var ErrInvalidEntity = common.NewError("invalid_entity", "invalid entity")

var (
	/*EntityNotFound code should be used to check whether an entity is found or not */
	EntityNotFound = "entity_not_found"
	/*EntityDuplicate code should be used to check if an entity is already present */
	EntityDuplicate = "duplicate_entity"
)

/*Entity - interface that reads and writes any implementing structure as JSON into the store */
type Entity interface {
	GetEntityMetadata() EntityMetadata
	GetVersion() string
	SetVersion(v string)
	SetKey(key Key)
	GetKey() Key
	GetScore() int64
	ComputeProperties()
	Validate(ctx context.Context) error
	Read(ctx context.Context, key Key) error
	Write(ctx context.Context) error
	Delete(ctx context.Context) error

	ProtoConverter
}

// ProtoConverter wraps the methods to convert the latest
// protocol format to previous version
type ProtoConverter interface {
	ToPreviousVersion(origin Entity) Entity
}

//// PreviousDecoder wraps the methods to decode data into
//// previous protocol
//type PreviousDecoder interface {
//	PreviousProtoDecode([]byte) (Entity, error)
//}

//AllocateEntities - allocate entities for the given entity type
func AllocateEntities(size int, entityMetadata EntityMetadata) []Entity {
	entities := make([]Entity, size)
	for i := 0; i < size; i++ {
		entities[i] = entityMetadata.Instance()
	}
	return entities
}
