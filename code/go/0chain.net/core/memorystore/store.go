package memorystore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

/*BATCH_SIZE size of the batch */
const BATCH_SIZE = 256

var storageAPI = &Store{}

/*GetStorageProvider - get the storage provider for the memorystore */
func GetStorageProvider() datastore.Store {
	return storageAPI
}

/*Store - just a struct to implement the datastore.Store interface */
type Store struct {
}

/*Read an entity from the datastore by providing the key */
func (ms *Store) Read(ctx context.Context, key datastore.Key, entity datastore.Entity) error {
	entity.SetKey(key)
	redisKey := GetEntityKey(entity)
	emd := entity.GetEntityMetadata()
	c := GetEntityCon(ctx, emd)
	c.Send("GET", redisKey)
	c.Flush()
	data, err := c.Receive()
	if err != nil {
		return err
	}

	if data == nil {
		return common.NewError(datastore.EntityNotFound, fmt.Sprintf("%v not found with id = %v", emd.GetName(), redisKey))
	}
	datastore.FromJSON(data, entity)
	entity.ComputeProperties()
	return nil
}

/*Write an entity to the datastore */
func (ms *Store) Write(ctx context.Context, entity datastore.Entity) error {
	return writeAux(ctx, entity, true)
}

func writeAux(ctx context.Context, entity datastore.Entity, overwrite bool) error {
	buffer := datastore.ToJSON(entity)
	redisKey := GetEntityKey(entity)
	emd := entity.GetEntityMetadata()
	c := GetEntityCon(ctx, emd)
	if overwrite {
		c.Send("SET", redisKey, buffer)
	} else {
		c.Send("SETNX", redisKey, buffer)
	}
	c.Flush()
	data, err := c.Receive()
	if err != nil {
		return err
	}
	if val, ok := data.(int64); ok && val == 0 {
		return common.NewError("duplicate_entity", fmt.Sprintf("%v with key %v already exists", emd.GetName(), entity.GetKey()))
	}
	ce, ok := entity.(datastore.CollectionEntity)
	if !ok {
		return nil
	}
	if ce.GetCollectionScore() == 0 {
		if entity.GetScore() != 0 {
			ce.SetCollectionScore(entity.GetScore())
		} else {
			ce.InitCollectionScore()
		}
	}
	err = datastore.AddToCollection(ce, ctx)
	return err
}

/*InsertIfNE - insert an entity only if it doesn't already exist in the datastore */
func (ms *Store) InsertIfNE(ctx context.Context, entity datastore.Entity) error {
	return writeAux(ctx, entity, false)
}

/*Delete an entity from the datastore
*  Given an entity id, the pattern is as follows
* entity.SetKey(id)
* memorydatastore.Delete(ctx,entity)
 */
func (ms *Store) Delete(ctx context.Context, entity datastore.Entity) error {
	redisKey := GetEntityKey(entity)
	c := GetEntityCon(ctx, entity.GetEntityMetadata())
	c.Send("DEL", redisKey)
	c.Flush()
	_, err := c.Receive()
	if err != nil {
		return err
	}
	if ce, ok := entity.(datastore.CollectionEntity); ok {
		return ms.DeleteFromCollection(ctx, ce)
	}
	return nil
}

/*MultiRead - allows reading multiple entities at the same time */
func (ms *Store) MultiRead(ctx context.Context, entityMetadata datastore.EntityMetadata, keys []datastore.Key, entities []datastore.Entity) error {
	if len(entities) <= BATCH_SIZE {
		return ms.multiReadAux(ctx, entityMetadata, keys, entities)
	}
	for start := 0; start < len(entities); start += BATCH_SIZE {
		end := start + BATCH_SIZE
		if end > len(entities) {
			end = len(entities)
		}
		err := ms.multiReadAux(ctx, entityMetadata, keys[start:end], entities[start:end])
		if err != nil {
			return err
		}
	}
	return nil
}

