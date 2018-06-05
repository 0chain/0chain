package persistencestore

import (
	"context"
	"fmt"

	"0chain.net/common"
	"0chain.net/datastore"
)

var storageAPI = &Store{}

/*GetStorageProvider - get the storage provider for the memorystore */
func GetStorageProvider() datastore.Store {
	return storageAPI
}

/*Store - just a struct to implement the datastore.Store interface */
type Store struct {
}

func getJSONSelect(table string) string {
	return fmt.Sprintf("SELECT JSON * FROM %v", table)
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
	entity.SetKey(key)
	c := GetCon(ctx)
	emd := entity.GetEntityMetadata()
	iter := c.Query(getJSONSelect(emd.GetName())).Iter()
	var json string
	valid := iter.Scan(&json)
	if !valid {
		return common.NewError(datastore.EntityNotFound, fmt.Sprintf("%v not found with id = %v", emd.GetName(), key))
	}
	datastore.FromJSON(json, entity)
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
	//TODO:
	return nil
}

/*MultiWrite - Write multiple entities to the store */
func (ps *Store) MultiWrite(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	//TODO:
	return nil
}

/*AddToCollection - Add to collection */
func (ps *Store) AddToCollection(ctx context.Context, entity datastore.CollectionEntity) error {
	// This may be NOOP for persistence stores
	return nil
}

/*IterateCollection - iterate the given collection */
func (ps *Store) IterateCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string, handler datastore.CollectionIteratorHandler) error {
	// This may not be the righ API for filtered queries
	return nil
}
