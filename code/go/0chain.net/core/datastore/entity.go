package datastore

import (
	"context"
	"fmt"

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
	SetKey(key Key)
	GetKey() Key
	GetScore() int64
	ComputeProperties()
	Validate(ctx context.Context) error
	Read(ctx context.Context, key Key) error
	Write(ctx context.Context) error
	Delete(ctx context.Context) error
	SetCacheData(data []byte, codec int, compress bool)
	GetCachedData(codec int, compress bool) []byte
}

//AllocateEntities - allocate entities for the given entity type
func AllocateEntities(size int, entityMetadata EntityMetadata) []Entity {
	entities := make([]Entity, size)
	for i := 0; i < size; i++ {
		entities[i] = entityMetadata.Instance()
	}
	return entities
}

// EncodedDataCache caches the encoded and compressed(if any) data for Entity
// that will be sent out to avoid doing duplicates encoding and compress actions
// for the same entity.
type EncodedDataCache struct {
	data map[string][]byte
}

// SetCacheData saves the data to cache
func (edc *EncodedDataCache) SetCacheData(data []byte, codec int, compress bool) {
	if edc.data == nil {
		edc.data = make(map[string][]byte)
	}

	key := fmt.Sprintf("%d:%v", codec, compress)
	edc.data[key] = data
}

// GetCachedData returns the cached data
func (edc *EncodedDataCache) GetCachedData(codec int, compress bool) []byte {
	key := fmt.Sprintf("%d:%v", codec, compress)
	return edc.data[key]
}