func (ms *Store) multiReadAux(ctx context.Context, entityMetadata datastore.EntityMetadata, keys []datastore.Key, entities []datastore.Entity) error {
	rkeys := make([]interface{}, len(keys))
	for idx, key := range keys {
		entity := entities[idx]
		entity.SetKey(datastore.ToKey(key))
		rkeys[idx] = GetEntityKey(entity)
	}
	c := GetEntityCon(ctx, entityMetadata)
	c.Send("MGET", rkeys...)
	c.Flush()
	data, err := c.Receive()
	if err != nil {
		return err
	}
	array, ok := data.([]interface{})
	if !ok {
		return fmt.Errorf("not a valid entity json: (%v)", rkeys)
	}
	for idx, ae := range array {
		if ae == nil {
			/* not setting this to nil so it's possible to reuse the same array used for block processing
			instead setting key to EmptyKey
			*/
			entities[idx].SetKey(datastore.EmptyKey)
			continue
		}
		entity := entities[idx]
		err = json.Unmarshal(ae.([]byte), entity)
		if err != nil {
			return err
		}
		entity.ComputeProperties()
	}
	return nil
}

/*MultiWrite allows writing multiple entities to the memorydatastore
* If the entities belong to a collection, then all entities should belong to
* the same collection (including partitioning)
 */
func (ms *Store) MultiWrite(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	if len(entities) <= BATCH_SIZE {
		return ms.multiWriteAux(ctx, entityMetadata, entities)
	}
	for start := 0; start < len(entities); start += BATCH_SIZE {
		end := start + BATCH_SIZE
		if end > len(entities) {
			end = len(entities)
		}
		err := ms.multiWriteAux(ctx, entityMetadata, entities[start:end])
		if err != nil {
			return err
		}
	}
	return nil
}
func (ms *Store) multiWriteAux(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	kvpair := make([]interface{}, 2*len(entities))
	hasCollectionEntity := false
	for idx, entity := range entities {
		if !hasCollectionEntity {
			_, hasCollectionEntity = entity.(datastore.CollectionEntity)
		}
		kvpair[2*idx] = GetEntityKey(entity)
		kvpair[2*idx+1] = bytes.NewBuffer(make([]byte, 0, 256))
		json.NewEncoder(kvpair[2*idx+1].(*bytes.Buffer)).Encode(entity)
	}
	c := GetEntityCon(ctx, entityMetadata)
	c.Send("MSET", kvpair...)
	c.Flush()
	_, err := c.Receive()
	if err != nil {
		return err
	}
	if hasCollectionEntity {
		err = ms.MultiAddToCollection(ctx, entityMetadata, entities)
	}
	return err
}

/*AddToCollection - default implementation for CollectionEntity interface */
func (ms *Store) AddToCollection(ctx context.Context, ce datastore.CollectionEntity) error {
	entityMetadata := ce.GetEntityMetadata()
	collectionName := ce.GetCollectionName()

	con := GetEntityCon(ctx, entityMetadata)
	con.Send("ZADD", collectionName, ce.GetCollectionScore(), ce.GetKey())
	con.Flush()
	_, err := con.Receive()
	if err != nil {
		return err
	}
	return nil
}

/*MultiAddToCollection adds multiple entities to a collection */
func (ms *Store) MultiAddToCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	if len(entities) <= BATCH_SIZE {
		return ms.multiAddToCollectionAux(ctx, entityMetadata, entities)
	}
	for start := 0; start < len(entities); start += BATCH_SIZE {
		end := start + BATCH_SIZE
		if end > len(entities) {
			end = len(entities)
		}
		err := ms.multiAddToCollectionAux(ctx, entityMetadata, entities[start:end])
		if err != nil {
			return err
		}
	}
	return nil
}

