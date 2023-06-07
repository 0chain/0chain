package ememorystore

import (
	"context"
	"encoding/binary"
	"strconv"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/linxGnu/grocksdb"
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
	var data *grocksdb.Slice
	var err error
	if emd.GetName() == "round" {
		rNumber, err := strconv.ParseInt(datastore.ToString(entity.GetKey()), 10, 64)
		if err != nil {
			return err
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(rNumber))
		data, err = c.Conn.Get(c.ReadOptions, key)
		if err != nil {
			return err
		}
	} else {
		data, err = c.Conn.Get(c.ReadOptions, []byte(key))
		if err != nil {
			return err
		}
	}
	defer data.Free()
	if emd.GetName() == "block" {
		return datastore.FromJSON(data.Data(), entity, false)
	}
	return datastore.FromJSON(data.Data(), entity)
}

func (ems *Store) Write(ctx context.Context, entity datastore.Entity) error {
	emd := entity.GetEntityMetadata()
	c := GetEntityCon(ctx, emd)
	data := datastore.ToJSON(entity).Bytes()
	if emd.GetName() == "round" {
		rNumber, err := strconv.ParseInt(datastore.ToString(entity.GetKey()), 10, 64)
		if err != nil {
			return err
		}
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(rNumber))
		if err := c.Conn.Put(key, data); err != nil {
			return err
		}
	} else {
		if err := c.Conn.Put([]byte(datastore.ToString(entity.GetKey())), data); err != nil {
			return err
		}
	}
	return nil
}

func (ems *Store) InsertIfNE(ctx context.Context, entity datastore.Entity) error {
	emd := entity.GetEntityMetadata()
	c := GetEntityCon(ctx, emd)
	_, err := c.Conn.Get(c.ReadOptions, []byte(datastore.ToString(entity.GetKey())))
	if err == nil {
		return common.NewError("entity_already_exists", "Entity already exists")
	}
	return ems.Write(ctx, entity)
}

func (ems *Store) Delete(ctx context.Context, entity datastore.Entity) error {
	emd := entity.GetEntityMetadata()
	c := GetEntityCon(ctx, emd)
	return c.Conn.Delete([]byte(datastore.ToString(entity.GetKey())))
}

func (ems *Store) MultiRead(ctx context.Context, entityMetadata datastore.EntityMetadata, keys []datastore.Key, entities []datastore.Entity) error {
	//TODO: even though rocksdb has MultiGet api, grocksdb doesn't seem to have one
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

// func (ems *Store) WBWrite(ctx context.Context, emd datastore.EntityMetadata, batch *AtomicWriteBatch) error {
// 	// Build []byte key and value
// 	c := GetEntityCon(ctx, emd)
// 	err :=
// }

func (ems *Store) Merge(ctx context.Context, entity datastore.Entity) error {
	c := GetEntityCon(ctx, entity.GetEntityMetadata())
	data := datastore.ToJSON(entity).Bytes()
	return c.Conn.Merge([]byte(datastore.ToString(entity.GetKey())), data)
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
