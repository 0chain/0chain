package memorystore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"0chain.net/common"
	"0chain.net/datastore"
)

type MemoryEntity interface {
	datastore.Entity
	Read(ctx context.Context, key datastore.Key) error
	Write(ctx context.Context) error
	Delete(ctx context.Context) error
}

/*GetEntityKey = entity name + entity id */
func GetEntityKey(entity MemoryEntity) datastore.Key {
	var key interface{} = entity.GetKey()
	switch v := key.(type) {
	case string:
		return datastore.ToKey(fmt.Sprintf("%s:%v", entity.GetEntityName(), v))
	case []byte:
		return datastore.ToKey(append(append([]byte(entity.GetEntityName()), ':'), v...))
	default:
		return datastore.EmptyKey
	}
}

/*Read an entity from the datastore by providing the key */
func Read(ctx context.Context, key datastore.Key, entity MemoryEntity) error {
	entity.SetKey(key)
	redisKey := GetEntityKey(entity)
	c := GetEntityCon(ctx, entity.GetEntityMetadata())
	c.Send("GET", redisKey)
	c.Flush()
	data, err := c.Receive()
	if err != nil {
		return err
	}

	if data == nil {
		return common.NewError(datastore.EntityNotFound, fmt.Sprintf("%v not found with id = %v", entity.GetEntityName(), redisKey))
	}
	datastore.FromJSON(data, entity)
	entity.ComputeProperties()
	return nil
}

/*Write an entity to the datastore */
func Write(ctx context.Context, entity MemoryEntity) error {
	return writeAux(ctx, entity, true)
}

func writeAux(ctx context.Context, entity MemoryEntity, overwrite bool) error {
	buffer := datastore.ToJSON(entity)
	redisKey := GetEntityKey(entity)
	c := GetEntityCon(ctx, entity.GetEntityMetadata())
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
		return common.NewError("duplicate_entity", fmt.Sprintf("%v with key %v already exists", entity.GetEntityName(), entity.GetKey()))
	}
	ce, ok := entity.(CollectionEntity)
	if !ok {
		return nil
	}
	if ce.GetCollectionScore() == 0 {
		ce.InitCollectionScore()
	}
	err = ce.AddToCollection(ctx, entity.GetEntityMetadata(), ce.GetCollectionName())
	return err
}

/*InsertIfNE - insert an entity only if it doesn't already exist in the datastore */
func InsertIfNE(ctx context.Context, entity MemoryEntity) error {
	return writeAux(ctx, entity, false)
}

/*Delete an entity from the datastore
*  Given an entity id, the pattern is as follows
* entity.SetKey(id)
* memorydatastore.Delete(ctx,entity)
 */
func Delete(ctx context.Context, entity MemoryEntity) error {
	redisKey := GetEntityKey(entity)
	c := GetEntityCon(ctx, entity.GetEntityMetadata())
	c.Send("DEL", redisKey)
	c.Flush()
	_, err := c.Receive()
	return err
}

func AllocateEntities(size int, entityMetadata datastore.EntityMetadata) ([]MemoryEntity, error) {
	entities := make([]MemoryEntity, size)
	for i := 0; i < size; i++ {
		entity, ok := entityMetadata.Instance().(MemoryEntity)
		if !ok {
			return nil, common.NewError("invalid_entity_provider", "Could not type cast to entity")
		}
		entities[i] = entity
	}
	return entities, nil
}

/*MultiRead - allows reading multiple entities at the same time */
func MultiRead(ctx context.Context, entityMetadata datastore.EntityMetadata, keys []datastore.Key, entities []MemoryEntity) error {
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
			instead setting key to nil
			entities[idx] = nil
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
func MultiWrite(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []MemoryEntity) error {
	if len(entities) <= BATCH_SIZE {
		return multiWriteAux(ctx, entityMetadata, entities)
	}
	for start := 0; start < len(entities); start += BATCH_SIZE {
		end := start + BATCH_SIZE
		if end > len(entities) {
			end = len(entities)
		}
		err := multiWriteAux(ctx, entityMetadata, entities[start:end])
		if err != nil {
			return err
		}
	}
	return nil
}
func multiWriteAux(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []MemoryEntity) error {
	kvpair := make([]interface{}, 2*len(entities))
	hasCollectionEntity := false
	for idx, entity := range entities {
		if !hasCollectionEntity {
			_, hasCollectionEntity = entity.(CollectionEntity)
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
		err = MultiAddToCollection(ctx, entityMetadata, entities)
	}
	return err
}
