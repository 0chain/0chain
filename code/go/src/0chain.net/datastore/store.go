package datastore

import (
	"context"
)

type Store interface {
	Read(ctx context.Context, key Key, entity Entity) error
	Write(ctx context.Context, entity Entity) error
	InsertIfNE(ctx context.Context, entity Entity) error
	Delete(ctx context.Context, entity Entity) error

	MultiRead(ctx context.Context, entityMetadata EntityMetadata, keys []Key, entities []Entity) error
	MultiWrite(ctx context.Context, entityMetadata EntityMetadata, entities []Entity) error
	MultiDelete(ctx context.Context, entityMetadata EntityMetadata, entities []Entity) error

	AddToCollection(ctx context.Context, entity CollectionEntity) error
	MultiAddToCollection(ctx context.Context, entityMetadata EntityMetadata, entities []Entity) error

	DeleteFromCollection(ctx context.Context, entity CollectionEntity) error
	MultiDeleteFromCollection(ctx context.Context, entityMetadata EntityMetadata, entities []Entity) error

	GetCollectionSize(ctx context.Context, entityMetadata EntityMetadata, collectionName string) int64
	IterateCollection(ctx context.Context, entityMetadata EntityMetadata, collectionName string, handler CollectionIteratorHandler) error
}