func (ms *Store) multiAddToCollectionAux(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	// Assuming all entities belong to the same collection.
	if len(entities) == 0 {
		return nil
	}
	svpair := make([]interface{}, 1+2*len(entities))
	ce := entities[0].(datastore.CollectionEntity)
	trackCollection(entityMetadata, ce)
	svpair[0] = ce.GetCollectionName()
	offset := 1
	for idx, entity := range entities {
		ce, ok := entity.(datastore.CollectionEntity)
		if !ok {
			return common.NewError("dev_error", "Entity needs to be CollectionEntity")
		}
		ind := offset + 2*idx
		score := ce.GetCollectionScore()
		if score == 0 {
			if entity.GetScore() == 0 {
				ce.InitCollectionScore()
			} else {
				ce.SetCollectionScore(entity.GetScore())
			}
		}
		svpair[ind] = ce.GetCollectionScore()
		svpair[ind+1] = ce.GetKey()
	}
	con := GetEntityCon(ctx, entityMetadata)
	con.Send("ZADD", svpair...)
	con.Flush()
	_, err := con.Receive()
	return err
}

/*MultiDelete - delete multiple entities from the store */
func (ms *Store) MultiDelete(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	if len(entities) <= BATCH_SIZE {
		return ms.multiDeleteAux(ctx, entityMetadata, entities)
	}
	for start := 0; start < len(entities); start += BATCH_SIZE {
		end := start + BATCH_SIZE
		if end > len(entities) {
			end = len(entities)
		}
		err := ms.multiDeleteAux(ctx, entityMetadata, entities[start:end])
		if err != nil {
			return err
		}
	}
	return nil
}

func (ms *Store) multiDeleteAux(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	rkeys := make([]interface{}, len(entities))

	hasCollectionEntity := false
	for idx, entity := range entities {
		rkeys[idx] = GetEntityKey(entity)
		if !hasCollectionEntity {
			_, hasCollectionEntity = entity.(datastore.CollectionEntity)
		}
	}
	c := GetEntityCon(ctx, entityMetadata)
	c.Send("DEL", rkeys...)
	c.Flush()
	_, err := c.Receive()
	if err != nil {
		return err
	}
	if hasCollectionEntity {
		ms.MultiDeleteFromCollection(ctx, entityMetadata, entities)
	}
	return nil
}

func (ms *Store) DeleteFromCollection(ctx context.Context, ce datastore.CollectionEntity) error {
	entityMetadata := ce.GetEntityMetadata()
	collectionName := ce.GetCollectionName()

	con := GetEntityCon(ctx, entityMetadata)
	con.Send("ZREM", collectionName, ce.GetKey())
	con.Flush()
	_, err := con.Receive()
	if err != nil {
		return err
	}
	return nil
}

func (ms *Store) MultiDeleteFromCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	if len(entities) <= BATCH_SIZE {
		return ms.multiDeleteFromCollectionAux(ctx, entityMetadata, entities)
	}
	for start := 0; start < len(entities); start += BATCH_SIZE {
		end := start + BATCH_SIZE
		if end > len(entities) {
			end = len(entities)
		}
		err := ms.multiDeleteFromCollectionAux(ctx, entityMetadata, entities[start:end])
		if err != nil {
			return err
		}
	}
	return nil
}

func (ms *Store) multiDeleteFromCollectionAux(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	// Assuming all entities belong to the same collection.
	if len(entities) == 0 {
		return nil
	}
	keys := make([]interface{}, 1+len(entities))
	ce := entities[0].(datastore.CollectionEntity)
	keys[0] = ce.GetCollectionName()
	for idx, entity := range entities {
		ce, ok := entity.(datastore.CollectionEntity)
		if !ok {
			return common.NewError("dev_error", "Entity needs to be CollectionEntity")
		}
		keys[idx+1] = ce.GetKey()
	}
	con := GetEntityCon(ctx, entityMetadata)
	con.Send("ZREM", keys...)
	con.Flush()
	_, err := con.Receive()
	return err
}

func (ms *Store) GetCollectionSize(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string) int64 {
	con := GetEntityCon(ctx, entityMetadata)
	con.Send("ZCARD", collectionName)
	con.Flush()
	data, err := con.Receive()
	if err != nil {
		return -1
	} else {
		val, ok := data.(int64)
		if !ok {
			return -1
		}
		return val
	}
}
