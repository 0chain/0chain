package persistencestore

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
)

/*BATCH_SIZE size of the batch */
const BATCH_SIZE = 16

var storageAPI = &Store{}

/*GetStorageProvider - get the storage provider for the memorystore */
func GetStorageProvider() datastore.Store {
	return storageAPI
}

/*Store - just a struct to implement the datastore.Store interface */
type Store struct {
}

func getJSONSelect(table string, primaryKey string) string {
	return fmt.Sprintf("SELECT JSON * FROM %v where %v = ?", table, primaryKey)
}

func getJSONSelectN(table string, primaryKey string, n int) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("SELECT JSON * FROM %v where %v in (?", table, primaryKey))
	for i := 1; i < n; i++ {
		buf.WriteString(",?")
	}
	buf.WriteString(")")
	return buf.String()
}

func getJSONInsert(table string) string {
	return fmt.Sprintf("INSERT INTO %v JSON ?", table)
}

func getJSONInsertIfNE(table string) string {
	return fmt.Sprintf("INSERT INTO %v JSON ? IF NOT EXISTS", table)
}

func getDeleteStmt(table string, primaryKey string) string {
	return fmt.Sprintf("DELETE FROM %v where %v = ?", table, primaryKey)
}

/*Read - read an entity from the store */
func (ps *Store) Read(ctx context.Context, key datastore.Key, entity datastore.Entity) error {
	c := GetCon(ctx)
	emd := entity.GetEntityMetadata()
	iter := c.Query(getJSONSelect(emd.GetName(), emd.GetIDColumnName()), key).Iter()
	var json string
	valid := iter.Scan(&json)
	if !valid {
		return common.NewError(datastore.EntityNotFound, fmt.Sprintf("%v not found with id = %v", emd.GetName(), key))
	}

	if err := datastore.FromJSON(json, entity); err != nil {
		return err
	}
	if err := iter.Close(); err != nil {
		return err
	}
	return nil
}

/*Write - write an entity to the store */
func (ps *Store) Write(ctx context.Context, entity datastore.Entity) error {
	c := GetCon(ctx)
	emd := entity.GetEntityMetadata()
	json := datastore.ToJSON(entity).String()
	err := c.Query(getJSONInsert(emd.GetName()), json).Exec()
	return err
}

/*InsertIfNE - insert an entity to the store if it doesn't exist */
func (ps *Store) InsertIfNE(ctx context.Context, entity datastore.Entity) error {
	c := GetCon(ctx)
	emd := entity.GetEntityMetadata()
	json := datastore.ToJSON(entity).String()
	err := c.Query(getJSONInsertIfNE(emd.GetName()), json).Exec()
	return err
}

/*Delete - Delete an entity from the store */
func (ps *Store) Delete(ctx context.Context, entity datastore.Entity) error {
	c := GetCon(ctx)
	emd := entity.GetEntityMetadata()
	err := c.Query(getDeleteStmt(emd.GetName(), emd.GetIDColumnName()), entity.GetKey()).Exec()
	return err
}

/*MultiRead - read multiple entities from the store */
func (ps *Store) MultiRead(ctx context.Context, entityMetadata datastore.EntityMetadata, keys []datastore.Key, entities []datastore.Entity) error {
	if len(entities) <= BATCH_SIZE {
		return ps.multiReadAux(ctx, entityMetadata, keys, entities)
	}
	for start := 0; start < len(entities); start += BATCH_SIZE {
		end := start + BATCH_SIZE
		if end > len(entities) {
			end = len(entities)
		}
		err := ps.multiReadAux(ctx, entityMetadata, keys[start:end], entities[start:end])
		if err != nil {
			return err
		}
	}
	return nil
}

func (ps *Store) multiReadAux(ctx context.Context, entityMetadata datastore.EntityMetadata, keys []datastore.Key, entities []datastore.Entity) error {
	ikeys := make([]interface{}, len(keys))
	for idx, key := range keys {
		ikeys[idx] = key
		entities[idx].SetKey(datastore.EmptyKey)
	}
	c := GetCon(ctx)
	iter := c.Query(getJSONSelectN(entityMetadata.GetName(), entityMetadata.GetIDColumnName(), len(keys)), ikeys...).Iter()
	var json string
	keyIdx := make(map[datastore.Key]datastore.Entity)
	for i := 0; i < len(keys); i++ {
		valid := iter.Scan(&json)
		if !valid {
			break
			//return common.NewError("not_all_keys_found", "Did not find entities for all the keys")
		}
		if err := datastore.FromJSON(json, entities[i]); err != nil {
			logging.Logger.Warn("decode entity failed", zap.Error(err))
		}
		keyIdx[entities[i].GetKey()] = entities[i]
	}
	if err := iter.Close(); err != nil {
		return err
	}
	// We may have gotten fewer rows , so we are mapping to the same sequences as keys
	for idx, key := range keys {
		entities[idx] = keyIdx[key]
	}
	return nil
}

/*MultiWrite - Write multiple entities to the store */
func (ps *Store) MultiWrite(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	if len(entities) <= BATCH_SIZE {
		return ps.multiWriteAux(ctx, entityMetadata, entities)
	}
	for start := 0; start < len(entities); start += BATCH_SIZE {
		end := start + BATCH_SIZE
		if end > len(entities) {
			end = len(entities)
		}
		err := ps.multiWriteAux(ctx, entityMetadata, entities[start:end])
		if err != nil {
			return err
		}
	}
	return nil
}

func (ps *Store) multiWriteAux(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	c := GetCon(ctx)
	sql := getJSONInsert(entityMetadata.GetName())
	batch := c.NewBatch(gocql.LoggedBatch)
	for _, entity := range entities {
		batch.Query(sql, datastore.ToJSON(entity).String())
	}
	err := c.ExecuteBatch(batch)
	if err != nil {
		ps.shouldReconnect(err, entityMetadata)
	}
	return err
}

/*MultiDelete - delete multiple entities from the store */
func (ps *Store) MultiDelete(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	// TODO
	return nil
}

/*AddToCollection - Add to collection */
func (ps *Store) AddToCollection(ctx context.Context, entity datastore.CollectionEntity) error {
	// This may be NOOP for persistence stores
	return nil
}

/*MultiAddToCollection - Add multiple entities to collection */
func (ps *Store) MultiAddToCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	return nil
}

/*IterateCollection - iterate the given collection */
func (ps *Store) IterateCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string, handler datastore.CollectionIteratorHandler) error {
	// This may not be the righ API for filtered queries
	return nil
}

func (ps *Store) DeleteFromCollection(ctx context.Context, entity datastore.CollectionEntity) error {
	return nil
}

func (ps *Store) MultiDeleteFromCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	return nil
}

func (ps *Store) GetCollectionSize(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string) int64 {
	return -1
}

func (ps *Store) shouldReconnect(err error, enittyMetadata datastore.EntityMetadata) bool {
	switch err {
	case gocql.ErrNoConnections:
		if err := initSession(1*time.Second, 0); err != nil {
			logging.Logger.Error("init session failed", zap.Error(err))
		}
		return true
	}
	return false
}
