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

	GetRangeFromCollection(ctx context.Context, entity Entity, entities []Entity, byScore, withScores bool, min, max string, offset, count int64) error

	HGet(ctx context.Context, entity Entity, hashTableName string, key Key) (string, error)
	HSet(ctx context.Context, entity Entity, hashTableName string, key, val Key) error
	HDel(ctx context.Context, entity Entity, hashTableName string, key Key) error

	StartTx(context.Context, Entity) context.Context
	SendTX(context.Context, Entity) error
}
