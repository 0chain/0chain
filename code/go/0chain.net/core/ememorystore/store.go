package ememorystore

import (
	"context"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

var storageAPI = &Store{}

/*GetStorageProvider - get the storage provider for the memorystore */
func GetStorageProvider() datastore.Store {
	return storageAPI
}

/*Store - just a struct to implement the datastore.Store interface */
type Store struct {
}

func (ems *Store) Read(ctx context.Context, key datastore.Key, entity datastore.Entity) error {
	entity.SetKey(key)
	emd := entity.GetEntityMetadata()
	c := GetEntityCon(ctx, emd)
	data, err := c.Conn.Get(c.ReadOptions, []byte(datastore.ToString(key)))
	if err != nil {
		return err
	}
	defer data.Free()
	err = datastore.FromJSON(data.Data(), entity)
	if err != nil {
		return err
	}
	return nil
}

func (ems *Store) Write(ctx context.Context, entity datastore.Entity) error {
	emd := entity.GetEntityMetadata()
	c := GetEntityCon(ctx, emd)
	data := datastore.ToJSON(entity).Bytes()
	err := c.Conn.Put([]byte(datastore.ToString(entity.GetKey())), data)
	if err != nil {
		return err
	}
	return nil
}

func (ems *Store) InsertIfNE(ctx context.Context, entity datastore.Entity) error {
	emd := entity.GetEntityMetadata()
	c := GetEntityCon(ctx, emd)
	_, err := c.Conn.Get(c.ReadOptions, []byte(datastore.ToString(entity.GetKey())))
	if err != nil {

	} else {
		return common.NewError("entity_already_exists", "Entity already exists")
	}
	ems.Write(ctx, entity)
	return nil
}

func (ems *Store) Delete(ctx context.Context, entity datastore.Entity) error {
	emd := entity.GetEntityMetadata()
	c := GetEntityCon(ctx, emd)
	return c.Conn.Delete([]byte(datastore.ToString(entity.GetKey())))
}

func (ems *Store) MultiRead(ctx context.Context, entityMetadata datastore.EntityMetadata, keys []datastore.Key, entities []datastore.Entity) error {
	//TODO: even though rocksdb has MultiGet api, gorocksdb doesn't seem to have one
	for idx, key := range keys {
		err := ems.Read(ctx, key, entities[idx])
		if err != nil {
			entities[idx].SetKey(datastore.EmptyKey)
		}
	}
	return nil
}

func (ems *Store) MultiWrite(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	c := GetEntityCon(ctx, entityMetadata)
	for _, entity := range entities {
		data := datastore.ToJSON(entity).Bytes()
		err := c.Conn.Put([]byte(datastore.ToString(entity.GetKey())), data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ems *Store) MultiDelete(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	c := GetEntityCon(ctx, entityMetadata)
	for _, entity := range entities {
		err := c.Conn.Delete([]byte(datastore.ToString(entity.GetKey())))
		if err != nil {
			return err
		}
	}
	return nil
}

func (ems *Store) AddToCollection(ctx context.Context, entity datastore.CollectionEntity) error {
	return nil
}

func (ems *Store) MultiAddToCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	return nil
}

func (ems *Store) DeleteFromCollection(ctx context.Context, entity datastore.CollectionEntity) error {
	return nil
}

func (ems *Store) MultiDeleteFromCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	return nil
}

func (ems *Store) GetCollectionSize(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string) int64 {
	return -1
}

func (ems *Store) IterateCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string, handler datastore.CollectionIteratorHandler) error {
	return nil
}
